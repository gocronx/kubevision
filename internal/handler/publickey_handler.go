package handler

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gocronx/kubevision/internal/middleware"
	bizerr "github.com/gocronx/kubevision/internal/pkg/errors"
	"github.com/gocronx/kubevision/internal/pkg/response"
	"github.com/gocronx/kubevision/internal/service"
)

const maxWebAuthnBody = 1 << 20

type PublicKeyHandler struct{ service *service.PublicKeyService }

func NewPublicKeyHandler(service *service.PublicKeyService) *PublicKeyHandler {
	return &PublicKeyHandler{service: service}
}

type registrationBeginRequest struct {
	Label    string `json:"label"`
	Password string `json:"password"`
	TOTPCode string `json:"totpCode"`
}
type loginBeginRequest struct {
	Username string `json:"username"`
}
type renameCredentialRequest struct {
	Label string `json:"label"`
}

func (h *PublicKeyHandler) Config(c *gin.Context) {
	response.Success(c, gin.H{"enabled": h.service.Enabled()})
}

func (h *PublicKeyHandler) BeginRegistration(c *gin.Context) {
	var req registrationBeginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, bizerr.CodeParamInvalid, "invalid request")
		return
	}
	result, err := h.service.BeginRegistration(c.Request.Context(), middleware.GetUserID(c), req.Label, req.Password, req.TOTPCode)
	respond(c, result, err)
}

func (h *PublicKeyHandler) FinishRegistration(c *gin.Context) {
	if !prepareWebAuthnRequest(c) {
		return
	}
	result, err := h.service.FinishRegistration(c.Request.Context(), middleware.GetUserID(c), c.GetHeader("X-WebAuthn-Ceremony"), c.Request)
	respond(c, result, err)
}

func (h *PublicKeyHandler) BeginLogin(c *gin.Context) {
	var req loginBeginRequest
	if c.Request.ContentLength != 0 {
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, bizerr.CodeParamInvalid, "invalid request")
			return
		}
	}
	result, err := h.service.BeginLogin(c.Request.Context(), req.Username)
	respond(c, result, err)
}

func (h *PublicKeyHandler) FinishLogin(c *gin.Context) {
	if !prepareWebAuthnRequest(c) {
		return
	}
	result, err := h.service.FinishLogin(c.Request.Context(), c.GetHeader("X-WebAuthn-Ceremony"), c.Request)
	respond(c, result, err)
}

func (h *PublicKeyHandler) List(c *gin.Context) {
	result, err := h.service.List(c.Request.Context(), middleware.GetUserID(c))
	respond(c, result, err)
}

func (h *PublicKeyHandler) Rename(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, strconv.IntSize)
	if err != nil {
		response.Error(c, bizerr.CodeParamInvalid, "invalid credential id")
		return
	}
	var req renameCredentialRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, bizerr.CodeParamInvalid, "invalid request")
		return
	}
	err = h.service.Rename(c.Request.Context(), middleware.GetUserID(c), uint(id), req.Label)
	respond(c, gin.H{"renamed": err == nil}, err)
}

func (h *PublicKeyHandler) Revoke(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, strconv.IntSize)
	if err != nil {
		response.Error(c, bizerr.CodeParamInvalid, "invalid credential id")
		return
	}
	err = h.service.Revoke(c.Request.Context(), middleware.GetUserID(c), uint(id))
	respond(c, gin.H{"revoked": err == nil}, err)
}

func (h *PublicKeyHandler) AdminRevoke(c *gin.Context) {
	role := middleware.GetUserRole(c)
	if role != "admin" && role != "super-admin" {
		response.Error(c, bizerr.CodeForbidden, "forbidden")
		return
	}
	userID, userErr := strconv.ParseUint(c.Param("id"), 10, strconv.IntSize)
	credentialID, credentialErr := strconv.ParseUint(c.Param("credentialId"), 10, strconv.IntSize)
	if userErr != nil || credentialErr != nil {
		response.Error(c, bizerr.CodeParamInvalid, "invalid user or credential id")
		return
	}
	err := h.service.AdminRevoke(c.Request.Context(), uint(userID), uint(credentialID))
	respond(c, gin.H{"revoked": err == nil}, err)
}

func prepareWebAuthnRequest(c *gin.Context) bool {
	if strings.TrimSpace(c.GetHeader("X-WebAuthn-Ceremony")) == "" {
		response.Error(c, bizerr.CodeParamInvalid, "ceremony id is required")
		return false
	}
	if c.Request.ContentLength > maxWebAuthnBody {
		response.Error(c, bizerr.CodeParamInvalid, "credential response is too large")
		return false
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxWebAuthnBody)
	return true
}

func respond(c *gin.Context, data any, err error) {
	if err == nil {
		response.Success(c, data)
		return
	}
	if business, ok := err.(*bizerr.BizError); ok {
		response.ErrorWithBizErr(c, business)
		return
	}
	response.Error(c, bizerr.CodeInternal, "internal server error")
}
