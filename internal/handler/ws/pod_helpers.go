package ws

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"

	"k8s.io/client-go/rest"

	"github.com/gocronx/kubevision/internal/repository"
)

// k8sRestConfig is a type alias kept local to the ws package so terminal.go
// and logs.go can reference it without importing the rest package header directly.
type k8sRestConfig = rest.Config

// Sentinel errors used by both the terminal and logs handlers.
var (
	errAccountDisabled  = errors.New("account is disabled")
	errTokenRevoked     = errors.New("token has been revoked")
	errPermissionDenied = errors.New("permission denied")
)

// checkWSPermission verifies that the given role has the requested
// resource:action permission by querying the role repository.
// Admin and super-admin roles always pass without a database lookup.
// Returns errPermissionDenied when the role lacks the required permission or
// when the role record cannot be fetched / parsed.
//
// Permission format: "<resource>:<action>" where "*" is a wildcard.
// Examples that grant access:
//   - "*:*"       — full access
//   - "pods:*"    — all pod actions
//   - "pods:exec" — exec permission only
func checkWSPermission(ctx context.Context, roleRepo repository.RoleRepo, role, resource, action string) error {
	// Built-in privileged roles bypass the database entirely.
	if role == "super-admin" || role == "admin" {
		return nil
	}

	roleRecord, err := roleRepo.GetByName(ctx, role)
	if err != nil {
		// Role not found in the database (e.g. stale JWT referencing a deleted
		// role) — deny rather than silently allowing unknown roles.
		return errPermissionDenied
	}

	if roleRecord.Permissions == "" {
		return errPermissionDenied
	}

	var permissions []string
	if err := json.Unmarshal([]byte(roleRecord.Permissions), &permissions); err != nil {
		// Malformed permissions JSON — deny and let admins investigate.
		return errPermissionDenied
	}

	for _, perm := range permissions {
		parts := strings.SplitN(perm, ":", 2)
		if len(parts) != 2 {
			continue
		}
		r, a := parts[0], parts[1]
		resourceMatch := r == "*" || strings.EqualFold(r, resource)
		actionMatch := a == "*" || strings.EqualFold(a, action)
		if resourceMatch && actionMatch {
			return nil
		}
	}
	return errPermissionDenied
}

// resolveClusterName converts a raw cluster ID string (numeric DB ID or
// cluster name) to the cluster name key used by the cluster.Manager.
func resolveClusterName(ctx context.Context, clusterIDStr string, clusterRepo repository.ClusterRepo) (string, error) {
	if id, err := strconv.ParseUint(clusterIDStr, 10, strconv.IntSize); err == nil {
		c, err := clusterRepo.GetByID(ctx, uint(id))
		if err != nil {
			return "", err
		}
		return c.Name, nil
	}
	// Fall back: treat the string directly as the cluster name.
	c, err := clusterRepo.GetByName(ctx, clusterIDStr)
	if err != nil {
		return "", err
	}
	return c.Name, nil
}
