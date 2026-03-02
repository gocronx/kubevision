package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/kubevision/kubevision/internal/middleware"
	bizerr "github.com/kubevision/kubevision/internal/pkg/errors"
	"github.com/kubevision/kubevision/internal/pkg/response"
	"github.com/kubevision/kubevision/internal/service"
)

// AuthHandler handles HTTP requests for authentication operations.
type AuthHandler struct {
	authService *service.AuthService
}

// NewAuthHandler creates a new AuthHandler with the given AuthService.
func NewAuthHandler(authService *service.AuthService) *AuthHandler {
	return &AuthHandler{
		authService: authService,
	}
}

// loginRequest is the expected JSON body for the login endpoint.
type loginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// refreshRequest is the expected JSON body for the refresh endpoint.
type refreshRequest struct {
	RefreshToken string `json:"refreshToken" binding:"required"`
}

// Login handles POST /api/v1/auth/login.
// It authenticates a user by username and password, returning access and refresh tokens.
func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, bizerr.CodeParamInvalid, "username and password are required")
		return
	}

	resp, err := h.authService.Login(c.Request.Context(), req.Username, req.Password)
	if err != nil {
		if bizErr, ok := err.(*bizerr.BizError); ok {
			response.ErrorWithBizErr(c, bizErr)
			return
		}
		response.Error(c, bizerr.CodeInternal, "internal server error")
		return
	}

	response.Success(c, resp)
}

// Refresh handles POST /api/v1/auth/refresh.
// It validates a refresh token and returns a new token pair.
func (h *AuthHandler) Refresh(c *gin.Context) {
	var req refreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, bizerr.CodeParamInvalid, "refreshToken is required")
		return
	}

	resp, err := h.authService.RefreshToken(c.Request.Context(), req.RefreshToken)
	if err != nil {
		if bizErr, ok := err.(*bizerr.BizError); ok {
			response.ErrorWithBizErr(c, bizErr)
			return
		}
		response.Error(c, bizerr.CodeInternal, "internal server error")
		return
	}

	response.Success(c, resp)
}

// Me handles GET /api/v1/users/me.
// It returns information about the currently authenticated user.
func (h *AuthHandler) Me(c *gin.Context) {
	userID := middleware.GetUserID(c)
	username := middleware.GetUsername(c)
	role := middleware.GetUserRole(c)

	if userID == 0 {
		response.Error(c, bizerr.CodeUnauthorized, "unauthorized")
		return
	}

	response.Success(c, service.UserInfo{
		ID:       userID,
		Username: username,
		Role:     role,
	})
}
