package middleware

import (
	"bytes"
	"io"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kubevision/kubevision/internal/model"
	"github.com/kubevision/kubevision/internal/service"
)

const maxRequestBodySize = 4 * 1024 // 4 KB

// writeMethods is the set of HTTP methods that trigger audit recording.
var writeMethods = map[string]bool{
	"POST":   true,
	"PUT":    true,
	"DELETE": true,
	"PATCH":  true,
}

// AuditMiddleware returns a Gin middleware that records mutating HTTP requests
// to the audit log via the provided AuditService.
//
// Capture strategy:
//   - Only POST / PUT / DELETE / PATCH requests are recorded.
//   - The request body is snapshotted (up to 4 KB) before the handler runs.
//   - Status code and duration are captured after c.Next() returns.
//   - The log entry is handed off to AuditService.Record (non-blocking).
func AuditMiddleware(auditService *service.AuditService) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !writeMethods[strings.ToUpper(c.Request.Method)] {
			c.Next()
			return
		}

		start := time.Now()

		// Snapshot the request body (bounded to maxRequestBodySize).
		var bodyStr string
		if c.Request.Body != nil {
			raw, err := io.ReadAll(io.LimitReader(c.Request.Body, maxRequestBodySize))
			if err == nil {
				bodyStr = string(raw)
				// Restore the FULL body for downstream handlers. raw holds the
				// first (up to) 4 KB; c.Request.Body still contains everything
				// beyond that. Concatenating them gives the complete payload.
				c.Request.Body = io.NopCloser(io.MultiReader(bytes.NewReader(raw), c.Request.Body))
			}
		}

		// Execute the actual handler.
		c.Next()

		durationMs := time.Since(start).Milliseconds()
		userID := GetUserID(c)
		username := GetUsername(c)

		resource := extractResource(c.FullPath())
		action := extractAction(c.Request.Method, c.FullPath())

		entry := model.AuditLog{
			CreatedAt:   time.Now().UTC(),
			UserID:      userID,
			Username:    username,
			Action:      action,
			Resource:    resource,
			Name:        extractNameParam(c),
			Namespace:   c.Param("namespace"),
			Cluster:     c.Param("id"),
			StatusCode:  c.Writer.Status(),
			DurationMs:  durationMs,
			ClientIP:    c.ClientIP(),
			RequestBody: bodyStr,
		}

		auditService.Record(entry)
	}
}

// Audit returns a no-op Gin middleware stub retained for compatibility.
func Audit() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
	}
}

// extractNameParam attempts to derive a meaningful resource name from common
// Gin path parameters.
func extractNameParam(c *gin.Context) string {
	if n := c.Param("name"); n != "" {
		return strings.TrimPrefix(n, "/")
	}
	return ""
}
