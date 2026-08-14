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
	"newpassword":  true,
	"oldpassword":  true,
	"bindpassword": true,
	"secret":       true,
	"totpcode":     true,
	"recoverycode": true,
	"kubeconfig":   true,
	"token":        true,
	"refreshtoken": true,
	"credential":   true,
	"privatekey":   true,
	"apikey":       true,
}

// redactSensitiveFields parses bodyStr as JSON and replaces the values of any
// sensitiveFields keys with "[REDACTED]". Invalid or truncated JSON is not
// retained because its sensitive fields cannot be redacted reliably.
func redactSensitiveFields(bodyStr string) string {
	if bodyStr == "" {
		return bodyStr
	}
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(bodyStr), &parsed); err != nil {
		return "[UNAVAILABLE]"
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
		if sensitiveFields[strings.ToLower(k)] {
			m[k] = "[REDACTED]"
			continue
		}
		switch nested := v.(type) {
		case map[string]interface{}:
			redactMapFields(nested)
		case []interface{}:
			for _, item := range nested {
				if object, ok := item.(map[string]interface{}); ok {
					redactMapFields(object)
				}
			}
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

// omitAuditBody prevents high-risk request payloads from ever reaching audit
// storage. Metadata about the operation is still recorded.
func omitAuditBody(c *gin.Context) bool {
	path := c.FullPath()
	return strings.HasPrefix(path, "/api/v1/auth/public-key/") ||
		path == "/api/v1/ai/chat" || path == "/api/v1/ai/continue-action" ||
		strings.Contains(path, "/package-releases/preview/") ||
		strings.HasSuffix(path, "/package-releases/install") ||
		strings.HasSuffix(path, "/package-releases/upgrade") ||
		strings.Contains(path, "/helm/repositories") ||
		strings.Contains(path, "/helm/charts/upload") ||
		strings.Contains(path, "/helm/upgrade-policies") ||
		(strings.Contains(path, "/resources/:resource") && strings.EqualFold(c.Param("resource"), "secrets"))
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
		if c.Request.Body != nil && !omitAuditBody(c) {
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
			Method:      c.Request.Method,
			Path:        c.FullPath(),
			Source:      "http",
			Outcome:     auditOutcome(c.Writer.Status()),
			RequestBody: bodyStr,
		}

		auditService.Record(entry)
	}
}

func auditOutcome(status int) string {
	if status >= 200 && status < 400 {
		return "succeeded"
	}
	if status == 401 || status == 403 {
		return "denied"
	}
	return "failed"
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
