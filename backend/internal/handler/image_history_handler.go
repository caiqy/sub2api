package handler

import (
	"errors"
	"strconv"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

type ImageHistoryHandler struct {
	imageHistoryService *service.ImageHistoryService
	apiKeyService       *service.APIKeyService
}

func NewImageHistoryHandler(imageHistoryService *service.ImageHistoryService, apiKeyService *service.APIKeyService) *ImageHistoryHandler {
	return &ImageHistoryHandler{
		imageHistoryService: imageHistoryService,
		apiKeyService:       apiKeyService,
	}
}

func (h *ImageHistoryHandler) List(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	page, pageSize := response.ParsePagination(c)
	mode, err := parseImageHistoryTab(c.Query("tab"))
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	status, err := parseImageHistoryStatus(c.Query("status"))
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	query := service.ImageHistoryListQuery{
		Mode:     mode,
		Status:   status,
		Page:     page,
		PageSize: pageSize,
	}
	if apiKeyIDStr := strings.TrimSpace(c.Query("api_key_id")); apiKeyIDStr != "" {
		apiKeyID, err := strconv.ParseInt(apiKeyIDStr, 10, 64)
		if err != nil || apiKeyID <= 0 {
			response.BadRequest(c, "Invalid api_key_id")
			return
		}
		apiKey, err := h.apiKeyService.GetByID(c.Request.Context(), apiKeyID)
		if err != nil {
			response.ErrorFrom(c, err)
			return
		}
		if apiKey.UserID != subject.UserID {
			response.Forbidden(c, "Not authorized to access this API key's image history")
			return
		}
		query.APIKeyID = apiKeyID
	}

	items, result, err := h.imageHistoryService.List(c.Request.Context(), subject.UserID, query)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	out := make([]dto.ImageHistoryListItem, 0, len(items))
	for i := range items {
		mapped := dto.ImageHistoryListItemFromService(&items[i])
		if mapped != nil {
			out = append(out, *mapped)
		}
	}
	response.PaginatedWithResult(c, out, &response.PaginationResult{
		Total:    result.Total,
		Page:     result.Page,
		PageSize: result.PageSize,
		Pages:    result.Pages,
	})
}

func (h *ImageHistoryHandler) GetByID(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	usageLogID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || usageLogID <= 0 {
		response.BadRequest(c, "Invalid image history ID")
		return
	}

	detail, err := h.imageHistoryService.GetDetail(c.Request.Context(), subject.UserID, usageLogID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, dto.ImageHistoryDetailFromService(detail))
}

func parseImageHistoryTab(value string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(value))
	switch normalized {
	case "", string(service.ImageHistoryModeGenerate), string(service.ImageHistoryModeEdit):
		return normalized, nil
	default:
		return "", errors.New("Invalid tab")
	}
}

func parseImageHistoryStatus(value string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(value))
	switch normalized {
	case "", string(service.ImageHistoryStatusSuccess), string(service.ImageHistoryStatusError):
		return normalized, nil
	default:
		return "", errors.New("Invalid status")
	}
}
