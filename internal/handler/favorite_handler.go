package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/gocronx/kubevision/internal/middleware"
	bizerr "github.com/gocronx/kubevision/internal/pkg/errors"
	"github.com/gocronx/kubevision/internal/pkg/response"
	"github.com/gocronx/kubevision/internal/service"
)

// FavoriteHandler handles HTTP requests for the favorites/bookmarks system.
type FavoriteHandler struct {
	favoriteService *service.FavoriteService
}

// NewFavoriteHandler creates a new FavoriteHandler.
func NewFavoriteHandler(favoriteService *service.FavoriteService) *FavoriteHandler {
	return &FavoriteHandler{favoriteService: favoriteService}
}

// List handles GET /api/v1/favorites.
// Returns all favorites for the authenticated user.
func (h *FavoriteHandler) List(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == 0 {
		response.Error(c, bizerr.CodeUnauthorized, "unauthorized")
		return
	}

	favs, err := h.favoriteService.ListFavorites(c.Request.Context(), userID)
	if err != nil {
		if bizErr, ok := err.(*bizerr.BizError); ok {
			response.ErrorWithBizErr(c, bizErr)
			return
		}
		response.Error(c, bizerr.CodeInternal, "internal server error")
		return
	}
	response.Success(c, favs)
}

// Create handles POST /api/v1/favorites.
// Adds a resource to the current user's favorites.
func (h *FavoriteHandler) Create(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == 0 {
		response.Error(c, bizerr.CodeUnauthorized, "unauthorized")
		return
	}

	var req service.AddFavoriteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, bizerr.CodeParamInvalid, "invalid request body")
		return
	}

	fav, err := h.favoriteService.AddFavorite(c.Request.Context(), userID, &req)
	if err != nil {
		if bizErr, ok := err.(*bizerr.BizError); ok {
			response.ErrorWithBizErr(c, bizErr)
			return
		}
		response.Error(c, bizerr.CodeInternal, "internal server error")
		return
	}
	response.Success(c, fav)
}

// Delete handles DELETE /api/v1/favorites/:id.
// Removes a specific favorite belonging to the current user.
func (h *FavoriteHandler) Delete(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == 0 {
		response.Error(c, bizerr.CodeUnauthorized, "unauthorized")
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, strconv.IntSize)
	if err != nil {
		response.Error(c, bizerr.CodeParamInvalid, "invalid favorite id")
		return
	}

	if err := h.favoriteService.RemoveFavorite(c.Request.Context(), userID, uint(id)); err != nil {
		if bizErr, ok := err.(*bizerr.BizError); ok {
			response.ErrorWithBizErr(c, bizErr)
			return
		}
		response.Error(c, bizerr.CodeInternal, "internal server error")
		return
	}
	response.Success(c, nil)
}

// Toggle handles POST /api/v1/favorites/toggle.
// Adds or removes a favorite depending on whether it already exists.
func (h *FavoriteHandler) Toggle(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == 0 {
		response.Error(c, bizerr.CodeUnauthorized, "unauthorized")
		return
	}

	var req service.AddFavoriteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, bizerr.CodeParamInvalid, "invalid request body")
		return
	}

	result, err := h.favoriteService.ToggleFavorite(c.Request.Context(), userID, &req)
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

// Reorder handles PUT /api/v1/favorites/reorder.
// Updates the sort order of the current user's favorites.
func (h *FavoriteHandler) Reorder(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == 0 {
		response.Error(c, bizerr.CodeUnauthorized, "unauthorized")
		return
	}

	var req service.ReorderFavoritesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, bizerr.CodeParamInvalid, "invalid request body")
		return
	}

	if err := h.favoriteService.ReorderFavorites(c.Request.Context(), userID, req.OrderedIDs); err != nil {
		if bizErr, ok := err.(*bizerr.BizError); ok {
			response.ErrorWithBizErr(c, bizErr)
			return
		}
		response.Error(c, bizerr.CodeInternal, "internal server error")
		return
	}
	response.Success(c, nil)
}

// Check handles GET /api/v1/favorites/check.
// Query params: cluster_id, resource_type, name, namespace.
func (h *FavoriteHandler) Check(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == 0 {
		response.Error(c, bizerr.CodeUnauthorized, "unauthorized")
		return
	}

	clusterID := c.Query("cluster_id")
	resourceType := c.Query("resource_type")
	name := c.Query("name")
	namespace := c.Query("namespace")

	if clusterID == "" || resourceType == "" || name == "" {
		response.Error(c, bizerr.CodeParamMissing, "cluster_id, resource_type, and name are required")
		return
	}

	result, err := h.favoriteService.CheckFavorite(c.Request.Context(), userID, clusterID, resourceType, name, namespace)
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
