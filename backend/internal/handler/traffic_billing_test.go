package handler

import (
	"context"
	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
	"testing"
)

type trafficSnapshotEstimator struct {
	countingEstimator
	consent  bool
	captured int64
}

func (e *trafficSnapshotEstimator) NewTrafficBillingSnapshot(_ context.Context, _ *service.User, n *int64) (service.TrafficBillingSnapshot, error) {
	if n != nil {
		e.captured = *n
	}
	return service.TrafficBillingSnapshot{Enabled: true, Consent: e.consent, InputBytes: n, ThresholdBytes: 10, PricePerMB: 0.01}, nil
}
func TestTrafficBillingHTTPAdmissionAndFrozenBytes(t *testing.T) {
	key := &service.APIKey{User: &service.User{ID: 1}}
	c := newInflightTestGinContext()
	c.Request = c.Request.WithContext(service.WithTrafficInputBytes(c.Request.Context(), 11))
	estimator := &trafficSnapshotEstimator{}
	_, err := reserveInflightBalance(c, nil, estimator, key, nil, service.InflightEstimateRequest{Model: "m", BodyBytes: 0})
	require.IsType(t, &service.TrafficBillingConsentError{}, err)
	status, code, _, _ := billingErrorDetails(err)
	require.Equal(t, 413, status)
	require.Equal(t, "traffic_billing_consent_required", code)
	require.EqualValues(t, 11, estimator.captured)
	c = newInflightTestGinContext()
	c.Request = c.Request.WithContext(service.WithTrafficInputBytes(c.Request.Context(), 10))
	done, err := reserveInflightBalance(c, nil, estimator, key, nil, service.InflightEstimateRequest{Model: "m"})
	require.NoError(t, err)
	done()
	snap, ok := service.TrafficBillingSnapshotFromContext(c.Request.Context())
	require.True(t, ok)
	require.False(t, snap.ConsentRequired())
	markTrafficDeliveryFailed(c)
	frozen := usageRecordContext(c.Request.Context(), context.Background())
	snap, ok = service.TrafficBillingSnapshotFromContext(frozen)
	require.True(t, ok)
	require.False(t, snap.Successful)
}

func TestTrafficBillingReservationGrowsWithoutDoubleHold(t *testing.T) {
	cache := newHandlerInflightCache(10)
	cfg := &config.Config{}
	cfg.Billing.InflightReservation.Enabled = true
	billing := service.NewBillingCacheService(cache, nil, nil, nil, nil, nil, cfg, nil)
	t.Cleanup(billing.Stop)
	c := newInflightTestGinContext()
	n := int64(12_000_000)
	snap := service.TrafficBillingSnapshot{Enabled: true, Consent: true, InputBytes: &n, ThresholdBytes: 10_000_000, PricePerMB: 0.01}
	c.Request = c.Request.WithContext(service.WithTrafficBillingSnapshot(c.Request.Context(), snap))
	key := &service.APIKey{User: &service.User{ID: 1}}
	estimator := &countingEstimator{priced: true}
	done, err := reserveInflightBalance(c, billing, estimator, key, nil, service.InflightEstimateRequest{Model: "m"})
	require.NoError(t, err)
	require.InDelta(t, 0.24, cache.total(), 1e-10)
	n = 15_000_000
	c.Request = c.Request.WithContext(service.WithTrafficBillingSnapshot(c.Request.Context(), snap))
	done2, err := reserveInflightBalance(c, billing, estimator, key, nil, service.InflightEstimateRequest{Model: "m"})
	require.NoError(t, err)
	require.InDelta(t, 0.30, cache.total(), 1e-10)
	done2()
	done()
	require.Zero(t, cache.total())
}

func TestTrafficBillingCachedContentFixedCharge(t *testing.T) {
	cache := newHandlerInflightCache(10)
	cfg := &config.Config{}
	cfg.Billing.InflightReservation.Enabled = true
	billing := service.NewBillingCacheService(cache, nil, nil, nil, nil, nil, cfg, nil)
	t.Cleanup(billing.Stop)
	h := &GatewayHandler{billingCacheService: billing}
	key := &service.APIKey{User: &service.User{ID: 1}}
	n := int64(12_000_000)
	s := service.TrafficBillingSnapshot{Enabled: true, Consent: false, Successful: true, InputBytes: &n, ThresholdBytes: 10_000_000, PricePerMB: 0.01}
	c := newInflightTestGinContext()
	c.Request = c.Request.WithContext(service.WithTrafficBillingSnapshot(c.Request.Context(), s))
	_, err := h.reserveGeminiCachedContentCharge(c, key, nil, 0)
	var consent *service.TrafficBillingConsentError
	require.ErrorAs(t, err, &consent)
	require.Zero(t, cache.total())
	s.Consent = true
	c.Request = c.Request.WithContext(service.WithTrafficBillingSnapshot(c.Request.Context(), s))
	done, err := h.reserveGeminiCachedContentCharge(c, key, nil, 0)
	require.NoError(t, err)
	require.InDelta(t, 0.24, cache.total(), 1e-10)
	done()
	require.Zero(t, cache.total())
}

func TestTrafficBillingUnbilledVoiceExclusion(t *testing.T) {
	for _, endpoint := range []string{"custom-voices", "custom-voices/voice1", "custom-voices/voice1/sample"} {
		require.Empty(t, grokVoiceInflightEstimate(endpoint, []byte("large body")).Model)
	}
	require.Equal(t, "tts", grokVoiceInflightEstimate("tts", nil).Model)
	require.Equal(t, "stt", grokVoiceInflightEstimate("stt", nil).Model)
}

func TestTrafficBillingAsyncAndVideoExclusion(t *testing.T) {
	c := newInflightTestGinContext()
	c.Request = c.Request.WithContext(service.WithTrafficBillingSnapshot(c.Request.Context(), service.TrafficBillingSnapshot{Excluded: true}))
	estimator := &trafficSnapshotEstimator{}
	_, err := reserveInflightBalance(c, nil, estimator, &service.APIKey{User: &service.User{ID: 1}}, nil, service.InflightEstimateRequest{Model: "m", BodyBytes: 99})
	require.NoError(t, err)
	require.Zero(t, estimator.captured)
	c = newInflightTestGinContext()
	_, err = reserveInflightBalance(c, nil, estimator, &service.APIKey{User: &service.User{ID: 1}}, nil, service.InflightEstimateRequest{Model: "m", BodyBytes: 99, Kind: service.InflightEstimateVideo})
	require.NoError(t, err)
	require.Zero(t, estimator.captured)
}
