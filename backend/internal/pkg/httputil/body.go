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
