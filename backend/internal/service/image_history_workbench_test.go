package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/require"
)

func TestImageHistoryList_BoundedBatchAndFullDetail(t *testing.T) {
	prompt := strings.Repeat("星🌟", 150) + ` "quoted" \\ end`
	body, err := json.Marshal(map[string]any{"prompt": prompt, "output_format": "webp", "had_source_image": true, "had_mask": true})
	require.NoError(t, err)
	response := `{"data":[{"b64_json":"` + strings.Repeat("A", 5<<20) + `"}]}`
	for _, pageSize := range []int{20, 50} {
		t.Run(fmt.Sprint(pageSize), func(t *testing.T) {
			repo := &imageHistoryRepoStub{details: make(map[int64]*UsageLogDetail)}
			ids := make([]int64, pageSize)
			for i := range ids {
				ids[i] = int64(i + 1)
				repo.logs = append(repo.logs, UsageLog{ID: ids[i], UserID: 7, APIKeyID: 4, HasDetail: true, ImageCount: 1, InboundEndpoint: stringPtr("/v1/images/edits")})
				repo.details[ids[i]] = &UsageLogDetail{RequestHeaders: "Content-Type: multipart/form-data; boundary=abc", RequestBody: RequestBodyPreviewSnapshot(string(body), 8<<20, true), ResponseBody: response}
			}
			svc := NewImageHistoryService(repo)
			items, _, err := svc.List(context.Background(), 7, ImageHistoryListQuery{PageSize: pageSize})
			require.NoError(t, err)
			require.Len(t, items, pageSize)
			require.Equal(t, 1, repo.summaryCalls)
			require.Equal(t, int64(7), repo.gotSummaryUserID)
			require.Equal(t, ids, repo.gotSummaryIDs)
			require.Zero(t, repo.gotDetail, "list must not read any full request/response detail")
			for _, item := range items {
				require.True(t, item.SummaryAvailable)
				require.True(t, utf8.ValidString(item.Prompt))
				require.Equal(t, string([]rune(prompt)[:256]), item.Prompt)
			}
			detail, err := svc.GetDetail(context.Background(), 7, ids[0])
			require.NoError(t, err)
			require.Equal(t, prompt, detail.Prompt)
			require.Equal(t, prompt, detail.Replay.Prompt)
			require.True(t, detail.HadSourceImage)
			require.True(t, detail.HadMask)
			require.Len(t, detail.Images, 1)
			require.Len(t, detail.Images[0].DataURL, len("data:image/webp;base64,")+(5<<20))
		})
	}
}

func TestImageHistoryList_LegacyBoundsFallbackAndRetention(t *testing.T) {
	multipart := "--abc\r\nContent-Disposition: form-data; name=\"prompt\"\r\n\r\nlarge edit\r\n--abc\r\nContent-Disposition: form-data; name=\"image[]\"; filename=\"src.png\"\r\n\r\n" + strings.Repeat("x", 400<<10) + "\r\n--abc--\r\n"
	fixtures := []*UsageLogDetail{
		{RequestBody: `{"prompt":"legacy retained"}`},
		{RequestBody: `{"prompt":""}`},
		{RequestBody: `{"prompt":"malformed"`},
		{RequestBody: `{"prompt":"large original","image":"data:image/png;base64,` + strings.Repeat("A", 400<<10) + `"}`, UpstreamRequestBody: `{"prompt":"must not leak upstream fallback"}`},
		{RequestHeaders: "Content-Type: multipart/form-data; boundary=abc", RequestBody: multipart},
		{UpstreamRequestBody: `{"prompt":"empty client fallback"}`},
		{RequestBody: " ", UpstreamRequestBody: `{"prompt":"not fallback"}`},
		{RequestBody: RequestBodyPreviewSnapshot(`{"model":"gpt-image-2","prompt":"","had_source_image":true}`, 1024, true)},
		nil, // retention deleted between page and batch
		{RequestBody: `{"prompt":{"invalid":"type"}}`},
		{RequestBody: "--abc\r\nContent-Disposition: form-data; name=\"prompt\"\r\n\r\nbounded raw edit\r\n--abc--\r\n"},
	}
	repo := &imageHistoryRepoStub{details: make(map[int64]*UsageLogDetail)}
	for i, detail := range fixtures {
		id := int64(i + 1)
		repo.logs = append(repo.logs, UsageLog{ID: id, UserID: 7, HasDetail: true, InboundEndpoint: stringPtr("/v1/images/edits")})
		repo.details[id] = detail
	}
	// No detail means no batch lookup for this ID even if an unrelated fixture exists.
	noDetailID := int64(len(fixtures) + 1)
	repo.logs = append(repo.logs, UsageLog{ID: noDetailID, UserID: 7, InboundEndpoint: stringPtr("/v1/images/edits")})
	repo.details[noDetailID] = &UsageLogDetail{RequestBody: `{"prompt":"must not fetch pruned detail"}`}
	svc := NewImageHistoryService(repo)
	items, _, err := svc.List(context.Background(), 7, ImageHistoryListQuery{})
	require.NoError(t, err)
	require.Len(t, items, len(fixtures)+1)
	for i, item := range items {
		available := i == 0 || i == 5 || i == 10
		require.Equal(t, available, item.SummaryAvailable, "fixture %d", i)
		if !available {
			require.Empty(t, item.Prompt)
		}
	}
	require.NotContains(t, repo.gotSummaryIDs, noDetailID)
	require.Zero(t, repo.gotDetail)
	// Unavailable list summaries do not bound the independent authorized detail.
	detail, err := svc.GetDetail(context.Background(), 7, 4)
	require.NoError(t, err)
	require.Equal(t, "large original", detail.Prompt)
	detail, err = svc.GetDetail(context.Background(), 7, 5)
	require.NoError(t, err)
	require.Equal(t, "large edit", detail.Prompt)
	require.True(t, detail.HadSourceImage)
	_, err = svc.GetDetail(context.Background(), 7, 9)
	require.ErrorIs(t, err, ErrUsageLogDetailNotFound)
	repo.gotDetail = 0
	_, err = svc.GetDetail(context.Background(), 8, 1)
	require.ErrorIs(t, err, ErrUsageLogNotFound)
	require.Zero(t, repo.gotDetail)
}

