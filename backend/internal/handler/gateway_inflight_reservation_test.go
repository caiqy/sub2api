package handler

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestRequestMaxOutputTokens(t *testing.T) {
	require.Equal(t, 1024, requestMaxOutputTokens([]byte(`{"max_tokens":1024}`)))
	require.Equal(t, 2048, requestMaxOutputTokens([]byte(`{"max_completion_tokens":2048}`)))
	require.Equal(t, 4096, requestMaxOutputTokens([]byte(`{"max_output_tokens":4096}`)))
	require.Equal(t, 512, requestMaxOutputTokens([]byte(`{"generationConfig":{"maxOutputTokens":512}}`)))
	require.Equal(t, 0, requestMaxOutputTokens([]byte(`{"max_tokens":"x"}`)))
	require.Equal(t, 0, requestMaxOutputTokens([]byte(`{}`)))
}

func TestTokenInflightEstimate_ServiceTierSnapshotRemainsSmall(t *testing.T) {
	for _, tc := range []struct{ tier, want string }{
		{strings.Repeat("x", 2<<20), ""},
		{" PRIORITY ", "priority"},
		{"ultrafast", "ultrafast"},
		{"flex", "flex"},
	} {
		body := []byte(`{"max_output_tokens":17,"service_tier":"` + tc.tier + `"}`)
		estimate := tokenInflightEstimate("gpt-6-astra", body)
		require.Equal(t, len(body), estimate.BodyBytes)
		require.Equal(t, 17, estimate.MaxTokens)
		require.LessOrEqual(t, len(estimate.ServiceTier), 9)
		require.Equal(t, tc.want, estimate.ServiceTier, "only billing-relevant tier scalars survive releasing the request body")
	}
}

func TestTokenInflightEstimate_NoneKeepsBillingMultiplierKey(t *testing.T) {
	estimate := tokenInflightEstimate("gpt-5.4", []byte(`{"reasoning":{"effort":"none"},"max_output_tokens":17}`))
	require.Equal(t, "none", estimate.ReasoningEffort, "none is a valid configured billing multiplier key, not a missing effort")
}

type countingEstimator struct {
	calls  int
	cost   float64
	priced bool
}

func (e *countingEstimator) EstimateInflightReservation(context.Context, *service.APIKey, service.InflightEstimateRequest) (float64, bool) {
	e.calls++
	return e.cost, e.priced
}

func newInflightTestGinContext() *gin.Context {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/", nil)
	return c
}

func TestReserveInflightBalance_SkipsWhenDisabledOrSubscription(t *testing.T) {
	cfg := &config.Config{}
	billing := service.NewBillingCacheService(nil, nil, nil, nil, nil, nil, cfg, nil)
	t.Cleanup(billing.Stop)
	est := &countingEstimator{cost: 1, priced: true}
	apiKey := &service.APIKey{User: &service.User{ID: 1}}

	done, err := reserveInflightBalance(newInflightTestGinContext(), billing, est, apiKey, nil, tokenInflightEstimate("m", []byte(`{}`)))
	require.NoError(t, err)
	done()
	require.Equal(t, 0, est.calls, "disabled switch must not even estimate")

	cfg.Billing.InflightReservation.Enabled = true
	apiKey.Group = &service.Group{SubscriptionType: service.SubscriptionTypeSubscription}
	done, err = reserveInflightBalance(newInflightTestGinContext(), billing, est, apiKey, &service.UserSubscription{}, tokenInflightEstimate("m", []byte(`{}`)))
	require.NoError(t, err)
	done()
	require.Equal(t, 0, est.calls, "subscription mode must be unaffected")
}

func TestReserveInflightBalance_UnpricedFailOpenByDefaultFailClosedOptIn(t *testing.T) {
	cache := newHandlerInflightCache(10)
	cfg := &config.Config{}
	cfg.Billing.InflightReservation = config.InflightReservationConfig{Enabled: true, TTLSeconds: 60}
	billing := service.NewBillingCacheService(cache, nil, nil, nil, nil, nil, cfg, nil)
	t.Cleanup(billing.Stop)
	apiKey := &service.APIKey{User: &service.User{ID: 1}}
	est := &countingEstimator{priced: false}

	done, err := reserveInflightBalance(newInflightTestGinContext(), billing, est, apiKey, nil, tokenInflightEstimate("unknown", nil))
	require.NoError(t, err)
	done()

	cfg.Billing.InflightReservation.FailClosedOnUnpriced = true
	_, err = reserveInflightBalance(newInflightTestGinContext(), billing, est, apiKey, nil, tokenInflightEstimate("unknown", nil))
	require.ErrorIs(t, err, service.ErrInsufficientBalance)
}

