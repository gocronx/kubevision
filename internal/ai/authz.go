package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gocronx/kubevision/internal/repository"
	"sigs.k8s.io/yaml"
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
		return permission{"pods", "list"}, true
	case "get_cluster_overview":
		return permission{"overview", "list"}, true
	case "query_prometheus":
		return permission{"prometheus", "query"}, true
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
	if tool == "query_prometheus" {
		return "permission denied: arbitrary Prometheus queries require an administrator role"
	}
	if reason := highRiskMutation(tool, args); reason != "" {
		return "permission denied: " + reason + " requires an administrator role"
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
	if !grants(perms, perm.resource, perm.action) && !grants(perms, "resources", perm.action) {
		return fmt.Sprintf("permission denied: your role %q lacks %s:%s", role, perm.resource, perm.action)
	}
	return ""
}

var adminOnlyKinds = map[string]bool{
	"clusterroles": true, "clusterrolebindings": true,
	"roles": true, "rolebindings": true, "serviceaccounts": true,
	"namespaces": true, "nodes": true, "persistentvolumes": true,
	"storageclasses": true,
}

// highRiskMutation blocks common Kubernetes privilege-escalation primitives
// for non-admin users even when they hold broad resource write permissions.
func highRiskMutation(tool string, args map[string]any) string {
	if !isMutation(tool) {
		return ""
	}
	kind := strings.ToLower(argString(args, "kind"))
	if adminOnlyKinds[kind] {
		return "mutating " + kind
	}
	if tool != "create_resource" && tool != "update_resource" && tool != "patch_resource" {
		return ""
	}
	raw := argString(args, "yaml")
	if raw == "" {
		raw = argString(args, "patch")
	}
	if raw == "" {
		return ""
	}
	var value any
	if err := yaml.Unmarshal([]byte(raw), &value); err != nil {
		return ""
	}
	if containsSensitivePodSetting(value) {
		return "using privileged workload settings"
	}
	return ""
}

func containsSensitivePodSetting(value any) bool {
	switch v := value.(type) {
	case map[string]any:
		if isPodSpec(v) && dangerousPodSpec(v) {
			return true
		}
		for _, child := range v {
			if containsSensitivePodSetting(child) {
				return true
			}
		}
	case []any:
		for _, child := range v {
			if containsSensitivePodSetting(child) {
				return true
			}
		}
	}
	return false
}

func isPodSpec(value map[string]any) bool {
	for _, key := range []string{"containers", "initContainers", "ephemeralContainers"} {
		if _, ok := value[key].([]any); ok {
			return true
		}
	}
	for _, key := range []string{"hostNetwork", "hostPID", "hostIPC"} {
		if enabled, _ := value[key].(bool); enabled {
			return true
		}
	}
	if _, ok := value["volumes"].([]any); ok {
		return true
	}
	return false
}

func dangerousPodSpec(spec map[string]any) bool {
	for _, key := range []string{"hostNetwork", "hostPID", "hostIPC"} {
		if enabled, _ := spec[key].(bool); enabled {
			return true
		}
	}
	for _, key := range []string{"serviceAccount", "serviceAccountName"} {
		if value, _ := spec[key].(string); strings.TrimSpace(value) != "" && value != "default" {
			return true
		}
	}
	if volumes, _ := spec["volumes"].([]any); containsMapKey(volumes, "hostPath") {
		return true
	}
	for _, key := range []string{"containers", "initContainers", "ephemeralContainers"} {
		containers, _ := spec[key].([]any)
		for _, item := range containers {
			container, _ := item.(map[string]any)
			if ports, _ := container["ports"].([]any); containsNonzeroInt(ports, "hostPort") {
				return true
			}
			security, _ := container["securityContext"].(map[string]any)
			if dangerousSecurityContext(security) {
				return true
			}
		}
	}
	return dangerousSecurityContext(mapValue(spec, "securityContext"))
}

func dangerousSecurityContext(security map[string]any) bool {
	if security == nil {
		return false
	}
	for _, key := range []string{"privileged", "allowPrivilegeEscalation"} {
		if enabled, _ := security[key].(bool); enabled {
			return true
		}
	}
	if mode, _ := security["procMount"].(string); strings.EqualFold(mode, "Unmasked") {
		return true
	}
	if profile := mapValue(security, "seccompProfile"); strings.EqualFold(argString(profile, "type"), "Unconfined") {
		return true
	}
	capabilities := mapValue(security, "capabilities")
	if added, _ := capabilities["add"].([]any); len(added) > 0 {
		return true
	}
	return false
}

func mapValue(value map[string]any, key string) map[string]any {
	result, _ := value[key].(map[string]any)
	return result
}

func containsMapKey(items []any, key string) bool {
	for _, item := range items {
		if value, ok := item.(map[string]any); ok {
			if _, exists := value[key]; exists {
				return true
			}
		}
	}
	return false
}

func containsNonzeroInt(items []any, key string) bool {
	for _, item := range items {
		if value, ok := item.(map[string]any); ok && argInt(value, key, 0) != 0 {
			return true
		}
	}
	return false
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
