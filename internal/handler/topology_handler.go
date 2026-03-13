package handler

import (
	"github.com/gin-gonic/gin"
	bizerr "github.com/gocronx/kubevision/internal/pkg/errors"
	"github.com/gocronx/kubevision/internal/pkg/response"
	"github.com/gocronx/kubevision/internal/service"
)

// TopologyHandler handles resource topology requests.
type TopologyHandler struct {
	topologyService *service.TopologyService
}

// NewTopologyHandler creates a new TopologyHandler.
func NewTopologyHandler(topologyService *service.TopologyService) *TopologyHandler {
	return &TopologyHandler{topologyService: topologyService}
}

// GetTopology handles GET /api/v1/clusters/:id/namespaces/:namespace/topology.
// Returns a graph of resource relationships for the given namespace.
func (h *TopologyHandler) GetTopology(c *gin.Context) {
	clusterID, err := parseClusterID(c)
	if err != nil {
		response.Error(c, bizerr.CodeParamInvalid, "invalid clusterID")
		return
	}

	namespace := c.Param("namespace")
	if namespace == "" {
		response.Error(c, bizerr.CodeParamMissing, "namespace is required")
		return
	}

	result, err := h.topologyService.GetNamespaceTopology(c.Request.Context(), clusterID, namespace)
	if err != nil {
		if bizErr, ok := err.(*bizerr.BizError); ok {
			response.ErrorWithBizErr(c, bizErr)
			return
		}
		response.Error(c, bizerr.CodeInternal, "internal server error")
		return
	}

	response.Success(c, result)
}
