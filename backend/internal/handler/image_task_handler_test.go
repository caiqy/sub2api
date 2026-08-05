package handler

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type asyncImageMemoryStore struct {
	mu    sync.RWMutex
	tasks map[string]*service.ImageTaskRecord
}

func (s *asyncImageMemoryStore) Save(_ context.Context, task *service.ImageTaskRecord, _ time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	copy := *task
	copy.Result = append(json.RawMessage(nil), task.Result...)
	copy.Error = append(json.RawMessage(nil), task.Error...)
	s.tasks[task.ID] = &copy
	return nil
}

func (s *asyncImageMemoryStore) Get(_ context.Context, id string) (*service.ImageTaskRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	task := s.tasks[id]
	if task == nil {
		return nil, service.ErrImageTaskNotFound
	}
	copy := *task
	copy.Result = append(json.RawMessage(nil), task.Result...)
	copy.Error = append(json.RawMessage(nil), task.Error...)
	return &copy, nil
}

func useAsyncImageSpoolDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	old := jsonRequestBodyHandleOptions
	jsonRequestBodyHandleOptions = service.RequestBodyHandleOptions{
		SpoolThresholdBytes: service.DefaultRequestBodySpoolThresholdBytes,
		PreviewLimitBytes:   service.DefaultRequestBodyPreviewLimitBytes,
		TempDir:             dir,
	}
	t.Cleanup(func() { jsonRequestBodyHandleOptions = old })
	return dir
}

func TestAsyncImageHandlerSpoolsReplaysAndCleansOwnedBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	spoolDir := useAsyncImageSpoolDir(t)
	store := &asyncImageMemoryStore{tasks: make(map[string]*service.ImageTaskRecord)}
	tasks := service.NewImageTaskServiceWithUploader(store, nil, time.Hour, time.Minute)
	release := make(chan struct{})
	gotBody := make(chan struct {
		bodyHash      [32]byte
		replayHash    [32]byte
		contentType   string
		contentLength int64
		err           error
	}, 1)
	h := &AsyncImageHandler{tasks: tasks}
	h.execute = func(_ string, c *gin.Context) {
		body, err := io.ReadAll(c.Request.Body)
		if err == nil {
			err = c.Request.Body.Close()
		}
		if err != nil {
			gotBody <- struct {
				bodyHash      [32]byte
				replayHash    [32]byte
				contentType   string
				contentLength int64
				err           error
			}{err: err}
			return
		}
		replay, err := c.Request.GetBody()
		if err != nil {
			gotBody <- struct {
				bodyHash      [32]byte
				replayHash    [32]byte
				contentType   string
				contentLength int64
				err           error
			}{err: err}
			return
		}
		replayed, err := io.ReadAll(replay)
		if closeErr := replay.Close(); err == nil {
			err = closeErr
		}
		gotBody <- struct {
			bodyHash      [32]byte
			replayHash    [32]byte
			contentType   string
			contentLength int64
			err           error
		}{
			bodyHash:      sha256.Sum256(body),
			replayHash:    sha256.Sum256(replayed),
			contentType:   c.Request.Header.Get("Content-Type"),
			contentLength: c.Request.ContentLength,
			err:           err,
		}
		<-release
		c.JSON(http.StatusOK, gin.H{"data": []gin.H{{"url": "https://example.test/image.png"}}})
	}

	router := gin.New()
	router.Use(func(c *gin.Context) {
		groupID := int64(3)
		c.Set(string(middleware2.ContextKeyAPIKey), &service.APIKey{
			ID:      9,
			UserID:  7,
			GroupID: &groupID,
			Group:   &service.Group{ID: groupID, Platform: service.PlatformOpenAI, AllowImageGeneration: true},
		})
		c.Next()
	})
	router.POST("/v1/images/generations/async", h.Submit)

	body := []byte(`{"model":"gpt-image-1","prompt":"` + strings.Repeat("x", int(service.DefaultRequestBodySpoolThresholdBytes)) + `"}`)
	wantHash := sha256.Sum256(body)
	request := httptest.NewRequest(http.MethodPost, "/v1/images/generations/async", strings.NewReader(string(body)))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusAccepted, recorder.Code)
	select {
	case got := <-gotBody:
		require.NoError(t, got.err)
		require.Equal(t, wantHash, got.bodyHash)
		require.Equal(t, wantHash, got.replayHash)
		require.Equal(t, "application/json", got.contentType)
		require.Equal(t, int64(len(body)), got.contentLength)
	case <-time.After(time.Second):
		t.Fatal("async image task did not read its body")
	}
	assertMatrixTempFiles(t, spoolDir, "sub2api-request-body-", true)
	close(release)
	require.Eventually(t, func() bool {
		entries, err := os.ReadDir(spoolDir)
		return err == nil && len(entries) == 0
	}, time.Second, 10*time.Millisecond)
}

