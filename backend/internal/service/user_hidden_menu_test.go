package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHydrateHiddenCustomMenuIDs(t *testing.T) {
	user := &User{HiddenCustomMenuResourceIDs: []int64{CustomMenuResourceID("88cdc7464b9f93ea")}}
	raw := `[{"id":"88cdc7464b9f93ea","label":"充值兑换","visibility":"user"}]`

	hydrateHiddenCustomMenuIDs(user, raw)

	require.Equal(t, []string{"88cdc7464b9f93ea"}, user.HiddenCustomMenuIDs)
}
