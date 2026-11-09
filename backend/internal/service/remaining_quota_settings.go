package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const SettingKeyRemainingQuotaExclusions = "admin_users_remaining_quota_excluded_user_ids"

func normalizeRemainingQuotaExclusions(ids []int64) ([]int64, error) {
	if len(ids) > 1000 {
		return nil, infraerrors.BadRequest("INVALID_QUOTA_EXCLUSIONS", "excluded_user_ids must contain at most 1000 IDs")
	}
	result := make([]int64, 0, len(ids))
	seen := make(map[int64]bool, len(ids))
	for _, id := range ids {
		if id <= 0 || id > 9007199254740991 {
			return nil, infraerrors.BadRequest("INVALID_QUOTA_EXCLUSIONS", "excluded_user_ids must contain positive JavaScript-safe integer IDs")
		}
		if !seen[id] {
			result = append(result, id)
			seen[id] = true
		}
	}
	return result, nil
}

// Read directly so all admin sessions and application instances share one saved list.
func (s *SettingService) GetRemainingQuotaExclusions(ctx context.Context) ([]int64, error) {
	if s == nil || s.settingRepo == nil {
		return nil, fmt.Errorf("remaining quota settings unavailable")
	}
	raw, err := s.settingRepo.GetValue(ctx, SettingKeyRemainingQuotaExclusions)
	if errors.Is(err, ErrSettingNotFound) {
		return []int64{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read remaining quota exclusions: %w", err)
	}
	var ids []int64
	if err := json.Unmarshal([]byte(raw), &ids); err != nil {
		return nil, fmt.Errorf("decode remaining quota exclusions: %w", err)
	}
	if ids == nil {
		return nil, fmt.Errorf("remaining quota exclusions must be an array")
	}
	normalized, err := normalizeRemainingQuotaExclusions(ids)
	if err != nil {
		return nil, fmt.Errorf("invalid saved remaining quota exclusions: %v", err)
	}
	return normalized, nil
}

func (s *SettingService) SetRemainingQuotaExclusions(ctx context.Context, ids []int64) ([]int64, error) {
	normalized, err := normalizeRemainingQuotaExclusions(ids)
	if err != nil {
		return nil, err
	}
	if s == nil || s.settingRepo == nil {
		return nil, fmt.Errorf("remaining quota settings unavailable")
	}
	data, err := json.Marshal(normalized)
	if err != nil {
		return nil, fmt.Errorf("encode remaining quota exclusions: %w", err)
	}
	if err := s.settingRepo.Set(ctx, SettingKeyRemainingQuotaExclusions, string(data)); err != nil {
		return nil, fmt.Errorf("save remaining quota exclusions: %w", err)
	}
	return normalized, nil
}
