package handler

import (
	"io"
	"strconv"

	"github.com/gin-gonic/gin"
	bizerr "github.com/kubevision/kubevision/internal/pkg/errors"
	"github.com/kubevision/kubevision/internal/pkg/response"
	"github.com/kubevision/kubevision/internal/repository"
	"github.com/kubevision/kubevision/internal/service"
)

// ResourceHandler handles HTTP requests for Kubernetes resource operations
// (list, get, create, update, delete, patch).
type ResourceHandler struct {
	resourceService *service.ResourceService
}

// NewResourceHandler creates a new ResourceHandler with the given ResourceService.
func NewResourceHandler(resourceService *service.ResourceService) *ResourceHandler {
	return &ResourceHandler{
		resourceService: resourceService,
	}
}

// List handles GET /api/v1/clusters/:clusterID/resources/:resource.
// Query params: namespace, labelSelector, fieldSelector, limit, continue.
func (h *ResourceHandler) List(c *gin.Context) {
	clusterID, err := parseClusterID(c)
	if err != nil {
		response.Error(c, bizerr.CodeParamInvalid, "invalid clusterID")
		return
	}

	resourceName := c.Param("resource")
	if resourceName == "" {
		response.Error(c, bizerr.CodeParamMissing, "resource type is required")
		return
	}

	namespace := c.Query("namespace")

	var opts repository.ListOptions
	if err := c.ShouldBindQuery(&opts); err != nil {
		response.Error(c, bizerr.CodeParamInvalid, "invalid query parameters")
		return
	}

	result, err := h.resourceService.ListResources(c.Request.Context(), clusterID, resourceName, namespace, opts)
	if err != nil {
		if bizErr, ok := err.(*bizerr.BizError); ok {
			response.ErrorWithBizErr(c, bizErr)
			return
		}
		response.Error(c, bizerr.CodeInternal, "internal server error")
		return
	}

	response.SuccessWithMeta(c, result.Items, &response.Meta{
		Total: result.Total,
		Stale: result.Stale,
	})
}

// Get handles GET /api/v1/clusters/:clusterID/resources/:resource/:name.
// Query param: namespace.
func (h *ResourceHandler) Get(c *gin.Context) {
	clusterID, err := parseClusterID(c)
	if err != nil {
		response.Error(c, bizerr.CodeParamInvalid, "invalid clusterID")
		return
	}

	resourceName := c.Param("resource")
	if resourceName == "" {
		response.Error(c, bizerr.CodeParamMissing, "resource type is required")
		return
	}

	name := c.Param("name")
	if name == "" {
		response.Error(c, bizerr.CodeParamMissing, "resource name is required")
		return
	}

	namespace := c.Query("namespace")

	res, err := h.resourceService.GetResource(c.Request.Context(), clusterID, resourceName, namespace, name)
	if err != nil {
		if bizErr, ok := err.(*bizerr.BizError); ok {
			response.ErrorWithBizErr(c, bizErr)
			return
		}
		response.Error(c, bizerr.CodeInternal, "internal server error")
		return
	}

	response.Success(c, res)
}

// Create handles POST /api/v1/clusters/:clusterID/resources/:resource.
// Query param: namespace. Body: JSON resource manifest.
func (h *ResourceHandler) Create(c *gin.Context) {
	clusterID, err := parseClusterID(c)
	if err != nil {
		response.Error(c, bizerr.CodeParamInvalid, "invalid clusterID")
		return
	}

	resourceName := c.Param("resource")
	if resourceName == "" {
		response.Error(c, bizerr.CodeParamMissing, "resource type is required")
		return
	}

	namespace := c.Query("namespace")

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		response.Error(c, bizerr.CodeParamInvalid, "failed to read request body")
		return
	}
	if len(body) == 0 {
		response.Error(c, bizerr.CodeParamMissing, "request body is required")
		return
	}

	res, err := h.resourceService.CreateResource(c.Request.Context(), clusterID, resourceName, namespace, body)
	if err != nil {
		if bizErr, ok := err.(*bizerr.BizError); ok {
			response.ErrorWithBizErr(c, bizErr)
			return
		}
		response.Error(c, bizerr.CodeInternal, "internal server error")
		return
	}

	response.Success(c, res)
}

// Update handles PUT /api/v1/clusters/:clusterID/resources/:resource/:name.
// Query param: namespace. Body: JSON resource manifest.
func (h *ResourceHandler) Update(c *gin.Context) {
	clusterID, err := parseClusterID(c)
	if err != nil {
		response.Error(c, bizerr.CodeParamInvalid, "invalid clusterID")
		return
	}

	resourceName := c.Param("resource")
	if resourceName == "" {
		response.Error(c, bizerr.CodeParamMissing, "resource type is required")
		return
	}

	name := c.Param("name")
	if name == "" {
		response.Error(c, bizerr.CodeParamMissing, "resource name is required")
		return
	}

	namespace := c.Query("namespace")

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		response.Error(c, bizerr.CodeParamInvalid, "failed to read request body")
		return
	}
	if len(body) == 0 {
		response.Error(c, bizerr.CodeParamMissing, "request body is required")
		return
	}

	res, err := h.resourceService.UpdateResource(c.Request.Context(), clusterID, resourceName, namespace, name, body)
	if err != nil {
		if bizErr, ok := err.(*bizerr.BizError); ok {
			response.ErrorWithBizErr(c, bizErr)
			return
		}
		response.Error(c, bizerr.CodeInternal, "internal server error")
		return
	}

	response.Success(c, res)
}