func TestAsyncImageHandlerSpoolCreateFailureReturns503WithoutTask(t *testing.T) {
	gin.SetMode(gin.TestMode)
	old := jsonRequestBodyHandleOptions
	jsonRequestBodyHandleOptions = service.RequestBodyHandleOptions{
		SpoolThresholdBytes: service.DefaultRequestBodySpoolThresholdBytes,
		PreviewLimitBytes:   service.DefaultRequestBodyPreviewLimitBytes,
		TempDir:             filepath.Join(t.TempDir(), "missing"),
	}
	t.Cleanup(func() { jsonRequestBodyHandleOptions = old })
	store := &asyncImageMemoryStore{tasks: make(map[string]*service.ImageTaskRecord)}
	tasks := service.NewImageTaskServiceWithUploader(store, nil, time.Hour, time.Minute)
	h := &AsyncImageHandler{tasks: tasks, execute: func(_ string, _ *gin.Context) { t.Error("image task must not execute") }}

	router := gin.New()
	router.Use(func(c *gin.Context) {
		groupID := int64(3)
		c.Set(string(middleware2.ContextKeyAPIKey), &service.APIKey{
			ID:      9,
			UserID:  7,
			GroupID: &groupID,
			Group:   &service.Group{ID: groupID, Platform: service.PlatformOpenAI, AllowImageGeneration: true},
		})
		c.Next()
	})
	router.POST("/v1/images/generations/async", h.Submit)

	body := `{"model":"gpt-image-1","prompt":"` + strings.Repeat("x", int(service.DefaultRequestBodySpoolThresholdBytes)) + `"}`
	request := httptest.NewRequest(http.MethodPost, "/v1/images/generations/async", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusServiceUnavailable, recorder.Code)
	require.Contains(t, recorder.Body.String(), "Failed to spool request body")
	require.Empty(t, store.tasks)
}

func TestAsyncImageHandlerOwnedBodyCleanupOnTerminalPaths(t *testing.T) {
	for _, tt := range []struct {
		name    string
		execute func(*gin.Context)
	}{
		{"success", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"data": []gin.H{{"url": "https://example.test/image.png"}}})
		}},
		{"failure", func(c *gin.Context) { c.JSON(http.StatusBadGateway, gin.H{"error": gin.H{"message": "failed"}}) }},
		{"panic", func(*gin.Context) { panic("image task panic") }},
	} {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			spoolDir := useAsyncImageSpoolDir(t)
			store := &asyncImageMemoryStore{tasks: make(map[string]*service.ImageTaskRecord)}
			tasks := service.NewImageTaskServiceWithUploader(store, nil, time.Hour, time.Minute)
			h := &AsyncImageHandler{tasks: tasks, execute: func(_ string, c *gin.Context) { tt.execute(c) }}

			router := gin.New()
			router.Use(func(c *gin.Context) {
				groupID := int64(3)
				c.Set(string(middleware2.ContextKeyAPIKey), &service.APIKey{
					ID:      9,
					UserID:  7,
					GroupID: &groupID,
					Group:   &service.Group{ID: groupID, Platform: service.PlatformOpenAI, AllowImageGeneration: true},
				})
				c.Next()
			})
			router.POST("/v1/images/generations/async", h.Submit)

			body := `{"model":"gpt-image-1","prompt":"` + strings.Repeat("x", int(service.DefaultRequestBodySpoolThresholdBytes)) + `"}`
			request := httptest.NewRequest(http.MethodPost, "/v1/images/generations/async", strings.NewReader(body))
			request.Header.Set("Content-Type", "application/json")
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, request)
			require.Equal(t, http.StatusAccepted, recorder.Code)

			var accepted struct {
				TaskID string `json:"task_id"`
			}
			require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &accepted))
			require.Eventually(t, func() bool {
				task, err := tasks.Get(context.Background(), service.ImageTaskOwner{UserID: 7, APIKeyID: 9}, accepted.TaskID)
				return err == nil && task.Status != service.ImageTaskStatusProcessing
			}, time.Second, 10*time.Millisecond)
			require.Eventually(t, func() bool {
				entries, err := os.ReadDir(spoolDir)
				return err == nil && len(entries) == 0
			}, time.Second, 10*time.Millisecond)
		})
	}
}

