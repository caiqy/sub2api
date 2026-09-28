package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"maps"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/modeltrace"
	"github.com/stretchr/testify/require"
)

type modelTraceSettingsStub struct {
	SettingRepository
	raw string
}

func (r *modelTraceSettingsStub) GetValue(context.Context, string) (string, error) {
	if r.raw == "" {
		return "", ErrSettingNotFound
	}
	return r.raw, nil
}
func (r *modelTraceSettingsStub) Set(_ context.Context, _, raw string) error { r.raw = raw; return nil }

type modelTraceAccountsStub struct {
	AccountRepository
	account *Account
}

func (r *modelTraceAccountsStub) GetByID(context.Context, int64) (*Account, error) {
	if r.account == nil {
		return nil, ErrAccountNotFound
	}
	return r.account, nil
}

type modelTraceTasksStub struct {
	ModelTraceRepository
	finished           *ModelTraceTask
	active             *ModelTraceTask
	progress           int
	progressVersions   []string
	enqueued           int
	maintained         int
	candidateCalls     int
	ids                []int64
	claims             atomic.Int32
	released           atomic.Bool
	checks             int
	failCheckAt        int
	onCheck            func(context.Context)
	fingerprintVersion string
	fingerprintData    []byte
	fingerprintErr     error
	updateErr          error
	updateCalls        int
}

func (r *modelTraceTasksStub) Check(ctx context.Context, _ int64, _ string) error {
	r.checks++
	if r.onCheck != nil {
		r.onCheck(ctx)
	}
	if r.checks == r.failCheckAt {
		return errors.New("task lease expired")
	}
	return nil
}

func (r *modelTraceTasksStub) Register(context.Context, string) error { return nil }
func (r *modelTraceTasksStub) Renew(context.Context, string) error    { return nil }
func (r *modelTraceTasksStub) Release(context.Context, string) error {
	r.released.Store(true)
	return nil
}
func (r *modelTraceTasksStub) Claim(context.Context, string) (*ModelTraceTask, error) {
	r.claims.Add(1)
	return nil, nil
}

func (r *modelTraceTasksStub) Progress(_ context.Context, task *ModelTraceTask, _ string) error {
	r.progress = task.CompletedRounds
	r.progressVersions = append(r.progressVersions, task.Version)
	return nil
}
func (r *modelTraceTasksStub) Finish(_ context.Context, task *ModelTraceTask, _ string) error {
	copy := *task
	r.finished = &copy
	return nil
}
func (r *modelTraceTasksStub) History(context.Context, int64, int, int) (*ModelTraceHistory, error) {
	return &ModelTraceHistory{Active: r.active}, nil
}
func (r *modelTraceTasksStub) Enqueue(_ context.Context, task *ModelTraceTask, _ string, _ int) (*ModelTraceTask, error) {
	r.enqueued++
	return task, nil
}
func (r *modelTraceTasksStub) Maintain(context.Context) error { r.maintained++; return nil }
func (r *modelTraceTasksStub) ReadFingerprint(context.Context) (string, []byte, error) {
	if r.fingerprintErr != nil {
		return "", nil, r.fingerprintErr
	}
	if r.fingerprintVersion == "" {
		return "", nil, sql.ErrNoRows
	}
	return r.fingerprintVersion, r.fingerprintData, nil
}
func (r *modelTraceTasksStub) TryUpdateFingerprint(ctx context.Context, fetch func(context.Context) (string, []byte, error)) (string, []byte, bool, error) {
	r.updateCalls++
	if r.updateErr != nil {
		return "", nil, false, r.updateErr
	}
	version, data, err := fetch(ctx)
	if err != nil {
		return "", nil, false, err
	}
	r.fingerprintVersion, r.fingerprintData = version, data
	return version, data, true, nil
}
func (r *modelTraceTasksStub) Candidates(context.Context, int, int64) ([]int64, error) {
	r.candidateCalls++
	if r.candidateCalls > 1 {
		return nil, nil
	}
	return r.ids, nil
}

