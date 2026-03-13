package handler

import (
	"strings"

	"github.com/gin-gonic/gin"
	bizerr "github.com/gocronx/kubevision/internal/pkg/errors"
	"github.com/gocronx/kubevision/internal/pkg/response"
	"github.com/gocronx/kubevision/internal/service"
)

const maxPromQLQueryLen = 1000

// validatePromQLQuery performs basic safety checks on a PromQL query string:
//   - Rejects queries longer than maxPromQLQueryLen characters.
//   - Rejects queries containing the broad wildcard pattern __name__=~".*"
//     which can result in extremely expensive full-metric-set scans.
func validatePromQLQuery(query string) error {
	if len(query) > maxPromQLQueryLen {
		return bizerr.New(bizerr.CodeParamInvalid, "query exceeds maximum allowed length")
	}
	if strings.Contains(query, `__name__=~".*"`) {
		return bizerr.New(bizerr.CodeParamInvalid, "query contains disallowed pattern")
	}
	return nil
}

// PluginHandler handles HTTP requests for plugin management.
type PluginHandler struct {
	pluginService *service.PluginService
}

// NewPluginHandler creates a new PluginHandler.
func NewPluginHandler(pluginService *service.PluginService) *PluginHandler {
	return &PluginHandler{pluginService: pluginService}
}

// List handles GET /api/v1/plugins.
func (h *PluginHandler) List(c *gin.Context) {
	plugins, err := h.pluginService.List(c.Request.Context())
	if err != nil {
		if bizErr, ok := err.(*bizerr.BizError); ok {
			response.ErrorWithBizErr(c, bizErr)
			return
		}
		response.Error(c, bizerr.CodeInternal, "internal server error")
		return
	}
	response.Success(c, plugins)
}

// GetConfig handles GET /api/v1/plugins/:name.
func (h *PluginHandler) GetConfig(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		response.Error(c, bizerr.CodeParamMissing, "plugin name required")
		return
	}

	cfg, err := h.pluginService.GetConfig(c.Request.Context(), name)
	if err != nil {
		if bizErr, ok := err.(*bizerr.BizError); ok {
			response.ErrorWithBizErr(c, bizErr)
			return
		}
		response.Error(c, bizerr.CodeInternal, "internal server error")
		return
	}
	response.Success(c, cfg)
}

// Configure handles PUT /api/v1/plugins/:name.
func (h *PluginHandler) Configure(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		response.Error(c, bizerr.CodeParamMissing, "plugin name required")
		return
	}

	var req service.PluginConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, bizerr.CodeParamInvalid, err.Error())
		return
	}

	cfg, err := h.pluginService.Configure(c.Request.Context(), name, &req)
	if err != nil {
		if bizErr, ok := err.(*bizerr.BizError); ok {
			response.ErrorWithBizErr(c, bizErr)
			return
		}
		response.Error(c, bizerr.CodeInternal, "internal server error")
		return
	}
	response.Success(c, cfg)
}

// HealthCheck handles GET /api/v1/plugins/:name/health.
func (h *PluginHandler) HealthCheck(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		response.Error(c, bizerr.CodeParamMissing, "plugin name required")
		return
	}

	if err := h.pluginService.HealthCheck(c.Request.Context(), name); err != nil {
		if bizErr, ok := err.(*bizerr.BizError); ok {
			response.ErrorWithBizErr(c, bizErr)
			return
		}
		response.Error(c, bizerr.CodeInternal, err.Error())
		return
	}
	response.Success(c, gin.H{"status": "healthy"})
}

// PrometheusQuery handles GET /api/v1/plugins/prometheus/query.
func (h *PluginHandler) PrometheusQuery(c *gin.Context) {
	p, ok := h.pluginService.GetPrometheus()
	if !ok {
		response.Error(c, bizerr.CodeNotFound, "prometheus plugin not available")
		return
	}

	query := c.Query("query")
	if query == "" {
		response.Error(c, bizerr.CodeParamMissing, "query parameter required")
		return
	}

	if err := validatePromQLQuery(query); err != nil {
		if bizErr, ok := err.(*bizerr.BizError); ok {
			response.ErrorWithBizErr(c, bizErr)
		} else {
			response.Error(c, bizerr.CodeParamInvalid, "invalid query")
		}
		return
	}

	result, err := p.Query(c.Request.Context(), query)
	if err != nil {
		response.Error(c, bizerr.CodeInternal, err.Error())
		return
	}
	response.Success(c, result)
}

// GrafanaDashboards handles GET /api/v1/plugins/grafana/dashboards.
func (h *PluginHandler) GrafanaDashboards(c *gin.Context) {
	p, ok := h.pluginService.GetGrafana()
	if !ok {
		response.Error(c, bizerr.CodeNotFound, "grafana plugin not available")
		return
	}

	dashboards, err := p.ListDashboards(c.Request.Context())
	if err != nil {
		response.Error(c, bizerr.CodeInternal, err.Error())
		return
	}
	response.Success(c, dashboards)
}

// ArgoCDApplications handles GET /api/v1/plugins/argocd/applications.
func (h *PluginHandler) ArgoCDApplications(c *gin.Context) {
	p, ok := h.pluginService.GetArgoCD()
	if !ok {
		response.Error(c, bizerr.CodeNotFound, "argocd plugin not available")
		return
	}

	apps, err := p.ListApplications(c.Request.Context())
	if err != nil {
		response.Error(c, bizerr.CodeInternal, err.Error())
		return
	}
	response.Success(c, apps)
}