func TestImageHistoryList_EmptyPageAndBatchFailure(t *testing.T) {
	repo := &imageHistoryRepoStub{}
	items, _, err := NewImageHistoryService(repo).List(context.Background(), 7, ImageHistoryListQuery{})
	require.NoError(t, err)
	require.Empty(t, items)
	require.Zero(t, repo.summaryCalls)
	repo.logs = []UsageLog{{ID: 1, HasDetail: true}}
	repo.summaryErr = errors.New("projection unavailable")
	_, _, err = NewImageHistoryService(repo).List(context.Background(), 7, ImageHistoryListQuery{})
	require.ErrorIs(t, err, repo.summaryErr)
}

func TestImageHistoryResponse_JSONMultiplicity(t *testing.T) {
	images := parseImageHistoryResponseImages(&UsageLogDetail{ResponseBody: `{"data":[{"b64_json":"QUJD"},{"b64_json":"QUJD"}]}`}, "png")
	require.Len(t, images, 2)
}

func TestImageHistoryResponse_JSONAndCompletedSSE(t *testing.T) {
	for _, tc := range []struct {
		name, body, format, image, url, errorMessage string
	}{
		{name: "malformed-types", body: `{"data":[{"b64_json":123,"url":{"invalid":"type"}}]}`},
		{name: "json-url", body: `{"data":[{"url":"https://example.invalid/result.webp","revised_prompt":"retained"}]}`, url: "https://example.invalid/result.webp"},
		{name: "json-format", body: `{"output_format":"webp","data":[{"b64_json":"QUJD"}]}`, format: "webp", image: "data:image/webp;base64,QUJD"},
		{name: "generation", body: "event: image_generation.partial_image\ndata: {\"b64_json\":\"UEFSVElBTA==\"}\n\nevent: image_generation.completed\ndata: {\"b64_json\":\"QUJD\",\"output_format\":\"jpeg\"}\n\ndata: [DONE]\n\n", format: "jpeg", image: "data:image/jpeg;base64,QUJD"},
		{name: "edit-typed-multiline", body: "data: {\"type\":\"image_edit.completed\",\r\ndata: \"b64_json\":\"QUJD\",\"output_format\":\"webp\"}\r\n\r\n", format: "webp", image: "data:image/webp;base64,QUJD"},
		{name: "completed-url", body: "event: image_edit.completed\ndata: {\"url\":\"https://example.invalid/final.png\"}\n\n", url: "https://example.invalid/final.png"},
		{name: "nested-dedup", body: "event: image_generation.completed\ndata: {\"item\":{\"b64_json\":\"QUJD\"}}\n\nevent: image_generation.completed\ndata: {\"output\":{\"b64_json\":\"QUJD\"}}\n\n", image: "data:image/png;base64,QUJD"},
		{name: "partial-only", body: "event: image_edit.partial_image\ndata: {\"b64_json\":\"UEFSVElBTA==\"}\n\n"},
		{name: "incomplete-frame", body: "event: image_generation.completed\ndata: {\"b64_json\":\"not-closed\""},
		{name: "error", body: "event: error\ndata: {\"error\":{\"message\":\"retained stream failure\"}}\n\n", errorMessage: "retained stream failure"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			detail := &UsageLogDetail{ResponseBody: tc.body}
			format := imageHistoryResponseOutputFormat(detail)
			require.Equal(t, tc.format, format)
			images := parseImageHistoryResponseImages(detail, format)
			if tc.image == "" && tc.url == "" {
				require.Empty(t, images)
			} else {
				require.Len(t, images, 1)
				require.Equal(t, tc.image, images[0].DataURL)
				require.Equal(t, tc.url, images[0].URL)
			}
			require.Equal(t, tc.errorMessage, parseImageHistoryErrorMessage(detail))
			// Client response missing: retained upstream response still works.
			detail.UpstreamResponseBody, detail.ResponseBody = detail.ResponseBody, ""
			require.Equal(t, images, parseImageHistoryResponseImages(detail, format))
		})
	}
}
