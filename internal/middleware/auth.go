package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/kubevision/kubevision/internal/auth"
	bizerr "github.com/kubevision/kubevision/internal/pkg/errors"
	"github.com/kubevision/kubevision/internal/pkg/response"
	"github.com/kubevision/kubevision/internal/repository"
)

const (
	contextKeyUserID   = "userID"
	contextKeyUsername = "username"
	contextKeyUserRole = "userRole"
)

// AuthMiddleware returns a Gin middleware that validates JWT tokens from the
// Authorization header. It verifies the token signature, expiration, and
// compares the token version against the database to support token revocation.
func AuthMiddleware(jwtManager *auth.JWTManager, userRepo repository.UserRepo) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. Extract the Bearer token from the Authorization header.
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			response.Error(c, bizerr.CodeUnauthorized, "missing authorization header")
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			response.Error(c, bizerr.CodeUnauthorized, "invalid authorization header format")
			c.Abort()
			return
		}

		tokenStr := parts[1]
		if tokenStr == "" {
			response.Error(c, bizerr.CodeUnauthorized, "empty token")
			c.Abort()
			return
		}

		// 2. Parse and validate the JWT.
		claims, err := jwtManager.ParseToken(tokenStr)
		if err != nil {
			response.Error(c, bizerr.CodeTokenExpired, "invalid or expired token")
			c.Abort()
			return
		}

		// 3. Verify token version against the database (supports token revocation).
		user, err := userRepo.GetByID(c.Request.Context(), claims.UserID)
		if err != nil {
			response.Error(c, bizerr.CodeUnauthorized, "user not found")
			c.Abort()
			return
		}

		if !user.IsActive {
			response.Error(c, bizerr.CodeForbidden, "account is disabled")
			c.Abort()
			return
		}

		if claims.TokenVersion != user.TokenVersion {
			response.Error(c, bizerr.CodeTokenExpired, "token has been revoked")
			c.Abort()
			return
		}

		// 4. Inject user information into the request context.
		c.Set(contextKeyUserID, claims.UserID)
		c.Set(contextKeyUsername, claims.Username)
		c.Set(contextKeyUserRole, claims.Role)

		// 5. Continue to the next handler.
		c.Next()
	}
}

// Auth returns a no-op Gin middleware stub used as a fallback when
// the full AuthMiddleware has not been wired in. It simply passes through.
func Auth() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
	}
}

// GetUserID extracts the authenticated user's ID from the Gin context.
func GetUserID(c *gin.Context) uint {
	v, exists := c.Get(contextKeyUserID)
	if !exists {
		return 0
	}
	uid, ok := v.(uint)
	if !ok {
		return 0
	}
	return uid
}

// GetUsername extracts the authenticated user's username from the Gin context.
func GetUsername(c *gin.Context) string {
	v, exists := c.Get(contextKeyUsername)
	if !exists {
		return ""
	}
	username, ok := v.(string)
	if !ok {
		return ""
	}
	return username
}

// GetUserRole extracts the authenticated user's role from the Gin context.
func GetUserRole(c *gin.Context) string {
	v, exists := c.Get(contextKeyUserRole)
	if !exists {
		return ""
	}
	role, ok := v.(string)
	if !ok {
		return ""
	}
	return role
}
