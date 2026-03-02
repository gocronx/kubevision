package handler

import (
	"github.com/gin-gonic/gin"
	bizerr "github.com/kubevision/kubevision/internal/pkg/errors"
	"github.com/kubevision/kubevision/internal/pkg/response"
	"github.com/kubevision/kubevision/internal/service"
)

// QuotaHandler handles HTTP requests for ResourceQuota aggregation.
type QuotaHandler struct {
	quotaService *service.QuotaService
}

// NewQuotaHandler creates a new QuotaHandler.
func NewQuotaHandler(quotaService *service.QuotaService) *QuotaHandler {
	return &QuotaHandler{quotaService: quotaService}
}

// GetQuotaSummary handles GET /api/v1/clusters/:id/quota-summary.
//
// Query params:
//
//	namespace - (optional) filter results to a single namespace; omit for all namespaces
//
// Response:
//
//	{"namespaces": [{"namespace": "...", "quotas": [...]}]}
func (h *QuotaHandler) GetQuotaSummary(c *gin.Context) {
	clusterID, err := parseClusterID(c)
	if err != nil {
		response.Error(c, bizerr.CodeParamInvalid, "invalid clusterID")
		return
	}

	namespace := c.Query("namespace")

	summary, err := h.quotaService.GetQuotaSummary(c.Request.Context(), clusterID, namespace)
	if err != nil {
		if bizErr, ok := err.(*bizerr.BizError); ok {
			response.ErrorWithBizErr(c, bizErr)
			return
		}
		response.Error(c, bizerr.CodeInternal, "internal server error")
		return
	}

	response.Success(c, summary)
}
