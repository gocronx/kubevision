package middleware

import "github.com/gin-gonic/gin"

// RBAC returns a Gin middleware that checks whether the authenticated user
// has the required permissions for the requested resource and action.
// TODO: implement RBAC enforcement logic.
func RBAC(requiredPermissions ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
	}
}
