package service

import (
	"context"
	"fmt"

	bizerr "github.com/kubevision/kubevision/internal/pkg/errors"
	"github.com/kubevision/kubevision/internal/repository"
)

// OverviewResponse holds the aggregated resource counts for a cluster overview.
type OverviewResponse struct {
	Pods        int `json:"pods"`
	Deployments int `json:"deployments"`
	Services    int `json:"services"`
	Nodes       int `json:"nodes"`
}

// OverviewService aggregates cluster-level resource counts.
type OverviewService struct {
	k8sRepo     repository.K8sResourceRepo
	clusterRepo repository.ClusterRepo
}

// NewOverviewService creates a new OverviewService.
func NewOverviewService(
	k8sRepo repository.K8sResourceRepo,
	clusterRepo repository.ClusterRepo,
) *OverviewService {
	return &OverviewService{
		k8sRepo:     k8sRepo,
		clusterRepo: clusterRepo,
	}
}

// GetOverview fetches counts for pods, deployments, services, and nodes for
// the given cluster. An empty namespace string is intentional: pods,
// deployments, and services are queried across all namespaces, and nodes are
// cluster-scoped.
func (s *OverviewService) GetOverview(
	ctx context.Context,
	clusterID uint,
) (*OverviewResponse, error) {
	// Validate that the cluster exists and get its name (used as the key for
	// the K8s repo, consistent with how QuotaService and TopologyService work).
	cluster, err := s.clusterRepo.GetByID(ctx, clusterID)
	if err != nil {
		return nil, bizerr.New(bizerr.CodeNotFound, fmt.Sprintf("cluster %d not found", clusterID))
	}

	clusterKey := cluster.Name

	// Count each resource type. Nodes are cluster-scoped so namespace="" is
	// correct. Pods, deployments, and services use namespace="" to query across
	// all namespaces.
	resourceTypes := []string{"pods", "deployments", "services", "nodes"}
	counts := make(map[string]int, len(resourceTypes))

	for _, rt := range resourceTypes {
		list, err := s.k8sRepo.List(ctx, clusterKey, rt, "", repository.ListOptions{})
		if err != nil {
			return nil, bizerr.New(
				bizerr.CodeInternal,
				fmt.Sprintf("failed to list %s: %s", rt, err.Error()),
			)
		}
		counts[rt] = int(list.Total)
	}

	return &OverviewResponse{
		Pods:        counts["pods"],
		Deployments: counts["deployments"],
		Services:    counts["services"],
		Nodes:       counts["nodes"],
	}, nil
}
