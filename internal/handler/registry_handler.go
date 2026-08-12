package handler

import (
	"errors"
	"strings"

	"github.com/gin-gonic/gin"
	bizerr "github.com/gocronx/kubevision/internal/pkg/errors"
	"github.com/gocronx/kubevision/internal/pkg/response"
	"github.com/gocronx/kubevision/internal/registry"
)

// RegistryHandler exposes read-only registry metadata without accepting URLs.
type RegistryHandler struct{ service *registry.Service }

func NewRegistryHandler(service *registry.Service) *RegistryHandler {
	return &RegistryHandler{service: service}
}

type registryQuery struct {
	Image  string `form:"image"`
	Prefix string `form:"prefix"`
	Limit  int    `form:"limit"`
	Cursor string `form:"cursor"`
}

func (h *RegistryHandler) ListTags(c *gin.Context) {
	var query registryQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.Error(c, bizerr.CodeParamInvalid, "invalid query parameters")
		return
	}
	if strings.TrimSpace(query.Image) == "" {
		response.Error(c, bizerr.CodeParamMissing, "image is required")
		return
	}
	page, err := h.service.Discover(c.Request.Context(), query.Image, query.Prefix, query.Limit, query.Cursor)
	if err != nil {
		switch {
		case errors.Is(err, registry.ErrNotAllowed), errors.Is(err, registry.ErrUnsafeAddress):
			response.Error(c, bizerr.CodeForbidden, "registry access is not allowed")
		case errors.Is(err, registry.ErrAuthentication):
			response.Error(c, bizerr.CodeUnauthorized, "registry authentication failed")
		default:
			if _, parseErr := registry.ParseReference(query.Image); parseErr != nil {
				response.Error(c, bizerr.CodeParamInvalid, "invalid image reference")
			} else {
				response.Error(c, bizerr.CodeK8sUnavailable, "registry is unavailable")
			}
		}
		return
	}
	response.SuccessWithMeta(c, page, &response.Meta{Total: int64(len(page.Tags)), Source: "registry"})
}
