//go:build unit

package admin

import (
	"fmt"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/domain"
	"github.com/stretchr/testify/require"
)

func TestAccountPlatformBinding(t *testing.T) {
	for _, platform := range append(domain.ConcretePlatformIDs(), "bogus", "composite", "", "openai ") {
		t.Run(platform, func(t *testing.T) {
			var req CreateAccountRequest
			err := bindGroupPlatformJSON(t, &req, fmt.Sprintf(`{"name":"a","platform":%q,"type":"apikey","credentials":{"api_key":"test"}}`, platform))
			if domain.IsConcretePlatform(platform) {
				require.NoError(t, err)
				require.Equal(t, platform, req.Platform)
			} else {
				require.Error(t, err)
			}
		})
	}
}
