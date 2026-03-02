package handler

import (
	"github.com/gin-gonic/gin"
	bizerr "github.com/kubevision/kubevision/internal/pkg/errors"
	"github.com/kubevision/kubevision/internal/pkg/response"
	"github.com/kubevision/kubevision/internal/service"
)

// CRDHandler handles HTTP requests for CRD discovery.
type CRDHandler struct {
	crdService *service.CRDService
}

// NewCRDHandler creates a new CRDHandler.
func NewCRDHandler(crdService *service.CRDService) *CRDHandler {
	return &CRDHandler{crdService: crdService}
}

// List handles GET /api/v1/clusters/:id/crds.
func (h *CRDHandler) List(c *gin.Context) {
	clusterID := c.Param("id")
	if clusterID == "" {
		response.Error(c, bizerr.CodeParamMissing, "cluster id required")
		return
	}

	crds, err := h.crdService.ListCached(c.Request.Context(), clusterID)
	if err != nil {
		response.Error(c, bizerr.CodeK8sUnavailable, err.Error())
		return
	}
	response.Success(c, crds)
}

// Refresh handles POST /api/v1/clusters/:id/crds/refresh.
func (h *CRDHandler) Refresh(c *gin.Context) {
	clusterID := c.Param("id")
	if clusterID == "" {
		response.Error(c, bizerr.CodeParamMissing, "cluster id required")
		return
	}

	crds, err := h.crdService.Discover(c.Request.Context(), clusterID)
	if err != nil {
		response.Error(c, bizerr.CodeK8sUnavailable, err.Error())
		return
	}
	response.Success(c, crds)
}
