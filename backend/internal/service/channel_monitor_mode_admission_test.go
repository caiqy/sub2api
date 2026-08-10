package service

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

type blockingChannelMonitorSettingsRepo struct {
	mu       sync.Mutex
	values   map[string]string
	barriers []chan struct{}
	started  chan struct{}
}

func (r *blockingChannelMonitorSettingsRepo) Get(context.Context, string) (*Setting, error) {
	panic("unexpected Get call")
}

func (r *blockingChannelMonitorSettingsRepo) GetValue(context.Context, string) (string, error) {
	panic("unexpected GetValue call")
}

func (r *blockingChannelMonitorSettingsRepo) Set(context.Context, string, string) error {
	panic("unexpected Set call")
}

func (r *blockingChannelMonitorSettingsRepo) GetMultiple(_ context.Context, keys []string) (map[string]string, error) {
	r.mu.Lock()
	values := make(map[string]string, len(keys))
	for _, key := range keys {
		values[key] = r.values[key]
	}
	var barrier chan struct{}
	if len(r.barriers) > 0 {
		barrier = r.barriers[0]
		r.barriers = r.barriers[1:]
	}
	r.mu.Unlock()
	if barrier != nil {
		r.started <- struct{}{}
		<-barrier
	}
	return values, nil
}

func (r *blockingChannelMonitorSettingsRepo) SetMultiple(context.Context, map[string]string) error {
	panic("unexpected SetMultiple call")
}

func (r *blockingChannelMonitorSettingsRepo) GetAll(context.Context) (map[string]string, error) {
	panic("unexpected GetAll call")
}

func (r *blockingChannelMonitorSettingsRepo) Delete(context.Context, string) error {
	panic("unexpected Delete call")
}

func (r *blockingChannelMonitorSettingsRepo) setMode(mode string) {
	r.mu.Lock()
	r.values[SettingKeyChannelMonitorMode] = mode
	r.mu.Unlock()
}

func TestChannelMonitorModeAdmissionKeepsPublishedModeAfterStaleRuntimeRead(t *testing.T) {
	oldReadRelease := make(chan struct{})
	postReadRelease := make(chan struct{})
	repo := &blockingChannelMonitorSettingsRepo{
		values: map[string]string{
			SettingKeyChannelMonitorEnabled: "true",
			SettingKeyChannelMonitorMode:    ChannelMonitorModeV1,
		},
		barriers: []chan struct{}{oldReadRelease, nil, postReadRelease},
		started:  make(chan struct{}, 2),
	}
	settings := &SettingService{settingRepo: repo}
	oldAdmissionDone := make(chan struct{})
	go func() {
		release, admitted := settings.admitChannelMonitorMode(context.Background(), ChannelMonitorModeV1)
		if admitted {
			release()
		}
		close(oldAdmissionDone)
	}()

	<-repo.started
	repo.setMode(ChannelMonitorModeV2)
	settings.notifyChannelMonitorRuntimeListeners()
	close(oldReadRelease)
	<-repo.started

	settings.channelMonitorModeAdmission.mu.Lock()
	desired := settings.channelMonitorModeAdmission.desired
	settings.channelMonitorModeAdmission.mu.Unlock()
	require.Equal(t, ChannelMonitorModeV2, desired, "a stale V1 runtime read must not overwrite the notified V2 mode")

	close(postReadRelease)
	<-oldAdmissionDone
	release, admitted := settings.admitChannelMonitorMode(context.Background(), ChannelMonitorModeV2)
	require.True(t, admitted, "a V2 entrant must proceed without another runtime notification")
	release()
}
