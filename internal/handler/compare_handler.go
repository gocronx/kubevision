package handler

import (
	"github.com/gin-gonic/gin"
	bizerr "github.com/gocronx/kubevision/internal/pkg/errors"
	"github.com/gocronx/kubevision/internal/pkg/response"
	"github.com/gocronx/kubevision/internal/service"
)

// CompareHandler handles the cross-cluster resource comparison endpoint.
type CompareHandler struct {
	compareService *service.CompareService
}

// NewCompareHandler creates a new CompareHandler.
func NewCompareHandler(compareService *service.CompareService) *CompareHandler {
	return &CompareHandler{compareService: compareService}
}

// Compare handles POST /api/v1/compare.
func (h *CompareHandler) Compare(c *gin.Context) {
	var req service.CompareRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, bizerr.CodeParamInvalid, err.Error())
		return
	}

	result, err := h.compareService.Compare(c.Request.Context(), &req)
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