func TestAsyncImageHandlerRunRejectedCleansOwnedBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	spoolDir := useAsyncImageSpoolDir(t)
	store := &asyncImageMemoryStore{tasks: make(map[string]*service.ImageTaskRecord)}
	tasks := service.NewImageTaskServiceWithUploader(store, nil, time.Hour, time.Minute)
	require.NoError(t, tasks.Shutdown(context.Background()))
	h := &AsyncImageHandler{tasks: tasks, execute: func(_ string, _ *gin.Context) { t.Error("rejected task must not execute") }}

	router := gin.New()
	router.Use(func(c *gin.Context) {
		groupID := int64(3)
		c.Set(string(middleware2.ContextKeyAPIKey), &service.APIKey{
			ID:      9,
			UserID:  7,
			GroupID: &groupID,
			Group:   &service.Group{ID: groupID, Platform: service.PlatformOpenAI, AllowImageGeneration: true},
		})
		c.Next()
	})
	router.POST("/v1/images/generations/async", h.Submit)

	body := `{"model":"gpt-image-1","prompt":"` + strings.Repeat("x", int(service.DefaultRequestBodySpoolThresholdBytes)) + `"}`
	request := httptest.NewRequest(http.MethodPost, "/v1/images/generations/async", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	require.Equal(t, http.StatusAccepted, recorder.Code)

	var accepted struct {
		TaskID string `json:"task_id"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &accepted))
	task, err := tasks.Get(context.Background(), service.ImageTaskOwner{UserID: 7, APIKeyID: 9}, accepted.TaskID)
	require.NoError(t, err)
	require.Equal(t, service.ImageTaskStatusFailed, task.Status)
	require.Eventually(t, func() bool {
		entries, err := os.ReadDir(spoolDir)
		return err == nil && len(entries) == 0
	}, time.Second, 10*time.Millisecond)
}

func TestAsyncImageHandlerRunWithBodyHandleSpoolOpenFailureFailsTask(t *testing.T) {
	gin.SetMode(gin.TestMode)
	spoolDir := t.TempDir()
	handle, err := service.NewRequestBodyHandleFromBytes(
		[]byte(`{"model":"gpt-image-1","prompt":"cat"}`),
		service.RequestBodyHandleOptions{SpoolThresholdBytes: 1, TempDir: spoolDir},
	)
	require.NoError(t, err)
	t.Cleanup(func() { service.CleanupRequestBodyHandle(handle) })
	store := &asyncImageMemoryStore{tasks: make(map[string]*service.ImageTaskRecord)}
	tasks := service.NewImageTaskServiceWithUploader(store, nil, time.Hour, time.Minute)
	h := &AsyncImageHandler{tasks: tasks, execute: func(_ string, _ *gin.Context) { t.Error("spool-open failure must not execute") }}
	task, err := tasks.Create(context.Background(), service.ImageTaskOwner{UserID: 7, APIKeyID: 9})
	require.NoError(t, err)

	entries, err := os.ReadDir(spoolDir)
	require.NoError(t, err)
	removed := false
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), "sub2api-request-body-") {
			require.NoError(t, os.Remove(filepath.Join(spoolDir, entry.Name())))
			removed = true
			break
		}
	}
	require.True(t, removed, "spooled request body file must exist")

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/generations/async", nil)
	h.runWithBodyHandle(task.ID, service.PlatformOpenAI, c, handle, context.Background())

	got, err := tasks.Get(context.Background(), service.ImageTaskOwner{UserID: 7, APIKeyID: 9}, task.ID)
	require.NoError(t, err)
	require.Equal(t, service.ImageTaskStatusFailed, got.Status)
	require.Equal(t, http.StatusServiceUnavailable, got.HTTPStatus)
	require.Eventually(t, func() bool {
		entries, err := os.ReadDir(spoolDir)
		return err == nil && len(entries) == 0
	}, time.Second, 10*time.Millisecond)
}

func TestAsyncImageHandlerSubmitAndPoll(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := &asyncImageMemoryStore{tasks: make(map[string]*service.ImageTaskRecord)}
	tasks := service.NewImageTaskServiceWithUploader(store, nil, time.Hour, time.Minute)
	release := make(chan struct{})
	h := &AsyncImageHandler{tasks: tasks}
	h.execute = func(_ string, c *gin.Context) {
		<-release
		c.JSON(http.StatusOK, gin.H{"created": 123, "data": []gin.H{{"url": "https://example.test/image.png"}}})
	}

	router := gin.New()
	router.Use(func(c *gin.Context) {
		groupID := int64(3)
		c.Set(string(middleware2.ContextKeyAPIKey), &service.APIKey{
			ID:      9,
			UserID:  7,
			GroupID: &groupID,
			Group:   &service.Group{ID: groupID, Platform: service.PlatformOpenAI, AllowImageGeneration: true},
		})
		c.Next()
	})
	router.POST("/v1/images/generations/async", h.Submit)
	router.GET("/v1/images/tasks/:task_id", h.Get)

	requestCtx, cancelRequest := context.WithCancel(context.Background())
	req := httptest.NewRequest(http.MethodPost, "/v1/images/generations/async", strings.NewReader(`{"model":"gpt-image-1","prompt":"cat"}`)).WithContext(requestCtx)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusAccepted, w.Code)
	require.Equal(t, "no-store", w.Header().Get("Cache-Control"))
	require.Equal(t, "3", w.Header().Get("Retry-After"))

	var accepted struct {
		TaskID  string `json:"task_id"`
		Status  string `json:"status"`
		PollURL string `json:"poll_url"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &accepted))
	require.Equal(t, service.ImageTaskStatusProcessing, accepted.Status)
	require.Equal(t, "/v1/images/tasks/"+accepted.TaskID, accepted.PollURL)
	require.Equal(t, accepted.PollURL, w.Header().Get("Location"))

	// The detached background request must survive completion of/cancellation
	// from the short submission request.
	cancelRequest()
	close(release)
	require.Eventually(t, func() bool {
		got, err := tasks.Get(context.Background(), service.ImageTaskOwner{UserID: 7, APIKeyID: 9}, accepted.TaskID)
		return err == nil && got.Status == service.ImageTaskStatusCompleted
	}, time.Second, 10*time.Millisecond)

	pollReq := httptest.NewRequest(http.MethodGet, accepted.PollURL, nil)
	pollWriter := httptest.NewRecorder()
	router.ServeHTTP(pollWriter, pollReq)
	require.Equal(t, http.StatusOK, pollWriter.Code)
	require.Equal(t, "no-store", pollWriter.Header().Get("Cache-Control"))
	require.Empty(t, pollWriter.Header().Get("Retry-After"))
	require.Contains(t, pollWriter.Body.String(), "https://example.test/image.png")
}

