package service

import (
	"context"
	"github.com/gin-gonic/gin"
)

type codexRoutingKey struct{}
type codexRoutingVerdict struct {
	results [2]CodexClientRestrictionDetectionResult
}

// PrepareCodexClientRouting snapshots inbound detection before protocol/identity rewriting.
// Only two account-policy variants survive; request bodies and headers are not retained.
func (s *OpenAIGatewayService) PrepareCodexClientRouting(c *gin.Context, body []byte) {
	if c == nil || c.Request == nil {
		return
	}
	verdict := codexRoutingVerdict{}
	for i := range verdict.results {
		account := &Account{Platform: PlatformOpenAI, Type: AccountTypeOAuth, Extra: map[string]any{"codex_cli_only": true, "codex_cli_only_allow_app_server": i == 1}}
		verdict.results[i] = s.detectCodexClientRestriction(c, account, body)
	}
	c.Request = c.Request.WithContext(context.WithValue(c.Request.Context(), codexRoutingKey{}, verdict))
}

// ContextWithCodexClientRouting preserves the WS lease cancellation context.
func ContextWithCodexClientRouting(ctx, source context.Context) context.Context {
	if verdict, ok := source.Value(codexRoutingKey{}).(codexRoutingVerdict); ok {
		return context.WithValue(ctx, codexRoutingKey{}, verdict)
	}
	return ctx
}

func HasCodexClientRouting(ctx context.Context) bool {
	_, ok := ctx.Value(codexRoutingKey{}).(codexRoutingVerdict)
	return ok
}

func codexClientRouteAllowed(ctx context.Context, account *Account) bool {
	if !account.IsCodexCLIOnlyRouteOnlyEnabled() {
		return true
	}
	verdict, ok := ctx.Value(codexRoutingKey{}).(codexRoutingVerdict)
	if !ok {
		// Internal metadata selectors have no business request snapshot.
		return true
	}
	i := 0
	if account.IsCodexCLIOnlyAppServerAllowed() {
		i = 1
	}
	return verdict.results[i].Matched
}
