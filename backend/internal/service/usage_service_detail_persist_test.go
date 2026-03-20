//go:build unit

package service

import (
	"context"
	"database/sql"
	"testing"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/pkg/usagestats"
	"github.com/stretchr/testify/require"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	_ "modernc.org/sqlite"
)

type usageServiceDetailPersisterStub struct {
	createCtxHasTx  bool
	createInserted  bool
	persistCtxHasTx bool
	persistCtxErr   error
	persistDeadline bool
	persistTrace    string
	events          []string
}

type usageServiceDetailContextKey string

func (s *usageServiceDetailPersisterStub) Create(ctx context.Context, log *UsageLog) (bool, error) {
	s.createCtxHasTx = dbent.TxFromContext(ctx) != nil
	s.events = append(s.events, "create")
	log.ID = 42
	log.CreatedAt = time.Now().UTC()
	log.DetailSnapshot = (&UsageLogDetailSnapshot{RequestBody: `{"persist":"later"}`}).Normalize()
	if s.createInserted {
		return true, nil
	}
	return false, nil
}

func (s *usageServiceDetailPersisterStub) PersistDetailBestEffort(ctx context.Context, log *UsageLog) {
	s.persistCtxHasTx = dbent.TxFromContext(ctx) != nil
	s.persistCtxErr = ctx.Err()
	_, s.persistDeadline = ctx.Deadline()
	trace, _ := ctx.Value(usageServiceDetailContextKey("trace")).(string)
	s.persistTrace = trace
	s.events = append(s.events, "detail")
}

