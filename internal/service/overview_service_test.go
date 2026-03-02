package service

import (
	"context"
	"errors"
	"testing"

	bizerr "github.com/kubevision/kubevision/internal/pkg/errors"
	"github.com/kubevision/kubevision/internal/repository"
)

// ---------------------------------------------------------------------------
// Tests: OverviewService.GetOverview
// ---------------------------------------------------------------------------

func TestOverviewService_GetOverview_ClusterNotFound(t *testing.T) {
	clusterRepo := newMockClusterRepo() // empty — no clusters seeded
	k8sRepo := newMockK8sRepo()
	svc := NewOverviewService(k8sRepo, clusterRepo)

	_, err := svc.GetOverview(context.Background(), 999)
	if err == nil {
		t.Fatal("expected error for missing cluster, got nil")
	}

	var bizErr *bizerr.BizError
	if !errors.As(err, &bizErr) {
		t.Fatalf("expected BizError, got %T: %v", err, err)
	}
	if bizErr.Code != bizerr.CodeNotFound {
		t.Errorf("expected code %d, got %d", bizerr.CodeNotFound, bizErr.Code)
	}
}

func TestOverviewService_GetOverview_K8sRepoError(t *testing.T) {
	clusterRepo := newMockClusterRepo()
	clusterRepo.addCluster(makeTestCluster(1, "prod"))
	k8sRepo := newMockK8sRepo()
	k8sRepo.listErr = errors.New("k8s unreachable")
	svc := NewOverviewService(k8sRepo, clusterRepo)

	_, err := svc.GetOverview(context.Background(), 1)
	if err == nil {
		t.Fatal("expected error when k8s repo fails, got nil")
	}

	var bizErr *bizerr.BizError
	if !errors.As(err, &bizErr) {
		t.Fatalf("expected BizError, got %T: %v", err, err)
	}
	if bizErr.Code != bizerr.CodeInternal {
		t.Errorf("expected code %d, got %d", bizerr.CodeInternal, bizErr.Code)
	}
}

