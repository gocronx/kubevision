package middleware

import "github.com/gin-gonic/gin"

// Audit returns a Gin middleware that records API requests to the audit log.
// TODO: implement audit logging with batch flush.
func Audit() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
	}
}
