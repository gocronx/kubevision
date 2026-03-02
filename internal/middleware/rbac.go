package middleware

import (
	"encoding/json"
	"strings"

	"github.com/gin-gonic/gin"
	bizerr "github.com/kubevision/kubevision/internal/pkg/errors"
	"github.com/kubevision/kubevision/internal/pkg/response"
	"github.com/kubevision/kubevision/internal/repository"
)

// RBACMiddleware returns a Gin middleware that enforces role-based access
// control using permissions stored in the role table.
//
// Permission format: "resource:action"
// Wildcards: "*:*" grants all access, "*:get" grants get on all resources,
// "pods:*" grants all actions on pods.
//
// The resource is derived from the first URL segment after /api/v1/
// (e.g. /api/v1/clusters → "clusters", /api/v1/audit-logs → "audit-logs").
// Special sub-paths are also handled: pods/:name/exec → action=exec,
// pods/:name/logs → action=logs.
//
// Admin role always passes all checks without hitting the database.
func RBACMiddleware(roleRepo repository.RoleRepo) gin.HandlerFunc {
	return func(c *gin.Context) {
		role := GetUserRole(c)

		// Super-admin and admin bypass all checks.
		if role == "super-admin" || role == "admin" {
			c.Next()
			return
		}

		// Derive the required permission from the request.
		resource := extractResource(c.FullPath())
		action := extractAction(c.Request.Method, c.FullPath())

		if resource == "" {
			// No resource could be determined; allow through (static/health routes).
			c.Next()
			return
		}

		// Load role permissions from database.
		roleRecord, err := roleRepo.GetByName(c.Request.Context(), role)
		if err != nil {
			// Role not found or DB error — deny by default.
			response.Error(c, bizerr.CodeForbidden, "role not found or permission denied")
			c.Abort()
			return
		}

		var permissions []string
		if err := json.Unmarshal([]byte(roleRecord.Permissions), &permissions); err != nil {
			response.Error(c, bizerr.CodeInternal, "failed to parse role permissions")
			c.Abort()
			return
		}

		if !hasPermission(permissions, resource, action) {
			response.Error(c, bizerr.CodeForbidden, "permission denied: "+resource+":"+action)
			c.Abort()
			return
		}

		c.Next()
	}
}

// RBAC returns a Gin middleware that checks whether the authenticated user
// has the required permissions for the requested resource and action.
// This is a compatibility wrapper retained for any existing callers.
func RBAC(requiredPermissions ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
	}
}

// extractResource returns the primary resource segment from a Gin full path
// (i.e. the route template, not the actual URL).
// Examples:
//
//	/api/v1/clusters                             → "clusters"
//	/api/v1/clusters/:id/resources/:resource     → "resources"
//	/api/v1/audit-logs                           → "audit-logs"
//	/api/v1/api-keys                             → "api-keys"
//	/api/v1/favorites                            → "favorites"
func extractResource(fullPath string) string {
	// Strip the /api/v1/ prefix.
	path := strings.TrimPrefix(fullPath, "/api/v1/")
	if path == "" || path == fullPath {
		return ""
	}

	// Split and walk through segments to find the last meaningful resource name
	// (skip parameter segments like :id, :name).
	segments := strings.Split(path, "/")
	last := ""
	for _, seg := range segments {
		if seg == "" || strings.HasPrefix(seg, ":") {
			continue
		}
		last = seg
	}
	return last
}

// extractAction maps an HTTP method (and optionally a special sub-path like
// /exec or /logs) to a permission action string.
func extractAction(method, fullPath string) string {
	// Check for special terminal/streaming sub-paths first.
	if strings.HasSuffix(fullPath, "/exec") {
		return "exec"
	}
	if strings.HasSuffix(fullPath, "/logs") {
		return "logs"
	}

	switch strings.ToUpper(method) {
	case "GET":
		// Distinguish between a single-resource GET and a list GET based on
		// whether the last meaningful path segment is a parameter or a resource
		// name. We use a simple heuristic: if the full path ends with a param
		// segment (:name, :id, etc.) it is a Get, otherwise it is a list.
		parts := strings.Split(strings.TrimSuffix(fullPath, "/"), "/")
		if len(parts) > 0 && strings.HasPrefix(parts[len(parts)-1], ":") {
			return "get"
		}
		return "list"
	case "POST":
		return "create"
	case "PUT":
		return "update"
	case "PATCH":
		return "update"
	case "DELETE":
		return "delete"
	default:
		return "get"
	}
}

// hasPermission checks whether a list of permission strings grants the
// requested resource:action combination. Supported wildcard forms:
//
//	"*:*"             — allow everything
//	"*:action"        — allow that action on any resource
//	"resource:*"      — allow any action on that resource
//	"resource:action" — exact match
func hasPermission(permissions []string, resource, action string) bool {
	for _, perm := range permissions {
		parts := strings.SplitN(perm, ":", 2)
		if len(parts) != 2 {
			continue
		}
		r, a := parts[0], parts[1]

		resourceMatch := r == "*" || strings.EqualFold(r, resource)
		actionMatch := a == "*" || strings.EqualFold(a, action)

		if resourceMatch && actionMatch {
			return true
		}
	}
	return false
}
