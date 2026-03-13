package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gocronx/kubevision/internal/model"
	"github.com/gocronx/kubevision/internal/service"
)

// sensitiveFields is the set of JSON field names whose values must be
// redacted before storing a request body in the audit log.
var sensitiveFields = map[string]bool{
	"password":     true,
	"newPassword":  true,
	"oldPassword":  true,
	"secret":       true,
	"kubeconfig":   true,
	"token":        true,
	"refreshToken": true,
}

// redactSensitiveFields parses bodyStr as JSON and replaces the values of any
// sensitiveFields keys with "[REDACTED]". If the body is not valid JSON it is
// returned unchanged so we do not lose audit data for non-JSON endpoints.
func redactSensitiveFields(bodyStr string) string {
	if bodyStr == "" {
		return bodyStr
	}
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(bodyStr), &parsed); err != nil {
		// Not a JSON object — return as-is (binary or text bodies).
		return bodyStr
	}
	redactMapFields(parsed)
	out, err := json.Marshal(parsed)
	if err != nil {
		return bodyStr
	}
	return string(out)
}

// redactMapFields recursively walks a decoded JSON map and replaces the values
// of any sensitive fields with "[REDACTED]".
func redactMapFields(m map[string]interface{}) {
	for k, v := range m {
		if sensitiveFields[k] {
			m[k] = "[REDACTED]"
			continue
		}
		// Recurse into nested objects.
		if nested, ok := v.(map[string]interface{}); ok {
			redactMapFields(nested)
		}
	}
}

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
				// Restore the FULL body for downstream handlers. raw holds the
				// first (up to) 4 KB; c.Request.Body still contains everything
				// beyond that. Concatenating them gives the complete payload.
				c.Request.Body = io.NopCloser(io.MultiReader(bytes.NewReader(raw), c.Request.Body))
				// Redact sensitive fields before storing in the audit log.
				bodyStr = redactSensitiveFields(string(raw))
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
