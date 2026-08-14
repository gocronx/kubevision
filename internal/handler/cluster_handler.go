package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
	bizerr "github.com/gocronx/kubevision/internal/pkg/errors"
	"github.com/gocronx/kubevision/internal/pkg/response"
	"github.com/gocronx/kubevision/internal/service"
)

// ClusterHandler handles HTTP requests for cluster management operations.
type ClusterHandler struct {
	clusterService *service.ClusterService
}

// NewClusterHandler creates a new ClusterHandler.
func NewClusterHandler(clusterService *service.ClusterService) *ClusterHandler {
	return &ClusterHandler{
		clusterService: clusterService,
	}
}

// List handles GET /api/v1/clusters.
func (h *ClusterHandler) List(c *gin.Context) {
	clusters, err := h.clusterService.List(c.Request.Context())
	if err != nil {
		if bizErr, ok := err.(*bizerr.BizError); ok {
			response.ErrorWithBizErr(c, bizErr)
			return
		}
		response.Error(c, bizerr.CodeInternal, "internal server error")
		return
	}
	response.Success(c, clusters)
}

// Get handles GET /api/v1/clusters/:id.
func (h *ClusterHandler) Get(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, strconv.IntSize)
	if err != nil {
		response.Error(c, bizerr.CodeParamInvalid, "invalid cluster id")
		return
	}

	cluster, err := h.clusterService.Get(c.Request.Context(), uint(id))
	if err != nil {
		if bizErr, ok := err.(*bizerr.BizError); ok {
			response.ErrorWithBizErr(c, bizErr)
			return
		}
		response.Error(c, bizerr.CodeInternal, "internal server error")
		return
	}
	response.Success(c, cluster)
}

// Create handles POST /api/v1/clusters.
func (h *ClusterHandler) Create(c *gin.Context) {
	var req service.AddClusterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, bizerr.CodeParamInvalid, "invalid request body")
		return
	}

	cluster, err := h.clusterService.Add(c.Request.Context(), &req)
	if err != nil {
		if bizErr, ok := err.(*bizerr.BizError); ok {
			response.ErrorWithBizErr(c, bizErr)
			return
		}
		response.Error(c, bizerr.CodeInternal, "internal server error")
		return
	}
	response.Success(c, cluster)
}

// Delete handles DELETE /api/v1/clusters/:id.
func (h *ClusterHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, strconv.IntSize)
	if err != nil {
		response.Error(c, bizerr.CodeParamInvalid, "invalid cluster id")
		return
	}

	if err := h.clusterService.Remove(c.Request.Context(), uint(id)); err != nil {
		if bizErr, ok := err.(*bizerr.BizError); ok {
			response.ErrorWithBizErr(c, bizErr)
			return
		}
		response.Error(c, bizerr.CodeInternal, "internal server error")
		return
	}
	response.Success(c, nil)
}