// handlerInflightCache 内存版余额缓存 + 在途预留（语义同 Redis Lua）。
type handlerInflightCache struct {
	service.BillingCache
	mu      sync.Mutex
	balance float64
	res     map[string]float64
	peak    float64
}

func newHandlerInflightCache(balance float64) *handlerInflightCache {
	return &handlerInflightCache{balance: balance, res: map[string]float64{}}
}

func (m *handlerInflightCache) GetUserBalance(context.Context, int64) (float64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.balance, nil
}

func (m *handlerInflightCache) GetUserPlatformQuotaCache(context.Context, int64, string) (*service.UserPlatformQuotaCacheEntry, bool, error) {
	return nil, false, nil
}

func (m *handlerInflightCache) ReserveInflightBalance(_ context.Context, _ int64, id string, amount, balance float64, _ time.Duration) (bool, float64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	sum := 0.0
	for _, v := range m.res {
		sum += v
	}
	if len(m.res) > 0 && balance-sum < amount {
		return false, sum, nil
	}
	m.res[id] = amount
	m.peak = max(m.peak, amount)
	return true, sum, nil
}

func (m *handlerInflightCache) ReleaseInflightBalance(_ context.Context, _ int64, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.res, id)
	return nil
}

func (m *handlerInflightCache) count() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.res)
}

func (m *handlerInflightCache) total() float64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	var total float64
	for _, amount := range m.res {
		total += amount
	}
	return total
}

func (m *handlerInflightCache) GrowInflightBalance(_ context.Context, _ int64, id string, amount, balance float64, _ time.Duration) (bool, bool, float64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	previous, alive := m.res[id]
	if !alive {
		return false, false, 0, nil
	}
	var total float64
	for _, cost := range m.res {
		total += cost
	}
	if amount > previous && balance-total < amount-previous {
		return false, true, previous, nil
	}
	reserved := max(previous, amount)
	m.res[id] = reserved
	return true, true, reserved, nil
}

func (m *handlerInflightCache) DeductUserBalance(_ context.Context, _ int64, amount float64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.balance -= amount
	return nil
}

