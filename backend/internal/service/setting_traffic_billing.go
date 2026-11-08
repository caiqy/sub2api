package service

import (
	"context"
	"math"
	"strconv"
	"sync"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

// Serialize read/merge/validate/write across partial settings updates.
var trafficBillingSettingsMu sync.Mutex

type TrafficBillingSettings struct {
	Enabled     bool
	ThresholdMB float64
	PricePerMB  float64
}

func parseTrafficBillingSettings(values map[string]string) SystemSettings {
	threshold := 10.0
	if v, err := strconv.ParseFloat(values[SettingKeyTrafficBillingThresholdMB], 64); err == nil {
		threshold = v
	}
	price, _ := strconv.ParseFloat(values[SettingKeyTrafficBillingPricePerMB], 64)
	return SystemSettings{TrafficBillingEnabled: values[SettingKeyTrafficBillingEnabled] == "true", TrafficBillingThresholdMB: threshold, TrafficBillingPricePerMB: price}
}

func (s *SettingService) GetTrafficBillingSettings(ctx context.Context) (*TrafficBillingSettings, error) {
	values, err := s.settingRepo.GetMultiple(ctx, []string{SettingKeyTrafficBillingEnabled, SettingKeyTrafficBillingThresholdMB, SettingKeyTrafficBillingPricePerMB})
	if err != nil {
		return nil, err
	}
	cfg := parseTrafficBillingSettings(values)
	return &TrafficBillingSettings{Enabled: cfg.TrafficBillingEnabled, ThresholdMB: cfg.TrafficBillingThresholdMB, PricePerMB: cfg.TrafficBillingPricePerMB}, nil
}

func (s *SettingService) validateTrafficBillingUpdates(ctx context.Context, updates map[string]string) error {
	keys := []string{SettingKeyTrafficBillingEnabled, SettingKeyTrafficBillingThresholdMB, SettingKeyTrafficBillingPricePerMB}
	changed := false
	for _, key := range keys {
		if _, ok := updates[key]; ok {
			changed = true
		}
	}
	if !changed {
		return nil
	}
	values, err := s.settingRepo.GetMultiple(ctx, keys)
	if err != nil {
		return err
	}
	if values == nil {
		values = make(map[string]string)
	}
	for _, key := range keys {
		if v, ok := updates[key]; ok {
			values[key] = v
		}
	}
	return ValidateTrafficBillingSettings(values)
}

// ValidateTrafficBillingSettings validates the merged values under the repository transaction lock.
func ValidateTrafficBillingSettings(values map[string]string) error {
	for _, key := range []string{SettingKeyTrafficBillingThresholdMB, SettingKeyTrafficBillingPricePerMB} {
		if raw, ok := values[key]; ok {
			if _, err := strconv.ParseFloat(raw, 64); err != nil {
				return infraerrors.BadRequest("INVALID_TRAFFIC_BILLING_SETTINGS", key+" must be a finite number")
			}
		}
	}
	cfg := parseTrafficBillingSettings(values)
	t, p := cfg.TrafficBillingThresholdMB, cfg.TrafficBillingPricePerMB
	if math.IsNaN(t) || math.IsInf(t, 0) || t < 0.000001 || t >= float64(math.MaxInt64)/1_000_000 || math.IsNaN(p) || math.IsInf(p, 0) || p < 0 || cfg.TrafficBillingEnabled && p <= 0 {
		return infraerrors.BadRequest("INVALID_TRAFFIC_BILLING_SETTINGS", "traffic threshold must be finite and positive; price must be finite and nonnegative, and positive when enabled")
	}
	return nil
}
