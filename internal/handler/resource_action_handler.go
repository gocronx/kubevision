package handler

import (
	"github.com/gin-gonic/gin"
	bizerr "github.com/kubevision/kubevision/internal/pkg/errors"
	"github.com/kubevision/kubevision/internal/pkg/response"
	"github.com/kubevision/kubevision/internal/service"
)

// ResourceActionHandler handles HTTP requests for workload lifecycle operations:
// scale, restart, rollback, and rollout history.
type ResourceActionHandler struct {
	actionService *service.ResourceActionService
}

// NewResourceActionHandler creates a new ResourceActionHandler.
func NewResourceActionHandler(actionService *service.ResourceActionService) *ResourceActionHandler {
	return &ResourceActionHandler{actionService: actionService}
}

// Scale handles PUT /api/v1/clusters/:id/namespaces/:namespace/:kind/:name/scale.
// Body: {"replicas": <int>}
// Supported kinds: deployments, statefulsets, replicasets.
func (h *ResourceActionHandler) Scale(c *gin.Context) {
	clusterID, err := parseClusterID(c)
	if err != nil {
		response.Error(c, bizerr.CodeParamInvalid, "invalid cluster id")
		return
	}

	kind := c.Param("kind")
	namespace := c.Param("namespace")
	name := c.Param("name")

	if kind == "" || namespace == "" || name == "" {
		response.Error(c, bizerr.CodeParamMissing, "kind, namespace, and name are required")
		return
	}

	var req service.ScaleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, bizerr.CodeParamInvalid, "invalid request body: replicas field is required")
		return
	}

	if err := h.actionService.Scale(c.Request.Context(), clusterID, kind, namespace, name, req); err != nil {
		if bizErr, ok := err.(*bizerr.BizError); ok {
			response.ErrorWithBizErr(c, bizErr)
			return
		}
		response.Error(c, bizerr.CodeInternal, "internal server error")
		return
	}

	response.Success(c, nil)
}

// Restart handles POST /api/v1/clusters/:id/namespaces/:namespace/:kind/:name/restart.
// No request body required.
// Supported kinds: deployments, statefulsets, daemonsets.
func (h *ResourceActionHandler) Restart(c *gin.Context) {
	clusterID, err := parseClusterID(c)
	if err != nil {
		response.Error(c, bizerr.CodeParamInvalid, "invalid cluster id")
		return
	}

	kind := c.Param("kind")
	namespace := c.Param("namespace")
	name := c.Param("name")

	if kind == "" || namespace == "" || name == "" {
		response.Error(c, bizerr.CodeParamMissing, "kind, namespace, and name are required")
		return
	}

	if err := h.actionService.Restart(c.Request.Context(), clusterID, kind, namespace, name); err != nil {
		if bizErr, ok := err.(*bizerr.BizError); ok {
			response.ErrorWithBizErr(c, bizErr)
			return
		}
		response.Error(c, bizerr.CodeInternal, "internal server error")
		return
	}

	response.Success(c, nil)
}

// RolloutHistory handles GET /api/v1/clusters/:id/namespaces/:namespace/deployments/:name/history.
// Returns a list of rollout revisions with change-cause annotations.
func (h *ResourceActionHandler) RolloutHistory(c *gin.Context) {
	clusterID, err := parseClusterID(c)
	if err != nil {
		response.Error(c, bizerr.CodeParamInvalid, "invalid cluster id")
		return
	}

	namespace := c.Param("namespace")
	name := c.Param("name")

	if namespace == "" || name == "" {
		response.Error(c, bizerr.CodeParamMissing, "namespace and name are required")
		return
	}

	revisions, err := h.actionService.RolloutHistory(c.Request.Context(), clusterID, namespace, name)
	if err != nil {
		if bizErr, ok := err.(*bizerr.BizError); ok {
			response.ErrorWithBizErr(c, bizErr)
			return
		}
		response.Error(c, bizerr.CodeInternal, "internal server error")
		return
	}

	response.Success(c, revisions)
}

// Rollback handles POST /api/v1/clusters/:id/namespaces/:namespace/deployments/:name/rollback.
// Optional body: {"revision": <int>}. Omitting revision (or passing 0) rolls back to the previous revision.
func (h *ResourceActionHandler) Rollback(c *gin.Context) {
	clusterID, err := parseClusterID(c)
	if err != nil {
		response.Error(c, bizerr.CodeParamInvalid, "invalid cluster id")
		return
	}

	namespace := c.Param("namespace")
	name := c.Param("name")

	if namespace == "" || name == "" {
		response.Error(c, bizerr.CodeParamMissing, "namespace and name are required")
		return
	}

	// Body is optional: if absent or empty, use zero-value (revision=0 → previous).
	var req service.RollbackRequest
	_ = c.ShouldBindJSON(&req) // ignore bind error — body is optional

	if err := h.actionService.Rollback(c.Request.Context(), clusterID, namespace, name, req); err != nil {
		if bizErr, ok := err.(*bizerr.BizError); ok {
			response.ErrorWithBizErr(c, bizErr)
			return
		}
		response.Error(c, bizerr.CodeInternal, "internal server error")
		return
	}

	response.Success(c, nil)
}
