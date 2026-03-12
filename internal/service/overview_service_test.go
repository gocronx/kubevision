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
	if resp.RunningPods != 0 {
		t.Errorf("RunningPods = %d, want 0", resp.RunningPods)
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
	if resp.ReadyNodes != 0 {
		t.Errorf("ReadyNodes = %d, want 0", resp.ReadyNodes)
	}
	if resp.Namespaces != 0 {
		t.Errorf("Namespaces = %d, want 0", resp.Namespaces)
	}
	if resp.Resources.CPU.Allocatable != 0 {
		t.Errorf("Resources.CPU.Allocatable = %d, want 0", resp.Resources.CPU.Allocatable)
	}
	if resp.Resources.Memory.Allocatable != 0 {
		t.Errorf("Resources.Memory.Allocatable = %d, want 0", resp.Resources.Memory.Allocatable)
	}
	if len(resp.RecentEvents) != 0 {
		t.Errorf("RecentEvents = %d, want 0", len(resp.RecentEvents))
	}
}

func TestOverviewService_GetOverview_WithCounts(t *testing.T) {
	clusterRepo := newMockClusterRepo()
	clusterRepo.addCluster(makeTestCluster(1, "prod"))

	// The service calls List for: pods, deployments, services, nodes, namespaces,
	// statefulsets, daemonsets, jobs, cronjobs, ingresses, persistentvolumes,
	// persistentvolumeclaims (12 calls), then events (13th call).
	callIndex := 0
	totals := []int64{12, 4, 7, 3, 5, 0, 0, 0, 0, 0, 0, 0, 0} // pods, deployments, services, nodes, namespaces, ss, ds, jobs, cj, ing, pv, pvc, events
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
	if resp.Namespaces != 5 {
		t.Errorf("Namespaces = %d, want 5", resp.Namespaces)
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

func TestOverviewService_GetOverview_QueriesAllResourceTypesIncludingNamespacesAndEvents(t *testing.T) {
	clusterRepo := newMockClusterRepo()
	clusterRepo.addCluster(makeTestCluster(1, "prod"))

	tracker := &trackingK8sRepo{}
	svc := NewOverviewService(tracker, clusterRepo)

	_, err := svc.GetOverview(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := map[string]bool{
		"pods":                  true,
		"deployments":          true,
		"services":             true,
		"nodes":                true,
		"namespaces":           true,
		"statefulsets":         true,
		"daemonsets":           true,
		"jobs":                 true,
		"cronjobs":             true,
		"ingresses":            true,
		"persistentvolumes":    true,
		"persistentvolumeclaims": true,
		"events":               true,
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
	if len(tracker.kinds) != 13 {
		t.Errorf("expected exactly 13 k8s List calls, got %d", len(tracker.kinds))
	}
}

func TestOverviewService_GetOverview_RunningPodsAndReadyNodes(t *testing.T) {
	clusterRepo := newMockClusterRepo()
	clusterRepo.addCluster(makeTestCluster(1, "prod"))

	// Build pod items: 2 Running, 1 Pending, 1 Succeeded.
	podItems := []repository.Resource{
		{Raw: map[string]any{"status": map[string]any{"phase": "Running"}}},
		{Raw: map[string]any{"status": map[string]any{"phase": "Running"}}},
		{Raw: map[string]any{"status": map[string]any{"phase": "Pending"}}},
		{Raw: map[string]any{"status": map[string]any{"phase": "Succeeded"}}},
	}

	// Build node items: 2 Ready, 1 NotReady.
	readyCondition := map[string]any{"type": "Ready", "status": "True"}
	notReadyCondition := map[string]any{"type": "Ready", "status": "False"}
	nodeItems := []repository.Resource{
		{Raw: map[string]any{"status": map[string]any{"conditions": []any{readyCondition}}}},
		{Raw: map[string]any{"status": map[string]any{"conditions": []any{readyCondition}}}},
		{Raw: map[string]any{"status": map[string]any{"conditions": []any{notReadyCondition}}}},
	}

	callIndex := 0
	k8sRepoCustom := &sequentialMockK8sRepoWithItems{
		calls: []sequentialCall{
			{total: 4, items: podItems},         // pods
			{total: 0, items: nil},              // deployments
			{total: 0, items: nil},              // services
			{total: 3, items: nodeItems},        // nodes
			{total: 0, items: nil},              // namespaces
			{total: 0, items: nil},              // statefulsets
			{total: 0, items: nil},              // daemonsets
			{total: 0, items: nil},              // jobs
			{total: 0, items: nil},              // cronjobs
			{total: 0, items: nil},              // ingresses
			{total: 0, items: nil},              // persistentvolumes
			{total: 0, items: nil},              // persistentvolumeclaims
			{total: 0, items: nil},              // events
		},
		index: &callIndex,
	}

	svc := NewOverviewService(k8sRepoCustom, clusterRepo)

	resp, err := svc.GetOverview(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Pods != 4 {
		t.Errorf("Pods = %d, want 4", resp.Pods)
	}
	if resp.RunningPods != 2 {
		t.Errorf("RunningPods = %d, want 2", resp.RunningPods)
	}
	if resp.Nodes != 3 {
		t.Errorf("Nodes = %d, want 3", resp.Nodes)
	}
	if resp.ReadyNodes != 2 {
		t.Errorf("ReadyNodes = %d, want 2", resp.ReadyNodes)
	}

	// Verify pod status distribution
	if resp.PodStatusDistribution.Running != 2 {
		t.Errorf("PodStatusDistribution.Running = %d, want 2", resp.PodStatusDistribution.Running)
	}
	if resp.PodStatusDistribution.Pending != 1 {
		t.Errorf("PodStatusDistribution.Pending = %d, want 1", resp.PodStatusDistribution.Pending)
	}
	if resp.PodStatusDistribution.Succeeded != 1 {
		t.Errorf("PodStatusDistribution.Succeeded = %d, want 1", resp.PodStatusDistribution.Succeeded)
	}
}

func TestOverviewService_GetOverview_ResourceAggregation(t *testing.T) {
	clusterRepo := newMockClusterRepo()
	clusterRepo.addCluster(makeTestCluster(1, "prod"))

	// Two nodes: each with 4 CPUs (4000m) and 8Gi memory.
	nodeItems := []repository.Resource{
		{Raw: map[string]any{
			"status": map[string]any{
				"allocatable": map[string]any{
					"cpu":    "4",
					"memory": "8Gi",
				},
				"conditions": []any{
					map[string]any{"type": "Ready", "status": "True"},
				},
			},
		}},
		{Raw: map[string]any{
			"status": map[string]any{
				"allocatable": map[string]any{
					"cpu":    "4",
					"memory": "8Gi",
				},
				"conditions": []any{
					map[string]any{"type": "Ready", "status": "True"},
				},
			},
		}},
	}

	// Two pods: each requests 500m CPU / 512Mi memory and limits 1 CPU / 1Gi memory.
	podItems := []repository.Resource{
		{Raw: map[string]any{
			"status": map[string]any{"phase": "Running"},
			"spec": map[string]any{
				"containers": []any{
					map[string]any{
						"resources": map[string]any{
							"requests": map[string]any{"cpu": "500m", "memory": "512Mi"},
							"limits":   map[string]any{"cpu": "1", "memory": "1Gi"},
						},
					},
				},
			},
		}},
		{Raw: map[string]any{
			"status": map[string]any{"phase": "Running"},
			"spec": map[string]any{
				"containers": []any{
					map[string]any{
						"resources": map[string]any{
							"requests": map[string]any{"cpu": "500m", "memory": "512Mi"},
							"limits":   map[string]any{"cpu": "1", "memory": "1Gi"},
						},
					},
				},
			},
		}},
	}

	callIndex := 0
	k8sRepoCustom := &sequentialMockK8sRepoWithItems{
		calls: []sequentialCall{
			{total: 2, items: podItems},  // pods
			{total: 0, items: nil},       // deployments
			{total: 0, items: nil},       // services
			{total: 2, items: nodeItems}, // nodes
			{total: 0, items: nil},       // namespaces
			{total: 0, items: nil},       // statefulsets
			{total: 0, items: nil},       // daemonsets
			{total: 0, items: nil},       // jobs
			{total: 0, items: nil},       // cronjobs
			{total: 0, items: nil},       // ingresses
			{total: 0, items: nil},       // persistentvolumes
			{total: 0, items: nil},       // persistentvolumeclaims
			{total: 0, items: nil},       // events
		},
		index: &callIndex,
	}

	svc := NewOverviewService(k8sRepoCustom, clusterRepo)

	resp, err := svc.GetOverview(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 2 nodes * 4000m = 8000 millicores allocatable
	wantCPUAllocatable := int64(8000)
	if resp.Resources.CPU.Allocatable != wantCPUAllocatable {
		t.Errorf("CPU.Allocatable = %d, want %d", resp.Resources.CPU.Allocatable, wantCPUAllocatable)
	}

	// 2 nodes * 8Gi = 2 * 8 * 1024 * 1024 * 1024 bytes
	wantMemAllocatable := int64(2 * 8 * 1024 * 1024 * 1024)
	if resp.Resources.Memory.Allocatable != wantMemAllocatable {
		t.Errorf("Memory.Allocatable = %d, want %d", resp.Resources.Memory.Allocatable, wantMemAllocatable)
	}

	// 2 pods * 500m = 1000 millicores requested
	wantCPURequests := int64(1000)
	if resp.Resources.CPU.Requests != wantCPURequests {
		t.Errorf("CPU.Requests = %d, want %d", resp.Resources.CPU.Requests, wantCPURequests)
	}

	// 2 pods * 1 CPU = 2000 millicores limited
	wantCPULimits := int64(2000)
	if resp.Resources.CPU.Limits != wantCPULimits {
		t.Errorf("CPU.Limits = %d, want %d", resp.Resources.CPU.Limits, wantCPULimits)
	}

	// 2 pods * 512Mi = 2 * 512 * 1024 * 1024 bytes requested
	wantMemRequests := int64(2 * 512 * 1024 * 1024)
	if resp.Resources.Memory.Requests != wantMemRequests {
		t.Errorf("Memory.Requests = %d, want %d", resp.Resources.Memory.Requests, wantMemRequests)
	}

	// 2 pods * 1Gi = 2 * 1024 * 1024 * 1024 bytes limited
	wantMemLimits := int64(2 * 1024 * 1024 * 1024)
	if resp.Resources.Memory.Limits != wantMemLimits {
		t.Errorf("Memory.Limits = %d, want %d", resp.Resources.Memory.Limits, wantMemLimits)
	}
}

// ---------------------------------------------------------------------------
// Helpers: sequential and tracking K8s repo implementations
// ---------------------------------------------------------------------------

// sequentialMockK8sRepo returns a different Total for each successive List call,
// cycling through the provided totals slice. This lets us assert that the service
// correctly maps each resource type to its own count.
// Items are synthetic: a slice of empty Resources whose length matches the total.
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
func (r *sequentialMockK8sRepo) Create(_ context.Context, _, _, _ string, _ map[string]any) (*repository.Resource, error) {
	return nil, nil
}
func (r *sequentialMockK8sRepo) Update(_ context.Context, _, _, _, _ string, _ map[string]any) (*repository.Resource, error) {
	return nil, nil
}
func (r *sequentialMockK8sRepo) Delete(_ context.Context, _, _, _, _ string) error {
	return nil
}
func (r *sequentialMockK8sRepo) Patch(_ context.Context, _, _, _, _ string, _ []byte) (*repository.Resource, error) {
	return nil, nil
}
func (r *sequentialMockK8sRepo) DryRunCreate(_ context.Context, _, _, _ string, _ map[string]any) (*repository.Resource, error) {
	return nil, nil
}
func (r *sequentialMockK8sRepo) DryRunUpdate(_ context.Context, _, _, _, _ string, _ map[string]any) (*repository.Resource, *repository.Resource, error) {
	return nil, nil, nil
}

// Compile-time check.
var _ repository.K8sResourceRepo = (*sequentialMockK8sRepo)(nil)

// ---------------------------------------------------------------------------

// sequentialCall holds the data to return for one specific List call.
type sequentialCall struct {
	total int64
	items []repository.Resource
}

// sequentialMockK8sRepoWithItems returns configurable items and totals per
// successive List call. It lets tests inject raw pod/node data to exercise
// the running-pod, ready-node, and resource-aggregation logic.
type sequentialMockK8sRepoWithItems struct {
	calls []sequentialCall
	index *int
}

func (r *sequentialMockK8sRepoWithItems) List(_ context.Context, _, _, _ string, _ repository.ListOptions) (*repository.ResourceList, error) {
	i := *r.index
	*r.index++
	if i >= len(r.calls) {
		return &repository.ResourceList{Items: []repository.Resource{}, Total: 0}, nil
	}
	call := r.calls[i]
	items := call.items
	if items == nil {
		items = make([]repository.Resource, call.total)
	}
	return &repository.ResourceList{Items: items, Total: call.total}, nil
}

func (r *sequentialMockK8sRepoWithItems) Get(_ context.Context, _, _, _, _ string) (*repository.Resource, error) {
	return nil, nil
}
func (r *sequentialMockK8sRepoWithItems) Create(_ context.Context, _, _, _ string, _ map[string]any) (*repository.Resource, error) {
	return nil, nil
}
func (r *sequentialMockK8sRepoWithItems) Update(_ context.Context, _, _, _, _ string, _ map[string]any) (*repository.Resource, error) {
	return nil, nil
}
func (r *sequentialMockK8sRepoWithItems) Delete(_ context.Context, _, _, _, _ string) error {
	return nil
}
func (r *sequentialMockK8sRepoWithItems) Patch(_ context.Context, _, _, _, _ string, _ []byte) (*repository.Resource, error) {
	return nil, nil
}
func (r *sequentialMockK8sRepoWithItems) DryRunCreate(_ context.Context, _, _, _ string, _ map[string]any) (*repository.Resource, error) {
	return nil, nil
}
func (r *sequentialMockK8sRepoWithItems) DryRunUpdate(_ context.Context, _, _, _, _ string, _ map[string]any) (*repository.Resource, *repository.Resource, error) {
	return nil, nil, nil
}

// Compile-time check.
var _ repository.K8sResourceRepo = (*sequentialMockK8sRepoWithItems)(nil)

// ---------------------------------------------------------------------------

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
func (r *trackingK8sRepo) Create(_ context.Context, _, _, _ string, _ map[string]any) (*repository.Resource, error) {
	return nil, nil
}
func (r *trackingK8sRepo) Update(_ context.Context, _, _, _, _ string, _ map[string]any) (*repository.Resource, error) {
	return nil, nil
}
func (r *trackingK8sRepo) Delete(_ context.Context, _, _, _, _ string) error { return nil }
func (r *trackingK8sRepo) Patch(_ context.Context, _, _, _, _ string, _ []byte) (*repository.Resource, error) {
	return nil, nil
}
func (r *trackingK8sRepo) DryRunCreate(_ context.Context, _, _, _ string, _ map[string]any) (*repository.Resource, error) {
	return nil, nil
}
func (r *trackingK8sRepo) DryRunUpdate(_ context.Context, _, _, _, _ string, _ map[string]any) (*repository.Resource, *repository.Resource, error) {
	return nil, nil, nil
}

// Compile-time check.
var _ repository.K8sResourceRepo = (*trackingK8sRepo)(nil)
