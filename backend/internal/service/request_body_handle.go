package service

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/tidwall/gjson"
)

const (
	DefaultRequestBodySpoolThresholdBytes int64 = 1 << 20
	DefaultRequestBodyPreviewLimitBytes   int64 = 256 << 10
	defaultRequestBodySpoolPrefix               = "sub2api-request-body-"
	requestBodyPreviewSnapshotKind              = "request_body_preview"
	requestBodyPreviewOmittedMarker             = "[inline binary payload omitted]"
	requestBodyCleanupRetryDelay                = 100 * time.Millisecond
	requestBodyCleanupRetryAttempts             = 2
)

type requestBodyPreviewSnapshot struct {
	Kind      string `json:"kind,omitempty"`
	Preview   string `json:"preview"`
	Truncated bool   `json:"truncated"`
	Size      int64  `json:"size"`
}

var requestBodySpoolCleanupOnce sync.Once

var ErrRequestBodySpool = errors.New("request body spool failed")

type RequestBodyHandleOptions struct {
	SpoolThresholdBytes int64
	PreviewLimitBytes   int64
	TempDir             string
	FilePrefix          string
}

type RequestBodyHandle struct {
	mu           sync.Mutex
	size         int64
	hash         string
	preview      string
	memory       []byte
	spoolPath    string
	spoolActive  bool
	spoolReaders int
	cleaned      bool
	retrying     bool
}

type requestBodySpoolReadCloser struct {
	io.ReadCloser
	state *requestBodySpoolReadCloserState
}

type requestBodySpoolReadCloserState struct {
	onClose  func() error
	once     sync.Once
	closeErr error
}

func (r requestBodySpoolReadCloser) Read(p []byte) (int, error) {
	n, err := r.ReadCloser.Read(p)
	if err != nil && !errors.Is(err, io.EOF) {
		return n, fmt.Errorf("%w: read spool file: %v", ErrRequestBodySpool, err)
	}
	return n, err
}

func (r requestBodySpoolReadCloser) Close() error {
	r.state.once.Do(func() {
		r.state.closeErr = r.ReadCloser.Close()
		if err := r.state.onClose(); r.state.closeErr == nil {
			r.state.closeErr = err
		}
	})
	return r.state.closeErr
}

func NewRequestBodyHandleFromReader(r io.Reader, opts RequestBodyHandleOptions) (*RequestBodyHandle, error) {
	if r == nil {
		return NewRequestBodyHandleFromBytes(nil, opts)
	}
	opts = normalizeRequestBodyHandleOptions(opts)
	runRequestBodySpoolCleanupOnce()

	tempDir := opts.TempDir
	prefix := opts.FilePrefix

	var memory bytes.Buffer
	var preview bytes.Buffer
	hasher := sha256.New()
	buf := make([]byte, 32*1024)
	var size int64
	var spool *os.File
	var spoolPath string

	cleanup := func() {
		if spool != nil {
			_ = spool.Close()
			_ = os.Remove(spoolPath)
		}
	}

	for {
		n, readErr := r.Read(buf)
		if n > 0 {
			chunk := buf[:n]
			size += int64(n)
			_, _ = hasher.Write(chunk)

			if opts.PreviewLimitBytes > int64(preview.Len()) {
				want := int(opts.PreviewLimitBytes - int64(preview.Len()))
				if want > len(chunk) {
					want = len(chunk)
				}
				_, _ = preview.Write(chunk[:want])
			}

			if spool == nil && size > opts.SpoolThresholdBytes {
				f, err := os.CreateTemp(tempDir, prefix)
				if err != nil {
					return nil, fmt.Errorf("%w: create temp file: %v", ErrRequestBodySpool, err)
				}
				spool = f
				spoolPath = f.Name()
				if _, err := spool.Write(memory.Bytes()); err != nil {
					cleanup()
					return nil, fmt.Errorf("%w: write buffered body: %v", ErrRequestBodySpool, err)
				}
				memory.Reset()
			}

			if spool != nil {
				if _, err := spool.Write(chunk); err != nil {
					cleanup()
					return nil, fmt.Errorf("%w: write body: %v", ErrRequestBodySpool, err)
				}
			} else if _, err := memory.Write(chunk); err != nil {
				return nil, err
			}
		}

		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			cleanup()
			return nil, readErr
		}
	}

	if spool != nil {
		if err := spool.Close(); err != nil {
			cleanup()
			return nil, fmt.Errorf("%w: close temp file: %v", ErrRequestBodySpool, err)
		}
		return &RequestBodyHandle{
			size:        size,
			hash:        hex.EncodeToString(hasher.Sum(nil)),
			preview:     sanitizeRequestBodyPreview(preview.String(), size > int64(preview.Len())),
			spoolPath:   spoolPath,
			spoolActive: true,
		}, nil
	}

	return &RequestBodyHandle{
		size:    size,
		hash:    hex.EncodeToString(hasher.Sum(nil)),
		preview: sanitizeRequestBodyPreview(preview.String(), size > int64(preview.Len())),
		memory:  memory.Bytes(),
	}, nil
}

