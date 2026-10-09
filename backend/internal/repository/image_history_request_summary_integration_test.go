//go:build integration

package repository

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// Exercises the projection against PostgreSQL TEXT, not a JSON mock. The existing
// integration harness accepts SUB2API_TEST_POSTGRES_IMAGE=postgres:14-alpine.
func TestImageHistoryRequestSummary_PostgresTextBounds(t *testing.T) {
	ctx := context.Background()
	tx := testEntTx(t)
	client := tx.Client()
	repo := newUsageLogRepositoryWithSQL(client, tx)
	user := mustCreateUser(t, client, &service.User{Email: "image-history-" + uuid.NewString() + "@test.invalid"})
	other := mustCreateUser(t, client, &service.User{Email: "image-history-other-" + uuid.NewString() + "@test.invalid"})
	key := mustCreateApiKey(t, client, &service.APIKey{UserID: user.ID, Key: "sk-" + uuid.NewString(), Name: "image history"})
	otherKey := mustCreateApiKey(t, client, &service.APIKey{UserID: other.ID, Key: "sk-" + uuid.NewString(), Name: "other"})
	account := mustCreateAccount(t, client, &service.Account{Name: "image-history"})
	limit := service.ImageHistoryRequestBodyLimit
	valid := `{"prompt":"retained"}`
	fixtures := []struct{ body, upstream, headers, wantBody, wantHeaders string }{
		{body: valid, headers: "Content-Type: application/json", wantBody: valid, wantHeaders: "Content-Type: application/json"},
		{body: valid + strings.Repeat(" ", limit-len(valid)), wantBody: valid + strings.Repeat(" ", limit-len(valid))},
		{body: strings.Repeat("星", limit/2), upstream: valid}, // Unicode chars under cap, UTF-8 octets over cap: no upstream fallback.
		{upstream: valid, wantBody: valid},
		{body: " ", upstream: valid, wantBody: " "}, // Only an actually empty original body permits fallback.
		{body: `{"prompt":"legacy malformed"`, wantBody: `{"prompt":"legacy malformed"`},
		{body: "--abc\r\nContent-Disposition: form-data; name=\"prompt\"\r\n\r\nlegacy multipart\r\n--abc--\r\n", wantBody: "--abc\r\nContent-Disposition: form-data; name=\"prompt\"\r\n\r\nlegacy multipart\r\n--abc--\r\n"},
		{body: `{"prompt":"too large","image":"data:image/png;base64,` + strings.Repeat("A", 5<<20) + `"}`, upstream: valid},
		{body: valid, headers: strings.Repeat("H", service.ImageHistoryRequestHeadersLimit+1), wantBody: valid},
		{body: valid, headers: strings.Repeat("H", service.ImageHistoryRequestHeadersLimit), wantBody: valid, wantHeaders: strings.Repeat("H", service.ImageHistoryRequestHeadersLimit)},
	}
	var ids []int64
	endpoint := "/v1/images/edits"
	for i, fixture := range fixtures {
		log := &service.UsageLog{UserID: user.ID, APIKeyID: key.ID, AccountID: account.ID, RequestID: uuid.NewString(), Model: "gpt-image-2", ImageCount: 1, InboundEndpoint: &endpoint}
		_, err := repo.Create(ctx, log)
		require.NoError(t, err)
		ids = append(ids, log.ID)
		// Keep large response fields in TEXT so any accidental full-row read is observable in fixtures.
		response := fmt.Sprintf(`{"data":[{"b64_json":"%s"}]}`, strings.Repeat("A", 5<<20))
		if i != 7 {
			response = `{"data":[]}`
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO usage_log_details (usage_log_id, detail_type, request_body, request_headers, upstream_request_body, upstream_request_headers, response_body, upstream_response_body) VALUES ($1,'image',$2,$3,$4,'Content-Type: application/json',$5,$5)`, log.ID, fixture.body, fixture.headers, fixture.upstream, response)
		require.NoError(t, err)
	}
	foreign := &service.UsageLog{UserID: other.ID, APIKeyID: otherKey.ID, AccountID: account.ID, RequestID: uuid.NewString(), Model: "gpt-image-2", InboundEndpoint: &endpoint}
	_, err := repo.Create(ctx, foreign)
	require.NoError(t, err)
	_, err = tx.ExecContext(ctx, `INSERT INTO usage_log_details (usage_log_id, detail_type, request_body) VALUES ($1,'image','{"prompt":"other owner"}')`, foreign.ID)
	require.NoError(t, err)
	out, err := repo.GetImageHistoryRequestSummariesByUser(ctx, user.ID, append(append([]int64(nil), ids...), foreign.ID))
	require.NoError(t, err)
	require.Len(t, out, len(fixtures))
	require.NotContains(t, out, foreign.ID)
	for i, id := range ids {
		require.Equal(t, fixtures[i].wantBody, out[id].RequestBody, "fixture %d", i)
		wantHeaders := fixtures[i].wantHeaders
		if fixtures[i].body == "" {
			wantHeaders = "Content-Type: application/json"
		}
		require.Equal(t, wantHeaders, out[id].RequestHeaders, "fixture %d", i)
		require.LessOrEqual(t, len(out[id].RequestBody), limit)
		require.LessOrEqual(t, len(out[id].RequestHeaders), service.ImageHistoryRequestHeadersLimit)
	}
	// A retained-out page ID is harmless when its detail disappears before the batch.
	_, err = tx.ExecContext(ctx, "DELETE FROM usage_log_details WHERE usage_log_id = $1", ids[0])
	require.NoError(t, err)
	out, err = repo.GetImageHistoryRequestSummariesByUser(ctx, user.ID, ids[:1])
	require.NoError(t, err)
	require.Empty(t, out)
	_, err = repo.GetDetailByUsageLogID(ctx, ids[0])
	require.ErrorIs(t, err, service.ErrUsageLogDetailNotFound)
}
