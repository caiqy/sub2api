package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

type codexTicketFuncUpstream struct {
	HTTPUpstream
	do func(*http.Request) (*http.Response, error)
}

func (u *codexTicketFuncUpstream) Do(req *http.Request, _ string, _ int64, _ int) (*http.Response, error) {
	return u.do(req)
}
func codexTicketResponse() *http.Response {
	h := http.Header{}
	h.Set(openAICodexTurnStateHeader, fakeCodexTicketState(292))
	return &http.Response{StatusCode: http.StatusOK, Header: h, Body: io.NopCloser(strings.NewReader("data: {}\n\n"))}
}

func TestCodexTicketProbeBypassesPluginDuringWiring(t *testing.T) {
	manager := &PluginManager{}
	manager.route.Store(&pluginRoute{pluginID: 1, rolloutPercent: 100, unavailable: "plugin must not handle synthetic probes"})
	var calls atomic.Int64
	upstream := &codexTicketFuncUpstream{do: func(req *http.Request) (*http.Response, error) {
		calls.Add(1)
		if HTTPUpstreamProfileFromContext(req.Context()) != HTTPUpstreamProfileOpenAIHarvest || !req.Close {
			return nil, errors.New("missing no-reuse transport profile")
		}
		return codexTicketResponse(), nil
	}}
	svc := ticketTestService(t, config.OpenAICodexTicketConfig{Enabled: true}, upstream)
	svc.SetPluginManager(manager)
	account := ticketTestAccount(41)
	// This binding rejects ordinary traffic; harvesting still uses the dedicated transport.
	request, _ := http.NewRequest(http.MethodPost, "https://example.com", nil)
	_, err := svc.doOpenAIUpstream(request, "", account)
	require.Error(t, err)
	start := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		<-start
		for i := 0; i < 200; i++ {
			svc.SetPluginManager(manager)
		}
	}()
	close(start)
	for i := 0; i < 20; i++ {
		state, status, err := svc.fireOpenAICodexTicketProbe(context.Background(), account, "test-token", "gpt-6-astra", "http://proxy.example.com:8080", time.Second)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, status)
		require.Len(t, state, 292)
	}
	wg.Wait()
	require.Equal(t, int64(20), calls.Load())
}

type codexTicketLifecycleRepo struct {
	AccountRepository
	account Account
	list    func(context.Context) ([]Account, error)
	persist func(context.Context) error
}

func (r *codexTicketLifecycleRepo) ListByPlatform(ctx context.Context, _ string) ([]Account, error) {
	if r.list != nil {
		return r.list(ctx)
	}
	return []Account{r.account}, nil
}
func (r *codexTicketLifecycleRepo) UpdateExtra(ctx context.Context, _ int64, _ map[string]any) error {
	if r.persist != nil {
		return r.persist(ctx)
	}
	return nil
}

type codexTicketLifecycleSettings struct {
	SettingRepository
	get func(context.Context, string) (string, error)
}

func (r *codexTicketLifecycleSettings) GetValue(ctx context.Context, key string) (string, error) {
	return r.get(ctx, key)
}