func (s *usageServiceDetailPersisterStub) GetByID(context.Context, int64) (*UsageLog, error) {
	return nil, nil
}
func (s *usageServiceDetailPersisterStub) GetDetailByUsageLogID(context.Context, int64) (*UsageLogDetail, error) {
	return nil, nil
}
func (s *usageServiceDetailPersisterStub) Delete(context.Context, int64) error { return nil }
func (s *usageServiceDetailPersisterStub) ListByUser(context.Context, int64, pagination.PaginationParams) ([]UsageLog, *pagination.PaginationResult, error) {
	return nil, nil, nil
}
func (s *usageServiceDetailPersisterStub) ListByAPIKey(context.Context, int64, pagination.PaginationParams) ([]UsageLog, *pagination.PaginationResult, error) {
	return nil, nil, nil
}
func (s *usageServiceDetailPersisterStub) ListByAccount(context.Context, int64, pagination.PaginationParams) ([]UsageLog, *pagination.PaginationResult, error) {
	return nil, nil, nil
}
func (s *usageServiceDetailPersisterStub) ListByUserAndTimeRange(context.Context, int64, time.Time, time.Time) ([]UsageLog, *pagination.PaginationResult, error) {
	return nil, nil, nil
}
func (s *usageServiceDetailPersisterStub) ListByAPIKeyAndTimeRange(context.Context, int64, time.Time, time.Time) ([]UsageLog, *pagination.PaginationResult, error) {
	return nil, nil, nil
}
func (s *usageServiceDetailPersisterStub) ListByAccountAndTimeRange(context.Context, int64, time.Time, time.Time) ([]UsageLog, *pagination.PaginationResult, error) {
	return nil, nil, nil
}
func (s *usageServiceDetailPersisterStub) ListByModelAndTimeRange(context.Context, string, time.Time, time.Time) ([]UsageLog, *pagination.PaginationResult, error) {
	return nil, nil, nil
}
func (s *usageServiceDetailPersisterStub) GetAccountWindowStats(context.Context, int64, time.Time) (*usagestats.AccountStats, error) {
	return nil, nil
}
func (s *usageServiceDetailPersisterStub) GetAccountTodayStats(context.Context, int64) (*usagestats.AccountStats, error) {
	return nil, nil
}
func (s *usageServiceDetailPersisterStub) GetDashboardStats(context.Context) (*usagestats.DashboardStats, error) {
	return nil, nil
}
func (s *usageServiceDetailPersisterStub) GetUsageTrendWithFilters(context.Context, time.Time, time.Time, string, int64, int64, int64, int64, string, *int16, *bool, *int8) ([]usagestats.TrendDataPoint, error) {
	return nil, nil
}
func (s *usageServiceDetailPersisterStub) GetModelStatsWithFilters(context.Context, time.Time, time.Time, int64, int64, int64, int64, *int16, *bool, *int8) ([]usagestats.ModelStat, error) {
	return nil, nil
}
func (s *usageServiceDetailPersisterStub) GetEndpointStatsWithFilters(context.Context, time.Time, time.Time, int64, int64, int64, int64, string, *int16, *bool, *int8) ([]usagestats.EndpointStat, error) {
	return nil, nil
}
func (s *usageServiceDetailPersisterStub) GetUpstreamEndpointStatsWithFilters(context.Context, time.Time, time.Time, int64, int64, int64, int64, string, *int16, *bool, *int8) ([]usagestats.EndpointStat, error) {
	return nil, nil
}
func (s *usageServiceDetailPersisterStub) GetGroupStatsWithFilters(context.Context, time.Time, time.Time, int64, int64, int64, int64, *int16, *bool, *int8) ([]usagestats.GroupStat, error) {
	return nil, nil
}
func (s *usageServiceDetailPersisterStub) GetUserBreakdownStats(context.Context, time.Time, time.Time, usagestats.UserBreakdownDimension, int) ([]usagestats.UserBreakdownItem, error) {
	return nil, nil
}
func (s *usageServiceDetailPersisterStub) GetAllGroupUsageSummary(context.Context, time.Time) ([]usagestats.GroupUsageSummary, error) {
	return nil, nil
}
func (s *usageServiceDetailPersisterStub) GetAPIKeyUsageTrend(context.Context, time.Time, time.Time, string, int) ([]usagestats.APIKeyUsageTrendPoint, error) {
	return nil, nil
}
func (s *usageServiceDetailPersisterStub) GetUserUsageTrend(context.Context, time.Time, time.Time, string, int) ([]usagestats.UserUsageTrendPoint, error) {
	return nil, nil
}
func (s *usageServiceDetailPersisterStub) GetUserSpendingRanking(context.Context, time.Time, time.Time, int) (*usagestats.UserSpendingRankingResponse, error) {
	return nil, nil
}
func (s *usageServiceDetailPersisterStub) GetBatchUserUsageStats(context.Context, []int64, time.Time, time.Time) (map[int64]*usagestats.BatchUserUsageStats, error) {
	return nil, nil
}
func (s *usageServiceDetailPersisterStub) GetBatchAPIKeyUsageStats(context.Context, []int64, time.Time, time.Time) (map[int64]*usagestats.BatchAPIKeyUsageStats, error) {
	return nil, nil
}
func (s *usageServiceDetailPersisterStub) GetUserDashboardStats(context.Context, int64) (*usagestats.UserDashboardStats, error) {
	return nil, nil
}
func (s *usageServiceDetailPersisterStub) GetAPIKeyDashboardStats(context.Context, int64) (*usagestats.UserDashboardStats, error) {
	return nil, nil
}
func (s *usageServiceDetailPersisterStub) GetUserUsageTrendByUserID(context.Context, int64, time.Time, time.Time, string) ([]usagestats.TrendDataPoint, error) {
	return nil, nil
}
func (s *usageServiceDetailPersisterStub) GetUserModelStats(context.Context, int64, time.Time, time.Time) ([]usagestats.ModelStat, error) {
	return nil, nil
}
func (s *usageServiceDetailPersisterStub) ListWithFilters(context.Context, pagination.PaginationParams, usagestats.UsageLogFilters) ([]UsageLog, *pagination.PaginationResult, error) {
	return nil, nil, nil
}
func (s *usageServiceDetailPersisterStub) GetGlobalStats(context.Context, time.Time, time.Time) (*usagestats.UsageStats, error) {
	return nil, nil
}
func (s *usageServiceDetailPersisterStub) GetStatsWithFilters(context.Context, usagestats.UsageLogFilters) (*usagestats.UsageStats, error) {
	return nil, nil
}
func (s *usageServiceDetailPersisterStub) GetAccountUsageStats(context.Context, int64, time.Time, time.Time) (*usagestats.AccountUsageStatsResponse, error) {
	return nil, nil
}
func (s *usageServiceDetailPersisterStub) GetUserStatsAggregated(context.Context, int64, time.Time, time.Time) (*usagestats.UsageStats, error) {
	return nil, nil
}
func (s *usageServiceDetailPersisterStub) GetAPIKeyStatsAggregated(context.Context, int64, time.Time, time.Time) (*usagestats.UsageStats, error) {
	return nil, nil
}
func (s *usageServiceDetailPersisterStub) GetAccountStatsAggregated(context.Context, int64, time.Time, time.Time) (*usagestats.UsageStats, error) {
	return nil, nil
}
func (s *usageServiceDetailPersisterStub) GetModelStatsAggregated(context.Context, string, time.Time, time.Time) (*usagestats.UsageStats, error) {
	return nil, nil
}
func (s *usageServiceDetailPersisterStub) GetDailyStatsAggregated(context.Context, int64, time.Time, time.Time) ([]map[string]any, error) {
	return nil, nil
}

