package handler

import (
	"errors"
	"io"
	"mime/multipart"
	"net/http"

	"github.com/Wei-Shaw/sub2api/internal/pkg/httputil"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

// ponytail: package-local seam only; production uses the existing responses body settings.
var jsonRequestBodyHandleOptions = openAIResponsesRequestBodyHandleOptions()

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

func (c *requestBodyCoordinator) ReadRaw() ([]byte, error) {
	if c == nil {
		return nil, errors.New("request body coordinator is nil")
	}
	return c.raw.ReadAll()
}

func (c *requestBodyCoordinator) SetEffectiveBytes(body []byte) error {
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

func (c *requestBodyCoordinator) Cleanup() {
	if c == nil {
		return
	}
	if c.form != nil {
		_ = c.form.RemoveAll()
	}
	service.CleanupRequestBodyHandle(c.raw)
	if c.effective != c.raw {
		service.CleanupRequestBodyHandle(c.effective)
	}
}