func TestAsyncImageHandlerSubmitUsesCompositeTargetAndCopiesDecision(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := &asyncImageMemoryStore{tasks: make(map[string]*service.ImageTaskRecord)}
	tasks := service.NewImageTaskServiceWithUploader(store, nil, time.Hour, time.Minute)
	executed := make(chan struct {
		platform string
		decision service.CompositeRouteDecision
	}, 1)
	h := &AsyncImageHandler{tasks: tasks}
	h.execute = func(platform string, c *gin.Context) {
		decision, ok := service.CompositeRouteDecisionFromContext(c.Request.Context())
		require.True(t, ok)
		executed <- struct {
			platform string
			decision service.CompositeRouteDecision
		}{platform: platform, decision: decision}
		c.JSON(http.StatusOK, gin.H{"data": []gin.H{{"url": "https://example.test/image.png"}}})
	}

	groupID := int64(31)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(string(middleware2.ContextKeyAPIKey), &service.APIKey{
			ID:      9,
			UserID:  7,
			GroupID: &groupID,
			Group:   &service.Group{ID: groupID, Platform: service.PlatformComposite, AllowImageGeneration: true},
		})
		c.Next()
	})
	router.POST("/v1/images/generations/async", h.Submit)

	ctx := service.WithCompositeRouteDecision(context.Background(), service.CompositeRouteDecision{
		Matched:        true,
		GroupID:        groupID,
		PublicModel:    "image-alias",
		TargetPlatform: service.PlatformOpenAI,
		UpstreamModel:  "gpt-image-1",
		Endpoint:       service.CompositeRouteEndpointImages,
	})
	req := httptest.NewRequest(http.MethodPost, "/v1/images/generations/async", strings.NewReader(`{"model":"image-alias","prompt":"cat"}`)).WithContext(ctx)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusAccepted, w.Code, w.Body.String())
	select {
	case got := <-executed:
		require.Equal(t, service.PlatformOpenAI, got.platform)
		require.Equal(t, groupID, got.decision.GroupID)
		require.Equal(t, "gpt-image-1", got.decision.UpstreamModel)
	case <-time.After(time.Second):
		t.Fatal("async image task did not execute")
	}
}