func NewRequestBodyHandleFromBytes(body []byte, opts RequestBodyHandleOptions) (*RequestBodyHandle, error) {
	return NewRequestBodyHandleFromReader(bytes.NewReader(body), opts)
}

func RequestBodyPreviewString(body []byte) string {
	truncated := int64(len(body)) > DefaultRequestBodyPreviewLimitBytes
	if int64(len(body)) > DefaultRequestBodyPreviewLimitBytes {
		body = body[:DefaultRequestBodyPreviewLimitBytes]
	}
	return sanitizeRequestBodyPreview(string(body), truncated)
}

func RequestBodyPreviewSnapshot(preview string, size int64, forceTruncated ...bool) string {
	providedBytes := len(preview)
	if size < 0 {
		size = int64(providedBytes)
	}
	truncated := size > int64(providedBytes) || len(forceTruncated) > 0 && forceTruncated[0]
	return marshalRequestBodyPreviewSnapshot(preview, size, truncated, int(DefaultRequestBodyPreviewLimitBytes))
}

func marshalRequestBodyPreviewSnapshot(preview string, size int64, truncated bool, maxBytes int) string {
	providedBytes := len(preview)
	preview = sanitizeRequestBodyPreview(preview, truncated)
	snapshot := requestBodyPreviewSnapshot{
		Kind:      requestBodyPreviewSnapshotKind,
		Preview:   preview,
		Truncated: truncated || len(preview) < providedBytes || isOmittedRequestBodyPreview(preview),
		Size:      size,
	}

	for {
		raw, _ := json.Marshal(snapshot)
		if maxBytes <= 0 || len(raw) <= maxBytes || snapshot.Preview == "" {
			return string(raw)
		}
		excess := len(raw) - maxBytes
		cut := len(snapshot.Preview) - excess
		if cut >= len(snapshot.Preview) {
			cut = len(snapshot.Preview) - 1
		}
		if cut < 0 {
			cut = 0
		}
		for cut > 0 && !utf8.ValidString(snapshot.Preview[:cut]) {
			cut--
		}
		snapshot.Preview = snapshot.Preview[:cut]
		snapshot.Truncated = true
	}
}

func parseRequestBodyPreviewSnapshot(raw string, previewLimits ...int) (requestBodyPreviewSnapshot, bool) {
	if !gjson.Valid(raw) {
		return requestBodyPreviewSnapshot{}, false
	}
	root := gjson.Parse(raw)
	if !root.IsObject() {
		return requestBodyPreviewSnapshot{}, false
	}
	kind := root.Get("kind")
	preview := root.Get("preview")
	truncated := root.Get("truncated")
	size := root.Get("size")
	if kind.Type != gjson.String || kind.String() != requestBodyPreviewSnapshotKind || preview.Type != gjson.String ||
		(truncated.Type != gjson.True && truncated.Type != gjson.False) || size.Type != gjson.Number {
		return requestBodyPreviewSnapshot{}, false
	}
	sizeValue, err := strconv.ParseInt(size.Raw, 10, 64)
	if err != nil {
		return requestBodyPreviewSnapshot{}, false
	}
	limit := 0
	if len(previewLimits) > 0 {
		limit = previewLimits[0]
	}
	previewValue, ok := boundedGJSONString(preview.Raw, limit)
	if !ok {
		return requestBodyPreviewSnapshot{}, false
	}
	return requestBodyPreviewSnapshot{
		Kind:      requestBodyPreviewSnapshotKind,
		Preview:   previewValue,
		Truncated: truncated.Bool(),
		Size:      sizeValue,
	}, true
}