type modelTraceProbeStub struct {
	account    *Account
	target     string
	calls      int
	failAt     int
	resolveErr error
	invalid    bool
	received   *Account
}

func (p *modelTraceProbeStub) ResolveModelTraceTarget(context.Context, int64, string) (*Account, string, error) {
	return p.account, p.target, p.resolveErr
}
func (p *modelTraceProbeStub) ProbeModelTrace(ctx context.Context, account *Account, _ string, challenge modeltrace.Challenge) (string, error) {
	p.calls++
	p.received = account
	if p.calls == p.failAt {
		return "", errors.New("https://user:password@proxy upstream api_key=secret cookie=secret")
	}
	if ctx.Err() != nil {
		return "", ctx.Err()
	}
	if p.invalid {
		return "raw sensitive invalid output", nil
	}
	return strings.Repeat("17,", challenge.ExpectedCount), nil
}

func TestModelTraceSettingsHotUpdateAndBounds(t *testing.T) {
	repo := &modelTraceSettingsStub{}
	s := &SettingService{settingRepo: repo}
	ctx := context.Background()
	defaults, err := s.GetModelTraceSettings(ctx)
	require.NoError(t, err)
	require.Equal(t, ModelTraceSettings{Model: "gpt-6-astra", Rounds: 1, IntervalMinutes: 60}, defaults)
	cfg := defaults
	cfg.Enabled = true
	require.NoError(t, s.SetModelTraceSettings(ctx, cfg))
	got, err := s.GetModelTraceSettings(ctx)
	require.NoError(t, err)
	require.Equal(t, cfg, got)
	saved := repo.raw
	for _, invalid := range []ModelTraceSettings{
		{Model: cfg.Model, Rounds: 0, IntervalMinutes: 60},
		{Model: cfg.Model, Rounds: 4, IntervalMinutes: 60},
		{Model: "unknown", Rounds: 1, IntervalMinutes: 60},
		{Model: cfg.Model, Rounds: 1, IntervalMinutes: 4},
		{Model: cfg.Model, Rounds: 1, IntervalMinutes: 10081},
	} {
		require.Error(t, s.SetModelTraceSettings(ctx, invalid))
		require.Equal(t, saved, repo.raw)
	}
	cfg.Enabled = false
	require.NoError(t, s.SetModelTraceSettings(ctx, cfg))
	got, err = s.GetModelTraceSettings(ctx)
	require.NoError(t, err)
	require.False(t, got.Enabled)
}

func TestModelTraceAutoSwitchesAndManualIsolation(t *testing.T) {
	for _, tc := range []struct {
		name        string
		global      bool
		enabled     any
		schedulable bool
		source      string
		probes      int
	}{
		{"both enabled", true, true, true, "auto", 1},
		{"global off", false, true, true, "auto", 0},
		{"account off", true, false, true, "auto", 0},
		{"missing defaults off", true, nil, true, "auto", 0},
		{"string is not enabled", true, "true", true, "auto", 0},
		{"not schedulable", true, true, false, "auto", 0},
		{"manual ignores switches", false, false, false, "manual", 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &modelTraceSettingsStub{}
			settings := &SettingService{settingRepo: cfg}
			require.NoError(t, settings.SetModelTraceSettings(context.Background(), ModelTraceSettings{Enabled: tc.global, Model: "gpt-6-astra", Rounds: 1, IntervalMinutes: 60}))
			original := &Account{ID: 1, Platform: PlatformOpenAI, Status: StatusActive, Schedulable: tc.schedulable, Extra: map[string]any{ModelTraceEnabledExtraKey: tc.enabled}}
			resolved := &Account{ID: 42, Platform: PlatformOpenAI}
			tasks := &modelTraceTasksStub{}
			probe := &modelTraceProbeStub{account: resolved, target: "gpt-6-astra"}
			s := &ModelTraceService{ctx: context.Background(), repo: tasks, accounts: &modelTraceAccountsStub{account: original}, settings: settings, prober: probe}
			s.execute(context.Background(), "owner", &ModelTraceTask{ID: 1, AccountID: 1, Source: tc.source, Model: probe.target, TargetModel: probe.target, Rounds: 1})
			require.Equal(t, tc.probes, probe.calls)
			require.Equal(t, int64(1), tasks.finished.AccountID)
			if tc.probes == 1 {
				require.Same(t, resolved, probe.received)
				require.Equal(t, "completed", tasks.finished.Status)
			} else {
				require.Equal(t, "failed", tasks.finished.Status)
				require.Empty(t, tasks.finished.Result)
			}
		})
	}
}

