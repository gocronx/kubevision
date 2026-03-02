package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Logger returns a Gin middleware that logs every request using the provided
// zap logger.
func Logger(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()

		fields := []zap.Field{
			zap.Int("status", status),
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.String("query", query),
			zap.String("ip", c.ClientIP()),
			zap.Duration("latency", latency),
			zap.Int("bodySize", c.Writer.Size()),
		}

		if rid, ok := c.Get("requestID"); ok {
			if ridStr, ok := rid.(string); ok {
				fields = append(fields, zap.String("requestID", ridStr))
			}
		}

		if len(c.Errors) > 0 {
			logger.Error("request completed with errors", append(fields, zap.String("errors", c.Errors.String()))...)
		} else {
			logger.Info("request completed", fields...)
		}
	}
}
