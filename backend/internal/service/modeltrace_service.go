package service

import (
	"context"
	"errors"
	"log"
	"slices"
	"sync"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/modeltrace"
	"github.com/google/uuid"
)

const ModelTraceProbeTimeout = 90 * time.Second

type ModelTraceTask struct {
	ID              int64              `json:"id"`
	AccountID       int64              `json:"account_id"`
	Source          string             `json:"source"`
	Status          string             `json:"status"`
	Model           string             `json:"model"`
	TargetModel     string             `json:"target_model"`
	Rounds          int                `json:"rounds"`
	CompletedRounds int                `json:"completed_rounds"`
	Result          string             `json:"result"`
	Winner          string             `json:"winner"`
	Probabilities   map[string]float64 `json:"probabilities"`
	Version         string             `json:"version"`
	CreatedAt       time.Time          `json:"created_at"`
	StartedAt       *time.Time         `json:"started_at,omitempty"`
	FinishedAt      *time.Time         `json:"finished_at,omitempty"`
	DurationMS      int64              `json:"duration_ms"`
	Error           string             `json:"error"`
}

type ModelTraceHistory struct {
	Items    []*ModelTraceTask `json:"items"`
	Total    int64             `json:"total"`
	Page     int               `json:"page"`
	PageSize int               `json:"page_size"`
	Pages    int               `json:"pages"`
	Active   *ModelTraceTask   `json:"active"`
}

type ModelTraceRepository interface {
	Register(ctx context.Context, owner string) error
	Renew(ctx context.Context, owner string) error
	Release(ctx context.Context, owner string) error
	Enqueue(ctx context.Context, task *ModelTraceTask, owner string, interval int) (*ModelTraceTask, error)
	Claim(ctx context.Context, owner string) (*ModelTraceTask, error)
	Check(ctx context.Context, taskID int64, owner string) error
	Progress(ctx context.Context, task *ModelTraceTask, owner string) error
	Finish(ctx context.Context, task *ModelTraceTask, owner string) error
	Maintain(ctx context.Context) error
	Candidates(ctx context.Context, interval int, after int64) ([]int64, error)
	History(ctx context.Context, accountID int64, page, pageSize int) (*ModelTraceHistory, error)
	ReadFingerprint(ctx context.Context) (string, []byte, error)
	TryUpdateFingerprint(ctx context.Context, fetch func(context.Context) (string, []byte, error)) (string, []byte, bool, error)
}

type modelTraceProber interface {
	ResolveModelTraceTarget(context.Context, int64, string) (*Account, string, error)
	ProbeModelTrace(context.Context, *Account, string, modeltrace.Challenge) (string, error)
}

type modelTraceEpoch struct {
	owner       string
	ctx         context.Context
	cancel      context.CancelFunc
	lastRenewed time.Time
}

type ModelTraceService struct {
	repo             ModelTraceRepository
	accounts         AccountRepository
	settings         *SettingService
	prober           modelTraceProber
	epochMu          sync.RWMutex
	epoch            *modelTraceEpoch
	ctx              context.Context
	cancel           context.CancelFunc
	wg               sync.WaitGroup
	scanAfter        int64
	fetchFingerprint func(context.Context) (string, []byte, error)
}

func NewModelTraceService(repo ModelTraceRepository, accounts AccountRepository, settings *SettingService, prober *AccountTestService) (*ModelTraceService, error) {
	s := &ModelTraceService{repo: repo, accounts: accounts, settings: settings, prober: prober}
	s.ctx, s.cancel = context.WithCancel(context.Background())
	ctx, cancel := context.WithTimeout(s.ctx, 10*time.Second)
	defer cancel()
	if err := s.refreshEpoch(ctx); err != nil {
		s.cancel()
		return nil, err
	}
	s.fetchFingerprint = fetchModelTraceFingerprint
	s.syncFingerprint(ctx)
	s.wg.Add(6)
	go s.control()
	go s.fingerprintLoop()
	for i := 0; i < 4; i++ {
		go s.worker()
	}
	return s, nil
}

