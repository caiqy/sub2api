//go:build unit

package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type probeControllerRecorder struct {
	droppedIDs []int64
}

func (r *probeControllerRecorder) DropProbeEntry(accountID int64) {
	r.droppedIDs = append(r.droppedIDs, accountID)
}

type probeToggleRepoStub struct {
	mockAccountRepoForGemini
	account               *Account
	clearTempUnschedCalls int
}

func (r *probeToggleRepoStub) GetByID(_ context.Context, id int64) (*Account, error) {
	if r.account != nil && r.account.ID == id {
		cloned := *r.account
		// Deep copy Extra map to avoid mutation across calls
		if r.account.Extra != nil {
			cloned.Extra = make(map[string]any, len(r.account.Extra))
			for k, v := range r.account.Extra {
				cloned.Extra[k] = v
			}
		}
		return &cloned, nil
	}
	return nil, ErrAccountNotFound
}

func (r *probeToggleRepoStub) Update(_ context.Context, account *Account) error {
	r.account = account
	return nil
}

func (r *probeToggleRepoStub) ClearTempUnschedulable(_ context.Context, _ int64) error {
	r.clearTempUnschedCalls++
	if r.account != nil {
		r.account.TempUnschedulableUntil = nil
		r.account.TempUnschedulableReason = ""
	}
	return nil
}

func (r *probeToggleRepoStub) BindGroups(_ context.Context, _ int64, _ []int64) error {
	return nil
}

func TestAdminService_UpdateAccount_ProbeToggleOff_DropsProbeEntry(t *testing.T) {
	repo := &probeToggleRepoStub{
		account: &Account{
			ID:       501,
			Platform: PlatformOpenAI,
			Type:     AccountTypeAPIKey,
			Status:   StatusActive,
			Extra:    map[string]any{"openai_probe_enabled": true},
		},
	}
	probeCtrl := &probeControllerRecorder{}
	blocker := &runtimeBlockRecorder{}
	svc := &adminServiceImpl{
		accountRepo:        repo,
		runtimeBlocker:     blocker,
		openaiProbeControl: probeCtrl,
	}

	_, err := svc.UpdateAccount(context.Background(), 501, &UpdateAccountInput{
		Extra: map[string]any{"openai_probe_enabled": false},
	})
	require.NoError(t, err)
	require.Equal(t, []int64{501}, probeCtrl.droppedIDs, "DropProbeEntry must be called")
}

func TestAdminService_UpdateAccount_ProbeToggleNoChange_DoesNotDropEntry(t *testing.T) {
	repo := &probeToggleRepoStub{
		account: &Account{
			ID:       502,
			Platform: PlatformOpenAI,
			Type:     AccountTypeAPIKey,
			Status:   StatusActive,
			Extra:    map[string]any{"openai_probe_enabled": false},
		},
	}
	probeCtrl := &probeControllerRecorder{}
	svc := &adminServiceImpl{
		accountRepo:        repo,
		runtimeBlocker:     &runtimeBlockRecorder{},
		openaiProbeControl: probeCtrl,
	}

	_, err := svc.UpdateAccount(context.Background(), 502, &UpdateAccountInput{
		Extra: map[string]any{"openai_probe_enabled": false},
	})
	require.NoError(t, err)
	require.Empty(t, probeCtrl.droppedIDs, "no flip means no drop")
}

func TestAdminService_UpdateAccount_ProbeToggleOff_LayeredProbeSource_ClearsTempUnsched(t *testing.T) {
	until := time.Now().Add(20 * time.Minute)
	probeReason, err := buildLayeredProbeTempUnschedReason("consecutive_failures", 5)
	require.NoError(t, err)

	repo := &probeToggleRepoStub{
		account: &Account{
			ID:                      504,
			Platform:                PlatformOpenAI,
			Type:                    AccountTypeAPIKey,
			Status:                  StatusActive,
			Extra:                   map[string]any{"openai_probe_enabled": true},
			TempUnschedulableUntil:  &until,
			TempUnschedulableReason: probeReason,
		},
	}
	probeCtrl := &probeControllerRecorder{}
	blocker := &runtimeBlockRecorder{}
	svc := &adminServiceImpl{
		accountRepo:        repo,
		runtimeBlocker:     blocker,
		openaiProbeControl: probeCtrl,
	}

	_, err = svc.UpdateAccount(context.Background(), 504, &UpdateAccountInput{
		Extra: map[string]any{"openai_probe_enabled": false},
	})
	require.NoError(t, err)
	require.Equal(t, []int64{504}, probeCtrl.droppedIDs, "Layer 1: drop called")
	require.Equal(t, 1, repo.clearTempUnschedCalls, "Layer 2: ClearTempUnschedulable called")
	require.Equal(t, []int64{504}, blocker.clearedIDs, "Layer 2: ClearAccountSchedulingBlock called")
}

func TestAdminService_UpdateAccount_ProbeToggleOff_NonLayeredSource_DoesNotClearTemp(t *testing.T) {
	until := time.Now().Add(20 * time.Minute)

	repo := &probeToggleRepoStub{
		account: &Account{
			ID:                      505,
			Platform:                PlatformOpenAI,
			Type:                    AccountTypeOAuth,
			Status:                  StatusActive,
			Extra:                   map[string]any{"openai_probe_enabled": true},
			TempUnschedulableUntil:  &until,
			TempUnschedulableReason: `{"version":1,"source":"oauth_401","kind":"token_invalid"}`,
		},
	}
	probeCtrl := &probeControllerRecorder{}
	blocker := &runtimeBlockRecorder{}
	svc := &adminServiceImpl{
		accountRepo:        repo,
		runtimeBlocker:     blocker,
		openaiProbeControl: probeCtrl,
	}

	_, err := svc.UpdateAccount(context.Background(), 505, &UpdateAccountInput{
		Extra: map[string]any{"openai_probe_enabled": false},
	})
	require.NoError(t, err)
	require.Equal(t, []int64{505}, probeCtrl.droppedIDs, "Layer 1 always runs")
	require.Equal(t, 0, repo.clearTempUnschedCalls, "Layer 2 must NOT clear non-layered_probe temp")
	require.Empty(t, blocker.clearedIDs)
}

func TestAdminService_UpdateAccount_ProbeToggleOff_NoTempState_OnlyDropsEntry(t *testing.T) {
	repo := &probeToggleRepoStub{
		account: &Account{
			ID:       506,
			Platform: PlatformOpenAI,
			Type:     AccountTypeAPIKey,
			Status:   StatusActive,
			Extra:    map[string]any{"openai_probe_enabled": true},
		},
	}
	probeCtrl := &probeControllerRecorder{}
	blocker := &runtimeBlockRecorder{}
	svc := &adminServiceImpl{
		accountRepo:        repo,
		runtimeBlocker:     blocker,
		openaiProbeControl: probeCtrl,
	}

	_, err := svc.UpdateAccount(context.Background(), 506, &UpdateAccountInput{
		Extra: map[string]any{"openai_probe_enabled": false},
	})
	require.NoError(t, err)
	require.Equal(t, []int64{506}, probeCtrl.droppedIDs, "Layer 1 runs even without temp state")
	require.Equal(t, 0, repo.clearTempUnschedCalls)
	require.Empty(t, blocker.clearedIDs)
}
