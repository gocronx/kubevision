package middleware

import "github.com/gin-gonic/gin"

// SecurityHeaders returns a Gin middleware that sets defensive HTTP response
// headers on every request.
//
// Header rationale:
//   - X-Frame-Options: DENY            — prevent clickjacking via iframe embedding
//   - X-Content-Type-Options: nosniff  — prevent MIME-type sniffing attacks
//   - Referrer-Policy: strict-origin-when-cross-origin — limit Referer leakage
//   - X-XSS-Protection: 0              — disable legacy XSS filter (modern browsers
//     do not need it and it can introduce vulnerabilities; CSP is preferred)
//   - Permissions-Policy              — disable access to sensitive browser APIs
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Header("X-XSS-Protection", "0")
		c.Header("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		c.Next()
	}
}
