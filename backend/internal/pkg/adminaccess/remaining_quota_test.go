package adminaccess

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestRemainingQuotaRequiresBothFinancialPermissions(t *testing.T) {
	for _, permissions := range [][]string{nil, {"users.read"}, {"users.balance.read"}, {"subscriptions.read"}, {"users.balance.read", "subscriptions.read"}} {
		require.Equal(t, len(permissions) == 2, RouteAllowed(Policy{GlobalPermissions: permissions}, "GET", "/api/v1/admin/users/remaining-quota"))
	}
	require.True(t, RouteAllowed(Policy{FullAccess: true}, "GET", "/api/v1/admin/users/remaining-quota"))
	require.False(t, RouteAllowed(Policy{GlobalPermissions: []string{"users.balance.read", "subscriptions.read", "users.update"}}, "PUT", "/api/v1/admin/users/remaining-quota/exclusions"))
	require.True(t, RouteAllowed(Policy{FullAccess: true}, "PUT", "/api/v1/admin/users/remaining-quota/exclusions"))
}