func TestUsageServiceCreate_PersistsDetailBestEffortOutsideTransaction(t *testing.T) {
	db, err := sql.Open("sqlite", "file:usage_service_detail_persist?mode=memory&cache=shared")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	drv := entsql.OpenDB(dialect.SQLite, db)
	client := dbent.NewClient(dbent.Driver(drv))
	require.NoError(t, err)
	t.Cleanup(func() { _ = client.Close() })

	repo := &usageServiceDetailPersisterStub{createInserted: true}
	userRepo := &mockUserRepo{updateBalanceFn: func(ctx context.Context, id int64, amount float64) error {
		repo.events = append(repo.events, "balance")
		require.NotNil(t, dbent.TxFromContext(ctx), "扣费仍应发生在主事务内")
		return nil
	}}

	svc := NewUsageService(repo, userRepo, client, nil)
	ctx := context.WithValue(context.Background(), usageServiceDetailContextKey("trace"), "detail-persist")
	_, err = svc.Create(ctx, CreateUsageLogRequest{
		UserID:     1,
		APIKeyID:   2,
		AccountID:  3,
		RequestID:  "req-service-detail-persist",
		Model:      "claude-3",
		ActualCost: 0.5,
	})
	require.NoError(t, err)
	require.True(t, repo.createCtxHasTx, "主 usage log 创建应运行在事务上下文")
	require.False(t, repo.persistCtxHasTx, "detail best-effort 应切换到事务外上下文")
	require.NoError(t, repo.persistCtxErr, "detail best-effort 不应继承已取消/已结束的请求状态")
	require.True(t, repo.persistDeadline, "detail best-effort 应带固定超时")
	require.Equal(t, "detail-persist", repo.persistTrace, "detail best-effort 应保留请求上下文值")
	require.Equal(t, []string{"create", "balance", "detail"}, repo.events)
}

func TestUsageServiceCreate_DoesNotPersistDetailWhenUsageLogNotInserted(t *testing.T) {
	db, err := sql.Open("sqlite", "file:usage_service_detail_skip_when_not_inserted?mode=memory&cache=shared")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	drv := entsql.OpenDB(dialect.SQLite, db)
	client := dbent.NewClient(dbent.Driver(drv))
	t.Cleanup(func() { _ = client.Close() })

	repo := &usageServiceDetailPersisterStub{}
	userRepo := &mockUserRepo{updateBalanceFn: func(ctx context.Context, id int64, amount float64) error {
		repo.events = append(repo.events, "balance")
		return nil
	}}

	svc := NewUsageService(repo, userRepo, client, nil)
	_, err = svc.Create(context.Background(), CreateUsageLogRequest{
		UserID:     1,
		APIKeyID:   2,
		AccountID:  3,
		RequestID:  "req-service-detail-skip",
		Model:      "claude-3",
		ActualCost: 0.5,
	})
	require.NoError(t, err)
	require.False(t, repo.persistCtxHasTx)
	require.Zero(t, repo.persistTrace)
	require.Equal(t, []string{"create"}, repo.events)
}

func TestUsageServiceCreate_PersistsDetailAfterOuterTransactionCommit(t *testing.T) {
	db, err := sql.Open("sqlite", "file:usage_service_detail_outer_tx_commit?mode=memory&cache=shared")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	drv := entsql.OpenDB(dialect.SQLite, db)
	client := dbent.NewClient(dbent.Driver(drv))
	t.Cleanup(func() { _ = client.Close() })

	outerTx, err := client.Tx(context.Background())
	require.NoError(t, err)
	t.Cleanup(func() { _ = outerTx.Rollback() })

	repo := &usageServiceDetailPersisterStub{createInserted: true}
	userRepo := &mockUserRepo{updateBalanceFn: func(ctx context.Context, id int64, amount float64) error {
		repo.events = append(repo.events, "balance")
		require.NotNil(t, dbent.TxFromContext(ctx), "外层事务内扣费应继续复用同一事务")
		return nil
	}}

	svc := NewUsageService(repo, userRepo, outerTx.Client(), nil)
	ctx := dbent.NewTxContext(context.WithValue(context.Background(), usageServiceDetailContextKey("trace"), "outer-tx-detail"), outerTx)

	_, err = svc.Create(ctx, CreateUsageLogRequest{
		UserID:     1,
		APIKeyID:   2,
		AccountID:  3,
		RequestID:  "req-service-detail-outer-tx",
		Model:      "claude-3",
		ActualCost: 0.5,
	})
	require.NoError(t, err)
	require.Equal(t, []string{"create", "balance"}, repo.events, "外层事务提交前不应提前执行 detail 持久化")

	require.NoError(t, outerTx.Commit())
	require.Equal(t, []string{"create", "balance", "detail"}, repo.events, "外层事务提交成功后应触发 detail best-effort")
}
