package service

import (
	"reflect"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

func TestProvideTokenRefreshServiceInjectsGrokOAuthService(t *testing.T) {
	cfg := &config.Config{}
	grokOAuth := NewGrokOAuthService(nil, nil)

	svc := ProvideTokenRefreshService(nil, nil, nil, nil, nil, grokOAuth, nil, nil, cfg, nil, nil, nil, nil, nil)
	svc.Stop()

	var grokRefresher *GrokTokenRefresher
	for _, registration := range svc.registrations {
		if r, ok := registration.refresher.(*GrokTokenRefresher); ok {
			grokRefresher = r
			break
		}
	}

	require.NotNil(t, grokRefresher)
	field := reflect.ValueOf(grokRefresher).Elem().FieldByName("grokOAuthService")
	require.False(t, field.IsNil())
}
