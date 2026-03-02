package service

import (
	"context"
	"testing"
)

func TestClusterService_List_Empty(t *testing.T) {
	clusterRepo := newMockClusterRepo()
	k8sRepo := newMockK8sRepo()
	_ = k8sRepo // ClusterService doesn't directly use k8sRepo

	// ClusterService requires concrete types from kubernetes packages, so we
	// test the parts that only interact with ClusterRepo.
	svc := &ClusterService{
		clusterRepo: clusterRepo,
	}

	ctx := context.Background()
	list, err := svc.List(ctx)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(list) != 0 {
		t.Errorf("expected empty list, got %d", len(list))
	}
}

func TestClusterService_List_WithClusters(t *testing.T) {
	clusterRepo := newMockClusterRepo()
	clusterRepo.addCluster(makeTestCluster(1, "prod"))
	clusterRepo.addCluster(makeTestCluster(2, "staging"))

	svc := &ClusterService{
		clusterRepo: clusterRepo,
	}

	ctx := context.Background()
	list, err := svc.List(ctx)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(list) != 2 {
		t.Errorf("expected 2 clusters, got %d", len(list))
	}
}

func TestClusterService_Get_Success(t *testing.T) {
	clusterRepo := newMockClusterRepo()
	clusterRepo.addCluster(makeTestCluster(1, "prod"))

	svc := &ClusterService{
		clusterRepo: clusterRepo,
	}

	ctx := context.Background()
	resp, err := svc.Get(ctx, 1)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if resp.Name != "prod" {
		t.Errorf("expected name 'prod', got %q", resp.Name)
	}
}

func TestClusterService_Get_NotFound(t *testing.T) {
	clusterRepo := newMockClusterRepo()

	svc := &ClusterService{
		clusterRepo: clusterRepo,
	}

	ctx := context.Background()
	_, err := svc.Get(ctx, 999)
	if err == nil {
		t.Fatal("expected error for non-existent cluster")
	}
}

func TestClusterService_ResolveClusterID_ByNumericID(t *testing.T) {
	clusterRepo := newMockClusterRepo()
	clusterRepo.addCluster(makeTestCluster(1, "prod"))

	svc := &ClusterService{
		clusterRepo: clusterRepo,
	}

	ctx := context.Background()
	name, err := svc.ResolveClusterID(ctx, "1")
	if err != nil {
		t.Fatalf("ResolveClusterID failed: %v", err)
	}
	if name != "prod" {
		t.Errorf("expected 'prod', got %q", name)
	}
}

func TestClusterService_ResolveClusterID_ByName(t *testing.T) {
	clusterRepo := newMockClusterRepo()
	clusterRepo.addCluster(makeTestCluster(1, "prod"))

	svc := &ClusterService{
		clusterRepo: clusterRepo,
	}

	ctx := context.Background()
	name, err := svc.ResolveClusterID(ctx, "prod")
	if err != nil {
		t.Fatalf("ResolveClusterID failed: %v", err)
	}
	if name != "prod" {
		t.Errorf("expected 'prod', got %q", name)
	}
}

func TestClusterService_ResolveClusterID_NotFound(t *testing.T) {
	clusterRepo := newMockClusterRepo()

	svc := &ClusterService{
		clusterRepo: clusterRepo,
	}

	ctx := context.Background()
	_, err := svc.ResolveClusterID(ctx, "unknown")
	if err == nil {
		t.Fatal("expected error for non-existent cluster")
	}
}

func TestToClusterResponse(t *testing.T) {
	c := makeTestCluster(1, "prod")
	resp := toClusterResponse(c)

	if resp.ID != 1 {
		t.Errorf("expected ID 1, got %d", resp.ID)
	}
	if resp.Name != "prod" {
		t.Errorf("expected name 'prod', got %q", resp.Name)
	}
	if resp.Status != "healthy" {
		t.Errorf("expected status 'healthy', got %q", resp.Status)
	}
}
