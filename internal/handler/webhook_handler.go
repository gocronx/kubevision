package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
	bizerr "github.com/gocronx/kubevision/internal/pkg/errors"
	"github.com/gocronx/kubevision/internal/pkg/response"
	"github.com/gocronx/kubevision/internal/service"
)

// WebhookHandler handles HTTP requests for webhook management.
type WebhookHandler struct {
	webhookService *service.WebhookService
}

// NewWebhookHandler creates a new WebhookHandler.
func NewWebhookHandler(webhookService *service.WebhookService) *WebhookHandler {
	return &WebhookHandler{webhookService: webhookService}
}

// List handles GET /api/v1/webhooks.
func (h *WebhookHandler) List(c *gin.Context) {
	whs, err := h.webhookService.List(c.Request.Context())
	if err != nil {
		if bizErr, ok := err.(*bizerr.BizError); ok {
			response.ErrorWithBizErr(c, bizErr)
			return
		}
		response.Error(c, bizerr.CodeInternal, "internal server error")
		return
	}
	response.Success(c, whs)
}

// Create handles POST /api/v1/webhooks.
func (h *WebhookHandler) Create(c *gin.Context) {
	var req service.WebhookRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, bizerr.CodeParamInvalid, err.Error())
		return
	}

	wh, err := h.webhookService.Create(c.Request.Context(), &req)
	if err != nil {
		if bizErr, ok := err.(*bizerr.BizError); ok {
			response.ErrorWithBizErr(c, bizErr)
			return
		}
		response.Error(c, bizerr.CodeInternal, "internal server error")
		return
	}
	response.Success(c, wh)
}

// Update handles PUT /api/v1/webhooks/:id.
func (h *WebhookHandler) Update(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, bizerr.CodeParamInvalid, "invalid webhook id")
		return
	}

	var req service.WebhookRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, bizerr.CodeParamInvalid, err.Error())
		return
	}

	wh, err := h.webhookService.Update(c.Request.Context(), id, &req)
	if err != nil {
		if bizErr, ok := err.(*bizerr.BizError); ok {
			response.ErrorWithBizErr(c, bizErr)
			return
		}
		response.Error(c, bizerr.CodeInternal, "internal server error")
		return
	}
	response.Success(c, wh)
}

// Delete handles DELETE /api/v1/webhooks/:id.
func (h *WebhookHandler) Delete(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, bizerr.CodeParamInvalid, "invalid webhook id")
		return
	}

	if err := h.webhookService.Delete(c.Request.Context(), id); err != nil {
		if bizErr, ok := err.(*bizerr.BizError); ok {
			response.ErrorWithBizErr(c, bizErr)
			return
		}
		response.Error(c, bizerr.CodeInternal, "internal server error")
		return
	}
	response.Success(c, nil)
}

// Test handles POST /api/v1/webhooks/:id/test.
func (h *WebhookHandler) Test(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, bizerr.CodeParamInvalid, "invalid webhook id")
		return
	}

	if err := h.webhookService.TestWebhook(c.Request.Context(), id); err != nil {
		if bizErr, ok := err.(*bizerr.BizError); ok {
			response.ErrorWithBizErr(c, bizErr)
			return
		}
		response.Error(c, bizerr.CodeInternal, "internal server error")
		return
	}
	response.Success(c, gin.H{"message": "test webhook sent successfully"})
}

// parseUintParam extracts a named URL parameter as a uint.
func parseUintParam(c *gin.Context, paramName string) (uint, error) {
	raw := c.Param(paramName)
	id, err := strconv.ParseUint(raw, 10, 64)
	if err != nil {
		return 0, err
	}
	return uint(id), nil
}
