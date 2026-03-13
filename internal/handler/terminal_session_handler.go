package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/gocronx/kubevision/internal/middleware"
	bizerr "github.com/gocronx/kubevision/internal/pkg/errors"
	"github.com/gocronx/kubevision/internal/pkg/response"
	"github.com/gocronx/kubevision/internal/service"
)

// TerminalSessionHandler handles HTTP requests for terminal session recordings.
type TerminalSessionHandler struct {
	sessionService *service.TerminalSessionService
}

// NewTerminalSessionHandler creates a new TerminalSessionHandler.
func NewTerminalSessionHandler(sessionService *service.TerminalSessionService) *TerminalSessionHandler {
	return &TerminalSessionHandler{sessionService: sessionService}
}

// List handles GET /api/v1/terminal-sessions.
// Admin users see all sessions; regular users see only their own.
func (h *TerminalSessionHandler) List(c *gin.Context) {
	// Parse optional pagination params.
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	if limit <= 0 || limit > 200 {
		limit = 50
	}

	role := middleware.GetUserRole(c)
	isAdmin := role == "admin"

	var sessions []service.TerminalSessionMeta
	var total int64
	var err error

	if isAdmin {
		sessions, total, err = h.sessionService.ListAll(c.Request.Context(), offset, limit)
	} else {
		userID := middleware.GetUserID(c)
		sessions, total, err = h.sessionService.ListByUser(c.Request.Context(), userID, offset, limit)
	}

	if err != nil {
		if bizErr, ok := err.(*bizerr.BizError); ok {
			response.ErrorWithBizErr(c, bizErr)
			return
		}
		response.Error(c, bizerr.CodeInternal, "internal server error")
		return
	}

	response.SuccessWithMeta(c, sessions, &response.Meta{Total: total})
}

// Get handles GET /api/v1/terminal-sessions/:id — returns metadata only.
func (h *TerminalSessionHandler) Get(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, bizerr.CodeParamInvalid, "invalid session id")
		return
	}

	sess, err := h.sessionService.GetByID(c.Request.Context(), id)
	if err != nil {
		if bizErr, ok := err.(*bizerr.BizError); ok {
			response.ErrorWithBizErr(c, bizErr)
			return
		}
		response.Error(c, bizerr.CodeInternal, "internal server error")
		return
	}

	// Return metadata only (strip recording field).
	response.Success(c, service.TerminalSessionMeta{
		ID:         sess.ID,
		CreatedAt:  sess.CreatedAt,
		UserID:     sess.UserID,
		Cluster:    sess.Cluster,
		Namespace:  sess.Namespace,
		Pod:        sess.Pod,
		Container:  sess.Container,
		DurationMs: sess.DurationMs,
		ExpiresAt:  sess.ExpiresAt,
	})
}

// Play handles GET /api/v1/terminal-sessions/:id/play — returns recording data.
func (h *TerminalSessionHandler) Play(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, bizerr.CodeParamInvalid, "invalid session id")
		return
	}

	sess, err := h.sessionService.GetByID(c.Request.Context(), id)
	if err != nil {
		if bizErr, ok := err.(*bizerr.BizError); ok {
			response.ErrorWithBizErr(c, bizErr)
			return
		}
		response.Error(c, bizerr.CodeInternal, "internal server error")
		return
	}

	response.Success(c, gin.H{
		"id":         sess.ID,
		"recording":  sess.Recording,
		"durationMs": sess.DurationMs,
	})
}
