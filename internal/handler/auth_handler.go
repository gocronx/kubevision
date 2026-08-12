package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/gocronx/kubevision/internal/middleware"
	bizerr "github.com/gocronx/kubevision/internal/pkg/errors"
	"github.com/gocronx/kubevision/internal/pkg/response"
	"github.com/gocronx/kubevision/internal/service"
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
	Provider string `json:"provider"`
}

// refreshRequest is the expected JSON body for the refresh endpoint.
type refreshRequest struct {
	RefreshToken string `json:"refreshToken" binding:"required"`
}

// twoFACodeRequest carries a TOTP code for enable/disable/verify operations.
type twoFACodeRequest struct {
	Code string `json:"code" binding:"required"`
}

// twoFAVerifyRequest carries the temp token and TOTP code for the verify step.
type twoFAVerifyRequest struct {
	TempToken string `json:"tempToken" binding:"required"`
	Code      string `json:"code" binding:"required"`
}

// twoFARecoveryRequest carries the temp token and recovery code.
type twoFARecoveryRequest struct {
	TempToken    string `json:"tempToken" binding:"required"`
	RecoveryCode string `json:"recoveryCode" binding:"required"`
}

// Login handles POST /api/v1/auth/login.
// When the user has 2FA enabled, responds with code 40102 and a short-lived tempToken.
func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, bizerr.CodeParamInvalid, "username and password are required")
		return
	}

	var result *service.LoginResult
	var err error
	if req.Provider == "directory" {
		result, err = h.authService.LoginDirectory(c.Request.Context(), req.Username, req.Password)
	} else {
		result, err = h.authService.Login(c.Request.Context(), req.Username, req.Password)
	}
	if err != nil {
		if bizErr, ok := err.(*bizerr.BizError); ok {
			response.ErrorWithBizErr(c, bizErr)
			return
		}
		response.Error(c, bizerr.CodeInternal, "internal server error")
		return
	}

	// If 2FA is required, return the temp token with the special business code.
	if result.TwoFARequired != nil {
		c.JSON(200, gin.H{
			"code":    bizerr.Code2FARequired,
			"message": "two-factor authentication required",
			"data":    result.TwoFARequired,
		})
		return
	}

	response.Success(c, result.FullTokens)
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

// Setup2FA handles POST /api/v1/auth/2fa/setup.
// Returns the TOTP secret, QR code URL, and one-time recovery codes.
// Requires the user to be authenticated. The secret is not yet active until Enable2FA is called.
func (h *AuthHandler) Setup2FA(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == 0 {
		response.Error(c, bizerr.CodeUnauthorized, "unauthorized")
		return
	}

	resp, err := h.authService.Setup2FA(c.Request.Context(), userID)
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

// Enable2FA handles POST /api/v1/auth/2fa/enable.
// Activates TOTP after the user confirms with a valid code.
// Requires the user to be authenticated and to have called Setup2FA first.
func (h *AuthHandler) Enable2FA(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == 0 {
		response.Error(c, bizerr.CodeUnauthorized, "unauthorized")
		return
	}

	var req twoFACodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, bizerr.CodeParamInvalid, "code is required")
		return
	}

	if err := h.authService.Enable2FA(c.Request.Context(), userID, req.Code); err != nil {
		if bizErr, ok := err.(*bizerr.BizError); ok {
			response.ErrorWithBizErr(c, bizErr)
			return
		}
		response.Error(c, bizerr.CodeInternal, "internal server error")
		return
	}

	response.Success(c, gin.H{"enabled": true})
}

// Disable2FA handles POST /api/v1/auth/2fa/disable.
// Deactivates TOTP and clears all 2FA data after verifying a valid code.
// Requires the user to be authenticated.
func (h *AuthHandler) Disable2FA(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == 0 {
		response.Error(c, bizerr.CodeUnauthorized, "unauthorized")
		return
	}

	var req twoFACodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, bizerr.CodeParamInvalid, "code is required")
		return
	}

	if err := h.authService.Disable2FA(c.Request.Context(), userID, req.Code); err != nil {
		if bizErr, ok := err.(*bizerr.BizError); ok {
			response.ErrorWithBizErr(c, bizErr)
			return
		}
		response.Error(c, bizerr.CodeInternal, "internal server error")
		return
	}

	response.Success(c, gin.H{"disabled": true})
}

// Verify2FA handles POST /api/v1/auth/2fa/verify.
// Exchanges a valid tempToken + TOTP code for full JWT tokens.
// This is a public endpoint (no auth middleware required — tempToken serves as auth).
func (h *AuthHandler) Verify2FA(c *gin.Context) {
	var req twoFAVerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, bizerr.CodeParamInvalid, "tempToken and code are required")
		return
	}

	resp, err := h.authService.Verify2FA(c.Request.Context(), req.TempToken, req.Code)
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

// Recovery2FA handles POST /api/v1/auth/2fa/recovery.
// Exchanges a valid tempToken + one-time recovery code for full JWT tokens.
// This is a public endpoint (no auth middleware required — tempToken serves as auth).
func (h *AuthHandler) Recovery2FA(c *gin.Context) {
	var req twoFARecoveryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, bizerr.CodeParamInvalid, "tempToken and recoveryCode are required")
		return
	}

	resp, err := h.authService.UseRecoveryCode(c.Request.Context(), req.TempToken, req.RecoveryCode)
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
