package handler

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/enttest"
	"github.com/Wei-Shaw/sub2api/internal/repository"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	_ "modernc.org/sqlite"
)

func TestPaymentHandlerGetCheckoutInfoRejectsHiddenPurchasePage(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db, err := sql.Open("sqlite", "file:payment_hidden_purchase?mode=memory&cache=shared&_fk=1")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	_, err = db.Exec("PRAGMA foreign_keys = ON")
	require.NoError(t, err)
	_, err = db.Exec(`CREATE TABLE user_avatars (
		user_id INTEGER PRIMARY KEY,
		storage_provider TEXT NOT NULL,
		storage_key TEXT NOT NULL DEFAULT '',
		url TEXT NOT NULL,
		content_type TEXT NOT NULL,
		byte_size INTEGER NOT NULL,
		sha256 TEXT NOT NULL
	)`)
	require.NoError(t, err)

	drv := entsql.OpenDB(dialect.SQLite, db)
	client := enttest.NewClient(t, enttest.WithOptions(dbent.Driver(drv)))
	t.Cleanup(func() { _ = client.Close() })

	user, err := client.User.Create().
		SetEmail("hidden-purchase@example.com").
		SetPasswordHash("hash").
		SetUsername("hidden-purchase").
		SetRole(service.RoleUser).
		SetStatus(service.StatusActive).
		Save(context.Background())
	require.NoError(t, err)
	require.NoError(t, client.UserResourceOverride.Create().
		SetUserID(user.ID).
		SetResourceType("ui").
		SetResourceID(1).
		SetEffect("deny").
		Exec(context.Background()))

	userRepo := repository.NewUserRepository(client, db)
	userSvc := service.NewUserService(userRepo, nil, nil, nil)
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
