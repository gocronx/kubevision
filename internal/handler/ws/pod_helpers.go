package ws

import (
	"context"
	"errors"
	"strconv"

	"k8s.io/client-go/rest"

	"github.com/kubevision/kubevision/internal/repository"
)

// k8sRestConfig is a type alias kept local to the ws package so terminal.go
// and logs.go can reference it without importing the rest package header directly.
type k8sRestConfig = rest.Config

// Sentinel errors used by both the terminal and logs handlers.
var (
	errAccountDisabled = errors.New("account is disabled")
	errTokenRevoked    = errors.New("token has been revoked")
)

// resolveClusterName converts a raw cluster ID string (numeric DB ID or
// cluster name) to the cluster name key used by the cluster.Manager.
func resolveClusterName(ctx context.Context, clusterIDStr string, clusterRepo repository.ClusterRepo) (string, error) {
	if id, err := strconv.ParseUint(clusterIDStr, 10, 64); err == nil {
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
