package service

import (
	"context"
	"encoding/json"
	"errors"
	"maps"
	"math"
	"slices"
	"strings"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/modeltrace"
)

const ModelTraceEnabledExtraKey = "modeltrace_enabled"
const ModelTraceLatestExtraKey = "modeltrace_latest"
const ModelTraceIntervalExtraKey = "modeltrace_interval_minutes"
const ModelTraceModelExtraKey = "modeltrace_model"
const ModelTraceQuarantineEnabledExtraKey = "modeltrace_quarantine_enabled"
const ModelTraceQuarantinedExtraKey = "modeltrace_quarantined"
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
	if account == nil || account.Platform != PlatformOpenAI || !cfg.Enabled || !account.isSchedulable(true) {
		return false
	}
	enabled, _ := account.Extra[ModelTraceEnabledExtraKey].(bool)
	return enabled
}

func modelTraceModelForAccount(account *Account, cfg ModelTraceSettings) string {
	if account != nil {
		if model, ok := account.Extra[ModelTraceModelExtraKey].(string); ok && strings.TrimSpace(model) != "" {
			return strings.TrimSpace(model)
		}
	}
	return cfg.Model
}

func (s *ModelTraceService) autoEligibleForTarget(ctx context.Context, account *Account, cfg ModelTraceSettings, selectedModel string) bool {
	if !modelTraceAutoEligible(account, cfg) || account.isModelRateLimitedWithContext(ctx, selectedModel) {
		return false
	}
	ctx = withOpenAIQuotaAutoPauseSettings(ctx, s.settings.GetOpenAIQuotaAutoPauseSettings(ctx))
	paused, _ := shouldAutoPauseOpenAIAccountByQuota(ctx, account)
	return !paused
}

func modelTraceAutoModelChanged(account *Account, task *ModelTraceTask) bool {
	configured, configuredOK := account.Extra[ModelTraceModelExtraKey].(string)
	configured = strings.TrimSpace(configured)
	return (task.ModelOverride && (!configuredOK || configured == "")) || (configured != "" && configured != task.Model)
}

// ValidateModelTraceAccountExtra rejects malformed user configuration before it reaches the scheduler.
func ValidateModelTraceAccountExtra(extra map[string]any) error {
	if value, ok := extra[ModelTraceIntervalExtraKey]; ok && value != nil {
		var minutes float64
		switch n := value.(type) {
		case float64:
			minutes = n
		case int:
			minutes = float64(n)
		case json.Number:
			parsed, err := n.Float64()
			if err != nil {
				return infraerrors.BadRequest("MODELTRACE_INTERVAL_INVALID", "account interval must be an integer between 5 and 10080")
			}
			minutes = parsed
		default:
			return infraerrors.BadRequest("MODELTRACE_INTERVAL_INVALID", "account interval must be an integer between 5 and 10080")
		}
		if math.IsNaN(minutes) || math.IsInf(minutes, 0) || math.Trunc(minutes) != minutes || minutes < 5 || minutes > 10080 {
			return infraerrors.BadRequest("MODELTRACE_INTERVAL_INVALID", "account interval must be an integer between 5 and 10080")
		}
	}
	if value, ok := extra[ModelTraceQuarantineEnabledExtraKey]; ok && value != nil {
		if _, valid := value.(bool); !valid {
			return infraerrors.BadRequest("MODELTRACE_QUARANTINE_INVALID", "quarantine policy must be a boolean")
		}
	}
	if value, ok := extra[ModelTraceModelExtraKey]; ok && value != nil {
		model, valid := value.(string)
		if !valid || (strings.TrimSpace(model) != "" && !slices.Contains(modeltrace.Models(), strings.TrimSpace(model))) {
			return infraerrors.BadRequest("MODELTRACE_MODEL_INVALID", "account model must be empty or supported by the fingerprint library")
		}
	}
	return nil
}

// Managed summaries cannot be erased or forged by account edits, including stale DTOs.
func MergeModelTraceExtra(incoming, current map[string]any) map[string]any {
	if incoming == nil {
		incoming = make(map[string]any)
	}
	delete(incoming, ModelTraceLatestExtraKey)
	delete(incoming, ModelTraceQuarantinedExtraKey)
	if latest, ok := current[ModelTraceLatestExtraKey]; ok {
		incoming[ModelTraceLatestExtraKey] = latest
	}
	if _, explicit := incoming[ModelTraceEnabledExtraKey]; !explicit {
		if enabled, ok := current[ModelTraceEnabledExtraKey]; ok {
			incoming[ModelTraceEnabledExtraKey] = enabled
		}
	}
	if _, explicit := incoming[ModelTraceQuarantineEnabledExtraKey]; !explicit {
		if enabled, ok := current[ModelTraceQuarantineEnabledExtraKey]; ok {
			incoming[ModelTraceQuarantineEnabledExtraKey] = enabled
		}
	}
	if interval, explicit := incoming[ModelTraceIntervalExtraKey]; !explicit {
		if saved, ok := current[ModelTraceIntervalExtraKey]; ok {
			incoming[ModelTraceIntervalExtraKey] = saved
		}
	} else if interval == nil {
		delete(incoming, ModelTraceIntervalExtraKey)
	}
	if model, explicit := incoming[ModelTraceModelExtraKey]; !explicit {
		if saved, ok := current[ModelTraceModelExtraKey]; ok {
			incoming[ModelTraceModelExtraKey] = saved
		}
	} else {
		value, ok := model.(string)
		if model == nil || (ok && strings.TrimSpace(value) == "") {
			delete(incoming, ModelTraceModelExtraKey)
		}
	}
	if incoming[ModelTraceQuarantineEnabledExtraKey] == true && current[ModelTraceQuarantinedExtraKey] == true {
		incoming[ModelTraceQuarantinedExtraKey] = true
	}
	return incoming
}

func modelTraceExtraWithoutUnsubmittedPolicy(extra map[string]any) map[string]any {
	extra = maps.Clone(extra)
	delete(extra, ModelTraceQuarantineEnabledExtraKey)
	delete(extra, ModelTraceIntervalExtraKey)
	delete(extra, ModelTraceModelExtraKey)
	delete(extra, ModelTraceQuarantinedExtraKey)
	return extra
}
