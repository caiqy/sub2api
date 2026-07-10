package httputil

import (
	"bytes"
	"compress/gzip"
	"compress/zlib"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/klauspost/compress/zstd"
)

const samplePayload = `{"model":"gpt-5.5","input":"hi","stream":false}`

func newRequestWithBody(t *testing.T, body []byte, encoding string) *http.Request {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, "/v1/responses", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	if encoding != "" {
		req.Header.Set("Content-Encoding", encoding)
	}
	req.ContentLength = int64(len(body))
	return req
}

func compressTestBody(t *testing.T, body []byte, encoding string) []byte {
	t.Helper()

	var buf bytes.Buffer
	switch encoding {
	case "gzip":
		w := gzip.NewWriter(&buf)
		if _, err := w.Write(body); err != nil {
			t.Fatalf("gzip write: %v", err)
		}
		if err := w.Close(); err != nil {
			t.Fatalf("gzip close: %v", err)
		}
	case "deflate":
		w := zlib.NewWriter(&buf)
		if _, err := w.Write(body); err != nil {
			t.Fatalf("zlib write: %v", err)
		}
		if err := w.Close(); err != nil {
			t.Fatalf("zlib close: %v", err)
		}
	case "zstd":
		w, err := zstd.NewWriter(&buf)
		if err != nil {
			t.Fatalf("zstd writer: %v", err)
		}
		if _, err := w.Write(body); err != nil {
			t.Fatalf("zstd write: %v", err)
		}
		if err := w.Close(); err != nil {
			t.Fatalf("zstd close: %v", err)
		}
	default:
		t.Fatalf("unsupported test encoding %q", encoding)
	}
	return buf.Bytes()
}

