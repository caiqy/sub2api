package middleware

import (
	"bytes"
	"io"
	"net/http"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type UsageDetailSnapshot = service.UsageLogDetailSnapshot

type usageDetailCollector struct {
	requestHeaders      string
	requestBody         string
	upstreamHeaders     string
	upstreamBody        string
	upstreamRespHeaders string
	upstreamRespBody    string
	responseHeaders     string
	responseBody        bytes.Buffer
	overrideHeaders     string
	overrideBody        string
	hasOverride         bool
}

type usageDetailResponseWriter struct {
	gin.ResponseWriter
	collector *usageDetailCollector
}

type replayRequestBody struct {
	closer io.Closer
	reader io.Reader
	err    error
}

func (w *usageDetailResponseWriter) Write(data []byte) (int, error) {
	if w.collector != nil && len(data) > 0 {
		w.collector.captureResponseChunk(data)
	}
	return w.ResponseWriter.Write(data)
}

func (w *usageDetailResponseWriter) WriteString(s string) (int, error) {
	if w.collector != nil && s != "" {
		w.collector.captureResponseChunk([]byte(s))
	}
	return w.ResponseWriter.WriteString(s)
}

func (w *usageDetailResponseWriter) ReadFrom(r io.Reader) (int64, error) {
	if readerFrom, ok := w.ResponseWriter.(io.ReaderFrom); ok {
		if w.collector == nil {
			return readerFrom.ReadFrom(r)
		}
		return readerFrom.ReadFrom(io.TeeReader(r, &usageDetailCaptureSink{collector: w.collector}))
	}
	return copyToResponseWriter(w, r)
}

func UsageDetailCapture() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestHeaders := ""
		if c.Request != nil {
			requestHeaders = service.FormatUsageDetailRequestHeadersText(c.Request)
		}
		collector := &usageDetailCollector{
			requestHeaders: requestHeaders,
			requestBody:    "",
		}
		if c.Request != nil && c.Request.Body != nil {
			body, replayData, err := captureRequestPrefix(c.Request.Body)
			collector.requestBody = body
			c.Request.Body = &replayRequestBody{
				closer: c.Request.Body,
				reader: bytes.NewReader(replayData),
				err:    err,
			}
		}
		c.Set(service.UsageDetailCaptureContextKey, collector)
		c.Writer = &usageDetailResponseWriter{ResponseWriter: c.Writer, collector: collector}
		c.Next()
		collector.responseHeaders = service.FormatUsageDetailResponseHeadersText(c.Writer.Status(), c.Writer.Header())
	}
}

func GetUsageDetailSnapshot(c *gin.Context) *UsageDetailSnapshot {
	return buildUsageDetailSnapshot(c)
}

func BuildUsageDetailSnapshot(c *gin.Context) *UsageDetailSnapshot {
	return buildUsageDetailSnapshot(c)
}

func buildUsageDetailSnapshot(c *gin.Context) *UsageDetailSnapshot {
	if c == nil {
		return nil
	}
	v, ok := c.Get(service.UsageDetailCaptureContextKey)
	if !ok {
		return nil
	}
	collector, ok := v.(*usageDetailCollector)
	if !ok || collector == nil {
		return nil
	}
	if collector.responseHeaders == "" {
		collector.responseHeaders = service.FormatUsageDetailResponseHeadersText(c.Writer.Status(), c.Writer.Header())
	}
	responseHeaders := collector.responseHeaders
	responseBody := collector.responseBody.String()
	if collector.hasOverride {
		responseHeaders = collector.overrideHeaders
	}
	if collector.hasOverride {
		responseBody = collector.overrideBody
	}
	return (&service.UsageLogDetailSnapshot{
		RequestHeaders:          collector.requestHeaders,
		RequestBody:             collector.requestBody,
		UpstreamRequestHeaders:  collector.upstreamHeaders,
		UpstreamRequestBody:     collector.upstreamBody,
		UpstreamResponseHeaders: collector.upstreamRespHeaders,
		UpstreamResponseBody:    collector.upstreamRespBody,
		ResponseHeaders:         responseHeaders,
		ResponseBody:            responseBody,
	}).Normalize()
}

func (r *replayRequestBody) Read(p []byte) (int, error) {
	if r == nil {
		return 0, io.EOF
	}
	if r.reader != nil {
		n, err := r.reader.Read(p)
		if err == io.EOF {
			r.reader = nil
			if r.err != nil {
				err = r.err
				r.err = nil
			}
		}
		return n, err
	}
	if r.err != nil {
		err := r.err
		r.err = nil
		return 0, err
	}
	return 0, io.EOF
}

func (r *replayRequestBody) Close() error {
	if r == nil || r.closer == nil {
		return nil
	}
	return r.closer.Close()
}

func (c *usageDetailCollector) captureResponseChunk(data []byte) {
	if c == nil || len(data) == 0 {
		return
	}
	_, _ = c.responseBody.Write(data)
}

func (c *usageDetailCollector) SetUsageUpstreamRequest(headers, body string) {
	if c == nil {
		return
	}
	c.upstreamHeaders = headers
	c.upstreamBody = body
}

func (c *usageDetailCollector) SetUsageResponseSnapshot(headers, body string) {
	if c == nil {
		return
	}
	c.overrideHeaders = headers
	c.overrideBody = body
	c.hasOverride = true
}

func (c *usageDetailCollector) SetUsageUpstreamResponse(headers, body string) {
	if c == nil {
		return
	}
	c.upstreamRespHeaders = headers
	c.upstreamRespBody = body
}

type usageDetailCaptureSink struct {
	collector *usageDetailCollector
}

func (s *usageDetailCaptureSink) Write(p []byte) (int, error) {
	if s != nil && s.collector != nil {
		s.collector.captureResponseChunk(p)
	}
	return len(p), nil
}

func copyToResponseWriter(w *usageDetailResponseWriter, r io.Reader) (int64, error) {
	buf := make([]byte, 32*1024)
	var written int64
	for {
		nr, er := r.Read(buf)
		if nr > 0 {
			nw, ew := w.Write(buf[:nr])
			written += int64(nw)
			if ew != nil {
				return written, ew
			}
			if nw != nr {
				return written, io.ErrShortWrite
			}
		}
		if er != nil {
			if er == io.EOF {
				return written, nil
			}
			return written, er
		}
	}
}

func captureRequestPrefix(body io.Reader) (captured string, replayPrefix []byte, err error) {
	payload, readErr := io.ReadAll(body)
	return string(payload), payload, readErr
}

func SetUsageUpstreamRequest(c *gin.Context, req *http.Request, body string) {
	service.SetUsageUpstreamRequest(c, req, body)
}
