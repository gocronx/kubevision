package handler

import (
	"github.com/gin-gonic/gin"
	bizerr "github.com/gocronx/kubevision/internal/pkg/errors"
	"github.com/gocronx/kubevision/internal/pkg/response"
	"github.com/gocronx/kubevision/internal/service"
)

// OverviewHandler handles HTTP requests for cluster overview aggregation.
type OverviewHandler struct {
	overviewService *service.OverviewService
}

// NewOverviewHandler creates a new OverviewHandler.
func NewOverviewHandler(overviewService *service.OverviewService) *OverviewHandler {
	return &OverviewHandler{overviewService: overviewService}
}

// GetOverview handles GET /api/v1/clusters/:id/overview.
//
// Response:
//
//	{"pods": 12, "deployments": 4, "services": 7, "nodes": 3}
func (h *OverviewHandler) GetOverview(c *gin.Context) {
	clusterID, err := parseClusterID(c)
	if err != nil {
		response.Error(c, bizerr.CodeParamInvalid, "invalid clusterID")
		return
	}

	overview, err := h.overviewService.GetOverview(c.Request.Context(), clusterID)
	if err != nil {
		if bizErr, ok := err.(*bizerr.BizError); ok {
			response.ErrorWithBizErr(c, bizErr)
			return
		}
		response.Error(c, bizerr.CodeInternal, "internal server error")
		return
	}

	response.Success(c, overview)
}
