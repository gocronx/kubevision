package handler

import (
	"github.com/gin-gonic/gin"
	bizerr "github.com/kubevision/kubevision/internal/pkg/errors"
	"github.com/kubevision/kubevision/internal/pkg/response"
	"github.com/kubevision/kubevision/internal/service"
)

// OAuthHandler handles HTTP requests for OAuth authentication.
type OAuthHandler struct {
	oauthService *service.OAuthService
}

// NewOAuthHandler creates a new OAuthHandler.
func NewOAuthHandler(oauthService *service.OAuthService) *OAuthHandler {
	return &OAuthHandler{oauthService: oauthService}
}

// ListProviders handles GET /api/v1/auth/oauth/providers.
func (h *OAuthHandler) ListProviders(c *gin.Context) {
	providers := h.oauthService.ListProviders()
	response.Success(c, providers)
}

// Authorize handles GET /api/v1/auth/oauth/:provider/authorize.
func (h *OAuthHandler) Authorize(c *gin.Context) {
	provider := c.Param("provider")
	if provider == "" {
		response.Error(c, bizerr.CodeParamMissing, "provider required")
		return
	}

	url, err := h.oauthService.GetAuthorizationURL(provider)
	if err != nil {
		if bizErr, ok := err.(*bizerr.BizError); ok {
			response.ErrorWithBizErr(c, bizErr)
			return
		}
		response.Error(c, bizerr.CodeInternal, "failed to generate auth URL")
		return
	}

	response.Success(c, gin.H{"authUrl": url})
}

// Callback handles GET /api/v1/auth/oauth/:provider/callback.
func (h *OAuthHandler) Callback(c *gin.Context) {
	provider := c.Param("provider")
	code := c.Query("code")
	state := c.Query("state")

	if provider == "" || code == "" || state == "" {
		response.Error(c, bizerr.CodeParamMissing, "provider, code, and state are required")
		return
	}

	result, err := h.oauthService.HandleCallback(c.Request.Context(), provider, code, state)
	if err != nil {
		if bizErr, ok := err.(*bizerr.BizError); ok {
			response.ErrorWithBizErr(c, bizErr)
			return
		}
		response.Error(c, bizerr.CodeInternal, "OAuth callback failed")
		return
	}

	response.Success(c, result)
}
