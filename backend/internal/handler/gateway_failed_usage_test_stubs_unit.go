//go:build unit

package handler

import (
	"context"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

type handlerSettingRepoStub struct{}

func (s *handlerSettingRepoStub) Get(context.Context, string) (*service.Setting, error) { return nil, service.ErrSettingNotFound }

func (s *handlerSettingRepoStub) GetValue(_ context.Context, key string) (string, error) {
	if key == service.SettingKeyGatewayRuntimeSettings {
		return "", service.ErrSettingNotFound
	}
	return "", nil
}

func (s *handlerSettingRepoStub) Set(context.Context, string, string) error { return nil }

func (s *handlerSettingRepoStub) GetMultiple(context.Context, []string) (map[string]string, error) {
	return map[string]string{}, nil
}

func (s *handlerSettingRepoStub) SetMultiple(context.Context, map[string]string) error { return nil }

func (s *handlerSettingRepoStub) GetAll(context.Context) (map[string]string, error) { return map[string]string{}, nil }

func (s *handlerSettingRepoStub) Delete(context.Context, string) error { return nil }

func newHandlerTestSettingService(cfg *config.Config) *service.SettingService {
	return service.NewSettingService(&handlerSettingRepoStub{}, cfg)
}

type stubAccountRepo struct {
	service.AccountRepository
	accounts map[int64]*service.Account
}

func (s *stubAccountRepo) GetByID(_ context.Context, id int64) (*service.Account, error) {
	if s.accounts == nil {
		return nil, service.ErrAccountNotFound
	}
	if account, ok := s.accounts[id]; ok {
		return account, nil
	}
	return nil, service.ErrAccountNotFound
}

func (s *stubAccountRepo) listAll() []service.Account {
	out := make([]service.Account, 0, len(s.accounts))
	for _, account := range s.accounts {
		if account != nil {
			out = append(out, *account)
		}
	}
	return out
}

func (s *stubAccountRepo) ListSchedulable(_ context.Context) ([]service.Account, error) {
	return s.listAll(), nil
}

func (s *stubAccountRepo) ListSchedulableByGroupID(_ context.Context, _ int64) ([]service.Account, error) {
	return s.listAll(), nil
}

func (s *stubAccountRepo) ListSchedulableByPlatform(_ context.Context, platform string) ([]service.Account, error) {
	return filterAccountsByPlatform(s.listAll(), platform), nil
}

func (s *stubAccountRepo) ListSchedulableByGroupIDAndPlatform(_ context.Context, _ int64, platform string) ([]service.Account, error) {
	return filterAccountsByPlatform(s.listAll(), platform), nil
}

func (s *stubAccountRepo) ListSchedulableByPlatforms(_ context.Context, platforms []string) ([]service.Account, error) {
	return filterAccountsByPlatforms(s.listAll(), platforms), nil
}

func (s *stubAccountRepo) ListSchedulableByGroupIDAndPlatforms(_ context.Context, _ int64, platforms []string) ([]service.Account, error) {
	return filterAccountsByPlatforms(s.listAll(), platforms), nil
}

func (s *stubAccountRepo) ListSchedulableUngroupedByPlatform(_ context.Context, platform string) ([]service.Account, error) {
	return filterAccountsByPlatform(s.listAll(), platform), nil
}

func (s *stubAccountRepo) ListSchedulableUngroupedByPlatforms(_ context.Context, platforms []string) ([]service.Account, error) {
	return filterAccountsByPlatforms(s.listAll(), platforms), nil
}

func (s *stubAccountRepo) ListTempUnschedulableByPlatform(_ context.Context, platform string, _ time.Time) ([]service.Account, error) {
	return filterAccountsByPlatform(s.listAll(), platform), nil
}

func (s *stubAccountRepo) UpdateLastUsed(_ context.Context, _ int64) error { return nil }

func (s *stubAccountRepo) BatchUpdateLastUsed(_ context.Context, _ map[int64]time.Time) error { return nil }

func (s *stubAccountRepo) SetRateLimited(_ context.Context, _ int64, _ time.Time) error { return nil }

func (s *stubAccountRepo) SetModelRateLimit(_ context.Context, _ int64, _ string, _ time.Time) error {
	return nil
}

func (s *stubAccountRepo) SetOverloaded(_ context.Context, _ int64, _ time.Time) error { return nil }

func (s *stubAccountRepo) SetTempUnschedulable(_ context.Context, _ int64, _ time.Time, _ string) error {
	return nil
}

func (s *stubAccountRepo) ClearTempUnschedulable(_ context.Context, _ int64) error { return nil }

func (s *stubAccountRepo) ClearRateLimit(_ context.Context, _ int64) error { return nil }

func (s *stubAccountRepo) ClearModelRateLimits(_ context.Context, _ int64) error { return nil }

type stubGroupRepo struct {
	service.GroupRepository
	group *service.Group
}

func (s *stubGroupRepo) GetByID(_ context.Context, id int64) (*service.Group, error) {
	if s.group != nil && s.group.ID == id {
		return s.group, nil
	}
	return nil, service.ErrGroupNotFound
}

func (s *stubGroupRepo) GetByIDLite(ctx context.Context, id int64) (*service.Group, error) {
	return s.GetByID(ctx, id)
}

type stubUsageLogRepo struct {
	service.UsageLogRepository
	lastLog *service.UsageLog
}

func (s *stubUsageLogRepo) Create(_ context.Context, log *service.UsageLog) (bool, error) {
	if log != nil {
		copied := *log
		s.lastLog = &copied
	}
	return true, nil
}

func (s *stubUsageLogRepo) PersistDetailBestEffort(_ context.Context, log *service.UsageLog) {
	if log != nil {
		copied := *log
		s.lastLog = &copied
	}
}

func filterAccountsByPlatform(accounts []service.Account, platform string) []service.Account {
	filtered := make([]service.Account, 0, len(accounts))
	for _, account := range accounts {
		if account.Platform == platform {
			filtered = append(filtered, account)
		}
	}
	return filtered
}

func filterAccountsByPlatforms(accounts []service.Account, platforms []string) []service.Account {
	allowed := make(map[string]struct{}, len(platforms))
	for _, platform := range platforms {
		allowed[platform] = struct{}{}
	}
	filtered := make([]service.Account, 0, len(accounts))
	for _, account := range accounts {
		if _, ok := allowed[account.Platform]; ok {
			filtered = append(filtered, account)
		}
	}
	return filtered
}
