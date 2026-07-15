package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestPaymentHandlerGetCheckoutInfoRejectsHiddenPurchasePage(t *testing.T) {
	gin.SetMode(gin.TestMode)

	user := &service.User{ID: 1, HiddenPurchasePage: true}
	userSvc := service.NewUserService(&hiddenUIUserRepo{oauthPendingFlowUserRepo: &oauthPendingFlowUserRepo{}, user: user}, nil, nil, nil)
	h := NewPaymentHandler(nil, nil, nil)
	h.SetUserService(userSvc)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/v1/payment/checkout-info", nil)
	ctx.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: user.ID})

	h.GetCheckoutInfo(ctx)

	require.Equal(t, http.StatusForbidden, recorder.Code)
	require.Contains(t, recorder.Body.String(), "PURCHASE_PAGE_HIDDEN")
}

func TestPaymentHandlerHiddenPurchasePageAllowsAdmin(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h := NewPaymentHandler(nil, nil, nil)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/v1/payment/checkout-info", nil)
	ctx.Set(string(middleware2.ContextKeyUserRole), service.RoleAdmin)

	rejected := h.rejectHiddenPurchasePageForUser(ctx, 42)

	require.False(t, rejected)
}