func TestReadRequestBodyWithPrealloc_PassesThroughIdentity(t *testing.T) {
	req := newRequestWithBody(t, []byte(samplePayload), "")
	got, err := ReadRequestBodyWithPrealloc(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(got) != samplePayload {
		t.Fatalf("body mismatch: got %q", got)
	}
}

func TestReadRequestBodyWithPrealloc_DecodesZstd(t *testing.T) {
	enc, _ := zstd.NewWriter(nil)
	compressed := enc.EncodeAll([]byte(samplePayload), nil)
	_ = enc.Close()

	req := newRequestWithBody(t, compressed, "zstd")
	got, err := ReadRequestBodyWithPrealloc(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(got) != samplePayload {
		t.Fatalf("body mismatch: got %q", got)
	}
	if req.Header.Get("Content-Encoding") != "" {
		t.Fatalf("Content-Encoding should be cleared after decoding")
	}
	if req.ContentLength != int64(len(samplePayload)) {
		t.Fatalf("ContentLength not updated: %d", req.ContentLength)
	}
}

func TestReadRequestBodyWithPrealloc_DecodesGzip(t *testing.T) {
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	if _, err := gw.Write([]byte(samplePayload)); err != nil {
		t.Fatalf("gzip write: %v", err)
	}
	if err := gw.Close(); err != nil {
		t.Fatalf("gzip close: %v", err)
	}

	req := newRequestWithBody(t, buf.Bytes(), "gzip")
	got, err := ReadRequestBodyWithPrealloc(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(got) != samplePayload {
		t.Fatalf("body mismatch: got %q", got)
	}
}

func TestReadRequestBodyWithPrealloc_DecodesDeflate(t *testing.T) {
	var buf bytes.Buffer
	zw := zlib.NewWriter(&buf)
	if _, err := zw.Write([]byte(samplePayload)); err != nil {
		t.Fatalf("zlib write: %v", err)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("zlib close: %v", err)
	}

	req := newRequestWithBody(t, buf.Bytes(), "deflate")
	got, err := ReadRequestBodyWithPrealloc(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(got) != samplePayload {
		t.Fatalf("body mismatch: got %q", got)
	}
}

func TestReadRequestBodyWithPrealloc_RejectsUnsupportedEncoding(t *testing.T) {
	req := newRequestWithBody(t, []byte(samplePayload), "br")
	_, err := ReadRequestBodyWithPrealloc(req)
	if err == nil {
		t.Fatal("expected error for unsupported encoding, got nil")
	}
	if !strings.Contains(err.Error(), "br") {
		t.Fatalf("error should mention encoding, got %v", err)
	}
}

func TestReadRequestBodyWithPrealloc_RejectsCorruptZstd(t *testing.T) {
	req := newRequestWithBody(t, []byte("not actually zstd"), "zstd")
	_, err := ReadRequestBodyWithPrealloc(req)
	if err == nil {
		t.Fatal("expected error for corrupt zstd body, got nil")
	}
}

func TestReadRequestBodyWithPrealloc_NilBody(t *testing.T) {
	req, err := http.NewRequest(http.MethodPost, "/v1/responses", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	got, err := ReadRequestBodyWithPrealloc(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil body, got %q", got)
	}
}

func TestReadRequestBodyWithPrealloc_RespectsIdentityEncoding(t *testing.T) {
	req := newRequestWithBody(t, []byte(samplePayload), "identity")
	got, err := ReadRequestBodyWithPrealloc(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(got) != samplePayload {
		t.Fatalf("body mismatch: got %q", got)
	}
	if req.Header.Get("Content-Encoding") != "identity" {
		t.Fatalf("Content-Encoding changed: %q", req.Header.Get("Content-Encoding"))
	}
	if req.ContentLength != int64(len(samplePayload)) {
		t.Fatalf("ContentLength changed: %d", req.ContentLength)
	}
}

func TestReadRequestBodyWithPrealloc_RejectsDecompressedBodyOverLimit(t *testing.T) {
	body := bytes.Repeat([]byte("a"), maxDecompressedBodySize+1)

	for _, encoding := range []string{"gzip", "zstd"} {
		t.Run(encoding, func(t *testing.T) {
			req := newRequestWithBody(t, compressTestBody(t, body, encoding), encoding)
			_, err := ReadRequestBodyWithPrealloc(req)

			var maxErr *http.MaxBytesError
			if !errors.As(err, &maxErr) {
				t.Fatalf("expected *http.MaxBytesError, got %v", err)
			}
			if maxErr.Limit != maxDecompressedBodySize {
				t.Fatalf("limit = %d, want %d", maxErr.Limit, maxDecompressedBodySize)
			}
		})
	}
}

func TestReadRequestBodyWithPrealloc_AllowsDecompressedBodyAtLimit(t *testing.T) {
	body := bytes.Repeat([]byte("a"), maxDecompressedBodySize)
	req := newRequestWithBody(t, compressTestBody(t, body, "gzip"), "gzip")

	got, err := ReadRequestBodyWithPrealloc(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != maxDecompressedBodySize {
		t.Fatalf("body length = %d, want %d", len(got), maxDecompressedBodySize)
	}
}

func TestReadRequestBodyWithPrealloc_DecompressedBodyLimit(t *testing.T) {
	for _, encoding := range []string{"gzip", "x-gzip", "deflate", "zstd"} {
		t.Run(encoding, func(t *testing.T) {
			for _, tc := range []struct {
				name string
				size int
			}{
				{name: "at limit", size: maxDecompressedBodySize},
				{name: "over limit", size: maxDecompressedBodySize + 1},
			} {
				t.Run(tc.name, func(t *testing.T) {
					size := tc.size
					body := bytes.Repeat([]byte("a"), size)
					req := newRequestWithBody(t, compressTestBody(t, body, strings.TrimPrefix(encoding, "x-")), encoding)

					got, err := ReadRequestBodyWithPrealloc(req)
					if size == maxDecompressedBodySize {
						if err != nil {
							t.Fatalf("ReadRequestBodyWithPrealloc: %v", err)
						}
						if len(got) != size {
							t.Fatalf("body length = %d, want %d", len(got), size)
						}
						return
					}

					var maxErr *http.MaxBytesError
					if !errors.As(err, &maxErr) {
						t.Fatalf("expected *http.MaxBytesError, got %v", err)
					}
				})
			}
		})
	}
}

func TestReadRequestBodyWithPrealloc_PreservesMetadataOnDecodeFailure(t *testing.T) {
	for _, tc := range []struct {
		name     string
		body     []byte
		encoding string
	}{
		{name: "corrupt gzip", body: []byte("not gzip"), encoding: "gzip"},
		{name: "corrupt zstd", body: []byte("not zstd"), encoding: "zstd"},
		{name: "decompressed body too large", body: compressTestBody(t, bytes.Repeat([]byte("a"), maxDecompressedBodySize+1), "gzip"), encoding: "gzip"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := newRequestWithBody(t, tc.body, tc.encoding)
			req.Header.Set("Content-Length", "123")
			wantEncoding := req.Header.Get("Content-Encoding")
			wantHeaderLength := req.Header.Get("Content-Length")
			wantContentLength := req.ContentLength

			_, err := ReadRequestBodyWithPrealloc(req)
			if err == nil {
				t.Fatal("expected decode error")
			}
			if req.Header.Get("Content-Encoding") != wantEncoding {
				t.Fatalf("Content-Encoding = %q, want %q", req.Header.Get("Content-Encoding"), wantEncoding)
			}
			if req.Header.Get("Content-Length") != wantHeaderLength {
				t.Fatalf("Content-Length = %q, want %q", req.Header.Get("Content-Length"), wantHeaderLength)
			}
			if req.ContentLength != wantContentLength {
				t.Fatalf("ContentLength = %d, want %d", req.ContentLength, wantContentLength)
			}
		})
	}
}

func TestNewDecodedRequestBodyReader(t *testing.T) {
	for _, encoding := range []string{"", "identity", "gzip", "x-gzip", "deflate", "zstd"} {
		t.Run(encoding, func(t *testing.T) {
			body := []byte(samplePayload)
			if encoding == "gzip" || encoding == "x-gzip" || encoding == "zstd" {
				body = compressTestBody(t, body, strings.TrimPrefix(encoding, "x-"))
			} else if encoding == "deflate" {
				var buf bytes.Buffer
				zw := zlib.NewWriter(&buf)
				if _, err := zw.Write(body); err != nil {
					t.Fatalf("zlib write: %v", err)
				}
				if err := zw.Close(); err != nil {
					t.Fatalf("zlib close: %v", err)
				}
				body = buf.Bytes()
			}

			r, err := NewDecodedRequestBodyReader(newRequestWithBody(t, body, encoding))
			if err != nil {
				t.Fatalf("NewDecodedRequestBodyReader: %v", err)
			}
			defer func() { _ = r.Close() }()
			got, err := io.ReadAll(r)
			if err != nil {
				t.Fatalf("ReadAll: %v", err)
			}
			if string(got) != samplePayload {
				t.Fatalf("body mismatch: got %q", got)
			}
		})
	}

	for _, tc := range []struct {
		name     string
		body     []byte
		encoding string
	}{
		{name: "unsupported", body: []byte(samplePayload), encoding: "br"},
		{name: "corrupt gzip", body: []byte("not gzip"), encoding: "gzip"},
		{name: "corrupt zstd", body: []byte("not zstd"), encoding: "zstd"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r, err := NewDecodedRequestBodyReader(newRequestWithBody(t, tc.body, tc.encoding))
			if err == nil {
				defer func() { _ = r.Close() }()
				_, err = io.ReadAll(r)
			}
			if err == nil {
				t.Fatal("expected decode error")
			}
		})
	}

	for _, size := range []int{maxDecompressedBodySize, maxDecompressedBodySize + 1} {
		t.Run("gzip limit", func(t *testing.T) {
			r, err := NewDecodedRequestBodyReader(newRequestWithBody(t, compressTestBody(t, bytes.Repeat([]byte("a"), size), "gzip"), "gzip"))
			if err != nil {
				t.Fatalf("NewDecodedRequestBodyReader: %v", err)
			}
			defer func() { _ = r.Close() }()
			_, err = io.ReadAll(r)
			var maxErr *http.MaxBytesError
			if size == maxDecompressedBodySize {
				if err != nil {
					t.Fatalf("ReadAll: %v", err)
				}
			} else if !errors.As(err, &maxErr) {
				t.Fatalf("expected *http.MaxBytesError, got %v", err)
			}
		})
	}

	t.Run("identity is not limited", func(t *testing.T) {
		r, err := NewDecodedRequestBodyReader(newRequestWithBody(t, bytes.Repeat([]byte("a"), maxDecompressedBodySize+1), "identity"))
		if err != nil {
			t.Fatalf("NewDecodedRequestBodyReader: %v", err)
		}
		defer func() { _ = r.Close() }()
		got, err := io.ReadAll(r)
		if err != nil {
			t.Fatalf("ReadAll: %v", err)
		}
		if len(got) != maxDecompressedBodySize+1 {
			t.Fatalf("body length = %d, want %d", len(got), maxDecompressedBodySize+1)
		}
	})
}
