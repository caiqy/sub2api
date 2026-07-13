package httputil

import (
	"bytes"
	"compress/gzip"
	"compress/zlib"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/klauspost/compress/zstd"
)

type requestBodyReadCloser struct {
	io.Reader
	close func() error
}

func (r requestBodyReadCloser) Close() error {
	return r.close()
}

const (
	requestBodyReadInitCap    = 512
	requestBodyReadMaxInitCap = 1 << 20
	jsonUTF8BOMLen            = 3
	// maxDecompressedBodySize limits the decompressed request body to 64 MB
	// to prevent decompression bomb attacks.
	maxDecompressedBodySize = 64 << 20
)

// ReadRequestBodyWithPrealloc reads request body with preallocated buffer based
// on content length, transparently decoding any Content-Encoding the upstream
// client used to compress the body (zstd, gzip, deflate).
func ReadRequestBodyWithPrealloc(req *http.Request) ([]byte, error) {
	if req == nil || req.Body == nil {
		return nil, nil
	}

	capHint := requestBodyReadInitCap
	if req.ContentLength > 0 {
		switch {
		case req.ContentLength < int64(requestBodyReadInitCap):
			capHint = requestBodyReadInitCap
		case req.ContentLength > int64(requestBodyReadMaxInitCap):
			capHint = requestBodyReadMaxInitCap
		default:
			capHint = int(req.ContentLength)
		}
	}

	encoding := strings.ToLower(strings.TrimSpace(req.Header.Get("Content-Encoding")))
	r, err := NewDecodedRequestBodyReader(req)
	if err != nil {
		return nil, fmt.Errorf("decode Content-Encoding %q: %w", encoding, err)
	}
	defer func() { _ = r.Close() }()

	buf := bytes.NewBuffer(make([]byte, 0, capHint))
	if _, err := io.Copy(buf, r); err != nil {
		if encoding != "" && encoding != "identity" {
			return nil, fmt.Errorf("decode Content-Encoding %q: %w", encoding, err)
		}
		return nil, err
	}
	decoded := buf.Bytes()
	if encoding == "" || encoding == "identity" {
		return decoded, nil
	}

	req.Header.Del("Content-Encoding")
	req.Header.Del("Content-Length")
	req.ContentLength = int64(len(decoded))

	return decoded, nil
}

// ReadLenientJSONRequestBodyWithPrealloc reads a request body and normalizes
// JSON string control bytes before strict validation.
func ReadLenientJSONRequestBodyWithPrealloc(req *http.Request, maxNormalizedBytes int64) ([]byte, error) {
	body, err := ReadRequestBodyWithPrealloc(req)
	if err != nil {
		return nil, err
	}
	return NormalizeLenientJSONRequestBody(body, maxNormalizedBytes)
}

// NewDecodedRequestBodyReader returns a reader for the decoded request body.
func NewDecodedRequestBodyReader(req *http.Request) (io.ReadCloser, error) {
	if req == nil || req.Body == nil {
		return nil, nil
	}

	switch encoding := strings.ToLower(strings.TrimSpace(req.Header.Get("Content-Encoding"))); encoding {
	case "", "identity":
		return req.Body, nil
	case "zstd":
		dec, err := zstd.NewReader(req.Body)
		if err != nil {
			return nil, err
		}
		return requestBodyReadCloser{
			Reader: http.MaxBytesReader(nil, io.NopCloser(dec), maxDecompressedBodySize),
			close: func() error {
				dec.Close()
				return req.Body.Close()
			},
		}, nil
	case "gzip", "x-gzip":
		gr, err := gzip.NewReader(req.Body)
		if err != nil {
			return nil, err
		}
		return requestBodyReadCloser{
			Reader: http.MaxBytesReader(nil, gr, maxDecompressedBodySize),
			close: func() error {
				_ = gr.Close()
				return req.Body.Close()
			},
		}, nil
	case "deflate":
		zr, err := zlib.NewReader(req.Body)
		if err != nil {
			return nil, err
		}
		return requestBodyReadCloser{
			Reader: http.MaxBytesReader(nil, zr, maxDecompressedBodySize),
			close: func() error {
				_ = zr.Close()
				return req.Body.Close()
			},
		}, nil
	default:
		return nil, errors.New("unsupported Content-Encoding")
	}
}

// NormalizeLenientJSONRequestBody escapes raw control bytes that broken
// OpenAI-compatible clients sometimes place inside JSON strings.
func NormalizeLenientJSONRequestBody(body []byte, maxNormalizedBytes int64) ([]byte, error) {
	if maxNormalizedBytes <= 0 {
		maxNormalizedBytes = maxDecompressedBodySize
	}

	body = trimUTF8BOM(body)
	if len(body) == 0 {
		return body, nil
	}
	if int64(len(body)) > maxNormalizedBytes {
		return nil, &http.MaxBytesError{Limit: maxNormalizedBytes}
	}

	var out []byte
	inString := false
	escaped := false
	for i, b := range body {
		if inString && isJSONControlByte(b) {
			if out == nil {
				capHint := len(body) + 6
				if int64(capHint) > maxNormalizedBytes {
					capHint = int(maxNormalizedBytes)
				}
				out = make([]byte, 0, capHint)
				out = append(out, body[:i]...)
			}
			if int64(len(out)+6) > maxNormalizedBytes {
				return nil, &http.MaxBytesError{Limit: maxNormalizedBytes}
			}
			out = appendJSONUnicodeEscape(out, b)
			escaped = false
			continue
		}

		switch {
		case escaped:
			escaped = false
		case inString && b == '\\':
			escaped = true
		case b == '"':
			inString = !inString
		}

		if out != nil {
			if int64(len(out)+1) > maxNormalizedBytes {
				return nil, &http.MaxBytesError{Limit: maxNormalizedBytes}
			}
			out = append(out, b)
		}
	}
	if out != nil {
		return out, nil
	}
	return body, nil
}

func trimUTF8BOM(body []byte) []byte {
	if len(body) >= jsonUTF8BOMLen && body[0] == 0xef && body[1] == 0xbb && body[2] == 0xbf {
		return body[jsonUTF8BOMLen:]
	}
	return body
}

func isJSONControlByte(b byte) bool {
	return b < 0x20 || b == 0x7f
}

func appendJSONUnicodeEscape(dst []byte, b byte) []byte {
	const hex = "0123456789abcdef"
	return append(dst, '\\', 'u', '0', '0', hex[b>>4], hex[b&0x0f])
}