func TestCodexTicketHarvesterStopCancelsInFlightWork(t *testing.T) {
	for _, stage := range []string{"settings-enabled", "settings-proxy", "accounts", "upstream", "persist"} {
		t.Run(stage, func(t *testing.T) {
			started := make(chan struct{})
			cancelled := make(chan struct{})
			// 取消后同一个 stage 可能被再次进入（例如探针路径重新读取设置），
			// 因此 started/cancelled 各自只关一次。
			var startedOnce, cancelledOnce sync.Once
			block := func(ctx context.Context) error {
				startedOnce.Do(func() { close(started) })
				<-ctx.Done()
				cancelledOnce.Do(func() { close(cancelled) })
				return ctx.Err()
			}
			account := ticketTestAccount(41)
			account.Status = StatusActive
			repo := &codexTicketLifecycleRepo{account: *account}
			upstream := &codexTicketFuncUpstream{do: func(req *http.Request) (*http.Response, error) {
				if stage == "upstream" {
					return nil, block(req.Context())
				}
				return codexTicketResponse(), nil
			}}
			svc := ticketTestService(t, config.OpenAICodexTicketConfig{Enabled: true, HarvestProxyURL: "http://proxy.example.com:8080", HarvestAttemptTimeoutSeconds: 25, Models: []string{"gpt-6-astra"}}, upstream)
			svc.accountRepo = repo
			if stage == "accounts" {
				repo.list = func(ctx context.Context) ([]Account, error) { return nil, block(ctx) }
			}
			if stage == "persist" {
				repo.persist = block
			}
			if strings.HasPrefix(stage, "settings-") {
				svc.settingService = NewSettingService(&codexTicketLifecycleSettings{get: func(ctx context.Context, key string) (string, error) {
					if stage == "settings-enabled" && key == SettingKeyOpenAICodexTicketEnabled || stage == "settings-proxy" && key == SettingKeyOpenAICodexTicketHarvestProxyURL {
						return "", block(ctx)
					}
					if key == SettingKeyOpenAICodexTicketEnabled {
						return "true", nil
					}
					return "", ErrSettingNotFound
				}}, svc.cfg)
			}
			svc.StartOpenAICodexTicketHarvester()
			t.Cleanup(svc.StopOpenAICodexTicketHarvester)
			select {
			case <-started:
			case <-time.After(3 * time.Second):
				t.Fatal("harvester did not reach " + stage)
			}
			// Repeated start must not create a second loop or overwrite the cancellation state.
			svc.StartOpenAICodexTicketHarvester()
			stopped := make(chan struct{})
			go func() { svc.StopOpenAICodexTicketHarvester(); close(stopped) }()
			select {
			case <-stopped:
			case <-time.After(time.Second):
				t.Fatal("stop waited for the probe timeout")
			}
			select {
			case <-cancelled:
			case <-time.After(time.Second):
				t.Fatal("in-flight operation did not receive cancellation")
			}
			svc.StopOpenAICodexTicketHarvester()
			svc.StartOpenAICodexTicketHarvester()
		})
	}
}

type codexTicketHeaderOnlyBody struct{ reads, closes int }

func (b *codexTicketHeaderOnlyBody) Read([]byte) (int, error) { b.reads++; return 0, io.EOF }
func (b *codexTicketHeaderOnlyBody) Close() error             { b.closes++; return nil }
func TestCodexTicketProbeClosesStreamWithoutDraining(t *testing.T) {
	body := &codexTicketHeaderOnlyBody{}
	svc := ticketTestService(t, config.OpenAICodexTicketConfig{}, &codexTicketFuncUpstream{do: func(*http.Request) (*http.Response, error) {
		response := codexTicketResponse()
		response.Body = body
		return response, nil
	}})
	_, _, err := svc.fireOpenAICodexTicketProbe(context.Background(), ticketTestAccount(41), "test-token", "gpt-6-astra", "", time.Second)
	require.NoError(t, err)
	require.Zero(t, body.reads)
	require.Equal(t, 1, body.closes)
}

