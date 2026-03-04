package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
	bizerr "github.com/kubevision/kubevision/internal/pkg/errors"
	"github.com/kubevision/kubevision/internal/pkg/response"
	"github.com/kubevision/kubevision/internal/service"
)

// TemplateHandler handles HTTP requests for resource templates.
type TemplateHandler struct {
	templateService *service.TemplateService
}

// NewTemplateHandler creates a new TemplateHandler.
func NewTemplateHandler(templateService *service.TemplateService) *TemplateHandler {
	return &TemplateHandler{templateService: templateService}
}

// List handles GET /api/v1/templates.
// Query param: category (optional).
func (h *TemplateHandler) List(c *gin.Context) {
	category := c.Query("category")
	templates, err := h.templateService.List(c.Request.Context(), category)
	if err != nil {
		if bizErr, ok := err.(*bizerr.BizError); ok {
			response.ErrorWithBizErr(c, bizErr)
			return
		}
		response.Error(c, bizerr.CodeInternal, "internal server error")
		return
	}
	response.Success(c, templates)
}

// Get handles GET /api/v1/templates/:id.
func (h *TemplateHandler) Get(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, bizerr.CodeParamInvalid, "invalid template id")
		return
	}

	tmpl, err := h.templateService.Get(c.Request.Context(), uint(id))
	if err != nil {
		if bizErr, ok := err.(*bizerr.BizError); ok {
			response.ErrorWithBizErr(c, bizErr)
			return
		}
		response.Error(c, bizerr.CodeInternal, "internal server error")
		return
	}
	response.Success(c, tmpl)
}

// Create handles POST /api/v1/templates.
func (h *TemplateHandler) Create(c *gin.Context) {
	var req service.CreateTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, bizerr.CodeParamInvalid, "invalid request body")
		return
	}

	tmpl, err := h.templateService.Create(c.Request.Context(), &req)
	if err != nil {
		if bizErr, ok := err.(*bizerr.BizError); ok {
			response.ErrorWithBizErr(c, bizErr)
			return
		}
		response.Error(c, bizerr.CodeInternal, "internal server error")
		return
	}
	response.Success(c, tmpl)
}

// Delete handles DELETE /api/v1/templates/:id.
func (h *TemplateHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, bizerr.CodeParamInvalid, "invalid template id")
		return
	}

	if err := h.templateService.Delete(c.Request.Context(), uint(id)); err != nil {
		if bizErr, ok := err.(*bizerr.BizError); ok {
			response.ErrorWithBizErr(c, bizErr)
			return
		}
		response.Error(c, bizerr.CodeInternal, "internal server error")
		return
	}
	response.Success(c, nil)
}