func (s *ModelTraceService) Shutdown(ctx context.Context) error {
	s.cancel()
	done := make(chan struct{})
	go func() { s.wg.Wait(); close(done) }()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-done:
		if epoch := s.currentEpoch(); epoch != nil {
			return s.repo.Release(ctx, epoch.owner)
		}
		return s.repo.Maintain(ctx)
	}
}

func (s *ModelTraceService) currentEpoch() *modelTraceEpoch {
	s.epochMu.RLock()
	defer s.epochMu.RUnlock()
	return s.epoch
}

// Only the control loop renews epochs. Maintenance uses the independent service lifetime.
func (s *ModelTraceService) refreshEpoch(ctx context.Context) error {
	if epoch := s.currentEpoch(); epoch != nil {
		if err := s.repo.Renew(ctx, epoch.owner); err == nil {
			epoch.lastRenewed = time.Now()
			return nil
		} else if time.Since(epoch.lastRenewed) < 30*time.Second {
			return err
		}
		s.epochMu.Lock()
		s.epoch = nil
		epoch.cancel()
		s.epochMu.Unlock()
		log.Print("[ModelTrace] execution epoch retired; maintenance and lease recovery remain active")
	}
	owner := uuid.NewString()
	if err := s.repo.Register(ctx, owner); err != nil {
		return err
	}
	executionCtx, cancel := context.WithCancel(s.ctx)
	s.epochMu.Lock()
	s.epoch = &modelTraceEpoch{owner: owner, ctx: executionCtx, cancel: cancel, lastRenewed: time.Now()}
	s.epochMu.Unlock()
	return nil
}

func (s *ModelTraceService) Models(ctx context.Context, accountID int64) ([]string, error) {
	models := modeltrace.Models()
	if accountID == 0 {
		return models, nil
	}
	if _, err := s.accounts.GetByID(ctx, accountID); err != nil {
		return nil, err
	}
	available := make([]string, 0, len(models))
	for _, model := range models {
		_, target, err := s.prober.ResolveModelTraceTarget(ctx, accountID, model)
		if err == nil && slices.Contains(models, target) {
			available = append(available, model)
		}
	}
	return available, nil
}

func (s *ModelTraceService) Create(ctx context.Context, accountID int64, model string, rounds int) (*ModelTraceTask, error) {
	return s.enqueue(ctx, accountID, model, rounds, "manual", ModelTraceSettings{})
}

func (s *ModelTraceService) enqueue(ctx context.Context, accountID int64, model string, rounds int, source string, cfg ModelTraceSettings) (*ModelTraceTask, error) {
	if s.ctx.Err() != nil {
		return nil, infraerrors.ServiceUnavailable("MODELTRACE_STOPPED", "ModelTrace worker is stopped")
	}
	epoch := s.currentEpoch()
	if epoch == nil || epoch.ctx.Err() != nil {
		return nil, infraerrors.ServiceUnavailable("MODELTRACE_RECOVERING", "ModelTrace worker is recovering its lease")
	}
	if err := validateModelTraceRequest(model, rounds); err != nil {
		return nil, err
	}
	account, err := s.accounts.GetByID(ctx, accountID)
	if err != nil {
		return nil, err
	}
	if account.Platform != PlatformOpenAI {
		return nil, infraerrors.BadRequest("MODELTRACE_ACCOUNT_UNSUPPORTED", "only OpenAI accounts support ModelTrace")
	}
	if source == "auto" && !modelTraceAutoEligible(account, cfg) {
		return nil, nil
	}
	history, err := s.repo.History(ctx, accountID, 1, 1)
	if err != nil {
		return nil, err
	}
	if history.Active != nil {
		return history.Active, nil
	}
	_, target, err := s.prober.ResolveModelTraceTarget(ctx, accountID, model)
	if err != nil {
		return nil, infraerrors.BadRequest("MODELTRACE_TARGET_UNAVAILABLE", "account credentials or model configuration are unavailable")
	}
	if !slices.Contains(modeltrace.Models(), target) {
		return nil, infraerrors.BadRequest("MODELTRACE_TARGET_UNSUPPORTED", "mapped target is not supported by the fingerprint library")
	}
	task := &ModelTraceTask{AccountID: accountID, Source: source, Model: model, TargetModel: target, Rounds: rounds, Version: modeltrace.Current().Version()}
	return s.repo.Enqueue(ctx, task, epoch.owner, cfg.IntervalMinutes)
}

