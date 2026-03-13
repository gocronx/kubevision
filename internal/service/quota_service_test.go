package service

import (
	"context"
	"errors"
	"testing"

	bizerr "github.com/gocronx/kubevision/internal/pkg/errors"
	"github.com/gocronx/kubevision/internal/repository"
)

// ---------------------------------------------------------------------------
// Tests: QuotaService.GetQuotaSummary
// ---------------------------------------------------------------------------

func TestQuotaService_GetQuotaSummary(t *testing.T) {
	t.Run("returns not-found error when cluster does not exist", func(t *testing.T) {
		clusterRepo := newMockClusterRepo()
		k8sRepo := newMockK8sRepo()
		svc := NewQuotaService(k8sRepo, clusterRepo)

		_, err := svc.GetQuotaSummary(context.Background(), 999, "")
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
	})

	t.Run("returns internal error when k8s repo list fails", func(t *testing.T) {
		clusterRepo := newMockClusterRepo()
		clusterRepo.addCluster(makeTestCluster(1, "prod"))
		k8sRepo := newMockK8sRepo()
		k8sRepo.listErr = errors.New("k8s unreachable")
		svc := NewQuotaService(k8sRepo, clusterRepo)

		_, err := svc.GetQuotaSummary(context.Background(), 1, "")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		var bizErr *bizerr.BizError
		if !errors.As(err, &bizErr) {
			t.Fatalf("expected BizError, got %T: %v", err, err)
		}
		if bizErr.Code != bizerr.CodeInternal {
			t.Errorf("expected code %d, got %d", bizerr.CodeInternal, bizErr.Code)
		}
	})

	t.Run("returns empty summaries when no resourcequotas exist", func(t *testing.T) {
		clusterRepo := newMockClusterRepo()
		clusterRepo.addCluster(makeTestCluster(1, "prod"))
		k8sRepo := newMockK8sRepo()
		k8sRepo.listResult = &repository.ResourceList{Items: []repository.Resource{}}
		svc := NewQuotaService(k8sRepo, clusterRepo)

		resp, err := svc.GetQuotaSummary(context.Background(), 1, "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(resp.Namespaces) != 0 {
			t.Errorf("expected 0 namespace summaries, got %d", len(resp.Namespaces))
		}
	})

	t.Run("groups resourcequotas by namespace", func(t *testing.T) {
		clusterRepo := newMockClusterRepo()
		clusterRepo.addCluster(makeTestCluster(1, "prod"))
		k8sRepo := newMockK8sRepo()
		k8sRepo.listResult = &repository.ResourceList{
			Items: []repository.Resource{
				{Name: "compute-quota", Namespace: "team-a", Raw: buildQuotaRaw("10", "5", "20Gi", "10Gi")},
				{Name: "network-quota", Namespace: "team-a", Raw: buildQuotaRaw("5", "2", "5Gi", "2Gi")},
				{Name: "default-quota", Namespace: "team-b", Raw: buildQuotaRaw("2", "1", "4Gi", "2Gi")},
			},
		}
		svc := NewQuotaService(k8sRepo, clusterRepo)

		resp, err := svc.GetQuotaSummary(context.Background(), 1, "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// Build a namespace -> count map for easy assertion.
		nsCount := make(map[string]int)
		for _, ns := range resp.Namespaces {
			nsCount[ns.Namespace] = len(ns.Quotas)
		}
		if nsCount["team-a"] != 2 {
			t.Errorf("team-a quota count = %d, want 2", nsCount["team-a"])
		}
		if nsCount["team-b"] != 1 {
			t.Errorf("team-b quota count = %d, want 1", nsCount["team-b"])
		}
	})

	t.Run("specific namespace filter returns only that namespace", func(t *testing.T) {
		clusterRepo := newMockClusterRepo()
		clusterRepo.addCluster(makeTestCluster(1, "prod"))
		k8sRepo := newMockK8sRepo()
		k8sRepo.listResult = &repository.ResourceList{
			Items: []repository.Resource{
				{Name: "quota-1", Namespace: "target-ns", Raw: buildQuotaRaw("10", "5", "20Gi", "10Gi")},
			},
		}
		svc := NewQuotaService(k8sRepo, clusterRepo)

		resp, err := svc.GetQuotaSummary(context.Background(), 1, "target-ns")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(resp.Namespaces) != 1 {
			t.Fatalf("expected 1 namespace, got %d", len(resp.Namespaces))
		}
		if resp.Namespaces[0].Namespace != "target-ns" {
			t.Errorf("namespace = %q, want %q", resp.Namespaces[0].Namespace, "target-ns")
		}
	})

	t.Run("specific namespace with no quotas returns empty quota list", func(t *testing.T) {
		clusterRepo := newMockClusterRepo()
		clusterRepo.addCluster(makeTestCluster(1, "prod"))
		k8sRepo := newMockK8sRepo()
		k8sRepo.listResult = &repository.ResourceList{Items: []repository.Resource{}}
		svc := NewQuotaService(k8sRepo, clusterRepo)

		resp, err := svc.GetQuotaSummary(context.Background(), 1, "empty-ns")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(resp.Namespaces) != 1 {
			t.Fatalf("expected 1 namespace entry, got %d", len(resp.Namespaces))
		}
		if resp.Namespaces[0].Namespace != "empty-ns" {
			t.Errorf("namespace = %q, want %q", resp.Namespaces[0].Namespace, "empty-ns")
		}
		if len(resp.Namespaces[0].Quotas) != 0 {
			t.Errorf("expected empty quota list, got %d entries", len(resp.Namespaces[0].Quotas))
		}
	})

	t.Run("resource without namespace is placed under 'default' namespace", func(t *testing.T) {
		clusterRepo := newMockClusterRepo()
		clusterRepo.addCluster(makeTestCluster(1, "prod"))
		k8sRepo := newMockK8sRepo()
		k8sRepo.listResult = &repository.ResourceList{
			Items: []repository.Resource{
				// Namespace is empty — the service should put this under "default".
				{Name: "no-ns-quota", Namespace: "", Raw: buildQuotaRaw("4", "2", "8Gi", "4Gi")},
			},
		}
		svc := NewQuotaService(k8sRepo, clusterRepo)

		resp, err := svc.GetQuotaSummary(context.Background(), 1, "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		found := false
		for _, ns := range resp.Namespaces {
			if ns.Namespace == "default" {
				found = true
				if len(ns.Quotas) != 1 {
					t.Errorf("expected 1 quota under 'default', got %d", len(ns.Quotas))
				}
			}
		}
		if !found {
			t.Error("expected a 'default' namespace entry for quota with empty namespace")
		}
	})

	t.Run("uses cluster name as key for k8s repo call", func(t *testing.T) {
		clusterRepo := newMockClusterRepo()
		clusterRepo.addCluster(makeTestCluster(7, "staging"))
		k8sRepo := newMockK8sRepo()
		k8sRepo.listResult = &repository.ResourceList{Items: []repository.Resource{}}
		svc := NewQuotaService(k8sRepo, clusterRepo)

		_, err := svc.GetQuotaSummary(context.Background(), 7, "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// The k8sRepo should have been called with clusterID == "staging" (the cluster name).
		if k8sRepo.lastListClusterID != "staging" {
			t.Errorf("k8sRepo called with clusterID=%q, want %q", k8sRepo.lastListClusterID, "staging")
		}
	})
}

// ---------------------------------------------------------------------------
// Tests: quotaEntryFromRaw
// ---------------------------------------------------------------------------

func TestQuotaEntryFromRaw(t *testing.T) {
	t.Run("returns entry with empty maps when raw is nil", func(t *testing.T) {
		entry := quotaEntryFromRaw("quota-a", nil)
		if entry.Name != "quota-a" {
			t.Errorf("Name = %q, want %q", entry.Name, "quota-a")
		}
		if len(entry.Hard) != 0 {
			t.Errorf("Hard should be empty, got %v", entry.Hard)
		}
		if len(entry.Used) != 0 {
			t.Errorf("Used should be empty, got %v", entry.Used)
		}
	})

	t.Run("returns entry with empty maps when raw has no status key", func(t *testing.T) {
		raw := map[string]interface{}{"spec": map[string]interface{}{}}
		entry := quotaEntryFromRaw("quota-b", raw)
		if len(entry.Hard) != 0 {
			t.Errorf("Hard should be empty, got %v", entry.Hard)
		}
		if len(entry.Used) != 0 {
			t.Errorf("Used should be empty, got %v", entry.Used)
		}
	})

	t.Run("correctly parses hard and used from raw status", func(t *testing.T) {
		raw := buildQuotaRaw("10", "3", "16Gi", "4Gi")
		entry := quotaEntryFromRaw("compute", raw)

		if entry.Hard["cpu"] != "10" {
			t.Errorf("Hard[cpu] = %q, want %q", entry.Hard["cpu"], "10")
		}
		if entry.Used["cpu"] != "3" {
			t.Errorf("Used[cpu] = %q, want %q", entry.Used["cpu"], "3")
		}
		if entry.Hard["memory"] != "16Gi" {
			t.Errorf("Hard[memory] = %q, want %q", entry.Hard["memory"], "16Gi")
		}
		if entry.Used["memory"] != "4Gi" {
			t.Errorf("Used[memory] = %q, want %q", entry.Used["memory"], "4Gi")
		}
	})

	t.Run("returns entry with empty used when status has no used key", func(t *testing.T) {
		raw := map[string]interface{}{
			"status": map[string]interface{}{
				"hard": map[string]interface{}{"cpu": "8"},
				// no "used" key
			},
		}
		entry := quotaEntryFromRaw("partial", raw)
		if len(entry.Used) != 0 {
			t.Errorf("Used should be empty when missing, got %v", entry.Used)
		}
		if entry.Hard["cpu"] != "8" {
			t.Errorf("Hard[cpu] = %q, want %q", entry.Hard["cpu"], "8")
		}
	})
}

// ---------------------------------------------------------------------------
// Helpers: build quota raw map
// ---------------------------------------------------------------------------

// buildQuotaRaw returns a raw resource map that mimics a ResourceQuota's
// status section with cpu and memory limits/usage.
func buildQuotaRaw(hardCPU, usedCPU, hardMem, usedMem string) map[string]interface{} {
	return map[string]interface{}{
		"status": map[string]interface{}{
			"hard": map[string]interface{}{
				"cpu":    hardCPU,
				"memory": hardMem,
			},
			"used": map[string]interface{}{
				"cpu":    usedCPU,
				"memory": usedMem,
			},
		},
	}
}
