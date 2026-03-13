package handler

import (
	"strings"

	"github.com/gin-gonic/gin"
	bizerr "github.com/gocronx/kubevision/internal/pkg/errors"
	"github.com/gocronx/kubevision/internal/pkg/response"
	"github.com/gocronx/kubevision/internal/service"
)

// SearchHandler handles HTTP requests for the global resource search endpoint.
type SearchHandler struct {
	searchService *service.SearchService
}

// NewSearchHandler creates a new SearchHandler with the given SearchService.
func NewSearchHandler(searchService *service.SearchService) *SearchHandler {
	return &SearchHandler{searchService: searchService}
}

// searchQueryParams holds the validated query parameters for a search request.
type searchQueryParams struct {
	Q         string `form:"q"`
	Namespace string `form:"namespace"`
	Types     string `form:"types"`
	Limit     int    `form:"limit"`
	Offset    int    `form:"offset"`
}

// Search handles GET /api/v1/clusters/:id/search.
//
// Query parameters:
//
//	q         - required search string
//	namespace - optional namespace filter
//	types     - optional comma-separated resource type filter (e.g. "pods,deployments")
//	limit     - per-type result limit (default 10, max 100)
//	offset    - per-type result offset for pagination
func (h *SearchHandler) Search(c *gin.Context) {
	clusterID, err := parseClusterID(c)
	if err != nil {
		response.Error(c, bizerr.CodeParamInvalid, "invalid cluster id")
		return
	}

	var params searchQueryParams
	if err := c.ShouldBindQuery(&params); err != nil {
		response.Error(c, bizerr.CodeParamInvalid, "invalid query parameters")
		return
	}

	if strings.TrimSpace(params.Q) == "" {
		response.Error(c, bizerr.CodeParamMissing, "query parameter 'q' is required")
		return
	}

	// Parse the comma-separated types filter into a slice.
	var types []string
	if params.Types != "" {
		for _, t := range strings.Split(params.Types, ",") {
			if trimmed := strings.TrimSpace(t); trimmed != "" {
				types = append(types, trimmed)
			}
		}
	}

	opts := service.SearchOptions{
		Query:     params.Q,
		Namespace: params.Namespace,
		Types:     types,
		Limit:     params.Limit,
		Offset:    params.Offset,
	}

	result, err := h.searchService.Search(c.Request.Context(), clusterID, opts)
	if err != nil {
		if bizErr, ok := err.(*bizerr.BizError); ok {
			response.ErrorWithBizErr(c, bizErr)
			return
		}
		response.Error(c, bizerr.CodeInternal, "internal server error")
		return
	}

	response.SuccessWithMeta(c, result, &response.Meta{
		Total: int64(result.Total),
	})
}
