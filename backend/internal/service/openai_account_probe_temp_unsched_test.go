package service

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
)

type probeTempUnschedRepo struct {
	AccountRepository
	until time.Time
}

func (r *probeTempUnschedRepo) SetTempUnschedulable(_ context.Context, _ int64, until time.Time, _ string) error {
	r.until = until
	return nil
}

func TestProbeSetTempUnschedulableUsesConfiguredCooldown(t *testing.T) {
	repo := &probeTempUnschedRepo{}
	probe := &openAIAccountProbe{
		service: &OpenAIGatewayService{
			accountRepo: repo,
			cfg: &config.Config{Gateway: config.GatewayConfig{OpenAIWS: config.GatewayOpenAIWSConfig{SchedulerLayered: config.GatewayOpenAIWSSchedulerLayeredConfig{
				ProbeTempUnschedulableSeconds: 1800,
			}}}},
		},
		ctx: context.Background(),
	}
	entry := &openAIAccountProbeEntry{}
	start := time.Now()

	probe.setTempUnschedulable(91, entry)

	if repo.until.IsZero() {
		t.Fatalf("expected SetTempUnschedulable to be called")
	}
	if repo.until.Before(start.Add(29*time.Minute)) || repo.until.After(start.Add(31*time.Minute)) {
		t.Fatalf("temp unschedulable until = %s, want about 30 minutes from %s", repo.until, start)
	}
}