func boundedGJSONString(raw string, maxBytes int) (string, bool) {
	if len(raw) < 2 || raw[0] != '"' || raw[len(raw)-1] != '"' {
		return "", false
	}
	if maxBytes <= 0 || len(raw) <= maxBytes+2 {
		value, err := strconv.Unquote(raw)
		return value, err == nil
	}
	end := maxBytes + 1
	if end > len(raw)-1 {
		end = len(raw) - 1
	}
	for end > 1 {
		value, err := strconv.Unquote(raw[:end] + `"`)
		if err == nil {
			return value, true
		}
		end--
	}
	return "", true
}

func sanitizeRequestBodyPreview(preview string, truncated bool) string {
	if int64(len(preview)) > DefaultRequestBodyPreviewLimitBytes {
		preview = preview[:DefaultRequestBodyPreviewLimitBytes]
		truncated = true
	}
	if isOmittedRequestBodyPreview(preview) {
		return preview
	}
	if hasInlineBinaryPayload(preview, truncated) {
		return requestBodyPreviewOmittedMarker
	}
	return preview
}

func isOmittedRequestBodyPreview(preview string) bool {
	return preview == requestBodyPreviewOmittedMarker || preview == "[multipart body omitted]"
}

type previewScope uint8

const (
	previewScopeNormal previewScope = iota
	previewScopeGeminiInlineData
	previewScopeAnthropicSource
	previewScopeImageArray
)

type previewTokenFrame struct {
	object        bool
	expectKey     bool
	key           string
	scope         previewScope
	hasData       bool
	hasBase64Type bool
}

func hasInlineBinaryPayload(raw string, _ bool) bool {
	if raw == "" {
		return false
	}
	sensitive, complete := scanPreviewJSONTokens(raw, true)
	return sensitive || !complete
}

func scanPreviewJSONTokens(raw string, anyStringDataURL ...bool) (bool, bool) {
	decoder := json.NewDecoder(strings.NewReader(raw))
	frames := make([]previewTokenFrame, 0, 16)
	rootValues := 0
	checkAnyStringDataURL := len(anyStringDataURL) > 0 && anyStringDataURL[0]
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			return false, len(frames) == 0 && rootValues == 1
		}
		if err != nil {
			return false, false
		}
		if len(frames) == 0 {
			rootValues++
			if rootValues > 1 {
				return false, false
			}
		}

		if delimiter, ok := token.(json.Delim); ok {
			switch delimiter {
			case '{', '[':
				scope := consumePreviewContainerValue(frames, delimiter)
				frames = append(frames, previewTokenFrame{object: delimiter == '{', expectKey: delimiter == '{', scope: scope})
			case '}', ']':
				if len(frames) == 0 {
					return false, false
				}
				frame := frames[len(frames)-1]
				frames = frames[:len(frames)-1]
				if frame.scope == previewScopeAnthropicSource && frame.hasData && frame.hasBase64Type {
					return true, true
				}
			}
			continue
		}
		value, isString := token.(string)
		if checkAnyStringDataURL && isString && isNonEmptyDataURL(value) {
			return true, true
		}

		if len(frames) == 0 {
			continue
		}
		frame := &frames[len(frames)-1]
		if frame.object && frame.expectKey {
			key, ok := token.(string)
			if !ok {
				return false, false
			}
			frame.key = key
			frame.expectKey = false
			continue
		}

		if frame.object {
			if isString && previewStringValueIsSensitive(frame, value) {
				return true, true
			}
			frame.key = ""
			frame.expectKey = true
		} else if frame.scope == previewScopeImageArray && isString && isNonEmptyDataURL(value) {
			return true, true
		}
	}
}

