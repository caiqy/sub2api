package service

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/modeltrace"
	"github.com/stretchr/testify/require"
)

func TestNextModelTraceRefreshBeijing(t *testing.T) {
	for _, tc := range []struct {
		now, expected time.Time
	}{
		{time.Date(2026, 9, 27, 1, 59, 59, 0, modelTraceShanghai), time.Date(2026, 9, 27, 2, 0, 0, 0, modelTraceShanghai)},
		{time.Date(2026, 9, 27, 2, 0, 0, 0, modelTraceShanghai), time.Date(2026, 9, 28, 2, 0, 0, 0, modelTraceShanghai)},
		{time.Date(2026, 9, 27, 23, 0, 0, 0, modelTraceShanghai), time.Date(2026, 9, 28, 2, 0, 0, 0, modelTraceShanghai)},
	} {
		require.Equal(t, tc.expected, nextModelTraceRefresh(tc.now.UTC()))
	}
}

func TestFetchModelTraceFromLimitsAndRedirects(t *testing.T) {
	const sha = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/commit":
			_, _ = w.Write([]byte(`{"object":{"type":"commit","sha":"` + sha + `"}}`))
		case "/bank/" + sha:
			_, _ = w.Write([]byte(`{"schema":"test"}`))
		case "/large":
			_, _ = w.Write([]byte(strings.Repeat("x", (4<<20)+1)))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	url := func(hash string) string { return server.URL + "/bank/" + hash }
	version, data, err := fetchModelTraceFrom(context.Background(), server.URL+"/commit", url)
	require.NoError(t, err)
	require.Equal(t, sha, version)
	require.JSONEq(t, `{"schema":"test"}`, string(data))
	_, _, err = fetchModelTraceFrom(context.Background(), server.URL+"/commit", func(string) string { return server.URL + "/large" })
	require.ErrorContains(t, err, "too large")
	_, _, err = fetchModelTraceFrom(context.Background(), server.URL+"/bank/"+sha, url)
	require.ErrorContains(t, err, "invalid upstream commit SHA")

	other := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { t.Error("unexpected redirect") }))
	defer other.Close()
	redirect := httptest.NewServer(http.RedirectHandler(other.URL, http.StatusFound))
	defer redirect.Close()
	_, _, err = fetchModelTraceFrom(context.Background(), redirect.URL, url)
	require.ErrorContains(t, err, "unexpected redirect")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, _, err = fetchModelTraceFrom(ctx, server.URL+"/commit", url)
	require.Error(t, err)
}

func modelTraceTestSnapshot(t *testing.T, version string) (*modeltrace.Snapshot, []byte) {
	t.Helper()
	data, err := os.ReadFile("../pkg/modeltrace/unified_bank.json")
	require.NoError(t, err)
	snapshot, err := modeltrace.ParseSnapshot(version, data)
	require.NoError(t, err)
	return snapshot, data
}

func TestModelTraceFingerprintRefreshFallbackAndRestore(t *testing.T) {
	original := modeltrace.Current()
	t.Cleanup(func() { modeltrace.Activate(original) })
	const version = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	snapshot, data := modelTraceTestSnapshot(t, version)
	repo := &modelTraceTasksStub{}
	s := &ModelTraceService{repo: repo, fetchFingerprint: func(context.Context) (string, []byte, error) { return version, data, nil }}
	s.refreshFingerprint(context.Background())
	require.Equal(t, snapshot.Version(), modeltrace.Current().Version())
	require.Equal(t, version, repo.fingerprintVersion)

	s.fetchFingerprint = func(context.Context) (string, []byte, error) { return version, []byte(`{}`), nil }
	s.refreshFingerprint(context.Background())
	require.Equal(t, version, repo.fingerprintVersion)
	require.Equal(t, version, modeltrace.Current().Version())
	s.fetchFingerprint = func(context.Context) (string, []byte, error) { return "", nil, errors.New("offline") }
	s.refreshFingerprint(context.Background())
	require.Equal(t, version, modeltrace.Current().Version())

	modeltrace.Activate(original)
	s.syncFingerprint(context.Background())
	require.Equal(t, version, modeltrace.Current().Version(), "another instance or restart loads the persisted bank")
	repo.fingerprintErr = errors.New("database unavailable")
	s.syncFingerprint(context.Background())
	require.Equal(t, version, modeltrace.Current().Version())
	repo.fingerprintErr = nil
	repo.fingerprintData = []byte(`{}`)
	s.syncFingerprint(context.Background())
	require.Equal(t, version, modeltrace.Current().Version(), "invalid stored data keeps the working bank")
}

func TestModelTraceQueuedTaskUsesExecutionSnapshot(t *testing.T) {
	original := modeltrace.Current()
	t.Cleanup(func() { modeltrace.Activate(original) })
	first, _ := modelTraceTestSnapshot(t, "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	second, _ := modelTraceTestSnapshot(t, "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")
	modeltrace.Activate(first)
	tasks := &modelTraceTasksStub{onCheck: func(context.Context) {
		if modeltrace.Current() == first {
			modeltrace.Activate(second)
		}
	}}
	account := &Account{ID: 1, Platform: PlatformOpenAI}
	probe := &modelTraceProbeStub{account: account, target: "gpt-6-astra"}
	s := &ModelTraceService{ctx: context.Background(), repo: tasks, accounts: &modelTraceAccountsStub{account: account}, prober: probe}
	queued := &ModelTraceTask{ID: 1, AccountID: 1, Source: "manual", Model: probe.target, TargetModel: probe.target, Rounds: 2, Version: original.Version()}
	s.execute(context.Background(), "owner", queued)
	require.Equal(t, "completed", tasks.finished.Status)
	require.Equal(t, first.Version(), tasks.finished.Version)
	require.Equal(t, []string{first.Version(), first.Version(), first.Version()}, tasks.progressVersions)
	require.Equal(t, second.Version(), modeltrace.Current().Version())
}
