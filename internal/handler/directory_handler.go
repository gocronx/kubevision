package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/gocronx/kubevision/internal/middleware"
	"github.com/gocronx/kubevision/internal/model"
	bizerr "github.com/gocronx/kubevision/internal/pkg/errors"
	"github.com/gocronx/kubevision/internal/pkg/response"
	"github.com/gocronx/kubevision/internal/service"
)

type DirectoryHandler struct{ service *service.DirectoryService }

func NewDirectoryHandler(service *service.DirectoryService) *DirectoryHandler {
	return &DirectoryHandler{service: service}
}

type directorySettingsRequest struct {
	Enabled            bool                         `json:"enabled"`
	ServerURL          string                       `json:"serverUrl"`
	StartTLS           bool                         `json:"startTls"`
	AllowPlaintext     bool                         `json:"allowPlaintext"`
	CABundle           string                       `json:"caBundle"`
	BindDN             string                       `json:"bindDn"`
	BindPassword       string                       `json:"bindPassword"`
	ConnectTimeoutSecs int                          `json:"connectTimeoutSecs"`
	SearchTimeoutSecs  int                          `json:"searchTimeoutSecs"`
	UserBaseDN         string                       `json:"userBaseDn"`
	UserFilter         string                       `json:"userFilter"`
	StableIDAttribute  string                       `json:"stableIdAttribute"`
	UsernameAttribute  string                       `json:"usernameAttribute"`
	DisplayAttribute   string                       `json:"displayAttribute"`
	EmailAttribute     string                       `json:"emailAttribute"`
	GroupAttribute     string                       `json:"groupAttribute"`
	FallbackRole       string                       `json:"fallbackRole"`
	RefreshMapping     bool                         `json:"refreshMapping"`
	Mappings           []model.DirectoryRoleMapping `json:"mappings"`
}

func (r directorySettingsRequest) settings() service.DirectorySettings {
	return service.DirectorySettings{Enabled: r.Enabled, ServerURL: r.ServerURL, StartTLS: r.StartTLS, AllowPlaintext: r.AllowPlaintext, CABundle: r.CABundle, BindDN: r.BindDN, BindPassword: r.BindPassword, ConnectTimeoutSecs: r.ConnectTimeoutSecs, SearchTimeoutSecs: r.SearchTimeoutSecs, UserBaseDN: r.UserBaseDN, UserFilter: r.UserFilter, StableIDAttribute: r.StableIDAttribute, UsernameAttribute: r.UsernameAttribute, DisplayAttribute: r.DisplayAttribute, EmailAttribute: r.EmailAttribute, GroupAttribute: r.GroupAttribute, FallbackRole: r.FallbackRole, RefreshMapping: r.RefreshMapping, Mappings: r.Mappings}
}

func requireDirectoryAdmin(c *gin.Context) bool {
	role := middleware.GetUserRole(c)
	if role != "admin" && role != "super-admin" {
		response.Error(c, bizerr.CodeForbidden, "administrator access required")
		return false
	}
	return true
}

func (h *DirectoryHandler) Get(c *gin.Context) {
	if !requireDirectoryAdmin(c) {
		return
	}
	settings, err := h.service.GetSettings(c.Request.Context())
	if err != nil {
		response.Error(c, bizerr.CodeInternal, "failed to read directory settings")
		return
	}
	response.Success(c, settings)
}

func (h *DirectoryHandler) Update(c *gin.Context) {
	if !requireDirectoryAdmin(c) {
		return
	}
	var req directorySettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, bizerr.CodeParamInvalid, "invalid directory settings")
		return
	}
	if err := h.service.SaveSettings(c.Request.Context(), req.settings()); err != nil {
		writeDirectoryError(c, err)
		return
	}
	response.Success(c, gin.H{"saved": true})
}

func (h *DirectoryHandler) Test(c *gin.Context) {
	if !requireDirectoryAdmin(c) {
		return
	}
	var req directorySettingsRequest
	var settings *service.DirectorySettings
	if c.Request.ContentLength > 0 {
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, bizerr.CodeParamInvalid, "invalid directory settings")
			return
		}
		s := req.settings()
		settings = &s
	}
	category := h.service.TestConnection(c.Request.Context(), settings)
	response.Success(c, gin.H{"ok": category == "ok", "category": category})
}

func (h *DirectoryHandler) Preview(c *gin.Context) {
	if !requireDirectoryAdmin(c) {
		return
	}
	var req struct {
		Identifier string `json:"identifier" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, bizerr.CodeParamInvalid, "identifier is required")
		return
	}
	preview, err := h.service.Preview(c.Request.Context(), req.Identifier)
	if err != nil {
		writeDirectoryError(c, err)
		return
	}
	response.Success(c, preview)
}

func writeDirectoryError(c *gin.Context, err error) {
	if e, ok := err.(*bizerr.BizError); ok {
		response.ErrorWithBizErr(c, e)
		return
	}
	response.Error(c, bizerr.CodeInternal, "directory operation failed")
}
