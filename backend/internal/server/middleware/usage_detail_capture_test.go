package middleware

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestUsageDetailCapture_SetUsageUpstreamRequest_PreservesHeaderTextAndBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var snapshot *UsageDetailSnapshot
	r := gin.New()
	r.Use(UsageDetailCapture())
	r.POST("/capture", func(c *gin.Context) {
		rawBody := "  {\"raw\":true}\n"
		upstreamReq, err := http.NewRequest(http.MethodPost, "https://example.com/v1/messages", strings.NewReader(rawBody))
		require.NoError(t, err)
		upstreamReq.Header.Add("X-Multi", "a")
		upstreamReq.Header.Add("X-Multi", "b")
		upstreamReq.Header.Set("Authorization", "Bearer secret-token")

		SetUsageUpstreamRequest(c, upstreamReq, rawBody)
		snapshot = BuildUsageDetailSnapshot(c)
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodPost, "/capture", strings.NewReader(`{"message":"hi"}`))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusNoContent, w.Code)
	require.NotNil(t, snapshot)
	require.Contains(t, snapshot.UpstreamRequestHeaders, ":method: POST")
	require.Contains(t, snapshot.UpstreamRequestHeaders, ":url: https://example.com/v1/messages")
	require.Contains(t, snapshot.UpstreamRequestHeaders, "Authorization: Bearer secret-token")
	require.Contains(t, snapshot.UpstreamRequestHeaders, "X-Multi: a")
	require.Contains(t, snapshot.UpstreamRequestHeaders, "X-Multi: b")
	require.Equal(t, "request_body_preview", gjson.Get(snapshot.UpstreamRequestBody, "kind").String())
	require.Equal(t, "  {\"raw\":true}\n", gjson.Get(snapshot.UpstreamRequestBody, "preview").String())
	require.False(t, gjson.Get(snapshot.UpstreamRequestBody, "truncated").Bool())
	require.Equal(t, int64(len("  {\"raw\":true}\n")), gjson.Get(snapshot.UpstreamRequestBody, "size").Int())
}

func TestUsageDetailCapture_SetUsageUpstreamRequest_WrapperSizeAndTruncation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tests := []struct {
		name      string
		body      string
		size      int64
		truncated bool
		omitted   bool
	}{
		{name: "small", body: `{"input":"hello"}`, size: int64(len(`{"input":"hello"}`))},
		{name: "large", body: `{"input":"preview"}`, size: 10 << 20, truncated: true},
		{name: "inline binary", body: `{"image_url":"data:image/png;base64,c2VjcmV0"}`, size: int64(len(`{"image_url":"data:image/png;base64,c2VjcmV0"}`)), truncated: true, omitted: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var snapshot *UsageDetailSnapshot
			r := gin.New()
			r.Use(UsageDetailCapture())
			r.POST("/capture", func(c *gin.Context) {
				upstreamReq, err := http.NewRequest(http.MethodPost, "https://example.com/v1/responses", strings.NewReader(tt.body))
				require.NoError(t, err)
				upstreamReq.ContentLength = tt.size
				SetUsageUpstreamRequest(c, upstreamReq, tt.body)
				snapshot = BuildUsageDetailSnapshot(c)
				c.Status(http.StatusNoContent)
			})

			r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/capture", nil))
			require.NotNil(t, snapshot)
			require.Equal(t, "request_body_preview", gjson.Get(snapshot.UpstreamRequestBody, "kind").String())
			require.Equal(t, tt.size, gjson.Get(snapshot.UpstreamRequestBody, "size").Int())
			require.Equal(t, tt.truncated, gjson.Get(snapshot.UpstreamRequestBody, "truncated").Bool())
			if tt.omitted {
				require.Contains(t, gjson.Get(snapshot.UpstreamRequestBody, "preview").String(), "omitted")
				require.NotContains(t, snapshot.UpstreamRequestBody, "c2VjcmV0")
			}
		})
	}
}

func TestUsageDetailCaptureMiddleware_DownstreamStillReadsFullBodyWithoutPreread(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var (
		downstream string
		snapshot   *UsageDetailSnapshot
	)

	r := gin.New()
	r.Use(UsageDetailCapture())
	r.POST("/capture", func(c *gin.Context) {
		raw, err := io.ReadAll(c.Request.Body)
		require.NoError(t, err)
		downstream = string(raw)
		snapshot = BuildUsageDetailSnapshot(c)
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodPost, "/capture", strings.NewReader(`{"message":"hi"}`))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, `{"message":"hi"}`, downstream)
	require.NotNil(t, snapshot)
	require.Equal(t, "", snapshot.RequestBody)
}

