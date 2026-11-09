package admin

import (
	"github.com/Wei-Shaw/sub2api/internal/pkg/adminaccess"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/gin-gonic/gin"
)

// GetRemainingQuota reports all undeleted users using the shared saved exclusion list.
func (h *UserHandler) GetRemainingQuota(c *gin.Context) {
	excluded, err := h.settingService.GetRemainingQuotaExclusions(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	summary, err := h.adminService.GetRemainingQuota(c.Request.Context(), excluded)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	summary.ExcludedUserIDs = excluded
	response.Success(c, summary)
}

func (h *UserHandler) SetRemainingQuotaExclusions(c *gin.Context) {
	if adminaccess.RestrictedFromContext(c.Request.Context()) != nil {
		response.Forbidden(c, "Only full administrators can save remaining quota exclusions")
		return
	}
	var req struct {
		ExcludedUserIDs *[]int64 `json:"excluded_user_ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.ExcludedUserIDs == nil {
		response.BadRequest(c, "excluded_user_ids must be an array of integer IDs")
		return
	}
	ids, err := h.settingService.SetRemainingQuotaExclusions(c.Request.Context(), *req.ExcludedUserIDs)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"excluded_user_ids": ids})
}