func (s *ModelTraceService) History(ctx context.Context, accountID int64, page, pageSize int) (*ModelTraceHistory, error) {
	if page < 1 || page > 1000000 || pageSize < 1 || pageSize > 100 {
		return nil, infraerrors.BadRequest("MODELTRACE_PAGINATION_INVALID", "page must be 1-1000000 and page_size 1-100")
	}
	if _, err := s.accounts.GetByID(ctx, accountID); err != nil {
		return nil, err
	}
	return s.repo.History(ctx, accountID, page, pageSize)
}

func (s *ModelTraceService) control() {
	defer s.wg.Done()
	heartbeat := time.NewTicker(5 * time.Second)
	scan := time.NewTicker(30 * time.Second)
	defer heartbeat.Stop()
	defer scan.Stop()
	// Scan and retention are isolated from heartbeat so large account sets cannot lose the lease.
	scanDone := make(chan struct{}, 1)
	scanDone <- struct{}{}
	for {
		select {
		case <-s.ctx.Done():
			return
		case <-heartbeat.C:
			ctx, cancel := context.WithTimeout(s.ctx, 4*time.Second)
			_ = s.refreshEpoch(ctx)
			cancel()
		case <-scan.C:
			select {
			case <-scanDone:
				s.wg.Add(1)
				go func() {
					defer s.wg.Done()
					defer func() { scanDone <- struct{}{} }()
					s.scan()
				}()
			default:
			}
		}
	}
}

func (s *ModelTraceService) scan() {
	ctx, cancel := context.WithTimeout(s.ctx, 25*time.Second)
	defer cancel()
	if err := s.repo.Maintain(ctx); err != nil {
		log.Print("[ModelTrace] maintenance failed")
		return
	}
	cfg, err := s.settings.GetModelTraceSettings(ctx)
	if err != nil || !cfg.Enabled {
		return
	}
	for ctx.Err() == nil {
		ids, err := s.repo.Candidates(ctx, cfg.IntervalMinutes, s.scanAfter)
		if err != nil {
			return
		}
		if len(ids) == 0 {
			s.scanAfter = 0
			return
		}
		for _, id := range ids {
			// DB reads make changes visible across instances without a cache invalidation protocol.
			current, err := s.settings.GetModelTraceSettings(ctx)
			if err != nil || !current.Enabled {
				return
			}
			_, _ = s.enqueue(ctx, id, current.Model, current.Rounds, "auto", current)
			s.scanAfter = id
		}
	}
}

func (s *ModelTraceService) worker() {
	defer s.wg.Done()
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			epoch := s.currentEpoch()
			if epoch == nil || epoch.ctx.Err() != nil {
				continue
			}
			ctx, cancel := context.WithTimeout(epoch.ctx, 5*time.Second)
			task, err := s.repo.Claim(ctx, epoch.owner)
			cancel()
			if err == nil && task != nil {
				s.execute(epoch.ctx, epoch.owner, task)
			}
		}
	}
}