func TestModelTracePartialFailureHasNoConclusionOrRetry(t *testing.T) {
	tasks := &modelTraceTasksStub{}
	original := &Account{ID: 1, Platform: PlatformOpenAI}
	probe := &modelTraceProbeStub{account: original, target: "gpt-6-astra", failAt: 3}
	s := &ModelTraceService{ctx: context.Background(), repo: tasks, accounts: &modelTraceAccountsStub{account: original}, prober: probe}
	s.execute(context.Background(), "owner", &ModelTraceTask{AccountID: 1, Source: "manual", Model: probe.target, TargetModel: probe.target, Rounds: 3})
	require.Equal(t, 3, probe.calls)
	require.Equal(t, 2, tasks.finished.CompletedRounds)
	require.Equal(t, 2, tasks.progress)
	require.Equal(t, "failed", tasks.finished.Status)
	require.Empty(t, tasks.finished.Result)
	require.Empty(t, tasks.finished.Probabilities)
	require.NotContains(t, tasks.finished.Error, "secret")
	require.NotContains(t, tasks.finished.Error, "password")
	require.Equal(t, "detection timed out", modelTraceSafeError(context.DeadlineExceeded))
	require.Equal(t, "detection interrupted", modelTraceSafeError(context.Canceled))
}

func TestModelTraceExpiredOwnerNeverStartsFirstOrNextProbe(t *testing.T) {
	for _, failAt := range []int{1, 2} {
		t.Run(fmt.Sprint(failAt), func(t *testing.T) {
			tasks := &modelTraceTasksStub{failCheckAt: failAt}
			original := &Account{ID: 1, Platform: PlatformOpenAI}
			probe := &modelTraceProbeStub{account: original, target: "gpt-6-astra"}
			s := &ModelTraceService{ctx: context.Background(), repo: tasks, accounts: &modelTraceAccountsStub{account: original}, prober: probe}
			s.execute(context.Background(), "owner", &ModelTraceTask{ID: 1, AccountID: 1, Source: "manual", Model: probe.target, TargetModel: probe.target, Rounds: 2})
			require.Equal(t, failAt-1, probe.calls, "expired owner must not send the next probe")
			require.Equal(t, failAt-1, tasks.finished.CompletedRounds)
			require.Equal(t, "failed", tasks.finished.Status)
			require.Empty(t, tasks.finished.Result)
		})
	}
}

type modelTraceRecoveryTasksStub struct {
	*modelTraceTasksStub
	dbDown         bool
	owners         []string
	enqueuedOwners []string
}