func TestOverviewService_GetOverview_EmptyCluster(t *testing.T) {
	clusterRepo := newMockClusterRepo()
	clusterRepo.addCluster(makeTestCluster(1, "prod"))
	k8sRepo := newMockK8sRepo()
	// Default mock returns empty list with Total=0.
	svc := NewOverviewService(k8sRepo, clusterRepo)

	resp, err := svc.GetOverview(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Pods != 0 {
		t.Errorf("Pods = %d, want 0", resp.Pods)
	}
	if resp.Deployments != 0 {
		t.Errorf("Deployments = %d, want 0", resp.Deployments)
	}
	if resp.Services != 0 {
		t.Errorf("Services = %d, want 0", resp.Services)
	}
	if resp.Nodes != 0 {
		t.Errorf("Nodes = %d, want 0", resp.Nodes)
	}
}

func TestOverviewService_GetOverview_WithCounts(t *testing.T) {
	clusterRepo := newMockClusterRepo()
	clusterRepo.addCluster(makeTestCluster(1, "prod"))

	// Use a custom implementation that returns different totals per resource type.
	// Because mockK8sRepo uses a single listResult field, we use sequentialMockK8sRepo
	// which cycles through the provided totals slice on successive List calls.
	callIndex := 0
	totals := []int64{12, 4, 7, 3} // pods, deployments, services, nodes
	k8sRepoCustom := &sequentialMockK8sRepo{totals: totals, index: &callIndex}

	svc := NewOverviewService(k8sRepoCustom, clusterRepo)

	resp, err := svc.GetOverview(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Pods != 12 {
		t.Errorf("Pods = %d, want 12", resp.Pods)
	}
	if resp.Deployments != 4 {
		t.Errorf("Deployments = %d, want 4", resp.Deployments)
	}
	if resp.Services != 7 {
		t.Errorf("Services = %d, want 7", resp.Services)
	}
	if resp.Nodes != 3 {
		t.Errorf("Nodes = %d, want 3", resp.Nodes)
	}
}

func TestOverviewService_GetOverview_UsesClusterName(t *testing.T) {
	clusterRepo := newMockClusterRepo()
	clusterRepo.addCluster(makeTestCluster(7, "staging"))
	k8sRepo := newMockK8sRepo()
	svc := NewOverviewService(k8sRepo, clusterRepo)

	_, err := svc.GetOverview(context.Background(), 7)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// The k8sRepo should have been called with "staging" (cluster name), not "7".
	if k8sRepo.lastListClusterID != "staging" {
		t.Errorf("k8sRepo called with clusterID=%q, want %q", k8sRepo.lastListClusterID, "staging")
	}
}

func TestOverviewService_GetOverview_QueriesAllFourResourceTypes(t *testing.T) {
	clusterRepo := newMockClusterRepo()
	clusterRepo.addCluster(makeTestCluster(1, "prod"))

	tracker := &trackingK8sRepo{}
	svc := NewOverviewService(tracker, clusterRepo)

	_, err := svc.GetOverview(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := map[string]bool{
		"pods":        true,
		"deployments": true,
		"services":    true,
		"nodes":       true,
	}
	for _, kind := range tracker.kinds {
		delete(expected, kind)
	}
	if len(expected) > 0 {
		missing := make([]string, 0, len(expected))
		for k := range expected {
			missing = append(missing, k)
		}
		t.Errorf("GetOverview did not query the following resource types: %v", missing)
	}
	if len(tracker.kinds) != 4 {
		t.Errorf("expected exactly 4 k8s List calls, got %d", len(tracker.kinds))
	}
}

// ---------------------------------------------------------------------------
// Helpers: sequential and tracking K8s repo implementations
// ---------------------------------------------------------------------------

// sequentialMockK8sRepo returns a different Total for each successive List call,
// cycling through the provided totals slice. This lets us assert that the service
// correctly maps each resource type to its own count.
type sequentialMockK8sRepo struct {
	totals []int64
	index  *int
}

func (r *sequentialMockK8sRepo) List(_ context.Context, _, _, _ string, _ repository.ListOptions) (*repository.ResourceList, error) {
	i := *r.index
	total := int64(0)
	if i < len(r.totals) {
		total = r.totals[i]
	}
	*r.index++
	return &repository.ResourceList{Items: make([]repository.Resource, total), Total: total}, nil
}

func (r *sequentialMockK8sRepo) Get(_ context.Context, _, _, _, _ string) (*repository.Resource, error) {
	return nil, nil
}
func (r *sequentialMockK8sRepo) Create(_ context.Context, _, _, _ string, _ map[string]interface{}) (*repository.Resource, error) {
	return nil, nil
}
func (r *sequentialMockK8sRepo) Update(_ context.Context, _, _, _, _ string, _ map[string]interface{}) (*repository.Resource, error) {
	return nil, nil
}
func (r *sequentialMockK8sRepo) Delete(_ context.Context, _, _, _, _ string) error {
	return nil
}
func (r *sequentialMockK8sRepo) Patch(_ context.Context, _, _, _, _ string, _ []byte) (*repository.Resource, error) {
	return nil, nil
}
func (r *sequentialMockK8sRepo) DryRunCreate(_ context.Context, _, _, _ string, _ map[string]interface{}) (*repository.Resource, error) {
	return nil, nil
}
func (r *sequentialMockK8sRepo) DryRunUpdate(_ context.Context, _, _, _, _ string, _ map[string]interface{}) (*repository.Resource, *repository.Resource, error) {
	return nil, nil, nil
}

// Compile-time check.
var _ repository.K8sResourceRepo = (*sequentialMockK8sRepo)(nil)

// trackingK8sRepo records every kind passed to List so tests can assert that
// all required resource types are queried.
type trackingK8sRepo struct {
	kinds []string
}

func (r *trackingK8sRepo) List(_ context.Context, _, kind, _ string, _ repository.ListOptions) (*repository.ResourceList, error) {
	r.kinds = append(r.kinds, kind)
	return &repository.ResourceList{Items: []repository.Resource{}, Total: 0}, nil
}
func (r *trackingK8sRepo) Get(_ context.Context, _, _, _, _ string) (*repository.Resource, error) {
	return nil, nil
}
func (r *trackingK8sRepo) Create(_ context.Context, _, _, _ string, _ map[string]interface{}) (*repository.Resource, error) {
	return nil, nil
}
func (r *trackingK8sRepo) Update(_ context.Context, _, _, _, _ string, _ map[string]interface{}) (*repository.Resource, error) {
	return nil, nil
}
func (r *trackingK8sRepo) Delete(_ context.Context, _, _, _, _ string) error { return nil }
func (r *trackingK8sRepo) Patch(_ context.Context, _, _, _, _ string, _ []byte) (*repository.Resource, error) {
	return nil, nil
}
func (r *trackingK8sRepo) DryRunCreate(_ context.Context, _, _, _ string, _ map[string]interface{}) (*repository.Resource, error) {
	return nil, nil
}
func (r *trackingK8sRepo) DryRunUpdate(_ context.Context, _, _, _, _ string, _ map[string]interface{}) (*repository.Resource, *repository.Resource, error) {
	return nil, nil, nil
}

// Compile-time check.
var _ repository.K8sResourceRepo = (*trackingK8sRepo)(nil)
