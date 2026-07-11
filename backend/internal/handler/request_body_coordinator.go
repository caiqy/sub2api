package handler

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"mime/multipart"
	"net/http"

	"github.com/Wei-Shaw/sub2api/internal/pkg/httputil"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

// ponytail: package-local seam only; production uses the existing responses body settings.
var jsonRequestBodyHandleOptions = openAIResponsesRequestBodyHandleOptions()

const multipartUploadPartLimit = 20 << 20

type requestBodyCoordinator struct {
	raw       *service.RequestBodyHandle
	effective *service.RequestBodyHandle
	form      *multipart.Form
}

func newJSONRequestBody(req *http.Request) (*requestBodyCoordinator, error) {
	reader, err := httputil.NewDecodedRequestBodyReader(req)
	if err != nil {
		return nil, err
	}
	if reader != nil {
		defer func() { _ = reader.Close() }()
	}

	raw, err := service.NewRequestBodyHandleFromReader(reader, jsonRequestBodyHandleOptions)
	if err != nil {
		return nil, err
	}
	if req != nil {
		req.Header.Del("Content-Encoding")
		req.Header.Del("Content-Length")
		req.ContentLength = raw.Size()
	}
	return &requestBodyCoordinator{raw: raw, effective: raw}, nil
}

func newMultipartRequestBody(req *http.Request) (*requestBodyCoordinator, error) {
	if req == nil || req.Body == nil {
		return nil, errors.New("multipart request body is required")
	}
	raw, err := service.NewRequestBodyHandleFromReader(req.Body, jsonRequestBodyHandleOptions)
	if err != nil {
		return nil, err
	}
	reader, err := raw.Open()
	if err != nil {
		service.CleanupRequestBodyHandle(raw)
		return nil, err
	}

	parsed := req.Clone(req.Context())
	parsed.Body = reader
	parsed.ContentLength = raw.Size()
	if err := parsed.ParseMultipartForm(0); err != nil {
		_ = reader.Close()
		service.CleanupRequestBodyHandle(raw)
		return nil, err
	}
	_ = reader.Close()
	for _, files := range parsed.MultipartForm.File {
		for _, file := range files {
			if file.Size > multipartUploadPartLimit {
				_ = parsed.MultipartForm.RemoveAll()
				service.CleanupRequestBodyHandle(raw)
				return nil, &http.MaxBytesError{Limit: multipartUploadPartLimit}
			}
		}
	}
	return &requestBodyCoordinator{raw: raw, effective: raw, form: parsed.MultipartForm}, nil
}

func (c *requestBodyCoordinator) ReadRaw() ([]byte, error) {
	if c == nil {
		return nil, errors.New("request body coordinator is nil")
	}
	return c.raw.ReadAll()
}

func (c *requestBodyCoordinator) SetEffectiveBytes(body []byte) error {
	if c.raw != nil && c.raw.Size() == int64(len(body)) {
		sum := sha256.Sum256(body)
		if c.raw.Hash() == hex.EncodeToString(sum[:]) {
			c.setEffective(c.raw)
			return nil
		}
	}
	handle, err := service.NewRequestBodyHandleFromBytes(body, jsonRequestBodyHandleOptions)
	if err != nil {
		return err
	}
	c.setEffective(handle)
	return nil
}

func (c *requestBodyCoordinator) SetEffectiveReader(reader io.Reader) error {
	handle, err := service.NewRequestBodyHandleFromReader(reader, jsonRequestBodyHandleOptions)
	if err != nil {
		return err
	}
	c.setEffective(handle)
	return nil
}

func (c *requestBodyCoordinator) setEffective(handle *service.RequestBodyHandle) {
	if handle == c.raw {
		if c.effective != nil && c.effective != c.raw {
			service.CleanupRequestBodyHandle(c.effective)
		}
		c.effective = c.raw
		return
	}
	if c.raw != nil && c.raw.Size() == handle.Size() && c.raw.Hash() == handle.Hash() {
		service.CleanupRequestBodyHandle(handle)
		handle = c.raw
	}
	if c.effective != nil && c.effective != c.raw && c.effective != handle {
		service.CleanupRequestBodyHandle(c.effective)
	}
	c.effective = handle
}

func (c *requestBodyCoordinator) Effective() *service.RequestBodyHandle {
	if c == nil {
		return nil
	}
	return c.effective
}

func uniqueRequestBodyHandles(handles ...*service.RequestBodyHandle) []*service.RequestBodyHandle {
	unique := make([]*service.RequestBodyHandle, 0, len(handles))
	for _, handle := range handles {
		if handle == nil {
			continue
		}
		alreadyIncluded := false
		for _, previous := range unique {
			if handle == previous {
				alreadyIncluded = true
				break
			}
		}
		if !alreadyIncluded {
			unique = append(unique, handle)
		}
	}
	return unique
}

func (c *requestBodyCoordinator) Cleanup() {
	if c == nil {
		return
	}
	if c.form != nil {
		_ = c.form.RemoveAll()
	}
	for _, handle := range uniqueRequestBodyHandles(c.raw, c.effective) {
		service.CleanupRequestBodyHandle(handle)
	}
}
