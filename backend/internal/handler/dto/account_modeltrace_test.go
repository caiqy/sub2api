package dto

import (
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestModelTraceExtraSurvivesCompactAccountProjection(t *testing.T) {
	extra := map[string]any{
		service.ModelTraceEnabledExtraKey: true,
		service.ModelTraceLatestExtraKey:  map[string]any{"task_id": 7, "result": "normal"},
	}
	account := &service.Account{ID: 1, Platform: service.PlatformOpenAI, Extra: extra}
	item := AccountListItemFromAccount(AccountFromServiceShallow(account))
	require.Equal(t, true, item.Extra[service.ModelTraceEnabledExtraKey])
	require.Equal(t, extra[service.ModelTraceLatestExtraKey], item.Extra[service.ModelTraceLatestExtraKey])
}
