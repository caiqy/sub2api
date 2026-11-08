package service

import (
	"context"
	"math"
	"testing"
)

func TestTrafficBillingSnapshot(t *testing.T) {
	n := int64(12_000_000)
	s := TrafficBillingSnapshot{Enabled: true, Consent: true, InputBytes: &n, ThresholdBytes: 10_000_000, PricePerMB: 0.01, Successful: true}
	ctx := WithTrafficBillingSnapshot(context.Background(), s)
	n = 1
	frozen, _ := TrafficBillingSnapshotFromContext(ctx)
	if amount, reason := frozen.Cost(); math.Abs(amount-0.24) > 1e-12 || reason != "charged" {
		t.Fatalf("cost=%g reason=%s", amount, reason)
	}
	for _, tc := range []struct {
		name   string
		mutate func(*TrafficBillingSnapshot)
		reason string
	}{
		{"off", func(s *TrafficBillingSnapshot) { s.Enabled = false }, "disabled"},
		{"failure", func(s *TrafficBillingSnapshot) { s.Successful = false }, "delivery_incomplete"},
		{"unknown", func(s *TrafficBillingSnapshot) { s.InputBytes = nil }, "input_unknown"},
		{"excluded", func(s *TrafficBillingSnapshot) { s.Excluded = true }, "excluded"},
		{"no consent", func(s *TrafficBillingSnapshot) { s.Consent = false }, "consent_required"},
		{"threshold", func(s *TrafficBillingSnapshot) { s.ThresholdBytes = *s.InputBytes }, "below_threshold"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			copy := frozen
			tc.mutate(&copy)
			if amount, reason := copy.Cost(); amount != 0 || reason != tc.reason {
				t.Fatalf("%g %s", amount, reason)
			}
		})
	}
	frozen.Consent = false
	if !frozen.ConsentRequired() {
		t.Fatal("missing consent gate")
	}
	frozen.ThresholdBytes = *frozen.InputBytes
	if frozen.ConsentRequired() {
		t.Fatal("threshold must be inclusive")
	}
}

func TestTrafficBillingExcludedDoesNotPersistBasis(t *testing.T) {
	log := &UsageLog{}
	cost := &CostBreakdown{}
	err := applyTrafficBilling(WithTrafficBillingSnapshot(context.Background(), TrafficBillingSnapshot{Excluded: true}), log, cost, true)
	if err != nil || log.TrafficInputBytes != nil || log.TrafficThresholdBytes != nil || log.TrafficPricePerMB != nil || log.TrafficBillingReason != "" || log.TrafficCost != 0 {
		t.Fatalf("excluded job acquired traffic basis: %+v err=%v", log, err)
	}
}

func TestTrafficBillingAmountOverflowRejected(t *testing.T) {
	n := int64(12_000_000)
	s := TrafficBillingSnapshot{Enabled: true, Consent: true, Successful: true, InputBytes: &n, ThresholdBytes: 10_000_000, PricePerMB: math.MaxFloat64}
	if s.ValidateCost() == nil {
		t.Fatal("overflow must be rejected")
	}
	boundary := s
	boundary.PricePerMB = 1e10 / (float64(n) * 2 / 1e6)
	if boundary.ValidateCost() == nil {
		t.Fatal("numeric(20,10) boundary must be rejected")
	}
	cost := &CostBreakdown{}
	log := &UsageLog{}
	if applyTrafficBilling(WithTrafficBillingSnapshot(context.Background(), s), log, cost, true) == nil {
		t.Fatal("invalid amount reached billing")
	}
	if log.TotalCost != 0 || cost.ActualCost != 0 {
		t.Fatal("invalid amount mutated bill")
	}
}

func TestTrafficBillingZeroModelSubscriptionSource(t *testing.T) {
	sub := &UserSubscription{ID: 7}
	params := &postUsageBillingParams{Cost: &CostBreakdown{ActualCost: 0.24}, User: &User{ID: 1}, APIKey: &APIKey{ID: 2}, Account: &Account{ID: 3}, Subscription: sub, IsSubscriptionBill: true}
	cmd := buildUsageBillingCommand("traffic-zero-model", &UsageLog{}, params)
	if cmd.SubscriptionID == nil || *cmd.SubscriptionID != 7 || math.Abs(cmd.SubscriptionCost-0.24) > 1e-12 || cmd.BalanceCost != 0 || cmd.AccountQuotaCost != 0 {
		t.Fatalf("wrong source: %+v", cmd)
	}
}

func TestTrafficBillingCustomerAndAccountIsolation(t *testing.T) {
	n := int64(12_000_000)
	ctx := WithTrafficBillingSnapshot(context.Background(), TrafficBillingSnapshot{Enabled: true, Consent: true, Successful: true, InputBytes: &n, ThresholdBytes: 10_000_000, PricePerMB: 0.01})
	for _, modelCost := range []float64{0, 2} {
		cost := &CostBreakdown{TotalCost: modelCost, ActualCost: 0}
		log := &UsageLog{TotalCost: modelCost}
		applyTrafficBilling(ctx, log, cost, true)
		if cost.TotalCost != modelCost || math.Abs(cost.ActualCost-0.24) > 1e-12 || math.Abs(log.TotalCost-modelCost-0.24) > 1e-12 {
			t.Fatalf("model=%g cost=%+v log=%+v", modelCost, cost, log)
		}
	}
	cost := &CostBreakdown{}
	log := &UsageLog{}
	applyTrafficBilling(ctx, log, cost, false)
	if log.TrafficCost != 0 || log.TrafficBillingReason != "delivery_incomplete" {
		t.Fatal("failed delivery charged")
	}
}
