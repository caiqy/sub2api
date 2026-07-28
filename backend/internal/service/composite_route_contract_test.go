package service

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCompositeRouteModelLengthContract(t *testing.T) {
	valid := strings.Repeat("m", 100)
	_, err := compositeRouteFromInput(1, CompositeRouteInput{
		PublicModel:    valid,
		TargetPlatform: PlatformOpenAI,
		UpstreamModel:  valid,
		Enabled:        true,
	})
	require.NoError(t, err)

	for _, input := range []CompositeRouteInput{
		{PublicModel: strings.Repeat("m", 101), TargetPlatform: PlatformOpenAI, Enabled: true},
		{PublicModel: valid, TargetPlatform: PlatformOpenAI, UpstreamModel: strings.Repeat("m", 101), Enabled: true},
	} {
		_, err := compositeRouteFromInput(1, input)
		require.Error(t, err)
	}
}