func (r *modelTraceRecoveryTasksStub) Register(_ context.Context, owner string) error {
	if r.dbDown {
		return errors.New("database unavailable")
	}
	r.owners = append(r.owners, owner)
	return nil
}
func (r *modelTraceRecoveryTasksStub) Renew(context.Context, string) error {
	if r.dbDown {
		return errors.New("database unavailable")
	}
	return nil
}
func (r *modelTraceRecoveryTasksStub) Maintain(ctx context.Context) error {
	if r.dbDown {
		return errors.New("database unavailable")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	r.maintained++
	return nil
}
func (r *modelTraceRecoveryTasksStub) Enqueue(ctx context.Context, task *ModelTraceTask, owner string, interval int) (*ModelTraceTask, error) {
	if r.dbDown {
		return nil, errors.New("database unavailable")
	}
	r.enqueuedOwners = append(r.enqueuedOwners, owner)
	return r.modelTraceTasksStub.Enqueue(ctx, task, owner, interval)
}

func TestModelTraceDBRecoveryMaintainsHistoryAndCreatesFreshEpoch(t *testing.T) {
	tasks := &modelTraceRecoveryTasksStub{modelTraceTasksStub: &modelTraceTasksStub{}}
	original := &Account{ID: 1, Platform: PlatformOpenAI}
	probe := &modelTraceProbeStub{account: original, target: "gpt-6-astra"}
	lifetime, stop := context.WithCancel(context.Background())
	defer stop()
	s := &ModelTraceService{ctx: lifetime, cancel: stop, repo: tasks, prober: probe,
		settings: &SettingService{settingRepo: &modelTraceSettingsStub{}}, accounts: &modelTraceAccountsStub{account: original}}
	require.NoError(t, s.refreshEpoch(lifetime))
	oldEpoch := s.currentEpoch()
	_, err := s.Create(lifetime, 1, probe.target, 1)
	require.NoError(t, err)
	oldEpoch.lastRenewed = time.Now().Add(-31 * time.Second)
	tasks.dbDown = true
	require.Error(t, s.refreshEpoch(lifetime))
	require.Nil(t, s.currentEpoch())
	require.ErrorIs(t, oldEpoch.ctx.Err(), context.Canceled)
	require.NoError(t, lifetime.Err(), "lease loss must not stop the maintenance lifetime")
	_, err = s.Create(lifetime, 1, probe.target, 1)
	require.Error(t, err)
	tasks.dbDown = false
	s.scan()
	require.Equal(t, 1, tasks.maintained, "maintenance must run after DB recovery even before a new epoch exists")
	require.NoError(t, s.refreshEpoch(lifetime))
	newEpoch := s.currentEpoch()
	require.NotEqual(t, oldEpoch.owner, newEpoch.owner)
	require.NoError(t, newEpoch.ctx.Err())
	_, err = s.Create(lifetime, 1, probe.target, 1)
	require.NoError(t, err)
	require.Equal(t, []string{oldEpoch.owner, newEpoch.owner}, tasks.enqueuedOwners)
	// Resuming the old worker cannot probe or finalize after epoch retirement.
	s.execute(oldEpoch.ctx, oldEpoch.owner, &ModelTraceTask{ID: 42, AccountID: 1, Source: "manual", Model: probe.target, TargetModel: probe.target, Rounds: 1})
	require.Zero(t, probe.calls)
	require.Nil(t, tasks.finished, "retired execution keeps the slot for lease recovery instead of freeing it early")
}

func TestModelTraceCancellationBetweenFenceAndProbeDoesNotSend(t *testing.T) {
	epoch, cancel := context.WithCancel(context.Background())
	defer cancel()
	tasks := &modelTraceTasksStub{onCheck: func(ctx context.Context) {
		_, bounded := ctx.Deadline()
		require.True(t, bounded, "the fence must already have a deadline")
		cancel()
	}}
	original := &Account{ID: 1, Platform: PlatformOpenAI}
	probe := &modelTraceProbeStub{account: original, target: "gpt-6-astra"}
	s := &ModelTraceService{ctx: context.Background(), repo: tasks, prober: probe, accounts: &modelTraceAccountsStub{account: original}}
	s.execute(epoch, "owner", &ModelTraceTask{ID: 1, AccountID: 1, Source: "manual", Model: probe.target, TargetModel: probe.target, Rounds: 1})
	require.Equal(t, 1, tasks.checks)
	require.Zero(t, probe.calls)
	require.Nil(t, tasks.finished)
}

func TestModelTraceInvalidOutputDoesNotCountAsCompleted(t *testing.T) {
	tasks := &modelTraceTasksStub{}
	original := &Account{ID: 1, Platform: PlatformOpenAI}
	probe := &modelTraceProbeStub{account: original, target: "gpt-6-astra", invalid: true}
	s := &ModelTraceService{ctx: context.Background(), repo: tasks, accounts: &modelTraceAccountsStub{account: original}, prober: probe}
	s.execute(context.Background(), "owner", &ModelTraceTask{AccountID: 1, Source: "manual", Model: probe.target, TargetModel: probe.target, Rounds: 3})
	require.Equal(t, 1, probe.calls)
	require.Zero(t, tasks.finished.CompletedRounds)
	require.Equal(t, "invalid or incomplete model output", tasks.finished.Error)
}

func TestModelTraceScanMaintainsHistoryWhenDisabled(t *testing.T) {
	tasks := &modelTraceTasksStub{ids: []int64{1}}
	settings := &SettingService{settingRepo: &modelTraceSettingsStub{}}
	s := &ModelTraceService{ctx: context.Background(), repo: tasks, settings: settings}
	s.scan()
	require.Equal(t, 1, tasks.maintained)
	require.Zero(t, tasks.candidateCalls)
	require.Zero(t, tasks.enqueued)
}

func TestModelTraceScanCreatesOnlyEligibleAutoTasks(t *testing.T) {
	tasks := &modelTraceTasksStub{ids: []int64{1}}
	settings := &SettingService{settingRepo: &modelTraceSettingsStub{}}
	require.NoError(t, settings.SetModelTraceSettings(context.Background(), ModelTraceSettings{Enabled: true, Model: "gpt-6-astra", Rounds: 1, IntervalMinutes: 60}))
	original := &Account{ID: 1, Platform: PlatformOpenAI, Status: StatusActive, Schedulable: true, Extra: map[string]any{ModelTraceEnabledExtraKey: true}}
	s := &ModelTraceService{ctx: context.Background(), epoch: &modelTraceEpoch{ctx: context.Background()}, repo: tasks, settings: settings, accounts: &modelTraceAccountsStub{account: original}, prober: &modelTraceProbeStub{account: original, target: "gpt-6-astra"}}
	s.scan()
	require.Equal(t, 1, tasks.enqueued)
	original.Extra[ModelTraceEnabledExtraKey] = false
	tasks.candidateCalls = 0
	s.scan()
	require.Equal(t, 1, tasks.enqueued)
}

func TestModelTraceExtraPreservesManagedSummaryAndSwitch(t *testing.T) {
	current := map[string]any{ModelTraceLatestExtraKey: map[string]any{"task_id": 8}, ModelTraceEnabledExtraKey: true}
	incoming := map[string]any{ModelTraceLatestExtraKey: "forged", "other": 1}
	got := MergeModelTraceExtra(incoming, current)
	require.Equal(t, current[ModelTraceLatestExtraKey], got[ModelTraceLatestExtraKey])
	require.Equal(t, true, got[ModelTraceEnabledExtraKey])
	got = MergeModelTraceExtra(map[string]any{ModelTraceEnabledExtraKey: false}, current)
	require.Equal(t, false, got[ModelTraceEnabledExtraKey])
	require.Equal(t, current[ModelTraceLatestExtraKey], got[ModelTraceLatestExtraKey])
}

func TestModelTraceQuarantineEligibilityAndPolicyMerge(t *testing.T) {
	account := &Account{ID: 1, Platform: PlatformOpenAI, Status: StatusActive, Schedulable: true,
		Extra: map[string]any{ModelTraceEnabledExtraKey: true, ModelTraceQuarantinedExtraKey: true}}
	cfg := ModelTraceSettings{Enabled: true}
	require.False(t, account.IsSchedulable())
	require.True(t, modelTraceAutoEligible(account, cfg), "own quarantine must not stop recovery probes")
	until := time.Now().Add(time.Hour)
	account.TempUnschedulableUntil = &until
	require.False(t, modelTraceAutoEligible(account, cfg), "another runtime blocker must still stop probes")
	account.TempUnschedulableUntil = nil
	account.Extra[ModelTraceEnabledExtraKey] = false
	require.False(t, modelTraceAutoEligible(account, cfg))
	account.Extra[ModelTraceEnabledExtraKey] = true
	cfg.Enabled = false
	require.False(t, modelTraceAutoEligible(account, cfg))

	current := map[string]any{
		ModelTraceQuarantineEnabledExtraKey: true, ModelTraceQuarantinedExtraKey: true,
		ModelTraceIntervalExtraKey: 30,
	}
	got := MergeModelTraceExtra(map[string]any{ModelTraceQuarantinedExtraKey: false}, current)
	require.Equal(t, true, got[ModelTraceQuarantinedExtraKey], "editor may not clear the runtime blocker")
	require.Equal(t, 30, got[ModelTraceIntervalExtraKey], "partial edits preserve the override")
	got = MergeModelTraceExtra(map[string]any{ModelTraceQuarantineEnabledExtraKey: false, ModelTraceIntervalExtraKey: nil}, current)
	require.NotContains(t, got, ModelTraceQuarantinedExtraKey, "disabling policy immediately lifts only its own blocker")
	require.NotContains(t, got, ModelTraceIntervalExtraKey, "explicit null restores global inheritance")
}

func TestModelTraceRecoveryProbeRespectsOtherBlockers(t *testing.T) {
	settings := &SettingService{settingRepo: &modelTraceSettingsStub{}}
	s := &ModelTraceService{settings: settings}
	account := &Account{ID: 1, Platform: PlatformOpenAI, Status: StatusActive, Schedulable: true,
		Extra: map[string]any{ModelTraceEnabledExtraKey: true, ModelTraceQuarantinedExtraKey: true}}
	cfg := ModelTraceSettings{Enabled: true}
	require.True(t, s.autoEligibleForTarget(context.Background(), account, cfg, "gpt-6-astra"))
	account.Extra[modelRateLimitsKey] = map[string]any{"gpt-6-astra": map[string]any{
		"rate_limit_reset_at": time.Now().Add(time.Hour).Format(time.RFC3339),
	}}
	require.False(t, s.autoEligibleForTarget(context.Background(), account, cfg, "gpt-6-astra"))
	account.Credentials = map[string]any{"model_mapping": map[string]any{"gpt-6-astra": "gpt-6-sol", "gpt-6-sol": "gpt-6-astra"}}
	account.Extra[modelRateLimitsKey] = map[string]any{"gpt-6-sol": map[string]any{
		"rate_limit_reset_at": time.Now().Add(time.Hour).Format(time.RFC3339),
	}}
	require.False(t, s.autoEligibleForTarget(context.Background(), account, cfg, "gpt-6-astra"), "check the selected model exactly once through its mapping")
}

func TestModelTraceAccountPolicyValidation(t *testing.T) {
	for _, interval := range []any{float64(5), 10080, nil} {
		require.NoError(t, ValidateModelTraceAccountExtra(map[string]any{ModelTraceIntervalExtraKey: interval}))
	}
	for _, interval := range []any{float64(4), float64(10081), float64(5.5), "60", true} {
		require.Error(t, ValidateModelTraceAccountExtra(map[string]any{ModelTraceIntervalExtraKey: interval}))
	}
	require.Error(t, ValidateModelTraceAccountExtra(map[string]any{ModelTraceQuarantineEnabledExtraKey: "true"}))
	require.NoError(t, ValidateModelTraceAccountExtra(map[string]any{ModelTraceQuarantineEnabledExtraKey: false}))
}

func TestModelTraceUnspecifiedEditInheritsLockedPolicy(t *testing.T) {
	readSnapshot := map[string]any{ModelTraceQuarantineEnabledExtraKey: false, ModelTraceIntervalExtraKey: 60}
	current := map[string]any{ModelTraceQuarantineEnabledExtraKey: true, ModelTraceQuarantinedExtraKey: true,
		ModelTraceIntervalExtraKey: 15}
	got := MergeModelTraceExtra(modelTraceExtraWithoutUnsubmittedPolicy(readSnapshot), current)
	require.Equal(t, true, got[ModelTraceQuarantineEnabledExtraKey])
	require.Equal(t, true, got[ModelTraceQuarantinedExtraKey])
	require.Equal(t, 15, got[ModelTraceIntervalExtraKey])
	require.Equal(t, false, readSnapshot[ModelTraceQuarantineEnabledExtraKey], "omit without mutating the source snapshot")
}

type modelTracePolicyRaceRepo struct {
	AccountRepository
	initial *Account
	current map[string]any
	saved   map[string]any
}

func (r *modelTracePolicyRaceRepo) GetByID(context.Context, int64) (*Account, error) {
	copy := *r.initial
	copy.Extra = maps.Clone(r.initial.Extra)
	return &copy, nil
}

func (r *modelTracePolicyRaceRepo) Update(_ context.Context, account *Account) error {
	r.saved = MergeModelTraceExtra(maps.Clone(account.Extra), r.current)
	return nil
}

func TestModelTraceAdminUnrelatedUpdateCannotUndoConcurrentQuarantine(t *testing.T) {
	repo := &modelTracePolicyRaceRepo{
		initial: &Account{ID: 1, Platform: PlatformOpenAI, Type: AccountTypeAPIKey,
			Status: StatusActive, Extra: map[string]any{ModelTraceQuarantineEnabledExtraKey: false, ModelTraceIntervalExtraKey: 60}},
		current: map[string]any{ModelTraceQuarantineEnabledExtraKey: true, ModelTraceQuarantinedExtraKey: true,
			ModelTraceIntervalExtraKey: 15},
	}
	_, err := (&adminServiceImpl{accountRepo: repo}).UpdateAccount(context.Background(), 1, &UpdateAccountInput{Name: "renamed"})
	require.NoError(t, err)
	require.Equal(t, true, repo.saved[ModelTraceQuarantineEnabledExtraKey])
	require.Equal(t, true, repo.saved[ModelTraceQuarantinedExtraKey])
	require.Equal(t, 15, repo.saved[ModelTraceIntervalExtraKey])
}

func TestModelTracePaginationBounds(t *testing.T) {
	s := &ModelTraceService{}
	for _, bounds := range [][2]int{{0, 20}, {1000001, 20}, {1, 0}, {1, 101}} {
		_, err := s.History(context.Background(), 1, bounds[0], bounds[1])
		require.Error(t, err)
	}
}

func TestModelTraceCreateReusesActiveTaskBeforeResolvingNewTarget(t *testing.T) {
	active := &ModelTraceTask{ID: 7, AccountID: 1, Status: "running", Source: "auto"}
	tasks := &modelTraceTasksStub{active: active}
	s := &ModelTraceService{ctx: context.Background(), epoch: &modelTraceEpoch{ctx: context.Background()}, repo: tasks,
		accounts: &modelTraceAccountsStub{account: &Account{ID: 1, Platform: PlatformOpenAI}},
		prober:   &modelTraceProbeStub{resolveErr: errors.New("credentials changed")}}
	got, err := s.Create(context.Background(), 1, "gpt-6-astra", 1)
	require.NoError(t, err)
	require.Same(t, active, got)
	require.Zero(t, tasks.enqueued)
}

func TestModelTraceBackgroundWorkersShutdown(t *testing.T) {
	tasks := &modelTraceTasksStub{}
	settings := &SettingService{settingRepo: &modelTraceSettingsStub{}}
	s, err := NewModelTraceService(tasks, &modelTraceAccountsStub{}, settings, nil)
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	t.Cleanup(func() { _ = s.Shutdown(ctx) })
	require.Eventually(t, func() bool { return tasks.claims.Load() > 0 }, 2*time.Second, 10*time.Millisecond)
	require.NoError(t, s.Shutdown(ctx))
	require.True(t, tasks.released.Load())
	stopped := tasks.claims.Load()
	time.Sleep(1100 * time.Millisecond)
	require.Equal(t, stopped, tasks.claims.Load())
	_, err = s.Create(context.Background(), 1, "gpt-6-astra", 1)
	require.Error(t, err)
}
