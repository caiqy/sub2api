package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUpdateServiceCompareVersions_FourSegmentVersion(t *testing.T) {
	require.Equal(t, -1, compareVersions("0.1.105.6", "0.1.105.7"))
	require.Equal(t, 1, compareVersions("0.1.105.7", "0.1.105.6"))
}
