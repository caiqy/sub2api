package admin

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/modeltrace"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type ModelTraceHandler struct {
	svc      *service.ModelTraceService
	settings *service.SettingService
}

func NewModelTraceHandler(svc *service.ModelTraceService, settings *service.SettingService) *ModelTraceHandler {
	return &ModelTraceHandler{svc: svc, settings: settings}
}

func modelTraceID(c *gin.Context, raw string) (int64, bool) {
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id < 1 {
		response.BadRequest(c, "account ID must be a positive integer")
		return 0, false
	}
	return id, true
}

func modelTraceError(c *gin.Context, err error) {
	var appErr *infraerrors.ApplicationError
	if errors.As(err, &appErr) {
		response.ErrorFrom(c, appErr)
		return
	}
	response.InternalError(c, "ModelTrace operation failed")
}

func modelTraceBody(c *gin.Context, target any) bool {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 4096)
	decoder := json.NewDecoder(c.Request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		response.BadRequest(c, "invalid ModelTrace request")
		return false
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		response.BadRequest(c, "invalid ModelTrace request")
		return false
	}
	return true
}

func (h *ModelTraceHandler) Models(c *gin.Context) {
	var id int64
	if values, present := c.Request.URL.Query()["account_id"]; present {
		if len(values) != 1 {
			response.BadRequest(c, "invalid account_id")
			return
		}
		var ok bool
		id, ok = modelTraceID(c, values[0])
		if !ok {
			return
		}
	}
	models, err := h.svc.Models(c.Request.Context(), id)
	if err != nil {
		modelTraceError(c, err)
		return
	}
	response.Success(c, gin.H{"models": models, "version": modeltrace.Version})
}

func (h *ModelTraceHandler) GetSettings(c *gin.Context) {
	cfg, err := h.settings.GetModelTraceSettings(c.Request.Context())
	if err != nil {
		modelTraceError(c, err)
		return
	}
	response.Success(c, cfg)
}

func (h *ModelTraceHandler) PutSettings(c *gin.Context) {
	var cfg service.ModelTraceSettings
	if !modelTraceBody(c, &cfg) {
		return
	}
	if err := h.settings.SetModelTraceSettings(c.Request.Context(), cfg); err != nil {
		modelTraceError(c, err)
		return
	}
	response.Success(c, cfg)
}

func (h *ModelTraceHandler) Create(c *gin.Context) {
	id, ok := modelTraceID(c, c.Param("id"))
	if !ok {
		return
	}
	var request struct {
		Model  string `json:"model"`
		Rounds *int   `json:"rounds"`
	}
	if !modelTraceBody(c, &request) {
		return
	}
	rounds := 1
	if request.Rounds != nil {
		rounds = *request.Rounds
	}
	task, err := h.svc.Create(c.Request.Context(), id, request.Model, rounds)
	if err != nil {
		modelTraceError(c, err)
		return
	}
	response.Success(c, task)
}

func (h *ModelTraceHandler) History(c *gin.Context) {
	id, ok := modelTraceID(c, c.Param("id"))
	if !ok {
		return
	}
	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil {
		response.BadRequest(c, "invalid page")
		return
	}
	size, err := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if err != nil {
		response.BadRequest(c, "invalid page_size")
		return
	}
	history, err := h.svc.History(c.Request.Context(), id, page, size)
	if err != nil {
		modelTraceError(c, err)
		return
	}
	response.Success(c, history)
}
