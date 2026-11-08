//go:build integration

package repository

import (
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/google/uuid"
)

func (s *UsageLogRepoSuite) TestTrafficBillingStatsKeepAccountCostsSeparate() {
	user := mustCreateUser(s.T(), s.client, &service.User{Email: "traffic-stats@test.com"})
	key := mustCreateApiKey(s.T(), s.client, &service.APIKey{UserID: user.ID, Key: "sk-traffic-stats", Name: "traffic"})
	account := mustCreateAccount(s.T(), s.client, &service.Account{Name: "traffic-stats"})
	now := time.Now().UTC().Truncate(time.Second)
	start, end := now.Add(-time.Hour), now.Add(time.Hour)
	input, threshold, price, multiplier := int64(12000000), int64(10000000), 0.01, 1.5
	row := &service.UsageLog{
		UserID: user.ID, APIKeyID: key.ID, AccountID: account.ID,
		RequestID: uuid.NewString(), Model: "traffic-test", InputTokens: 1,
		TotalCost: 1.24, ActualCost: 2.24, AccountRateMultiplier: &multiplier,
		TrafficCost: 0.24, TrafficInputBytes: &input, TrafficThresholdBytes: &threshold,
		TrafficPricePerMB: &price, TrafficBillingReason: "charged", CreatedAt: now,
	}
	inserted, err := s.repo.Create(s.ctx, row)
	s.Require().NoError(err)
	s.Require().True(inserted)
	saved, err := s.repo.GetByID(s.ctx, row.ID)
	s.Require().NoError(err)
	s.Require().InDelta(0.24, saved.TrafficCost, 1e-8)
	s.Require().Equal(&input, saved.TrafficInputBytes)
	s.Require().Equal(&threshold, saved.TrafficThresholdBytes)
	s.Require().Equal(&price, saved.TrafficPricePerMB)
	s.Require().Equal("charged", saved.TrafficBillingReason)

	customer, err := s.repo.GetUserStatsAggregated(s.ctx, user.ID, start, end)
	s.Require().NoError(err)
	s.Require().InDelta(1.24, customer.TotalCost, 1e-8)
	s.Require().InDelta(2.24, customer.TotalActualCost, 1e-8)
	accountStats, err := s.repo.GetAccountStatsAggregated(s.ctx, account.ID, start, end)
	s.Require().NoError(err)
	s.Require().InDelta(1, accountStats.TotalCost, 1e-8)
	s.Require().InDelta(2, accountStats.TotalActualCost, 1e-8)
	window, err := s.repo.GetAccountWindowStats(s.ctx, account.ID, start)
	s.Require().NoError(err)
	s.Require().InDelta(1.5, window.Cost, 1e-8)
	s.Require().InDelta(1, window.StandardCost, 1e-8)
	s.Require().InDelta(2.24, window.UserCost, 1e-8)
	batch, err := s.repo.GetAccountWindowStatsBatch(s.ctx, []int64{account.ID}, start)
	s.Require().NoError(err)
	s.Require().InDelta(1.5, batch[account.ID].Cost, 1e-8)
	history, err := s.repo.GetAccountUsageStats(s.ctx, account.ID, start, end)
	s.Require().NoError(err)
	s.Require().InDelta(1.5, history.Summary.TotalCost, 1e-8)
	s.Require().InDelta(1, history.Summary.TotalStandardCost, 1e-8)
	s.Require().InDelta(2.24, history.Summary.TotalUserCost, 1e-8)
	models, err := s.repo.GetModelStatsWithFilters(s.ctx, start, end, 0, 0, account.ID, 0, nil, nil, nil)
	s.Require().NoError(err)
	s.Require().Len(models, 1)
	s.Require().InDelta(1, models[0].Cost, 1e-8)
	s.Require().InDelta(1.5, models[0].ActualCost, 1e-8)
	endpoints, err := s.repo.GetEndpointStatsWithFilters(s.ctx, start, end, 0, 0, account.ID, 0, "", nil, nil, nil)
	s.Require().NoError(err)
	s.Require().Len(endpoints, 1)
	s.Require().InDelta(1, endpoints[0].Cost, 1e-8)
	s.Require().InDelta(1.5, endpoints[0].ActualCost, 1e-8)
	dashboard, err := s.repo.GetDashboardStatsWithRange(s.ctx, start, end)
	s.Require().NoError(err)
	s.Require().InDelta(1.24, dashboard.TotalCost, 1e-8)
	s.Require().InDelta(2.24, dashboard.TotalActualCost, 1e-8)
	s.Require().InDelta(1.5, dashboard.TotalAccountCost, 1e-8)

	// An explicit account-pricing override remains the account basis, including zero.
	for _, override := range []float64{0, 0.4} {
		_, err = s.tx.ExecContext(s.ctx, "UPDATE usage_logs SET account_stats_cost = $1 WHERE id = $2", override, row.ID)
		s.Require().NoError(err)
		window, err = s.repo.GetAccountWindowStats(s.ctx, account.ID, start)
		s.Require().NoError(err)
		s.Require().InDelta(override*multiplier, window.Cost, 1e-8)
	}
}