func TestSetUsageUpstreamRequest_DoesNotFallbackToGetBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var snapshot *UsageDetailSnapshot
	r := gin.New()
	r.Use(UsageDetailCapture())
	r.POST("/capture", func(c *gin.Context) {
		req, err := http.NewRequest(http.MethodPost, "https://example.com/v1/responses", strings.NewReader("ignored"))
		require.NoError(t, err)
		called := 0
		req.GetBody = func() (io.ReadCloser, error) {
			called++
			return io.NopCloser(strings.NewReader("should-not-be-read")), nil
		}

		service.SetUsageUpstreamRequest(c, req, "")
		snapshot = BuildUsageDetailSnapshot(c)
		require.Equal(t, 0, called)
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodPost, "/capture", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.NotNil(t, snapshot)
	require.Equal(t, "request_body_preview", gjson.Get(snapshot.UpstreamRequestBody, "kind").String())
	require.Equal(t, "", gjson.Get(snapshot.UpstreamRequestBody, "preview").String())
	require.True(t, gjson.Get(snapshot.UpstreamRequestBody, "truncated").Bool())
	require.Equal(t, int64(len("ignored")), gjson.Get(snapshot.UpstreamRequestBody, "size").Int())
}

func TestUsageDetailCaptureMiddleware_CapturesRequestAndResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var snapshotRequest *UsageDetailSnapshot
	var downstreamRequestBody string
	r := gin.New()
	r.Use(UsageDetailCapture())
	r.POST("/capture", func(c *gin.Context) {
		raw, err := io.ReadAll(c.Request.Body)
		require.NoError(t, err)
		downstreamRequestBody = string(raw)
		service.SetUsageRequestBody(c, "request-preview")

		c.Header("X-Trace", "abc")
		_, err = c.Writer.Write([]byte("hello "))
		require.NoError(t, err)
		_, err = c.Writer.Write([]byte("world"))
		require.NoError(t, err)

		snapshotRequest = BuildUsageDetailSnapshot(c)
	})

	req := httptest.NewRequest(http.MethodPost, "/capture", strings.NewReader(`{"message":"hi"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test", "1")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.NotNil(t, snapshotRequest)
	require.Contains(t, snapshotRequest.RequestHeaders, "Content-Type: application/json")
	require.Contains(t, snapshotRequest.RequestHeaders, "X-Test: 1")
	require.Equal(t, `{"message":"hi"}`, downstreamRequestBody)
	require.Equal(t, "request-preview", snapshotRequest.RequestBody)
	require.Contains(t, snapshotRequest.ResponseHeaders, "X-Trace: abc")
	require.Equal(t, "hello world", snapshotRequest.ResponseBody)
}

func TestUsageDetailCaptureMiddleware_RequestHeadersIncludeMethodAndAbsoluteURL(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var snapshot *UsageDetailSnapshot
	r := gin.New()
	r.Use(UsageDetailCapture())
	r.POST("/capture", func(c *gin.Context) {
		snapshot = BuildUsageDetailSnapshot(c)
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodPost, "https://api.example.com/capture?debug=1", strings.NewReader(`{"message":"hi"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusNoContent, w.Code)
	require.NotNil(t, snapshot)
	require.Contains(t, snapshot.RequestHeaders, ":method: POST")
	require.Contains(t, snapshot.RequestHeaders, ":url: https://api.example.com/capture?debug=1")
	require.Contains(t, snapshot.RequestHeaders, "Content-Type: application/json")
}

func TestUsageDetailCaptureMiddleware_RequestHeadersUseFirstForwardedValues(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var snapshot *UsageDetailSnapshot
	r := gin.New()
	r.Use(UsageDetailCapture())
	r.POST("/capture", func(c *gin.Context) {
		snapshot = BuildUsageDetailSnapshot(c)
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodPost, "/capture?debug=1", strings.NewReader(`{"message":"hi"}`))
	req.Host = "origin.example.com"
	req.Header.Set("X-Forwarded-Proto", " , https, http")
	req.Header.Set("X-Forwarded-Host", " , api.example.com, fallback.example.com")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusNoContent, w.Code)
	require.NotNil(t, snapshot)
	require.Contains(t, snapshot.RequestHeaders, ":url: https://api.example.com/capture?debug=1")
}

func TestUsageDetailCaptureMiddleware_RequestHeadersStillIncludeMetaWhenHeadersEmpty(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var snapshot *UsageDetailSnapshot
	r := gin.New()
	r.Use(UsageDetailCapture())
	r.GET("/empty", func(c *gin.Context) {
		snapshot = BuildUsageDetailSnapshot(c)
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "https://api.example.com/empty", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusNoContent, w.Code)
	require.NotNil(t, snapshot)
	require.Equal(t, ":method: GET\n:url: https://api.example.com/empty\n", snapshot.RequestHeaders)
}

func TestUsageDetailCaptureMiddleware_HandlesEmptyBodyAndHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var snapshotRequest *UsageDetailSnapshot
	r := gin.New()
	r.Use(UsageDetailCapture())
	r.GET("/empty", func(c *gin.Context) {
		snapshotRequest = GetUsageDetailSnapshot(c)
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/empty", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusNoContent, w.Code)
	require.NotNil(t, snapshotRequest)
	require.Equal(t, "", snapshotRequest.RequestBody)
	require.Equal(t, "", snapshotRequest.ResponseBody)
	require.Equal(t, ":method: GET\n:url: http://example.com/empty\n", snapshotRequest.RequestHeaders)
	require.Equal(t, ":status: 200\n", snapshotRequest.ResponseHeaders)
}

func TestUsageDetailCaptureMiddleware_RestoresPartialBodyAndErrorToDownstream(t *testing.T) {
	gin.SetMode(gin.TestMode)

	expectedErr := errors.New("boom")
	var (
		downstreamBody []byte
		downstreamErr  error
		snapshot       *UsageDetailSnapshot
	)

	r := gin.New()
	r.Use(UsageDetailCapture())
	r.POST("/capture", func(c *gin.Context) {
		downstreamBody, downstreamErr = io.ReadAll(c.Request.Body)
		snapshot = BuildUsageDetailSnapshot(c)
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodPost, "/capture", nil)
	req.Body = &failingReadCloser{
		chunks: [][]byte{[]byte("par"), []byte("tial")},
		err:    expectedErr,
	}
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusNoContent, w.Code)
	require.ErrorIs(t, downstreamErr, expectedErr)
	require.Equal(t, []byte("partial"), downstreamBody)
	require.NotNil(t, snapshot)
	require.Equal(t, "", snapshot.RequestBody)
}

func TestUsageDetailCaptureMiddleware_CapturesResponseViaReadFromPath(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var snapshot *UsageDetailSnapshot
	r := gin.New()
	r.Use(UsageDetailCapture())
	r.GET("/copy", func(c *gin.Context) {
		rf, ok := c.Writer.(io.ReaderFrom)
		require.True(t, ok)

		src := &plainReader{data: []byte("copied response body")}
		written, err := rf.ReadFrom(src)
		require.NoError(t, err)
		require.Equal(t, int64(len("copied response body")), written)

		snapshot = BuildUsageDetailSnapshot(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/copy", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.NotNil(t, snapshot)
	require.Equal(t, "copied response body", snapshot.ResponseBody)
}

func TestUsageDetailCaptureMiddleware_CapturesFullRequestAndResponseBodies(t *testing.T) {
	gin.SetMode(gin.TestMode)

	oversizedRequest := strings.Repeat("r", 64*1024+16)
	oversizedResponse := strings.Repeat("s", 64*1024+32)
	var (
		downstreamRequestBody string
		snapshot              *UsageDetailSnapshot
	)

	r := gin.New()
	r.Use(UsageDetailCapture())
	r.POST("/truncate", func(c *gin.Context) {
		requestBody, err := io.ReadAll(c.Request.Body)
		require.NoError(t, err)
		downstreamRequestBody = string(requestBody)

		_, err = c.Writer.Write([]byte(oversizedResponse))
		require.NoError(t, err)

		snapshot = BuildUsageDetailSnapshot(c)
	})

	req := httptest.NewRequest(http.MethodPost, "/truncate", strings.NewReader(oversizedRequest))
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.NotNil(t, snapshot)
	require.Equal(t, oversizedRequest, downstreamRequestBody)
	require.Equal(t, "", snapshot.RequestBody)
	require.Equal(t, oversizedResponse, snapshot.ResponseBody)
}

func TestUsageDetailCapture_SetUsageResponseSnapshot_AllowsExplicitEmptyOverride(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var snapshot *UsageDetailSnapshot
	r := gin.New()
	r.Use(UsageDetailCapture())
	r.GET("/empty-override", func(c *gin.Context) {
		_, err := c.Writer.Write([]byte("local fallback body"))
		require.NoError(t, err)
		service.SetUsageResponseSnapshot(c, "", "")
		snapshot = BuildUsageDetailSnapshot(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/empty-override", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.NotNil(t, snapshot)
	require.Equal(t, "", snapshot.ResponseHeaders)
	require.Equal(t, "", snapshot.ResponseBody)
}

type failingReadCloser struct {
	chunks [][]byte
	err    error
	index  int
	closed bool
}

func (r *failingReadCloser) Read(p []byte) (int, error) {
	if r.index < len(r.chunks) {
		n := copy(p, r.chunks[r.index])
		r.index++
		return n, nil
	}
	if r.err != nil {
		err := r.err
		r.err = nil
		return 0, err
	}
	return 0, io.EOF
}

func (r *failingReadCloser) Close() error {
	r.closed = true
	return nil
}

type plainReader struct {
	data []byte
	off  int
}

func (r *plainReader) Read(p []byte) (int, error) {
	if r.off >= len(r.data) {
		return 0, io.EOF
	}
	n := copy(p, r.data[r.off:])
	r.off += n
	return n, nil
}
