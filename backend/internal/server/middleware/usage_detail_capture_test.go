package middleware

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestUsageDetailCaptureMiddleware_CapturesRequestAndResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var snapshotRequest *UsageDetailSnapshot
	r := gin.New()
	r.Use(UsageDetailCapture())
	r.POST("/capture", func(c *gin.Context) {
		firstRead, err := io.ReadAll(c.Request.Body)
		require.NoError(t, err)
		c.Request.Body = io.NopCloser(strings.NewReader(string(firstRead)))
		secondRead, err := io.ReadAll(c.Request.Body)
		require.NoError(t, err)
		require.Equal(t, string(firstRead), string(secondRead))

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
	require.Equal(t, `{"message":"hi"}`, snapshotRequest.RequestBody)
	require.Contains(t, snapshotRequest.ResponseHeaders, "X-Trace: abc")
	require.Equal(t, "hello world", snapshotRequest.ResponseBody)
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
	require.Equal(t, "", snapshotRequest.RequestHeaders)
	require.Equal(t, "", snapshotRequest.ResponseHeaders)
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
	require.Equal(t, "partial", snapshot.RequestBody)
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
	require.Equal(t, oversizedRequest, snapshot.RequestBody)
	require.Equal(t, oversizedResponse, snapshot.ResponseBody)
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