func TestCodexTicketPolicyExemptsCredentialShadows(t *testing.T) {
	parentID := int64(41)
	parent := ticketTestAccount(parentID)
	shadow := ticketTestAccount(42)
	shadow.ParentAccountID = &parentID
	shadow.Status = StatusActive
	cfg := config.OpenAICodexTicketConfig{Enabled: true, FailClosed: true, HarvestProxyURL: "http://proxy.example.com:8080"}
	upstream := &httpUpstreamRecorder{}
	svc := ticketTestService(t, cfg, upstream)
	svc.accountRepo = &codexTicketRefreshRepo{accounts: []Account{*shadow}}
	require.True(t, svc.openAICodexTicketBlocksAccount(context.Background(), parent, "gpt-6-astra"))
	for _, accountType := range []string{AccountTypeOAuth, AccountTypeSetupToken} {
		shadow.Type = accountType
		require.False(t, svc.openAICodexTicketBlocksAccount(context.Background(), shadow, "gpt-6-astra"))
		headers := http.Header{}
		headers.Set(openAICodexTurnStateHeader, "client-state")
		require.NoError(t, svc.applyOpenAICodexTicket(context.Background(), shadow, "gpt-6-astra", headers))
		require.Equal(t, "client-state", headers.Get(openAICodexTurnStateHeader))
		require.Empty(t, OpenAICodexTicketStatuses(shadow, cfg, time.Now()))
		svc.probeOnceOpenAICodexTicket(context.Background(), shadow, "gpt-6-astra")
	}
	svc.refreshOpenAICodexTickets(context.Background())
	require.Empty(t, upstream.requests)
}

func TestCodexTicketNoProxyDoesNotInjectExistingTicket(t *testing.T) {
	for _, failClosed := range []bool{false, true} {
		svc := ticketTestService(t, config.OpenAICodexTicketConfig{Enabled: true, FailClosed: failClosed, HarvestProxyURL: "  "}, nil)
		account := ticketTestAccount(701)
		svc.storeOpenAICodexTicket(context.Background(), account, &openAICodexTicket{Model: "gpt-6-astra", State: fakeCodexTicketState(292), Length: 292, ExpiresAt: time.Now().Add(time.Hour)})
		h := http.Header{}
		h.Set(openAICodexTurnStateHeader, "client-state")
		err := svc.applyOpenAICodexTicket(context.Background(), account, "gpt-6-astra", h)
		if failClosed {
			require.ErrorIs(t, err, ErrOpenAICodexTicketUnavailable)
		} else {
			require.NoError(t, err)
		}
		require.Equal(t, "client-state", h.Get(openAICodexTurnStateHeader))
	}
}

func TestCodexTicketInactiveDoesNotParticipate(t *testing.T) {
	account := ticketTestAccount(702)
	account.Status = "inactive"
	upstream := &httpUpstreamRecorder{}
	svc := ticketTestService(t, config.OpenAICodexTicketConfig{Enabled: true, FailClosed: true, HarvestProxyURL: "http://proxy.example.com:8080"}, upstream)
	require.False(t, isOpenAICodexTicketParticipant(account))
	require.Empty(t, OpenAICodexTicketStatuses(account, svc.cfg.Gateway.OpenAICodexTicket, time.Now()))
	require.False(t, svc.openAICodexTicketBlocksAccount(context.Background(), account, "gpt-6-astra"))
	h := http.Header{}
	require.NoError(t, svc.applyOpenAICodexTicket(context.Background(), account, "gpt-6-astra", h))
	svc.probeOnceOpenAICodexTicket(context.Background(), account, "gpt-6-astra")
	require.Empty(t, upstream.requests)
}

func TestCodexTicketMergeDropsRetiredProxy(t *testing.T) {
	key := openAICodexTicketExtraKey("gpt-6-astra")
	input := map[string]any{"codex_harvest_proxy_url": "http://user:synthetic-secret@proxy", key: "forged", "keep": true}
	current := map[string]any{"codex_harvest_proxy_url": "old", key: "persisted"}
	require.Equal(t, map[string]any{"keep": true, key: "persisted"}, MergeOpenAICodexTicketExtra(input, current))
	require.Equal(t, map[string]any{"keep": true}, MergeOpenAICodexTicketExtra(input, nil))
	require.Contains(t, input, "codex_harvest_proxy_url")
}

