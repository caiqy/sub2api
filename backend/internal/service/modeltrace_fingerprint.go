package service

import (
	"context"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/modeltrace"
)

const modelTraceCommitURL = "https://api.github.com/repos/xqy2006/ModelTrace/git/ref/heads/main"

var modelTraceShanghai = time.FixedZone("Asia/Shanghai", 8*60*60)

func nextModelTraceRefresh(now time.Time) time.Time {
	local := now.In(modelTraceShanghai)
	next := time.Date(local.Year(), local.Month(), local.Day(), 2, 0, 0, 0, modelTraceShanghai)
	if !next.After(now) {
		next = next.AddDate(0, 0, 1)
	}
	return next
}

func (s *ModelTraceService) fingerprintLoop() {
	defer s.wg.Done()
	poll := time.NewTicker(time.Minute)
	defer poll.Stop()
	refresh := time.NewTimer(time.Until(nextModelTraceRefresh(time.Now())))
	defer refresh.Stop()
	for {
		select {
		case <-s.ctx.Done():
			return
		case <-poll.C:
			ctx, cancel := context.WithTimeout(s.ctx, 10*time.Second)
			s.syncFingerprint(ctx)
			cancel()
		case <-refresh.C:
			ctx, cancel := context.WithTimeout(s.ctx, 25*time.Second)
			s.refreshFingerprint(ctx)
			cancel()
			refresh.Reset(time.Until(nextModelTraceRefresh(time.Now())))
		}
	}
}

func (s *ModelTraceService) syncFingerprint(ctx context.Context) {
	version, data, err := s.repo.ReadFingerprint(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return
	}
	if err != nil {
		log.Printf("[ModelTrace] fingerprint sync failed: %v", err)
		return
	}
	if version == modeltrace.Current().Version() {
		return
	}
	snapshot, err := modeltrace.ParseSnapshot(version, data)
	if err != nil {
		log.Printf("[ModelTrace] stored fingerprint rejected: %v", err)
		return
	}
	modeltrace.Activate(snapshot)
	log.Printf("[ModelTrace] fingerprint activated: %s", snapshot.Version())
}

func (s *ModelTraceService) refreshFingerprint(ctx context.Context) {
	var snapshot *modeltrace.Snapshot
	version, _, updated, err := s.repo.TryUpdateFingerprint(ctx, func(ctx context.Context) (string, []byte, error) {
		version, data, err := s.fetchFingerprint(ctx)
		if err != nil {
			return "", nil, err
		}
		snapshot, err = modeltrace.ParseSnapshot(version, data)
		return version, data, err
	})
	if err != nil {
		log.Printf("[ModelTrace] fingerprint refresh failed: %v", err)
		return
	}
	if updated {
		modeltrace.Activate(snapshot)
		log.Printf("[ModelTrace] fingerprint updated: %s", version)
	}
}

func fetchModelTraceFingerprint(ctx context.Context) (string, []byte, error) {
	return fetchModelTraceFrom(ctx, modelTraceCommitURL, func(sha string) string {
		return "https://raw.githubusercontent.com/xqy2006/ModelTrace/" + sha + "/data/unified_bank.json"
	})
}

func fetchModelTraceFrom(ctx context.Context, commitURL string, bankURL func(string) string) (string, []byte, error) {
	client := &http.Client{Timeout: 20 * time.Second, CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) >= 3 || req.URL.Host != via[0].URL.Host || req.URL.Scheme != via[0].URL.Scheme {
			return errors.New("modeltrace: unexpected redirect")
		}
		return nil
	}}
	get := func(url string, limit int64) ([]byte, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("User-Agent", "sub2api-modeltrace")
		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}
		defer func() { _ = resp.Body.Close() }()
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("modeltrace: upstream HTTP %d", resp.StatusCode)
		}
		body, err := io.ReadAll(io.LimitReader(resp.Body, limit+1))
		if err != nil {
			return nil, err
		}
		if int64(len(body)) > limit {
			return nil, errors.New("modeltrace: upstream response too large")
		}
		return body, nil
	}
	metadata, err := get(commitURL, 64<<10)
	if err != nil {
		return "", nil, err
	}
	var ref struct {
		Object struct {
			SHA  string `json:"sha"`
			Type string `json:"type"`
		} `json:"object"`
	}
	if err := json.Unmarshal(metadata, &ref); err != nil {
		return "", nil, fmt.Errorf("modeltrace: invalid upstream commit: %w", err)
	}
	commit := ref.Object.SHA
	if ref.Object.Type != "commit" || len(commit) != 40 || strings.ToLower(commit) != commit {
		return "", nil, errors.New("modeltrace: invalid upstream commit SHA")
	}
	if _, err := hex.DecodeString(commit); err != nil {
		return "", nil, errors.New("modeltrace: invalid upstream commit SHA")
	}
	data, err := get(bankURL(commit), 4<<20)
	return commit, data, err
}
