package admin

import (
	"context"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/adminaccess"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type remainingQuotaAdminStub struct {
	service.AdminService
	excluded []int64
	calls    int
}

func (s *remainingQuotaAdminStub) GetRemainingQuota(_ context.Context, ids []int64) (*service.RemainingQuotaSummary, error) {
	s.excluded = ids
	s.calls++
	return &service.RemainingQuotaSummary{Balance: 2, Subscription: 3, Total: 5, UnlimitedSubscriptions: 1}, nil
}

type quotaSettingsRepo struct {
	service.SettingRepository
	value             string
	readErr, writeErr error
	writes            int
}

func (r *quotaSettingsRepo) GetValue(_ context.Context, key string) (string, error) {
	if key != service.SettingKeyRemainingQuotaExclusions {
		return "", service.ErrSettingNotFound
	}
	if r.readErr != nil {
		return "", r.readErr
	}
	if r.value == "" {
		return "", service.ErrSettingNotFound
	}
	return r.value, nil
}
func (r *quotaSettingsRepo) Set(_ context.Context, key, value string) error {
	r.writes++
	if r.writeErr != nil {
		return r.writeErr
	}
	r.value = value
	return nil
}
func quotaTestRouter(repo *quotaSettingsRepo, stub *remainingQuotaAdminStub, principal *adminaccess.Principal) *gin.Engine {
	gin.SetMode(gin.TestMode)
	h := &UserHandler{adminService: stub, settingService: service.NewSettingService(repo, &config.Config{})}
	r := gin.New()
	if principal != nil {
		r.Use(func(c *gin.Context) {
			c.Request = c.Request.WithContext(adminaccess.WithPrincipal(c.Request.Context(), principal))
		})
	}
	r.GET("/remaining-quota", h.GetRemainingQuota)
	r.PUT("/remaining-quota/exclusions", h.SetRemainingQuotaExclusions)
	return r
}
func TestRemainingQuotaHandlerUsesSavedExclusions(t *testing.T) {
	repo := &quotaSettingsRepo{value: `[2,3]`}
	stub := &remainingQuotaAdminStub{}
	r := quotaTestRouter(repo, stub, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/remaining-quota?exclude_user_ids=99&status=active", nil))
	require.Equal(t, 200, w.Code)
	require.Equal(t, []int64{2, 3}, stub.excluded)
	require.Contains(t, w.Body.String(), `"excluded_user_ids":[2,3]`)
	repo.value = ""
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/remaining-quota", nil))
	require.Equal(t, 200, w.Code)
	require.Contains(t, w.Body.String(), `"excluded_user_ids":[]`)
	repo.readErr = errors.New("database unavailable")
	before := stub.calls
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/remaining-quota", nil))
	require.Equal(t, 500, w.Code)
	require.Equal(t, before, stub.calls)
}
func TestRemainingQuotaExclusionsHandler(t *testing.T) {
	for _, tc := range []struct {
		body   string
		status int
		stored string
	}{
		{`{"excluded_user_ids":[1,2,1]}`, 200, `[1,2]`}, {`{"excluded_user_ids":[]}`, 200, `[]`},
		{`{}`, 400, ""}, {`{"excluded_user_ids":null}`, 400, ""}, {`{"excluded_user_ids":[0]}`, 400, ""}, {`{"excluded_user_ids":[-1]}`, 400, ""}, {`{"excluded_user_ids":[1.5]}`, 400, ""}, {`{"excluded_user_ids":["1"]}`, 400, ""}, {`{"excluded_user_ids":[9007199254740992]}`, 400, ""},
	} {
		t.Run(tc.body, func(t *testing.T) {
			repo := &quotaSettingsRepo{}
			r := quotaTestRouter(repo, &remainingQuotaAdminStub{}, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, httptest.NewRequest("PUT", "/remaining-quota/exclusions", strings.NewReader(tc.body)))
			require.Equal(t, tc.status, w.Code)
			require.Equal(t, tc.stored, repo.value)
		})
	}
	for _, full := range []bool{false, true} {
		repo := &quotaSettingsRepo{}
		r := quotaTestRouter(repo, &remainingQuotaAdminStub{}, &adminaccess.Principal{Policy: adminaccess.Policy{FullAccess: full}})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("PUT", "/remaining-quota/exclusions", strings.NewReader(`{"excluded_user_ids":[]}`)))
		if full {
			require.Equal(t, 200, w.Code)
		} else {
			require.Equal(t, 403, w.Code)
			require.Zero(t, repo.writes)
		}
	}
	repo := &quotaSettingsRepo{value: `[7]`, writeErr: errors.New("write failed")}
	r := quotaTestRouter(repo, &remainingQuotaAdminStub{}, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("PUT", "/remaining-quota/exclusions", strings.NewReader(`{"excluded_user_ids":[1]}`)))
	require.Equal(t, 500, w.Code)
	require.Equal(t, `[7]`, repo.value)
}