func consumePreviewContainerValue(frames []previewTokenFrame, delimiter json.Delim) previewScope {
	if len(frames) == 0 {
		return previewScopeNormal
	}
	parent := &frames[len(frames)-1]
	if !parent.object {
		if parent.scope == previewScopeImageArray && delimiter == '[' {
			return previewScopeImageArray
		}
		return previewScopeNormal
	}
	key := parent.key
	parent.key = ""
	parent.expectKey = true
	if delimiter == '{' {
		switch key {
		case "inlineData", "inline_data":
			return previewScopeGeminiInlineData
		case "source":
			return previewScopeAnthropicSource
		}
	}
	if delimiter == '[' && isExactImagePreviewKey(key) {
		return previewScopeImageArray
	}
	return previewScopeNormal
}

func previewStringValueIsSensitive(frame *previewTokenFrame, value string) bool {
	key := frame.key
	nonEmpty := strings.TrimSpace(value) != ""
	if key == "b64_json" && nonEmpty {
		return true
	}
	if isExactImagePreviewKey(key) && isNonEmptyDataURL(value) {
		return true
	}
	if frame.scope == previewScopeGeminiInlineData && key == "data" && nonEmpty {
		return true
	}
	if frame.scope == previewScopeAnthropicSource {
		switch key {
		case "data":
			frame.hasData = nonEmpty
		case "type":
			frame.hasBase64Type = value == "base64"
		}
		return frame.hasData && frame.hasBase64Type
	}
	return false
}

func isExactImagePreviewKey(key string) bool {
	switch key {
	case "image_url", "url", "image", "images", "mask", "mask_image_url":
		return true
	default:
		return false
	}
}

func isNonEmptyDataURL(value string) bool {
	value = strings.TrimSpace(value)
	if len(value) < len("data:") || !strings.EqualFold(value[:len("data:")], "data:") {
		return false
	}
	comma := strings.IndexByte(value, ',')
	return comma >= 0 && strings.TrimSpace(value[comma+1:]) != ""
}

func containsNonEmptyDataURL(raw string) bool {
	for i := 0; i+len("data:") <= len(raw); i++ {
		if !strings.EqualFold(raw[i:i+len("data:")], "data:") {
			continue
		}
		comma := i + len("data:")
		for comma < len(raw) && raw[comma] != ',' {
			comma++
		}
		return comma < len(raw) && strings.TrimSpace(raw[comma+1:]) != ""
	}
	return false
}

func normalizeRequestBodyHandleOptions(opts RequestBodyHandleOptions) RequestBodyHandleOptions {
	if opts.SpoolThresholdBytes <= 0 {
		opts.SpoolThresholdBytes = DefaultRequestBodySpoolThresholdBytes
	}
	if opts.PreviewLimitBytes <= 0 {
		opts.PreviewLimitBytes = DefaultRequestBodyPreviewLimitBytes
	}
	if opts.TempDir == "" {
		opts.TempDir = os.TempDir()
	}
	if opts.FilePrefix == "" {
		opts.FilePrefix = defaultRequestBodySpoolPrefix
	}
	return opts
}

func runRequestBodySpoolCleanupOnce() {
	requestBodySpoolCleanupOnce.Do(func() {
		// ponytail: best-effort startup sweep; dedicated scheduler if temp churn ever matters.
		_ = CleanupStaleRequestBodySpoolFiles(os.TempDir(), defaultRequestBodySpoolPrefix, 24*time.Hour, time.Now())
	})
}

