package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gocronx/kubevision/internal/auth"
	bizerr "github.com/gocronx/kubevision/internal/pkg/errors"
	"github.com/gocronx/kubevision/internal/pkg/response"
	"github.com/gocronx/kubevision/internal/repository"
	"github.com/gocronx/kubevision/internal/service"
)

const (
	contextKeyUserID   = "userID"
	contextKeyUsername = "username"
	contextKeyUserRole = "userRole"
)

// AuthMiddleware returns a Gin middleware that validates requests via either:
//  1. X-API-Key header — validated against the API key store.
//  2. Authorization: Bearer <jwt> header — standard JWT validation.
//
// On success it injects userID, username, and userRole into the Gin context.
func AuthMiddleware(
	jwtManager *auth.JWTManager,
	userRepo repository.UserRepo,
	apiKeyService *service.APIKeyService,
) gin.HandlerFunc {
	return func(c *gin.Context) {
		// ── Path 1: API Key authentication ──────────────────────────────────
		if apiKey := c.GetHeader("X-API-Key"); apiKey != "" {
			if apiKeyService == nil {
				response.Error(c, bizerr.CodeUnauthorized, "API key authentication not configured")
				c.Abort()
				return
			}

			record, err := apiKeyService.Validate(c.Request.Context(), apiKey)
			if err != nil {
				if bizErr, ok := err.(*bizerr.BizError); ok {
					response.ErrorWithBizErr(c, bizErr)
				} else {
					response.Error(c, bizerr.CodeUnauthorized, "invalid API key")
				}
				c.Abort()
				return
			}

			// Load full user to get current role and active status.
			user, err := userRepo.GetByID(c.Request.Context(), record.UserID)
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

			c.Set(contextKeyUserID, user.ID)
			c.Set(contextKeyUsername, user.Username)
			c.Set(contextKeyUserRole, user.Role)
			c.Next()
			return
		}

		// ── Path 2: JWT Bearer token authentication ──────────────────────────
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

		// Parse and validate the JWT.
		claims, err := jwtManager.ParseToken(tokenStr)
		if err != nil {
			response.Error(c, bizerr.CodeTokenExpired, "invalid or expired token")
			c.Abort()
			return
		}

		// Verify token version against the database (supports token revocation).
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

		// Inject user information into the request context.
		// Use the role from the database (not the JWT claim) so that
		// role changes take effect immediately without re-login.
		c.Set(contextKeyUserID, claims.UserID)
		c.Set(contextKeyUsername, claims.Username)
		c.Set(contextKeyUserRole, user.Role)

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
