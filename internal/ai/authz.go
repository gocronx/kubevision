package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gocronx/kubevision/internal/repository"
)

// authorizer checks whether a user's role permits a given tool invocation,
// mirroring the "resource:action" permission model used by the HTTP RBAC
// middleware so the assistant cannot exceed the user's own privileges.
type authorizer struct {
	roles repository.RoleRepo
}

func newAuthorizer(roles repository.RoleRepo) *authorizer {
	return &authorizer{roles: roles}
}

// permission describes the RBAC check a tool requires.
type permission struct {
	resource string // plural resource name, e.g. "pods"
	action   string // get | list | create | update | delete | logs
}

// requiredPermission returns the RBAC permission a tool needs given its
// arguments. The boolean is false for tools that need no special permission
// beyond authentication (none currently).
func requiredPermission(tool string, args map[string]any) (permission, bool) {
	res := strings.ToLower(argString(args, "kind"))
	switch tool {
	case "get_resource":
		return permission{res, "get"}, true
	case "list_resources":
		return permission{res, "list"}, true
	case "get_pod_logs":
		return permission{"pods", "logs"}, true
	case "get_cluster_overview":
		return permission{"nodes", "list"}, true
	case "query_prometheus":
		return permission{"pods", "list"}, true
	case "create_resource":
		return permission{res, "create"}, true
	case "update_resource":
		return permission{res, "update"}, true
	case "patch_resource":
		return permission{res, "update"}, true
	case "delete_resource":
		return permission{res, "delete"}, true
	default:
		return permission{}, false
	}
}

// authorize reports an error message when the role may not run the tool, or an
// empty string when permitted. super-admin and admin bypass all checks, matching
// middleware.RBACMiddleware.
func (a *authorizer) authorize(ctx context.Context, role, tool string, args map[string]any) string {
	if role == "super-admin" || role == "admin" {
		return ""
	}

	perm, needed := requiredPermission(tool, args)
	if !needed {
		return ""
	}
	if perm.resource == "" {
		return "missing required argument: kind"
	}

	record, err := a.roles.GetByName(ctx, role)
	if err != nil || record == nil {
		return fmt.Sprintf("permission denied: role %q not found", role)
	}
	var perms []string
	if err := json.Unmarshal([]byte(record.Permissions), &perms); err != nil {
		return "permission denied: cannot read role permissions"
	}
	if !grants(perms, perm.resource, perm.action) {
		return fmt.Sprintf("permission denied: your role %q lacks %s:%s", role, perm.resource, perm.action)
	}
	return ""
}

// grants applies the same wildcard semantics as the HTTP RBAC middleware:
// "*:*", "*:action", "resource:*", or an exact "resource:action".
func grants(perms []string, resource, action string) bool {
	for _, p := range perms {
		parts := strings.SplitN(p, ":", 2)
		if len(parts) != 2 {
			continue
		}
		r, act := parts[0], parts[1]
		resourceMatch := r == "*" || strings.EqualFold(r, resource)
		actionMatch := act == "*" || strings.EqualFold(act, action)
		if resourceMatch && actionMatch {
			return true
		}
	}
	return false
}

// argString safely extracts a string argument from a decoded tool-call payload.
func argString(args map[string]any, key string) string {
	if args == nil {
		return ""
	}
	if v, ok := args[key].(string); ok {
		return v
	}
	return ""
}
