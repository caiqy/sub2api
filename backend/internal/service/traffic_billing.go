package service

import (
	"context"
	"fmt"
	"math"
	"sync/atomic"
)

// TrafficBillingSnapshot never retains request bodies or a Gin context.
// DeliveryFailed carries only the shared HTTP writer failure signal.
type TrafficBillingSnapshot struct {
	Enabled        bool
	Consent        bool
	InputBytes     *int64
	ThresholdBytes int64
	PricePerMB     float64
	Successful     bool
	Excluded       bool
	DeliveryFailed *atomic.Bool
}

type trafficBillingStartKey struct{}
type trafficBillingStart struct {
	Snapshot TrafficBillingSnapshot
	Err      error
}

// CaptureTrafficBillingStart freezes settings after authentication without changing
// admission for routes which never produce a synchronous bill.
func CaptureTrafficBillingStart(ctx context.Context, settings *SettingService, user *User) context.Context {
	snapshot, err := newTrafficBillingSnapshot(ctx, settings, user, nil)
	return context.WithValue(ctx, trafficBillingStartKey{}, trafficBillingStart{snapshot, err})
}

func newTrafficBillingSnapshot(ctx context.Context, settings *SettingService, user *User, inputBytes *int64) (TrafficBillingSnapshot, error) {
	s := TrafficBillingSnapshot{ThresholdBytes: 10_000_000, InputBytes: inputBytes}
	if existing, ok := TrafficBillingSnapshotFromContext(ctx); ok && existing.Excluded {
		return existing, nil
	}
	if start, ok := ctx.Value(trafficBillingStartKey{}).(trafficBillingStart); ok {
		start.Snapshot.InputBytes = inputBytes
		return start.Snapshot, start.Err
	}
	if user != nil {
		s.Consent = user.AllowLargeRequests
	}
	if settings == nil {
		return s, nil
	}
	config, err := settings.GetTrafficBillingSettings(ctx)
	if err != nil {
		return s, fmt.Errorf("traffic billing settings: %w: %v", ErrBillingServiceUnavailable, err)
	}
	s.Enabled, s.ThresholdBytes, s.PricePerMB = config.Enabled, int64(config.ThresholdMB*1e6), config.PricePerMB
	return s, nil
}

func (s *GatewayService) NewTrafficBillingSnapshot(ctx context.Context, user *User, inputBytes *int64) (TrafficBillingSnapshot, error) {
	if _, frozen := ctx.Value(trafficBillingStartKey{}).(trafficBillingStart); !frozen && s.userRepo != nil && user != nil && s.settingService != nil {
		fresh, err := s.userRepo.GetByID(ctx, user.ID)
		if err != nil {
			return TrafficBillingSnapshot{}, fmt.Errorf("traffic billing consent: %w: %v", ErrBillingServiceUnavailable, err)
		}
		user = fresh
	}
	return newTrafficBillingSnapshot(ctx, s.settingService, user, inputBytes)
}
func (s *OpenAIGatewayService) NewTrafficBillingSnapshot(ctx context.Context, user *User, inputBytes *int64) (TrafficBillingSnapshot, error) {
	if _, frozen := ctx.Value(trafficBillingStartKey{}).(trafficBillingStart); !frozen && s.userRepo != nil && user != nil && s.settingService != nil {
		fresh, err := s.userRepo.GetByID(ctx, user.ID)
		if err != nil {
			return TrafficBillingSnapshot{}, fmt.Errorf("traffic billing consent: %w: %v", ErrBillingServiceUnavailable, err)
		}
		user = fresh
	}
	return newTrafficBillingSnapshot(ctx, s.settingService, user, inputBytes)
}

type trafficBillingContextKey struct{}
type trafficInputBytesContextKey struct{}

func WithTrafficInputBytes(ctx context.Context, n int64) context.Context {
	return context.WithValue(ctx, trafficInputBytesContextKey{}, n)
}
func TrafficInputBytesFromContext(ctx context.Context) *int64 {
	if ctx == nil {
		return nil
	}
	n, ok := ctx.Value(trafficInputBytesContextKey{}).(int64)
	if !ok {
		return nil
	}
	return &n
}

type TrafficBillingConsentError struct{ Snapshot TrafficBillingSnapshot }

func (e *TrafficBillingConsentError) Error() string { return e.Snapshot.ConsentMessage() }

func WithTrafficBillingSnapshot(ctx context.Context, snapshot TrafficBillingSnapshot) context.Context {
	if snapshot.InputBytes != nil {
		n := *snapshot.InputBytes
		snapshot.InputBytes = &n
	}
	return context.WithValue(ctx, trafficBillingContextKey{}, snapshot)
}

func TrafficBillingSnapshotFromContext(ctx context.Context) (TrafficBillingSnapshot, bool) {
	if ctx == nil {
		return TrafficBillingSnapshot{}, false
	}
	s, ok := ctx.Value(trafficBillingContextKey{}).(TrafficBillingSnapshot)
	if s.DeliveryFailed != nil && s.DeliveryFailed.Load() {
		s.Successful = false
	}
	return s, ok
}

func (s TrafficBillingSnapshot) ConsentRequired() bool {
	return !s.Excluded && s.Enabled && !s.Consent && s.InputBytes != nil && *s.InputBytes > s.ThresholdBytes
}

func (s TrafficBillingSnapshot) ConsentMessage() string {
	return fmt.Sprintf("Enable large requests in your profile to accept traffic charges (threshold %d bytes, price %g $/MB)", s.ThresholdBytes, s.PricePerMB)
}

func (s TrafficBillingSnapshot) Cost() (float64, string) {
	switch {
	case s.Excluded:
		return 0, "excluded"
	case !s.Enabled:
		return 0, "disabled"
	case s.InputBytes == nil || *s.InputBytes < 0:
		return 0, "input_unknown"
	case !s.Successful:
		return 0, "delivery_incomplete"
	case *s.InputBytes <= s.ThresholdBytes:
		return 0, "below_threshold"
	case !s.Consent:
		return 0, "consent_required"
	default:
		return float64(*s.InputBytes) * 2 / 1e6 * s.PricePerMB, "charged"
	}
}

func (s TrafficBillingSnapshot) ValidateCost() error {
	amount, _ := s.Cost()
	if math.IsNaN(amount) || math.IsInf(amount, 0) || amount < 0 || amount >= 1e10 {
		return fmt.Errorf("traffic billing amount outside numeric(20,10): input_bytes=%v price_per_mb=%g: %w", s.InputBytes, s.PricePerMB, ErrBillingServiceUnavailable)
	}
	return nil
}

func applyTrafficBilling(ctx context.Context, log *UsageLog, cost *CostBreakdown, delivered bool) error {
	s, ok := TrafficBillingSnapshotFromContext(ctx)
	if !ok || s.Excluded || log == nil || cost == nil {
		return nil
	}
	s.Successful = s.Successful && delivered
	if err := s.ValidateCost(); err != nil {
		return err
	}
	amount, reason := s.Cost()
	log.TrafficCost = amount
	log.TrafficInputBytes = s.InputBytes
	log.TrafficThresholdBytes = &s.ThresholdBytes
	log.TrafficPricePerMB = &s.PricePerMB
	log.TrafficBillingReason = reason
	if amount > 0 && log.AccountStatsCost == nil {
		model := cost.TotalCost
		log.AccountStatsCost = &model
	}
	log.TotalCost += amount
	log.ActualCost += amount
	cost.ActualCost += amount
	return nil
}
