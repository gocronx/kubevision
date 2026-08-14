package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/gocronx/kubevision/internal/middleware"
	bizerr "github.com/gocronx/kubevision/internal/pkg/errors"
	"github.com/gocronx/kubevision/internal/pkg/response"
	"github.com/gocronx/kubevision/internal/service"
)

// APIKeyHandler handles HTTP requests for API key management.
type APIKeyHandler struct {
	apiKeyService *service.APIKeyService
}

// NewAPIKeyHandler creates a new APIKeyHandler.
func NewAPIKeyHandler(apiKeyService *service.APIKeyService) *APIKeyHandler {
	return &APIKeyHandler{apiKeyService: apiKeyService}
}

// Generate handles POST /api/v1/api-keys.
// Creates a new API key for the authenticated user. The plain-text key is
// returned only in this response and cannot be retrieved again.
func (h *APIKeyHandler) Generate(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == 0 {
		response.Error(c, bizerr.CodeUnauthorized, "unauthorized")
		return
	}

	var req service.GenerateKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, bizerr.CodeParamInvalid, "name is required")
		return
	}

	result, err := h.apiKeyService.Generate(c.Request.Context(), userID, req.Name, req.ExpiresAt)
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

// List handles GET /api/v1/api-keys.
// Returns all API keys for the authenticated user (without the key hash).
func (h *APIKeyHandler) List(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == 0 {
		response.Error(c, bizerr.CodeUnauthorized, "unauthorized")
		return
	}

	keys, err := h.apiKeyService.ListByUser(c.Request.Context(), userID)
	if err != nil {
		if bizErr, ok := err.(*bizerr.BizError); ok {
			response.ErrorWithBizErr(c, bizErr)
			return
		}
		response.Error(c, bizerr.CodeInternal, "internal server error")
		return
	}

	response.Success(c, keys)
}

// Revoke handles DELETE /api/v1/api-keys/:id.
// Deletes an API key owned by the authenticated user.
func (h *APIKeyHandler) Revoke(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == 0 {
		response.Error(c, bizerr.CodeUnauthorized, "unauthorized")
		return
	}

	keyID, err := strconv.ParseUint(c.Param("id"), 10, strconv.IntSize)
	if err != nil {
		response.Error(c, bizerr.CodeParamInvalid, "invalid key id")
		return
	}

	if err := h.apiKeyService.Revoke(c.Request.Context(), uint(keyID), userID); err != nil {
		if bizErr, ok := err.(*bizerr.BizError); ok {
			response.ErrorWithBizErr(c, bizErr)
			return
		}
		response.Error(c, bizerr.CodeInternal, "internal server error")
		return
	}

	response.Success(c, nil)
}