// When object storage is not configured the feature is fully disabled: the
// endpoints must return 404 without creating a task or writing to Redis.
func TestAsyncImageHandlerDisabledReturns404(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := &asyncImageMemoryStore{tasks: make(map[string]*service.ImageTaskRecord)}
	tasks := service.NewImageTaskServiceWithOptions(store, time.Hour, time.Minute) // enabled == false
	h := &AsyncImageHandler{tasks: tasks}

	router := gin.New()
	router.Use(func(c *gin.Context) {
		groupID := int64(3)
		c.Set(string(middleware2.ContextKeyAPIKey), &service.APIKey{
			ID:      9,
			UserID:  7,
			GroupID: &groupID,
			Group:   &service.Group{ID: groupID, Platform: service.PlatformOpenAI, AllowImageGeneration: true},
		})
		c.Next()
	})
	router.POST("/v1/images/generations/async", h.Submit)
	router.GET("/v1/images/tasks/:task_id", h.Get)

	req := httptest.NewRequest(http.MethodPost, "/v1/images/generations/async", strings.NewReader(`{"model":"gpt-image-1","prompt":"cat"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusNotFound, w.Code)
	require.Contains(t, w.Body.String(), "not enabled")

	pollReq := httptest.NewRequest(http.MethodGet, "/v1/images/tasks/imgtask_missing", nil)
	pollWriter := httptest.NewRecorder()
	router.ServeHTTP(pollWriter, pollReq)
	require.Equal(t, http.StatusNotFound, pollWriter.Code)

	// No task was created / persisted.
	require.Empty(t, store.tasks)
}

func TestAsyncImageHandlerShutdownCancelsExecution(t *testing.T) {
	gin.SetMode(gin.TestMode)
	spoolDir := useAsyncImageSpoolDir(t)
	store := &asyncImageMemoryStore{tasks: make(map[string]*service.ImageTaskRecord)}
	tasks := service.NewImageTaskServiceWithUploader(store, nil, time.Hour, time.Minute)
	started := make(chan struct{})
	cancelled := make(chan struct{})
	h := &AsyncImageHandler{tasks: tasks}
	h.execute = func(_ string, c *gin.Context) {
		close(started)
		<-c.Request.Context().Done()
		close(cancelled)
	}

	router := gin.New()
	router.Use(func(c *gin.Context) {
		groupID := int64(3)
		c.Set(string(middleware2.ContextKeyAPIKey), &service.APIKey{
			ID:      9,
			UserID:  7,
			GroupID: &groupID,
			Group:   &service.Group{ID: groupID, Platform: service.PlatformOpenAI, AllowImageGeneration: true},
		})
		c.Next()
	})
	router.POST("/v1/images/generations/async", h.Submit)

	body := `{"model":"gpt-image-1","prompt":"` + strings.Repeat("x", int(service.DefaultRequestBodySpoolThresholdBytes)) + `"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/images/generations/async", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusAccepted, w.Code)
	<-started

	shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	require.NoError(t, tasks.Shutdown(shutdownCtx))
	select {
	case <-cancelled:
	case <-time.After(time.Second):
		t.Fatal("gateway image execution was not cancelled")
	}
	require.Eventually(t, func() bool {
		entries, err := os.ReadDir(spoolDir)
		return err == nil && len(entries) == 0
	}, time.Second, 10*time.Millisecond)
}