func (h *RequestBodyHandle) Open() (io.ReadCloser, error) {
	if h == nil {
		return nil, errors.New("request body handle is nil")
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.cleaned {
		return nil, errors.New("request body handle has been cleaned up")
	}
	if h.spoolActive {
		file, err := os.Open(h.spoolPath)
		if err != nil {
			return nil, fmt.Errorf("%w: open spool file: %v", ErrRequestBodySpool, err)
		}
		h.spoolReaders++
		return requestBodySpoolReadCloser{
			ReadCloser: file,
			state:      &requestBodySpoolReadCloserState{onClose: h.releaseSpoolReader},
		}, nil
	}
	return io.NopCloser(bytes.NewReader(h.memory)), nil
}

func (h *RequestBodyHandle) ReadAll() ([]byte, error) {
	if h == nil {
		return nil, errors.New("request body handle is nil")
	}
	h.mu.Lock()
	spooled := h.spoolActive
	h.mu.Unlock()
	r, err := h.Open()
	if err != nil {
		return nil, err
	}
	defer func() { _ = r.Close() }()
	body, err := io.ReadAll(r)
	if err != nil {
		if spooled {
			return nil, fmt.Errorf("%w: read spool file: %v", ErrRequestBodySpool, err)
		}
		return nil, err
	}
	return body, nil
}

func (h *RequestBodyHandle) PreviewString() string {
	if h == nil {
		return ""
	}
	return h.preview
}

func (h *RequestBodyHandle) Hash() string {
	if h == nil {
		return ""
	}
	return h.hash
}

func (h *RequestBodyHandle) Size() int64 {
	if h == nil {
		return 0
	}
	return h.size
}

func (h *RequestBodyHandle) Cleanup() error {
	return h.cleanup(os.Remove)
}

func (h *RequestBodyHandle) cleanup(remove func(string) error) error {
	if h == nil {
		return nil
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.cleaned && (!h.spoolActive || h.spoolReaders > 0) {
		return nil
	}
	if h.spoolActive {
		if h.spoolReaders > 0 {
			h.memory = nil
			h.cleaned = true
			return nil
		}
		if err := remove(h.spoolPath); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	h.memory = nil
	h.spoolActive = false
	h.cleaned = true
	return nil
}

func (h *RequestBodyHandle) releaseSpoolReader() error {
	h.mu.Lock()
	if h.spoolReaders > 0 {
		h.spoolReaders--
	}
	shouldCleanup := h.cleaned && h.spoolActive && h.spoolReaders == 0
	h.mu.Unlock()
	if !shouldCleanup {
		return nil
	}
	if err := h.Cleanup(); err != nil {
		logger.LegacyPrintf("service.request_body_handle", "[RequestBodyHandle] cleanup after reader close failed path=%q err=%v", h.spoolPath, err)
		h.scheduleCleanupRetry()
		return err
	}
	return nil
}

// CleanupRequestBodyHandle releases a request body and retries transient spool deletion failures.
func CleanupRequestBodyHandle(h *RequestBodyHandle) {
	if h == nil {
		return
	}
	if err := h.Cleanup(); err != nil {
		logger.LegacyPrintf("service.request_body_handle", "[RequestBodyHandle] cleanup failed path=%q err=%v", h.spoolPath, err)
		h.scheduleCleanupRetry()
	}
}

func (h *RequestBodyHandle) scheduleCleanupRetry() {
	h.mu.Lock()
	if !h.spoolActive || h.retrying {
		h.mu.Unlock()
		return
	}
	h.retrying = true
	h.mu.Unlock()

	// ponytail: two delayed retries; stale sweep handles files that remain unavailable.
	go func() {
		defer func() {
			h.mu.Lock()
			h.retrying = false
			h.mu.Unlock()
		}()
		retryRequestBodyHandleCleanup(func() error {
			err := h.Cleanup()
			if err != nil {
				logger.LegacyPrintf("service.request_body_handle", "[RequestBodyHandle] cleanup retry failed path=%q err=%v", h.spoolPath, err)
			}
			return err
		})
	}()
}

func retryRequestBodyHandleCleanup(cleanup func() error) {
	for attempt := 0; attempt < requestBodyCleanupRetryAttempts; attempt++ {
		time.Sleep(requestBodyCleanupRetryDelay)
		if cleanup() == nil {
			return
		}
	}
}

func CleanupStaleRequestBodySpoolFiles(dir, prefix string, olderThan time.Duration, now time.Time) error {
	if dir == "" || prefix == "" {
		return nil
	}
	matches, err := filepath.Glob(filepath.Join(dir, prefix+"*"))
	if err != nil {
		return err
	}
	for _, path := range matches {
		info, err := os.Stat(path)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return err
		}
		if info.IsDir() || strings.TrimPrefix(filepath.Base(path), prefix) == filepath.Base(path) {
			continue
		}
		if now.Sub(info.ModTime()) <= olderThan {
			continue
		}
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	return nil
}
