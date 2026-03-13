package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/gocronx/kubevision/internal/middleware"
	bizerr "github.com/gocronx/kubevision/internal/pkg/errors"
	"github.com/gocronx/kubevision/internal/pkg/response"
	"github.com/gocronx/kubevision/internal/service"
)

// UserHandler handles HTTP requests for user management.
type UserHandler struct {
	userService *service.UserService
}

// NewUserHandler creates a new UserHandler.
func NewUserHandler(userService *service.UserService) *UserHandler {
	return &UserHandler{userService: userService}
}

// List handles GET /api/v1/users.
// Returns all users (admin+).
func (h *UserHandler) List(c *gin.Context) {
	users, err := h.userService.ListUsers(c.Request.Context())
	if err != nil {
		if bizErr, ok := err.(*bizerr.BizError); ok {
			response.ErrorWithBizErr(c, bizErr)
			return
		}
		response.Error(c, bizerr.CodeInternal, "internal server error")
		return
	}
	response.Success(c, users)
}

// Get handles GET /api/v1/users/:id.
// Returns the detail of a single user (admin+).
func (h *UserHandler) Get(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, bizerr.CodeParamInvalid, "invalid user id")
		return
	}

	user, err := h.userService.GetUser(c.Request.Context(), uint(id))
	if err != nil {
		if bizErr, ok := err.(*bizerr.BizError); ok {
			response.ErrorWithBizErr(c, bizErr)
			return
		}
		response.Error(c, bizerr.CodeInternal, "internal server error")
		return
	}
	response.Success(c, user)
}

// createUserRequest is the request body for creating a user.
type createUserRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	Role     string `json:"role"     binding:"required"`
}

// Create handles POST /api/v1/users.
// Creates a new user (admin+).
func (h *UserHandler) Create(c *gin.Context) {
	var req createUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, bizerr.CodeParamInvalid, "username, password, and role are required")
		return
	}

	user, err := h.userService.CreateUser(c.Request.Context(), req.Username, req.Password, req.Role)
	if err != nil {
		if bizErr, ok := err.(*bizerr.BizError); ok {
			response.ErrorWithBizErr(c, bizErr)
			return
		}
		response.Error(c, bizerr.CodeInternal, "internal server error")
		return
	}
	response.Success(c, user)
}

// updateUserRequest is the request body for updating a user.
type updateUserRequest struct {
	Role     string `json:"role"`
	IsActive *bool  `json:"isActive"`
}

// Update handles PUT /api/v1/users/:id.
// Updates a user's role and active status (admin+).
func (h *UserHandler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, bizerr.CodeParamInvalid, "invalid user id")
		return
	}

	var req updateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, bizerr.CodeParamInvalid, "invalid request body")
		return
	}

	callerID := middleware.GetUserID(c)
	if callerID == 0 {
		response.Error(c, bizerr.CodeUnauthorized, "unauthorized")
		return
	}

	// Default isActive to true if not provided.
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	user, err := h.userService.UpdateUser(c.Request.Context(), uint(id), callerID, req.Role, isActive)
	if err != nil {
		if bizErr, ok := err.(*bizerr.BizError); ok {
			response.ErrorWithBizErr(c, bizErr)
			return
		}
		response.Error(c, bizerr.CodeInternal, "internal server error")
		return
	}
	response.Success(c, user)
}

// Delete handles DELETE /api/v1/users/:id.
// Deletes a user (admin+).
func (h *UserHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, bizerr.CodeParamInvalid, "invalid user id")
		return
	}

	callerID := middleware.GetUserID(c)
	if callerID == 0 {
		response.Error(c, bizerr.CodeUnauthorized, "unauthorized")
		return
	}

	if err := h.userService.DeleteUser(c.Request.Context(), uint(id), callerID); err != nil {
		if bizErr, ok := err.(*bizerr.BizError); ok {
			response.ErrorWithBizErr(c, bizErr)
			return
		}
		response.Error(c, bizerr.CodeInternal, "internal server error")
		return
	}
	response.Success(c, nil)
}

// resetPasswordRequest is the request body for an admin password reset.
type resetPasswordRequest struct {
	NewPassword string `json:"newPassword" binding:"required"`
}

// ResetPassword handles PUT /api/v1/users/:id/reset-password.
// Allows an admin to reset another user's password (admin+).
func (h *UserHandler) ResetPassword(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, bizerr.CodeParamInvalid, "invalid user id")
		return
	}

	var req resetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, bizerr.CodeParamInvalid, "newPassword is required")
		return
	}

	if err := h.userService.ResetPassword(c.Request.Context(), uint(id), req.NewPassword); err != nil {
		if bizErr, ok := err.(*bizerr.BizError); ok {
			response.ErrorWithBizErr(c, bizErr)
			return
		}
		response.Error(c, bizerr.CodeInternal, "internal server error")
		return
	}
	response.Success(c, nil)
}

// changePasswordRequest is the request body for changing one's own password.
type changePasswordRequest struct {
	OldPassword string `json:"oldPassword" binding:"required"`
	NewPassword string `json:"newPassword" binding:"required"`
}

// ChangePassword handles PUT /api/v1/users/me/password.
// Allows any authenticated user to change their own password.
// Because TokenVersion is bumped, the frontend should redirect to login after success.
func (h *UserHandler) ChangePassword(c *gin.Context) {
	callerID := middleware.GetUserID(c)
	if callerID == 0 {
		response.Error(c, bizerr.CodeUnauthorized, "unauthorized")
		return
	}

	var req changePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, bizerr.CodeParamInvalid, "oldPassword and newPassword are required")
		return
	}

	if err := h.userService.ChangePassword(c.Request.Context(), callerID, req.OldPassword, req.NewPassword); err != nil {
		if bizErr, ok := err.(*bizerr.BizError); ok {
			response.ErrorWithBizErr(c, bizErr)
			return
		}
		response.Error(c, bizerr.CodeInternal, "internal server error")
		return
	}
	response.Success(c, nil)
}
