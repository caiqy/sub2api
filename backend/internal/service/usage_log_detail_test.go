//go:build unit

package service

import (
	"strconv"
	"testing"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/stretchr/testify/require"
)

func TestUsageLogDetailSnapshot_PreservesRawContent(t *testing.T) {
	original := &UsageLogDetailSnapshot{
		RequestHeaders:  "  Authorization: Bearer token\nX-Test: 1  ",
		RequestBody:     "  {\"message\":\" hello \"}  ",
		ResponseHeaders: "  Content-Type: application/json  ",
		ResponseBody:    "  {\"result\":\" ok \"}  ",
	}

	snapshot := original.Normalize()
	require.NotNil(t, snapshot)
	require.NotSame(t, original, snapshot)
	require.Equal(t, original.RequestHeaders, snapshot.RequestHeaders)
	require.Equal(t, original.RequestBody, snapshot.RequestBody)
	require.Equal(t, original.ResponseHeaders, snapshot.ResponseHeaders)
	require.Equal(t, original.ResponseBody, snapshot.ResponseBody)
}

func TestUsageLogDetailSnapshot_NormalizeNilSafeAndReturnsSnapshotCopy(t *testing.T) {
	var nilSnapshot *UsageLogDetailSnapshot
	require.Nil(t, nilSnapshot.Normalize())

	emptyOriginal := &UsageLogDetailSnapshot{}
	emptyNormalized := emptyOriginal.Normalize()
	require.NotNil(t, emptyNormalized)
	require.NotSame(t, emptyOriginal, emptyNormalized)
	require.Equal(t, "", emptyNormalized.RequestHeaders)
	require.Equal(t, "", emptyNormalized.RequestBody)
	require.Equal(t, "", emptyNormalized.ResponseHeaders)
	require.Equal(t, "", emptyNormalized.ResponseBody)
}

func TestUsageLog_HasDetailAndDetailSnapshotRepresentDifferentLifecycles(t *testing.T) {
	snapshot := (&UsageLogDetailSnapshot{RequestBody: "raw"}).Normalize()

	writeModel := &UsageLog{DetailSnapshot: snapshot}
	queryModel := &UsageLog{HasDetail: true}

	require.False(t, writeModel.HasDetail)
	require.NotNil(t, writeModel.DetailSnapshot)
	require.True(t, queryModel.HasDetail)
	require.Nil(t, queryModel.DetailSnapshot)
}

func TestErrUsageLogDetailNotFoundIncludesRetentionHint(t *testing.T) {
	require.Equal(t, "USAGE_LOG_DETAIL_NOT_FOUND", infraerrors.Reason(ErrUsageLogDetailNotFound))
	require.Contains(t, infraerrors.Message(ErrUsageLogDetailNotFound), strconv.Itoa(UsageLogDetailRetentionLimit))
}