func TestGatewayInflightReservation_HTTPRetainsOnlyBodyEstimateScalars(t *testing.T) {
	group := &service.Group{ID: 62, Platform: service.PlatformAnthropic, Status: service.StatusActive, Hydrated: true, RateMultiplier: 1}
	account := &service.Account{ID: 163, Platform: service.PlatformAnthropic, Type: service.AccountTypeAPIKey, Status: service.StatusActive, Schedulable: true, Concurrency: 1, Priority: 1, Credentials: map[string]any{"api_key": "local-fixture"}}
	upstream := &gatewayAcceptedWireCapturingUpstream{}
	env := newTerminalGatewayMessagesEnv(t, group, upstream, account)
	cache := newHandlerInflightCache(100)
	env.handler.cfg.Billing.InflightReservation = config.InflightReservationConfig{Enabled: true, TTLSeconds: 60, DefaultMaxTokens: 1}
	captureTerminalGatewayUsageBilling(t, env, group, upstream, cache)
	env.apiKey.User.Balance = 100
	request := httptest.NewRequest(http.MethodPost, "/v1/messages", strings.NewReader(`{"model":"claude-sonnet-4-5","max_tokens":1,"messages":[{"role":"user","content":"`+strings.Repeat("x", 4096)+`"}]}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	env.router().ServeHTTP(recorder, request)
	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	cache.mu.Lock()
	peak := cache.peak
	cache.mu.Unlock()
	require.Greater(t, peak, 0.001, "input estimate must survive releasing the original body before account waits")
	require.Eventually(t, func() bool { return cache.count() == 0 }, time.Second, time.Millisecond)
}

func TestOpenAIWSInflightReservation_ActualGenerationGrowsToPeakAcrossTransports(t *testing.T) {
	for _, transport := range []string{service.OpenAIWSIngressModePassthrough, service.OpenAIWSIngressModeDedicated, "http-bridge"} {
		t.Run(transport, func(t *testing.T) {
			cache := newHandlerInflightCache(100)
			observed := make(chan float64, 3)
			input, cheap, expensive := 0.0, 0.001, 0.006
			mode := transport
			if transport == "http-bridge" {
				mode = service.OpenAIWSIngressModeDedicated
			}
			got := runOpenAIResponsesWebSocketUsageLogCase(t, openAIResponsesWSUsageLogCase{
				inflightCache: cache, ingressMode: mode, httpBridge: transport == "http-bridge",
				observeUpstream: func([]byte) { observed <- cache.total() },
				firstPayload:    `{"type":"response.create","model":"gpt-5.4","max_output_tokens":1000,"input":"first"}`,
				midPayload:      `{"type":"response.create","model":"gpt-6-astra","max_output_tokens":1000,"input":"second"}`,
				secondPayload:   `{"type":"response.create","model":"gpt-5.4","max_output_tokens":1000,"input":"third"}`,
				group: &service.Group{ID: 4201, Platform: service.PlatformOpenAI, RateMultiplier: 1, ModelPricing: []service.ChannelModelPricing{
					{Models: []string{"gpt-5.4"}, InputPrice: &input, OutputPrice: &cheap},
					{Models: []string{"gpt-6-astra"}, InputPrice: &input, OutputPrice: &expensive},
				}},
			})
			require.Len(t, got.logs, 3)
			require.Equal(t, []float64{1, 6, 6}, []float64{<-observed, <-observed, <-observed}, "only actual accepted generations raise the single session hold")
			require.Eventually(t, func() bool { return cache.count() == 0 }, time.Second, time.Millisecond)
		})
	}
}

func TestOpenAIWSInflightReservation_SessionUpdateDoesNotGrowUntilGeneration(t *testing.T) {
	cache := newHandlerInflightCache(100)
	observed := make(chan float64, 3)
	input, cheap, expensive := 0.0, 0.001, 0.006
	got := runOpenAIResponsesWebSocketUsageLogCase(t, openAIResponsesWSUsageLogCase{
		inflightCache: cache, ingressMode: service.OpenAIWSIngressModePassthrough,
		observeUpstream: func([]byte) { observed <- cache.total() },
		firstPayload:    `{"type":"response.create","model":"gpt-5.4","max_output_tokens":1000,"input":"first"}`,
		midPayload:      `{"type":"session.update","session":{"model":"gpt-6-astra"}}`,
		secondPayload:   `{"type":"response.create","max_output_tokens":1000,"input":"second"}`,
		group: &service.Group{ID: 4201, Platform: service.PlatformOpenAI, RateMultiplier: 1, ModelPricing: []service.ChannelModelPricing{
			{Models: []string{"gpt-5.4"}, InputPrice: &input, OutputPrice: &cheap},
			{Models: []string{"gpt-6-astra"}, InputPrice: &input, OutputPrice: &expensive},
		}},
	})
	require.Len(t, got.logs, 2)
	require.Equal(t, []float64{1, 1, 6}, []float64{<-observed, <-observed, <-observed})
	require.Eventually(t, func() bool { return cache.count() == 0 }, time.Second, time.Millisecond)
}

func TestOpenAIWSInflightReservation_UnpricedThenPricedGeneration(t *testing.T) {
	for _, transport := range []string{service.OpenAIWSIngressModePassthrough, service.OpenAIWSIngressModeDedicated, "http-bridge"} {
		t.Run(transport, func(t *testing.T) {
			cache := newHandlerInflightCache(100)
			observed := make(chan float64, 2)
			zero, expensive := 0.0, 0.006
			mode := transport
			if transport == "http-bridge" {
				mode = service.OpenAIWSIngressModeDedicated
			}
			got := runOpenAIResponsesWebSocketUsageLogCase(t, openAIResponsesWSUsageLogCase{
				inflightCache: cache, ingressMode: mode, httpBridge: transport == "http-bridge",
				observeUpstream: func([]byte) { observed <- cache.total() },
				firstPayload:    `{"type":"response.create","model":"gpt-5.4","max_output_tokens":1000,"input":"first"}`,
				secondPayload:   `{"type":"response.create","model":"gpt-6-astra","max_output_tokens":1000,"input":"second"}`,
				group: &service.Group{ID: 4201, Platform: service.PlatformOpenAI, RateMultiplier: 1, ModelPricing: []service.ChannelModelPricing{
					{Models: []string{"gpt-5.4"}, InputPrice: &zero, OutputPrice: &zero},
					{Models: []string{"gpt-6-astra"}, InputPrice: &zero, OutputPrice: &expensive},
				}},
			})
			require.Len(t, got.logs, 2)
			require.Equal(t, []float64{0, 6}, []float64{<-observed, <-observed})
			require.Eventually(t, func() bool { return cache.count() == 0 }, time.Second, time.Millisecond)
		})
	}
}

func TestInflightUsageTask_LifecycleAcrossBothHandlers(t *testing.T) {
	for _, controller := range []string{"gateway", "openai"} {
		for _, mode := range []string{"inline", "stopped", "drop", "mandatory", "panic", "cancel"} {
			t.Run(controller+"/"+mode, func(t *testing.T) {
				cache := newHandlerInflightCache(100)
				cfg := &config.Config{}
				cfg.Billing.InflightReservation = config.InflightReservationConfig{Enabled: true, TTLSeconds: 60}
				billing := service.NewBillingCacheService(cache, nil, nil, nil, nil, nil, cfg, nil)
				t.Cleanup(billing.Stop)
				reservation, err := billing.ReserveInflight(context.Background(), &service.User{ID: 1}, nil, nil, 6)
				require.NoError(t, err)
				t.Cleanup(reservation.HandlerDone)
				parent, cancel := context.WithCancel(service.WithInflightReservation(context.Background(), reservation))
				defer cancel()
				var pool *service.UsageRecordWorkerPool
				if mode != "inline" && mode != "cancel" {
					pool = service.NewUsageRecordWorkerPoolWithOptions(service.UsageRecordWorkerPoolOptions{WorkerCount: 1, QueueSize: 1, TaskTimeout: time.Second, OverflowPolicy: config.UsageRecordOverflowPolicyDrop})
					t.Cleanup(pool.Stop)
				}
				if mode == "stopped" {
					pool.Stop()
				}
				if mode == "drop" || mode == "mandatory" {
					started, release := make(chan struct{}), make(chan struct{})
					t.Cleanup(func() { close(release) })
					pool.Submit(func(context.Context) { close(started); <-release })
					<-started
					pool.Submit(func(context.Context) { <-release })
				}
				if mode == "cancel" {
					cancel()
				}
				var calls atomic.Int32
				validContext := make(chan bool, 1)
				task := func(ctx context.Context) {
					calls.Add(1)
					validContext <- ctx.Err() == nil && service.InflightReservationFromContext(ctx) == reservation
					if mode == "panic" {
						panic("local lifecycle fixture")
					}
					_ = cache.DeductUserBalance(ctx, 1, 2)
				}
				if controller == "gateway" {
					h := &GatewayHandler{usageRecordWorkerPool: pool}
					if mode == "mandatory" {
						h.submitMandatoryUsageRecordTask(parent, task)
					} else {
						h.submitUsageRecordTask(parent, task)
					}
				} else {
					h := &OpenAIGatewayHandler{usageRecordWorkerPool: pool}
					if mode == "mandatory" {
						h.submitMandatoryUsageRecordTask(parent, task)
					} else {
						h.submitUsageRecordTask(parent, task)
					}
				}
				reservation.HandlerDone()
				require.Eventually(t, func() bool { return cache.count() == 0 }, time.Second, time.Millisecond)
				if mode == "drop" {
					require.Zero(t, calls.Load())
				} else {
					require.Equal(t, int32(1), calls.Load())
					require.True(t, <-validContext, "detached task owns the exact handle, even after request cancellation")
				}
			})
		}
	}
}

func TestInflightUsageTask_SessionHolderReplacementKeepsQueuedHandle(t *testing.T) {
	cache := newHandlerInflightCache(100)
	cfg := &config.Config{}
	cfg.Billing.InflightReservation = config.InflightReservationConfig{Enabled: true, TTLSeconds: 60}
	billing := service.NewBillingCacheService(cache, nil, nil, nil, nil, nil, cfg, nil)
	t.Cleanup(billing.Stop)
	user := &service.User{ID: 1}
	old, err := billing.ReserveInflight(context.Background(), user, nil, nil, 2)
	require.NoError(t, err)
	var holder atomic.Pointer[service.InflightReservation]
	holder.Store(old)
	parent := service.WithInflightReservationHolder(context.Background(), &holder)
	task, abandon := wrapUsageRecordTaskContext(parent, func(ctx context.Context) {
		require.Same(t, old, service.InflightReservationFromContext(ctx))
	})
	t.Cleanup(abandon)
	old.HandlerDone()
	next, err := billing.ReserveInflight(parent, user, nil, nil, 6)
	require.NoError(t, err)
	t.Cleanup(next.HandlerDone)
	require.NotSame(t, old, next)
	holder.Store(next)
	task(context.Background())
	abandon()
	require.Equal(t, 1, cache.count(), "late old usage completion must not release the holder's replacement")
	require.Equal(t, 6.0, cache.total())
	next.HandlerDone()
	require.Zero(t, cache.count())
}

func TestWrapUsageRecordTaskContext_HandsReservationToBillingTask(t *testing.T) {
	cache := newHandlerInflightCache(1)
	cfg := &config.Config{}
	cfg.Billing.InflightReservation = config.InflightReservationConfig{Enabled: true, TTLSeconds: 60}
	billing := service.NewBillingCacheService(cache, nil, nil, nil, nil, nil, cfg, nil)
	t.Cleanup(billing.Stop)
	apiKey := &service.APIKey{User: &service.User{ID: 5}}

	c := newInflightTestGinContext()
	done, err := reserveInflightBalance(c, billing, &countingEstimator{cost: 0.9, priced: true}, apiKey, nil, tokenInflightEstimate("m", nil))
	require.NoError(t, err)
	require.Equal(t, 1, cache.count())

	ran := false
	task, abandon := wrapUsageRecordTaskContext(c.Request.Context(), func(context.Context) { ran = true })
	done() // handler returns; billing still pending
	require.Equal(t, 1, cache.count(), "reservation held until the billing task finishes")
	task(context.Background())
	require.True(t, ran)
	require.Equal(t, 0, cache.count())
	abandon() // idempotent with the task's own done

	// Dropped task: the submitter abandons it and the reservation is released.
	c2 := newInflightTestGinContext()
	done2, err := reserveInflightBalance(c2, billing, &countingEstimator{cost: 0.9, priced: true}, apiKey, nil, tokenInflightEstimate("m", nil))
	require.NoError(t, err)
	_, abandon2 := wrapUsageRecordTaskContext(c2.Request.Context(), func(context.Context) {})
	done2()
	require.Equal(t, 1, cache.count())
	abandon2()
	require.Equal(t, 0, cache.count())
}

// 新接入的端点（独立 web_search）：在途预留超过余额时拒绝，且不残留预留。
func TestWebSearch_RejectsWhenInflightExceedsBalance(t *testing.T) {
	cache := newHandlerInflightCache(1.5)
	cfg := &config.Config{}
	cfg.Billing.InflightReservation = config.InflightReservationConfig{Enabled: true, TTLSeconds: 60}
	billingCache := service.NewBillingCacheService(cache, nil, nil, nil, nil, nil, cfg, nil)
	t.Cleanup(billingCache.Stop)
	billing := service.NewBillingService(cfg, nil)
	gw := service.NewGatewayService(
		nil, nil, nil, nil, nil, nil, nil, nil, cfg, nil, nil, billing, nil, nil,
		nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, service.NewModelPricingResolver(nil, billing), nil, nil, nil,
	)
	h := &GatewayHandler{gatewayService: gw, billingCacheService: billingCache}

	groupID := int64(3)
	perK := 1000.0 // $1 per search
	apiKey := &service.APIKey{
		ID: 9, User: &service.User{ID: 42, Balance: 1.5}, GroupID: &groupID,
		Group: &service.Group{ID: groupID, Platform: service.PlatformGrok, RateMultiplier: 1, SearchPricePer1k: &perK},
	}

	// Another in-flight request of this user already holds $1.
	held, err := billingCache.ReserveInflight(context.Background(), apiKey.User, apiKey.Group, nil, 1.0)
	require.NoError(t, err)
	defer held.HandlerDone()

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/web_search", bytes.NewBufferString(`{"query":"sub2api"}`))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set(string(middleware2.ContextKeyAPIKey), apiKey)

	h.WebSearch(c)

	require.NotEqual(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), "balance")
	require.Equal(t, 1, cache.count(), "rejected request must not leave a reservation behind")
}