func TestCodexTicketRefreshBoundsAttemptsAndRotatesFailures(t *testing.T) {
	var mu sync.Mutex
	seen := map[string]int{}
	upstream := &codexTicketFuncUpstream{do: func(req *http.Request) (*http.Response, error) {
		mu.Lock()
		seen[req.Header.Get("Authorization")]++
		mu.Unlock()
		return nil, io.EOF
	}}
	repo := &codexTicketRefreshRepo{}
	for i := int64(1); i <= 5; i++ {
		account := ticketTestAccount(i)
		account.Credentials["access_token"] = fmt.Sprint(i)
		repo.accounts = append(repo.accounts, *account)
	}
	svc := ticketTestService(t, config.OpenAICodexTicketConfig{Enabled: true, HarvestProxyURL: "http://proxy.example.com:8080", MaxConcurrentProbes: 2, Models: []string{"gpt-6-astra"}}, upstream)
	svc.accountRepo = repo
	svc.refreshOpenAICodexTickets(context.Background())
	require.Len(t, seen, 2, "excess targets must wait for a later cycle")
	svc.refreshOpenAICodexTickets(context.Background())
	require.Len(t, seen, 4, "failed first targets must not starve later targets")
	svc.refreshOpenAICodexTickets(context.Background())
	require.Len(t, seen, 5)
	var total int
	for _, count := range seen {
		total += count
	}
	require.Equal(t, 6, total)
}

func TestCodexTicketConcurrentProbesHonorChangedLimit(t *testing.T) {
	started := make(chan struct{}, 10)
	release := make(chan struct{})
	var wg sync.WaitGroup
	t.Cleanup(func() { close(release); wg.Wait() })
	upstream := &codexTicketFuncUpstream{do: func(req *http.Request) (*http.Response, error) {
		started <- struct{}{}
		select {
		case <-release:
		case <-req.Context().Done():
		}
		return nil, io.EOF
	}}
	svc := ticketTestService(t, config.OpenAICodexTicketConfig{Enabled: true, HarvestProxyURL: "http://proxy.example.com:8080", MaxConcurrentProbes: 2}, upstream)
	launch := func(id int64) <-chan struct{} {
		done := make(chan struct{})
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer close(done)
			svc.probeOnceOpenAICodexTicket(context.Background(), ticketTestAccount(id), "gpt-6-astra")
		}()
		return done
	}
	for i := int64(1); i <= 2; i++ {
		launch(i)
		select {
		case <-started:
		case <-time.After(time.Second):
			t.Fatal("probe did not start")
		}
	}
	// Both existing probes have finished reading config and are blocked in transport.
	svc.cfg.Gateway.OpenAICodexTicket.MaxConcurrentProbes = 1
	done := launch(3)
	select {
	case <-done:
	case <-started:
		t.Fatal("lowered limit ignored existing probes")
	case <-time.After(time.Second):
		t.Fatal("capacity must skip without waiting")
	}
	svc.cfg.Gateway.OpenAICodexTicket.MaxConcurrentProbes = 3
	launch(4)
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("increased limit did not admit probe")
	}
	done = launch(5)
	select {
	case <-done:
	case <-started:
		t.Fatal("increased limit reset in-flight count")
	case <-time.After(time.Second):
		t.Fatal("capacity must skip without waiting")
	}
}

func TestCodexTicketProbeLogsDoNotExposeSecrets(t *testing.T) {
	sink, cleanup := captureStructuredLog(t)
	defer cleanup()
	const secret = "synthetic-proxy-password"
	svc := ticketTestService(t, config.OpenAICodexTicketConfig{Enabled: true, HarvestProxyURL: "http://user:" + secret + "@proxy.example.com:8080"}, &codexTicketFuncUpstream{do: func(*http.Request) (*http.Response, error) {
		return nil, errors.New("proxy connection failed: " + secret)
	}})
	svc.probeOnceOpenAICodexTicket(context.Background(), ticketTestAccount(703), "gpt-6-astra")
	require.True(t, sink.ContainsFieldValue("reason", openAICodexTicketMissTransport))
	sink.mu.Lock()
	defer sink.mu.Unlock()
	for _, event := range sink.events {
		require.NotContains(t, fmt.Sprint(event), secret)
	}
}