// Delete handles DELETE /api/v1/clusters/:clusterID/resources/:resource/:name.
// Query param: namespace.
func (h *ResourceHandler) Delete(c *gin.Context) {
	clusterID, err := parseClusterID(c)
	if err != nil {
		response.Error(c, bizerr.CodeParamInvalid, "invalid clusterID")
		return
	}

	resourceName := c.Param("resource")
	if resourceName == "" {
		response.Error(c, bizerr.CodeParamMissing, "resource type is required")
		return
	}

	name := c.Param("name")
	if name == "" {
		response.Error(c, bizerr.CodeParamMissing, "resource name is required")
		return
	}

	namespace := c.Query("namespace")

	err = h.resourceService.DeleteResource(c.Request.Context(), clusterID, resourceName, namespace, name)
	if err != nil {
		if bizErr, ok := err.(*bizerr.BizError); ok {
			response.ErrorWithBizErr(c, bizErr)
			return
		}
		response.Error(c, bizerr.CodeInternal, "internal server error")
		return
	}

	response.Success(c, nil)
}

// Patch handles PATCH /api/v1/clusters/:clusterID/resources/:resource/:name.
// Query param: namespace. Body: JSON patch data.
func (h *ResourceHandler) Patch(c *gin.Context) {
	clusterID, err := parseClusterID(c)
	if err != nil {
		response.Error(c, bizerr.CodeParamInvalid, "invalid clusterID")
		return
	}

	resourceName := c.Param("resource")
	if resourceName == "" {
		response.Error(c, bizerr.CodeParamMissing, "resource type is required")
		return
	}

	name := c.Param("name")
	if name == "" {
		response.Error(c, bizerr.CodeParamMissing, "resource name is required")
		return
	}

	namespace := c.Query("namespace")

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		response.Error(c, bizerr.CodeParamInvalid, "failed to read request body")
		return
	}
	if len(body) == 0 {
		response.Error(c, bizerr.CodeParamMissing, "request body is required")
		return
	}

	res, err := h.resourceService.PatchResource(c.Request.Context(), clusterID, resourceName, namespace, name, body)
	if err != nil {
		if bizErr, ok := err.(*bizerr.BizError); ok {
			response.ErrorWithBizErr(c, bizErr)
			return
		}
		response.Error(c, bizerr.CodeInternal, "internal server error")
		return
	}

	response.Success(c, res)
}

// DryRunCreate handles POST /api/v1/clusters/:clusterID/resources/:resource/dry-run.
// It performs a Kubernetes server-side dry-run create and returns what the resource
// would look like after creation (with API server defaults applied) without
// actually persisting anything.
//
// Query param: namespace. Body: JSON resource manifest.
//
// Response data shape:
//
//	{
//	  "current": null,
//	  "proposed": { ... resource object ... },
//	  "valid": true,
//	  "errors": []
//	}
func (h *ResourceHandler) DryRunCreate(c *gin.Context) {
	clusterID, err := parseClusterID(c)
	if err != nil {
		response.Error(c, bizerr.CodeParamInvalid, "invalid clusterID")
		return
	}

	resourceName := c.Param("resource")
	if resourceName == "" {
		response.Error(c, bizerr.CodeParamMissing, "resource type is required")
		return
	}

	namespace := c.Query("namespace")

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		response.Error(c, bizerr.CodeParamInvalid, "failed to read request body")
		return
	}
	if len(body) == 0 {
		response.Error(c, bizerr.CodeParamMissing, "request body is required")
		return
	}

	result, err := h.resourceService.DryRunCreateResource(c.Request.Context(), clusterID, resourceName, namespace, body)
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

// DryRunUpdate handles PUT /api/v1/clusters/:clusterID/resources/:resource/:name/dry-run.
// It performs a Kubernetes server-side dry-run update and returns both the current
// live resource and what the resource would look like after the update, without
// actually applying the change.
//
// Query param: namespace. Body: JSON resource manifest.
//
// Response data shape:
//
//	{
//	  "current": { ... current resource ... },
//	  "proposed": { ... proposed resource ... },
//	  "valid": true,
//	  "errors": []
//	}
func (h *ResourceHandler) DryRunUpdate(c *gin.Context) {
	clusterID, err := parseClusterID(c)
	if err != nil {
		response.Error(c, bizerr.CodeParamInvalid, "invalid clusterID")
		return
	}

	resourceName := c.Param("resource")
	if resourceName == "" {
		response.Error(c, bizerr.CodeParamMissing, "resource type is required")
		return
	}

	name := c.Param("name")
	if name == "" {
		response.Error(c, bizerr.CodeParamMissing, "resource name is required")
		return
	}

	namespace := c.Query("namespace")

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		response.Error(c, bizerr.CodeParamInvalid, "failed to read request body")
		return
	}
	if len(body) == 0 {
		response.Error(c, bizerr.CodeParamMissing, "request body is required")
		return
	}

	result, err := h.resourceService.DryRunUpdateResource(c.Request.Context(), clusterID, resourceName, namespace, name, body)
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

// parseClusterID extracts and validates the cluster :id URL parameter as a uint.
func parseClusterID(c *gin.Context) (uint, error) {
	raw := c.Param("id")
	id, err := strconv.ParseUint(raw, 10, 64)
	if err != nil {
		return 0, err
	}
	return uint(id), nil
}
