package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/gocronx/kubevision/internal/middleware"
	"github.com/gocronx/kubevision/internal/operation"
	bizerr "github.com/gocronx/kubevision/internal/pkg/errors"
	"github.com/gocronx/kubevision/internal/pkg/response"
)

type OperationHandler struct{ manager *operation.Manager }

func NewOperationHandler(manager *operation.Manager) *OperationHandler {
	return &OperationHandler{manager: manager}
}

func (h *OperationHandler) List(c *gin.Context) {
	limit, err := strconv.Atoi(c.DefaultQuery("limit", "100"))
	if err != nil {
		response.Error(c, bizerr.CodeParamInvalid, "invalid limit")
		return
	}
	items, err := h.manager.List(c.Request.Context(), middleware.GetUserID(c), canViewAllOperations(c), limit)
	writeOperationResult(c, items, err)
}

func (h *OperationHandler) Get(c *gin.Context) {
	item, err := h.manager.Get(c.Request.Context(), c.Param("id"), middleware.GetUserID(c), canViewAllOperations(c))
	writeOperationResult(c, item, err)
}

func (h *OperationHandler) Retry(c *gin.Context) {
	principal := operation.Principal{UserID: middleware.GetUserID(c), Username: middleware.GetUsername(c), Role: middleware.GetUserRole(c)}
	item, err := h.manager.Retry(c.Request.Context(), c.Param("id"), principal, canViewAllOperations(c))
	writeOperationResult(c, item, err)
}

func canViewAllOperations(c *gin.Context) bool {
	role := middleware.GetUserRole(c)
	return c.Query("all") == "true" && (role == "admin" || role == "super-admin")
}

func writeOperationResult(c *gin.Context, data interface{}, err error) {
	if err == nil {
		response.Success(c, data)
		return
	}
	if business, ok := err.(*bizerr.BizError); ok {
		response.ErrorWithBizErr(c, business)
		return
	}
	response.Error(c, bizerr.CodeInternal, "failed to load operation")
}
