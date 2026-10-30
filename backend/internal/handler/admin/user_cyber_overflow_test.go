package admin

import (
	"context"
	"net/http"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type cyberOverflowAdminStub struct {
	*stubAdminService
	input *service.UpdateUserInput
}

func (s *cyberOverflowAdminStub) UpdateUser(_ context.Context, id int64, input *service.UpdateUserInput) (*service.User, error) {
	s.input = input
	return &service.User{ID: id, CyberOverflowExempt: input.CyberOverflowExempt != nil && *input.CyberOverflowExempt}, nil
}
func TestAdminUserCyberOverflowMapping(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &cyberOverflowAdminStub{stubAdminService: newStubAdminService()}
	h := NewUserHandler(svc, nil, nil, nil, nil, nil, nil)
	router := gin.New()
	router.PUT("/users/:id", h.Update)
	for _, value := range []bool{true, false} {
		rec := doJSON(t, router, http.MethodPut, "/users/1", map[string]any{"cyber_overflow_exempt": value})
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.NotNil(t, svc.input.CyberOverflowExempt)
		require.Equal(t, value, *svc.input.CyberOverflowExempt)
		require.Contains(t, rec.Body.String(), `"cyber_overflow_exempt":`+map[bool]string{true: "true", false: "false"}[value])
	}
	rec := doJSON(t, router, http.MethodPut, "/users/1", map[string]any{"username": "new"})
	require.Equal(t, http.StatusOK, rec.Code)
	require.Nil(t, svc.input.CyberOverflowExempt)
}
