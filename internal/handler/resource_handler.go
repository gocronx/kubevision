package handler

import (
	"io"
	"strconv"

	"github.com/gin-gonic/gin"
	bizerr "github.com/gocronx/kubevision/internal/pkg/errors"
	"github.com/gocronx/kubevision/internal/pkg/response"
	"github.com/gocronx/kubevision/internal/repository"
	"github.com/gocronx/kubevision/internal/service"
)

// ResourceHandler handles HTTP requests for Kubernetes resource operations
// (list, get, create, update, delete, patch, batch operations).
type ResourceHandler struct {
	resourceService       *service.ResourceService
	resourceActionService *service.ResourceActionService
	podMetricsService     *service.PodMetricsService
}

// NewResourceHandler creates a new ResourceHandler with the given services.
func NewResourceHandler(resourceService *service.ResourceService, resourceActionService ...*service.ResourceActionService) *ResourceHandler {
	h := &ResourceHandler{
		resourceService: resourceService,
	}
	if len(resourceActionService) > 0 {
		h.resourceActionService = resourceActionService[0]
	}
	return h
}

// WithPodMetrics enables optional metrics-server data on Pod responses.
func (h *ResourceHandler) WithPodMetrics(metrics *service.PodMetricsService) *ResourceHandler {
	h.podMetricsService = metrics
	return h
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
	if resourceName == "pods" && c.Query("includeMetrics") == "true" {
		items := make([]*repository.Resource, len(result.Items))
		for i := range result.Items {
			items[i] = &result.Items[i]
		}
		h.attachPodMetrics(c, clusterID, namespace, items)
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
	if resourceName == "pods" && c.Query("includeMetrics") == "true" {
		h.attachPodMetrics(c, clusterID, namespace, []*repository.Resource{res})
	}

	response.Success(c, res)
}

func (h *ResourceHandler) attachPodMetrics(c *gin.Context, clusterID uint, namespace string, resources []*repository.Resource) {
	if h.podMetricsService == nil {
		for _, item := range resources {
			item.MetricsStatus = "unavailable"
		}
		return
	}
	metrics, err := h.podMetricsService.List(c.Request.Context(), clusterID, namespace)
	if err != nil {
		for _, item := range resources {
			item.MetricsStatus = "unavailable"
		}
		return
	}
	for _, item := range resources {
		usage, found := metrics[item.Namespace+"/"+item.Name]
		if !found {
			item.MetricsStatus = "pending"
			continue
		}
		service.ApplyPodResourceAllocations(usage, item.Raw)
		item.Metrics = usage
		item.MetricsStatus = "available"
	}
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

	body, err := io.ReadAll(io.LimitReader(c.Request.Body, 2*1024*1024))
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

	body, err := io.ReadAll(io.LimitReader(c.Request.Body, 2*1024*1024))
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

	body, err := io.ReadAll(io.LimitReader(c.Request.Body, 2*1024*1024))
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

	body, err := io.ReadAll(io.LimitReader(c.Request.Body, 2*1024*1024))
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

	body, err := io.ReadAll(io.LimitReader(c.Request.Body, 2*1024*1024))
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

// BatchDelete handles POST /api/v1/clusters/:clusterID/resources/batch-delete.
// Body: JSON array of {resource, name, namespace} items.
func (h *ResourceHandler) BatchDelete(c *gin.Context) {
	clusterID, err := parseClusterID(c)
	if err != nil {
		response.Error(c, bizerr.CodeParamInvalid, "invalid clusterID")
		return
	}

	var items []struct {
		Resource  string `json:"resource"`
		Name      string `json:"name"`
		Namespace string `json:"namespace"`
	}
	if err := c.ShouldBindJSON(&items); err != nil {
		response.Error(c, bizerr.CodeParamInvalid, "invalid request body")
		return
	}
	if len(items) == 0 {
		response.Error(c, bizerr.CodeParamMissing, "at least one item is required")
		return
	}
	if len(items) > 50 {
		response.Error(c, bizerr.CodeParamInvalid, "maximum 50 items per batch")
		return
	}

	type batchResult struct {
		Resource  string `json:"resource"`
		Name      string `json:"name"`
		Namespace string `json:"namespace"`
		Success   bool   `json:"success"`
		Error     string `json:"error,omitempty"`
	}

	results := make([]batchResult, 0, len(items))
	for _, item := range items {
		err := h.resourceService.DeleteResource(c.Request.Context(), clusterID, item.Resource, item.Namespace, item.Name)
		r := batchResult{
			Resource:  item.Resource,
			Name:      item.Name,
			Namespace: item.Namespace,
			Success:   err == nil,
		}
		if err != nil {
			r.Error = err.Error()
		}
		results = append(results, r)
	}

	response.Success(c, results)
}

// BatchRestart handles POST /api/v1/clusters/:clusterID/batch-restart.
// Body: JSON array of {kind, name, namespace} items.
func (h *ResourceHandler) BatchRestart(c *gin.Context) {
	if h.resourceActionService == nil {
		response.Error(c, bizerr.CodeInternal, "restart service not available")
		return
	}

	clusterID, err := parseClusterID(c)
	if err != nil {
		response.Error(c, bizerr.CodeParamInvalid, "invalid clusterID")
		return
	}

	var items []struct {
		Kind      string `json:"kind"`
		Name      string `json:"name"`
		Namespace string `json:"namespace"`
	}
	if err := c.ShouldBindJSON(&items); err != nil {
		response.Error(c, bizerr.CodeParamInvalid, "invalid request body")
		return
	}
	if len(items) == 0 {
		response.Error(c, bizerr.CodeParamMissing, "at least one item is required")
		return
	}
	if len(items) > 50 {
		response.Error(c, bizerr.CodeParamInvalid, "maximum 50 items per batch")
		return
	}

	type batchResult struct {
		Kind      string `json:"kind"`
		Name      string `json:"name"`
		Namespace string `json:"namespace"`
		Success   bool   `json:"success"`
		Error     string `json:"error,omitempty"`
	}

	results := make([]batchResult, 0, len(items))
	for _, item := range items {
		err := h.resourceActionService.Restart(c.Request.Context(), clusterID, item.Kind, item.Namespace, item.Name)
		r := batchResult{
			Kind:      item.Kind,
			Name:      item.Name,
			Namespace: item.Namespace,
			Success:   err == nil,
		}
		if err != nil {
			r.Error = err.Error()
		}
		results = append(results, r)
	}

	response.Success(c, results)
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
