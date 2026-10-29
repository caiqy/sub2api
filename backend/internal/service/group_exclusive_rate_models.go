package service

import (
	"context"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/domain"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

type GroupExclusiveRateModels = domain.GroupExclusiveRateModels

func NormalizeGroupExclusiveRateModels(cfg GroupExclusiveRateModels) (GroupExclusiveRateModels, error) {
	models := make([]string, 0, len(cfg.Models))
	seen := make(map[string]bool, len(cfg.Models))
	for _, model := range cfg.Models {
		model = strings.TrimSpace(model)
		if model != "" && !seen[model] {
			models = append(models, model)
			seen[model] = true
		}
	}
	cfg.Models = models
	if cfg.Enabled && len(models) == 0 {
		return cfg, infraerrors.BadRequest("EXCLUSIVE_RATE_MODELS_REQUIRED", "开启专属倍率模型限制时至少选择一个模型")
	}
	return cfg, nil
}

func (s *GatewayService) ResolveUserGroupModelRateMultiplier(ctx context.Context, userID int64, group *Group, clientModel string) float64 {
	if group == nil {
		return 1
	}
	if !group.ExclusiveRateModels.AppliesTo(clientModel) {
		return group.RateMultiplier
	}
	return s.ResolveUserGroupRateMultiplier(ctx, userID, group.ID, group.RateMultiplier)
}

func (s *OpenAIGatewayService) ResolveUserGroupModelRateMultiplier(ctx context.Context, userID int64, group *Group, clientModel string) float64 {
	if group == nil {
		return 1
	}
	if !group.ExclusiveRateModels.AppliesTo(clientModel) {
		return group.RateMultiplier
	}
	return s.ResolveUserGroupRateMultiplier(ctx, userID, group.ID, group.RateMultiplier)
}

type rateClientModelCtxKey struct{}

func withRateClientModel(ctx context.Context, models []string) context.Context {
	if len(models) > 0 {
		return context.WithValue(ctx, rateClientModelCtxKey{}, models[0])
	}
	return ctx
}

func rateClientModelFromContext(ctx context.Context) string {
	if model, ok := ctx.Value(rateClientModelCtxKey{}).(string); ok {
		return model
	}
	model, _ := RequestedPublicModelFromContext(ctx)
	return model
}

func requestedRateModel(original, fallback string) string {
	if original != "" {
		return original
	}
	return fallback
}
