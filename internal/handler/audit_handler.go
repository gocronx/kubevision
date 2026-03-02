package handler

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kubevision/kubevision/internal/middleware"
	bizerr "github.com/kubevision/kubevision/internal/pkg/errors"
	"github.com/kubevision/kubevision/internal/pkg/response"
	"github.com/kubevision/kubevision/internal/repository"
)

// AuditHandler handles HTTP requests for audit log operations.
type AuditHandler struct {
	auditRepo repository.AuditRepo
}

// NewAuditHandler creates a new AuditHandler.
func NewAuditHandler(auditRepo repository.AuditRepo) *AuditHandler {
	return &AuditHandler{auditRepo: auditRepo}
}

// List handles GET /api/v1/audit-logs.
// Only admin and ops roles can view audit logs.
// Query params: action, cluster, since (RFC3339), offset, limit.
func (h *AuditHandler) List(c *gin.Context) {
	role := middleware.GetUserRole(c)
	if role != "admin" && role != "ops" {
		response.Error(c, bizerr.CodeForbidden, "only admin and ops roles may view audit logs")
		return
	}

	filter := repository.AuditFilter{
		Action:  c.Query("action"),
		Cluster: c.Query("cluster"),
	}

	// Parse since timestamp (RFC3339).
	if s := c.Query("since"); s != "" {
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			filter.Since = t
		}
	}

	// Pagination — default limit 50, max 500.
	filter.Limit = 50
	filter.Offset = 0
	if l := c.Query("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= 500 {
			filter.Limit = n
		}
	}
	if o := c.Query("offset"); o != "" {
		if n, err := strconv.Atoi(o); err == nil && n >= 0 {
			filter.Offset = n
		}
	}

	logs, total, err := h.auditRepo.List(c.Request.Context(), filter)
	if err != nil {
		response.Error(c, bizerr.CodeInternal, "failed to list audit logs")
		return
	}

	response.Paginated(c, logs, total, c.GetHeader("X-Request-ID"))
}
