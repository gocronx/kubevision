package handler

import (
	"context"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gocronx/kubevision/internal/middleware"
	"github.com/gocronx/kubevision/internal/packages"
	bizerr "github.com/gocronx/kubevision/internal/pkg/errors"
	"github.com/gocronx/kubevision/internal/pkg/response"
)

type PackageHandler struct {
	service *packages.Service
	resolve func(context.Context, string) (string, error)
}

func NewPackageHandler(service *packages.Service, resolve func(context.Context, string) (string, error)) *PackageHandler {
	return &PackageHandler{service: service, resolve: resolve}
}

func (h *PackageHandler) List(c *gin.Context) {
	cluster, ok := h.cluster(c)
	if !ok {
		return
	}
	limit, err := queryInt(c, "limit", 0)
	if err != nil {
		response.Error(c, bizerr.CodeParamInvalid, "invalid limit")
		return
	}
	items, err := h.service.List(c.Request.Context(), packageActor(c), cluster, packages.ListOptions{Namespace: c.Query("namespace"), State: c.Query("state"), Label: c.Query("label"), Limit: limit})
	writePackageResult(c, items, err)
}

func (h *PackageHandler) Get(c *gin.Context) {
	cluster, ok := h.cluster(c)
	if !ok {
		return
	}
	revision, err := queryInt(c, "revision", 0)
	if err != nil || revision < 0 {
		response.Error(c, bizerr.CodeParamInvalid, "invalid revision")
		return
	}
	item, err := h.service.Get(c.Request.Context(), packageActor(c), cluster, c.Param("namespace"), c.Param("name"), revision)
	writePackageResult(c, item, err)
}

func (h *PackageHandler) History(c *gin.Context) {
	cluster, ok := h.cluster(c)
	if !ok {
		return
	}
	items, err := h.service.History(c.Request.Context(), packageActor(c), cluster, c.Param("namespace"), c.Param("name"))
	writePackageResult(c, items, err)
}

type packageRollbackRequest struct {
	Revision       int  `json:"revision" binding:"required,min=1"`
	Wait           bool `json:"wait"`
	Atomic         bool `json:"atomic"`
	TimeoutSeconds int  `json:"timeoutSeconds"`
}

func (h *PackageHandler) Rollback(c *gin.Context) {
	cluster, ok := h.cluster(c)
	if !ok {
		return
	}
	var req packageRollbackRequest
	if c.ShouldBindJSON(&req) != nil {
		response.Error(c, bizerr.CodeParamInvalid, "revision is required")
		return
	}
	err := h.service.Rollback(c.Request.Context(), packageActor(c), cluster, c.Param("namespace"), c.Param("name"), packages.RollbackOptions{Revision: req.Revision, Wait: req.Wait, Atomic: req.Atomic, Timeout: time.Duration(req.TimeoutSeconds) * time.Second})
	writePackageResult(c, nil, err)
}

type packageRemoveRequest struct {
	Confirmation   string `json:"confirmation" binding:"required"`
	KeepHistory    bool   `json:"keepHistory"`
	Wait           bool   `json:"wait"`
	TimeoutSeconds int    `json:"timeoutSeconds"`
}

func (h *PackageHandler) Remove(c *gin.Context) {
	cluster, ok := h.cluster(c)
	if !ok {
		return
	}
	var req packageRemoveRequest
	if c.ShouldBindJSON(&req) != nil {
		response.Error(c, bizerr.CodeParamInvalid, "confirmation is required")
		return
	}
	err := h.service.Remove(c.Request.Context(), packageActor(c), cluster, c.Param("namespace"), c.Param("name"), packages.RemoveOptions{Confirmation: req.Confirmation, KeepHistory: req.KeepHistory, Wait: req.Wait, Timeout: time.Duration(req.TimeoutSeconds) * time.Second})
	writePackageResult(c, nil, err)
}

func (h *PackageHandler) cluster(c *gin.Context) (string, bool) {
	cluster, err := h.resolve(c.Request.Context(), c.Param("id"))
	if err != nil {
		response.Error(c, bizerr.CodeNotFound, "cluster not found")
		return "", false
	}
	return cluster, true
}

func packageActor(c *gin.Context) packages.Actor {
	return packages.Actor{UserID: middleware.GetUserID(c), Username: middleware.GetUsername(c), Role: middleware.GetUserRole(c), ClientIP: c.ClientIP()}
}
func queryInt(c *gin.Context, name string, fallback int) (int, error) {
	raw := c.Query(name)
	if raw == "" {
		return fallback, nil
	}
	return strconv.Atoi(raw)
}
func writePackageResult(c *gin.Context, data interface{}, err error) {
	if err == nil {
		response.Success(c, data)
		return
	}
	if b, ok := err.(*bizerr.BizError); ok {
		response.ErrorWithBizErr(c, b)
		return
	}
	response.Error(c, bizerr.CodeInternal, "internal server error")
}
