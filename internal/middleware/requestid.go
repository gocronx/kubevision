package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const headerXRequestID = "X-Request-ID"

// RequestID returns a Gin middleware that ensures every request has a unique
// request ID. If the client sends one via the X-Request-ID header it is reused;
// otherwise a new UUID is generated.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		rid := c.GetHeader(headerXRequestID)
		if rid == "" {
			rid = uuid.New().String()
		}
		c.Set("requestID", rid)
		c.Header(headerXRequestID, rid)
		c.Next()
	}
}
