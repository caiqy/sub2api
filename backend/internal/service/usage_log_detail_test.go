//go:build unit

package service

import (
	"testing"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/stretchr/testify/require"
)

// ResetUsageLogDetailRetentionLimitsForTest resets package-level runtime state.
// Do not use it in parallel tests.
func ResetUsageLogDetailRetentionLimitsForTest(t *testing.T) {
	t.Helper()
	oldNormal, oldImage := GetUsageLogDetailRetentionLimits()
	SetUsageLogDetailRetentionLimits(UsageLogDetailRetentionLimitDefault, ImageUsageLogDetailRetentionLimitDefault)
	t.Cleanup(func() { SetUsageLogDetailRetentionLimits(oldNormal, oldImage) })
}

func TestUsageLogDetailTypeFromUsageLog(t *testing.T) {
	imageEndpoint := "/v1/images/generations"
	imageEditsEndpoint := "/v1/images/edits"
	imageBillingMode := string(BillingModeImage)
	imageBillingModeWithWhitespace := " IMAGE "

	tests := []struct {
		name string
		log  *UsageLog
		want UsageLogDetailType
	}{
		{name: "nil_log", log: nil, want: UsageLogDetailTypeNormal},
		{name: "inbound_images_endpoint", log: &UsageLog{InboundEndpoint: &imageEndpoint}, want: UsageLogDetailTypeImage},
		{name: "inbound_image_edits_endpoint", log: &UsageLog{InboundEndpoint: &imageEditsEndpoint}, want: UsageLogDetailTypeImage},
		{name: "upstream_images_endpoint", log: &UsageLog{UpstreamEndpoint: &imageEndpoint}, want: UsageLogDetailTypeImage},
		{name: "image_billing_mode", log: &UsageLog{BillingMode: &imageBillingMode}, want: UsageLogDetailTypeImage},
		{name: "image_billing_mode_trim_case_insensitive", log: &UsageLog{BillingMode: &imageBillingModeWithWhitespace}, want: UsageLogDetailTypeImage},
		{name: "image_count_fallback", log: &UsageLog{ImageCount: 1}, want: UsageLogDetailTypeImage},
		{name: "normal", log: &UsageLog{Model: "gpt-5"}, want: UsageLogDetailTypeNormal},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, UsageLogDetailTypeFromUsageLog(tc.log))
		})
	}
}

func TestUsageLogDetailRetentionLimits(t *testing.T) {
	ResetUsageLogDetailRetentionLimitsForTest(t)

	normal, image := GetUsageLogDetailRetentionLimits()
	require.Equal(t, UsageLogDetailRetentionLimitDefault, normal)
	require.Equal(t, ImageUsageLogDetailRetentionLimitDefault, image)

	SetUsageLogDetailRetentionLimits(12, 0)
	normal, image = GetUsageLogDetailRetentionLimits()
	require.Equal(t, 12, normal)
	require.Equal(t, 0, image)

	SetUsageLogDetailRetentionLimits(-1, -1)
	normal, image = GetUsageLogDetailRetentionLimits()
	require.Equal(t, UsageLogDetailRetentionLimitDefault, normal)
	require.Equal(t, ImageUsageLogDetailRetentionLimitDefault, image)
}

func TestUsageLogDetailSnapshot_PreservesRawContent(t *testing.T) {
	original := &UsageLogDetailSnapshot{
		RequestHeaders:         "  Authorization: Bearer token\nX-Test: 1  ",
		RequestBody:            "  {\"message\":\" hello \"}  ",
		UpstreamRequestHeaders: "  X-Upstream-Test: 1\nAuthorization: Bearer upstream  ",
		UpstreamRequestBody:    "  {\"upstream\":\" raw body \"}  ",
		ResponseHeaders:        "  Content-Type: application/json  ",
		ResponseBody:           "  {\"result\":\" ok \"}  ",
	}

	snapshot := original.Normalize()
	require.NotNil(t, snapshot)
	require.NotSame(t, original, snapshot)
	require.Equal(t, original.RequestHeaders, snapshot.RequestHeaders)
	require.Equal(t, original.RequestBody, snapshot.RequestBody)
	require.Equal(t, original.UpstreamRequestHeaders, snapshot.UpstreamRequestHeaders)
	require.Equal(t, original.UpstreamRequestBody, snapshot.UpstreamRequestBody)
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
	require.Equal(t, "", emptyNormalized.UpstreamRequestHeaders)
	require.Equal(t, "", emptyNormalized.UpstreamRequestBody)
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

func TestErrUsageLogDetailNotFoundMentionsConfiguredRetentionPools(t *testing.T) {
	require.Equal(t, "USAGE_LOG_DETAIL_NOT_FOUND", infraerrors.Reason(ErrUsageLogDetailNotFound))
	msg := infraerrors.Message(ErrUsageLogDetailNotFound)
	require.Contains(t, msg, "regular and image usage details are retained separately")
	require.Contains(t, msg, "retention limit of 0")
	require.NotContains(t, msg, "500")
}