func (s *ModelTraceService) execute(epochCtx context.Context, owner string, task *ModelTraceTask) {
	ctx, cancel := context.WithTimeout(epochCtx, 5*time.Minute)
	defer cancel()
	task.Status = "failed"
	task.Probabilities = map[string]float64{}
	defer func() {
		if recover() != nil {
			task.Error = "internal detection failure"
			task.Status = "failed"
			task.Result, task.Winner = "", ""
			task.Probabilities = map[string]float64{}
		}
		// Retired owners must not release the account slot before the old request's drain window.
		if epochCtx.Err() != nil && s.ctx.Err() == nil {
			return
		}
		finishCtx, finishCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer finishCancel()
		if err := s.repo.Finish(finishCtx, task, owner); err != nil {
			log.Print("[ModelTrace] task finalization failed; lease recovery will mark it interrupted")
		}
	}()
	original, err := s.accounts.GetByID(ctx, task.AccountID)
	if err != nil {
		task.Error = "account is unavailable"
		return
	}
	if task.Source == "auto" {
		cfg, err := s.settings.GetModelTraceSettings(ctx)
		if err != nil || !modelTraceAutoEligible(original, cfg) {
			task.Error = "automatic detection disabled or account is not schedulable"
			return
		}
	}
	account, target, err := s.prober.ResolveModelTraceTarget(ctx, task.AccountID, task.Model)
	if err != nil || target != task.TargetModel || original.Platform != PlatformOpenAI {
		task.Error = "account or model configuration changed before execution"
		return
	}
	snapshot := modeltrace.Current()
	if !slices.Contains(snapshot.Models(), target) {
		task.Error = "mapped target is not supported by the fingerprint library"
		return
	}
	task.Version = snapshot.Version()
	if err := s.repo.Progress(ctx, task, owner); err != nil {
		task.Error = "task interrupted while saving fingerprint version"
		return
	}
	samples := make([]modeltrace.Sample, 0, task.Rounds)
	for i := 0; i < task.Rounds; i++ {
		// Deletion must stop subsequent rounds; never switch to an account pool.
		if _, err := s.accounts.GetByID(ctx, task.AccountID); err != nil {
			task.Error = "account is unavailable"
			return
		}
		challenge, err := modeltrace.NewChallenge()
		if err != nil {
			task.Error = "challenge generation failed"
			return
		}
		roundCtx, roundCancel := context.WithTimeout(ctx, ModelTraceProbeTimeout)
		// Start the deadline BEFORE the DB fence: a pause/slow check cannot extend this send window.
		checkCtx, checkCancel := context.WithTimeout(roundCtx, 5*time.Second)
		err = s.repo.Check(checkCtx, task.ID, owner)
		checkCancel()
		roundDeadline, _ := roundCtx.Deadline()
		if err != nil || roundCtx.Err() != nil || !time.Now().Before(roundDeadline) {
			roundCancel()
			task.Error = "task interrupted before sending probe"
			return
		}
		output, err := s.prober.ProbeModelTrace(roundCtx, account, target, challenge)
		roundCancel()
		if err != nil {
			task.Error = modelTraceSafeError(err)
			return
		}
		sample := modeltrace.Sample{Output: output, ExpectedCount: challenge.ExpectedCount}
		if _, err := snapshot.ScoreSamples([]modeltrace.Sample{sample}); err != nil {
			task.Error = "invalid or incomplete model output"
			return
		}
		samples = append(samples, sample)
		task.CompletedRounds++
		if err := s.repo.Progress(ctx, task, owner); err != nil {
			task.Error = "task interrupted while saving progress"
			return
		}
	}
	result, err := snapshot.ScoreSamples(samples)
	if err != nil {
		task.Error = "model output scoring failed"
		return
	}
	task.Status, task.Result = "completed", "degraded"
	if result.Winner == task.TargetModel {
		task.Result = "normal"
	}
	task.Winner, task.Probabilities = result.Winner, result.Probabilities
}

func modelTraceSafeError(err error) string {
	if errors.Is(err, context.DeadlineExceeded) {
		return "detection timed out"
	}
	if errors.Is(err, context.Canceled) {
		return "detection interrupted"
	}
	var status modelTraceHTTPError
	if errors.As(err, &status) {
		return status.Error()
	}
	// Never persist upstream messages, URLs, response bodies, tokens, or proxy credentials.
	return "upstream request failed (authentication, quota, rate limit, transport or response error)"
}
