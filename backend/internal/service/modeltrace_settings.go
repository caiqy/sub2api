package service

import (
	"context"
	"encoding/json"
	"errors"
	"slices"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/modeltrace"
)

const ModelTraceEnabledExtraKey = "modeltrace_enabled"
const ModelTraceLatestExtraKey = "modeltrace_latest"
const modelTraceSettingsKey = "modeltrace_settings"

type ModelTraceSettings struct {
	Enabled         bool   `json:"enabled"`
	Model           string `json:"model"`
	Rounds          int    `json:"rounds"`
	IntervalMinutes int    `json:"interval_minutes"`
}

func validateModelTraceRequest(model string, rounds int) error {
	if !slices.Contains(modeltrace.Models(), model) {
		return infraerrors.BadRequest("MODELTRACE_MODEL_UNSUPPORTED", "model is not supported by the fingerprint library")
	}
	if rounds < 1 || rounds > 3 {
		return infraerrors.BadRequest("MODELTRACE_ROUNDS_INVALID", "rounds must be between 1 and 3")
	}
	return nil
}

func (s *SettingService) GetModelTraceSettings(ctx context.Context) (ModelTraceSettings, error) {
	cfg := ModelTraceSettings{Model: "gpt-6-astra", Rounds: 1, IntervalMinutes: 60}
	raw, err := s.settingRepo.GetValue(ctx, modelTraceSettingsKey)
	if errors.Is(err, ErrSettingNotFound) {
		return cfg, nil
	}
	if err != nil {
		return cfg, err
	}
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		return ModelTraceSettings{}, errors.New("invalid stored ModelTrace settings")
	}
	if err := validateModelTraceSettings(cfg); err != nil {
		return ModelTraceSettings{}, err
	}
	return cfg, nil
}

func validateModelTraceSettings(cfg ModelTraceSettings) error {
	if err := validateModelTraceRequest(cfg.Model, cfg.Rounds); err != nil {
		return err
	}
	if cfg.IntervalMinutes < 5 || cfg.IntervalMinutes > 10080 {
		return infraerrors.BadRequest("MODELTRACE_INTERVAL_INVALID", "interval_minutes must be between 5 and 10080")
	}
	return nil
}

func (s *SettingService) SetModelTraceSettings(ctx context.Context, cfg ModelTraceSettings) error {
	if err := validateModelTraceSettings(cfg); err != nil {
		return err
	}
	raw, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	return s.settingRepo.Set(ctx, modelTraceSettingsKey, string(raw))
}

func modelTraceAutoEligible(account *Account, cfg ModelTraceSettings) bool {
	if account == nil || account.Platform != PlatformOpenAI || !cfg.Enabled || !account.IsSchedulable() {
		return false
	}
	enabled, _ := account.Extra[ModelTraceEnabledExtraKey].(bool)
	return enabled
}

// Managed summaries cannot be erased or forged by account edits, including stale DTOs.
func MergeModelTraceExtra(incoming, current map[string]any) map[string]any {
	if incoming == nil {
		incoming = make(map[string]any)
	}
	delete(incoming, ModelTraceLatestExtraKey)
	if latest, ok := current[ModelTraceLatestExtraKey]; ok {
		incoming[ModelTraceLatestExtraKey] = latest
	}
	if _, explicit := incoming[ModelTraceEnabledExtraKey]; !explicit {
		if enabled, ok := current[ModelTraceEnabledExtraKey]; ok {
			incoming[ModelTraceEnabledExtraKey] = enabled
		}
	}
	return incoming
}
