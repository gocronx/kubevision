package service

import (
	"context"
	"testing"

	"github.com/kubevision/kubevision/internal/repository"
)

func TestTopologyService_GetNamespaceTopology_EmptyNamespace(t *testing.T) {
	k8sRepo := newMockK8sRepo()
	clusterRepo := newMockClusterRepo()
	clusterRepo.addCluster(makeTestCluster(1, "prod"))

	svc := NewTopologyService(k8sRepo, clusterRepo)
	ctx := context.Background()

	result, err := svc.GetNamespaceTopology(ctx, 1, "default")
	if err != nil {
		t.Fatalf("GetNamespaceTopology failed: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Nodes == nil {
		t.Error("expected Nodes to be initialized (not nil)")
	}
	if result.Edges == nil {
		t.Error("expected Edges to be initialized (not nil)")
	}
}

func TestTopologyService_GetNamespaceTopology_ClusterNotFound(t *testing.T) {
	k8sRepo := newMockK8sRepo()
	clusterRepo := newMockClusterRepo()

	svc := NewTopologyService(k8sRepo, clusterRepo)
	ctx := context.Background()

	_, err := svc.GetNamespaceTopology(ctx, 999, "default")
	if err == nil {
		t.Fatal("expected error for non-existent cluster")
	}
}

func TestTopologyService_GetNamespaceTopology_WithResources(t *testing.T) {
	k8sRepo := newMockK8sRepo()
	clusterRepo := newMockClusterRepo()
	clusterRepo.addCluster(makeTestCluster(1, "prod"))

	// Prepare mock data with ownership.
	k8sRepo.listResult = &repository.ResourceList{
		Items: []repository.Resource{
			{
				Name: "nginx-deploy",
				Kind: "Deployment",
				Raw: map[string]interface{}{
					"metadata": map[string]interface{}{
						"name":      "nginx-deploy",
						"namespace": "default",
						"labels":    map[string]interface{}{"app": "nginx"},
					},
					"status": map[string]interface{}{
						"conditions": []interface{}{
							map[string]interface{}{
								"type":   "Available",
								"status": "True",
							},
						},
					},
				},
			},
		},
	}

	svc := NewTopologyService(k8sRepo, clusterRepo)
	ctx := context.Background()

	result, err := svc.GetNamespaceTopology(ctx, 1, "default")
	if err != nil {
		t.Fatalf("GetNamespaceTopology failed: %v", err)
	}

	// The same listResult is returned for all resource types, so we should have multiple nodes.
	if len(result.Nodes) == 0 {
		t.Error("expected at least some nodes")
	}
}

func TestExtractResourceStatus(t *testing.T) {
	tests := []struct {
		name     string
		resType  string
		raw      map[string]interface{}
		expected string
	}{
		{
			"pod running",
			"pods",
			map[string]interface{}{"status": map[string]interface{}{"phase": "Running"}},
			"Running",
		},
		{
			"deployment available",
			"deployments",
			map[string]interface{}{
				"status": map[string]interface{}{
					"conditions": []interface{}{
						map[string]interface{}{"type": "Available", "status": "True"},
					},
				},
			},
			"Available",
		},
		{
			"deployment progressing",
			"deployments",
			map[string]interface{}{
				"status": map[string]interface{}{
					"conditions": []interface{}{
						map[string]interface{}{"type": "Progressing", "status": "True"},
					},
				},
			},
			"Progressing",
		},
		{
			"no status",
			"pods",
			map[string]interface{}{},
			"",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := extractResourceStatus(tc.resType, tc.raw)
			if got != tc.expected {
				t.Errorf("extractResourceStatus(%q, ...) = %q, want %q", tc.resType, got, tc.expected)
			}
		})
	}
}

func TestMatchLabels(t *testing.T) {
	tests := []struct {
		name     string
		selector map[string]string
		labels   map[string]string
		want     bool
	}{
		{"exact match", map[string]string{"app": "nginx"}, map[string]string{"app": "nginx", "env": "prod"}, true},
		{"no match", map[string]string{"app": "nginx"}, map[string]string{"app": "redis"}, false},
		{"empty selector", map[string]string{}, map[string]string{"app": "nginx"}, false},
		{"nil labels", map[string]string{"app": "nginx"}, nil, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := matchLabels(tc.selector, tc.labels)
			if got != tc.want {
				t.Errorf("matchLabels = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestKindToResourceType(t *testing.T) {
	tests := []struct {
		kind string
		want string
	}{
		{"Deployment", "deployments"},
		{"ReplicaSet", "replicasets"},
		{"StatefulSet", "statefulsets"},
		{"DaemonSet", "daemonsets"},
		{"Job", "jobs"},
		{"CronJob", "cronjobs"},
		{"Service", "services"},
		{"Unknown", "Unknown"},
	}

	for _, tc := range tests {
		t.Run(tc.kind, func(t *testing.T) {
			got := kindToResourceType(tc.kind)
			if got != tc.want {
				t.Errorf("kindToResourceType(%q) = %q, want %q", tc.kind, got, tc.want)
			}
		})
	}
}
