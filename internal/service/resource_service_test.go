package service

import (
	"context"
	"errors"
	"testing"

	bizerr "github.com/kubevision/kubevision/internal/pkg/errors"
	"github.com/kubevision/kubevision/internal/kubernetes/resource"
	"github.com/kubevision/kubevision/internal/repository"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// newResourceSvc creates a ResourceService backed by the provided mocks and
// a real resource.Registry so the resource-type validation logic is exercised.
func newResourceSvc(k8s *mockK8sRepo, cluster *mockClusterRepo) *ResourceService {
	reg := resource.NewRegistry()
	return NewResourceService(k8s, reg, cluster)
}

// validBody is a minimal but valid JSON body for a Kubernetes resource.
const validBody = `{"apiVersion":"v1","kind":"Pod","metadata":{"name":"test-pod"}}`

// invalidBody is JSON that cannot be decoded into map[string]interface{}.
const invalidBody = `not-json`

// ---------------------------------------------------------------------------
// Tests: ListResources
// ---------------------------------------------------------------------------

func TestResourceService_ListResources(t *testing.T) {
	t.Run("returns not-found error for unknown cluster", func(t *testing.T) {
		clusterRepo := newMockClusterRepo()
		k8sRepo := newMockK8sRepo()
		svc := newResourceSvc(k8sRepo, clusterRepo)

		_, err := svc.ListResources(context.Background(), 999, "pods", "default", repository.ListOptions{})
		assertBizError(t, err, bizerr.CodeNotFound)
	})

	t.Run("returns param-invalid error for unknown resource type", func(t *testing.T) {
		clusterRepo := newMockClusterRepo()
		clusterRepo.addCluster(makeTestCluster(1, "prod"))
		k8sRepo := newMockK8sRepo()
		svc := newResourceSvc(k8sRepo, clusterRepo)

		_, err := svc.ListResources(context.Background(), 1, "unknownresource", "default", repository.ListOptions{})
		assertBizError(t, err, bizerr.CodeParamInvalid)
	})

	t.Run("returns internal error when k8s repo fails", func(t *testing.T) {
		clusterRepo := newMockClusterRepo()
		clusterRepo.addCluster(makeTestCluster(1, "prod"))
		k8sRepo := newMockK8sRepo()
		k8sRepo.listErr = errors.New("k8s down")
		svc := newResourceSvc(k8sRepo, clusterRepo)

		_, err := svc.ListResources(context.Background(), 1, "pods", "default", repository.ListOptions{})
		assertBizError(t, err, bizerr.CodeInternal)
	})

	t.Run("returns resource list on success", func(t *testing.T) {
		clusterRepo := newMockClusterRepo()
		clusterRepo.addCluster(makeTestCluster(1, "prod"))
		k8sRepo := newMockK8sRepo()
		k8sRepo.listResult = &repository.ResourceList{
			Items: []repository.Resource{
				{Name: "pod-1", Namespace: "default", Kind: "Pod"},
			},
			Total: 1,
		}
		svc := newResourceSvc(k8sRepo, clusterRepo)

		result, err := svc.ListResources(context.Background(), 1, "pods", "default", repository.ListOptions{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result.Items) != 1 {
			t.Errorf("expected 1 item, got %d", len(result.Items))
		}
		if result.Items[0].Name != "pod-1" {
			t.Errorf("item name = %q, want %q", result.Items[0].Name, "pod-1")
		}
	})

	t.Run("passes namespace and options through to k8s repo", func(t *testing.T) {
		clusterRepo := newMockClusterRepo()
		clusterRepo.addCluster(makeTestCluster(2, "staging"))
		k8sRepo := newMockK8sRepo()
		k8sRepo.listResult = &repository.ResourceList{Items: []repository.Resource{}}
		svc := newResourceSvc(k8sRepo, clusterRepo)

		_, err := svc.ListResources(context.Background(), 2, "deployments", "team-a", repository.ListOptions{Limit: 50})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if k8sRepo.lastListClusterID != "staging" {
			t.Errorf("clusterID = %q, want %q", k8sRepo.lastListClusterID, "staging")
		}
		if k8sRepo.lastListNamespace != "team-a" {
			t.Errorf("namespace = %q, want %q", k8sRepo.lastListNamespace, "team-a")
		}
	})
}

// ---------------------------------------------------------------------------
// Tests: GetResource
// ---------------------------------------------------------------------------

func TestResourceService_GetResource(t *testing.T) {
	t.Run("returns not-found error for unknown cluster", func(t *testing.T) {
		svc := newResourceSvc(newMockK8sRepo(), newMockClusterRepo())

		_, err := svc.GetResource(context.Background(), 999, "pods", "default", "my-pod")
		assertBizError(t, err, bizerr.CodeNotFound)
	})

	t.Run("returns param-invalid for unknown resource type", func(t *testing.T) {
		clusterRepo := newMockClusterRepo()
		clusterRepo.addCluster(makeTestCluster(1, "prod"))
		svc := newResourceSvc(newMockK8sRepo(), clusterRepo)

		_, err := svc.GetResource(context.Background(), 1, "badkind", "default", "my-thing")
		assertBizError(t, err, bizerr.CodeParamInvalid)
	})

	t.Run("returns not-found error when k8s repo Get fails", func(t *testing.T) {
		clusterRepo := newMockClusterRepo()
		clusterRepo.addCluster(makeTestCluster(1, "prod"))
		k8sRepo := newMockK8sRepo()
		k8sRepo.getErr = errors.New("resource not found")
		svc := newResourceSvc(k8sRepo, clusterRepo)

		_, err := svc.GetResource(context.Background(), 1, "pods", "default", "my-pod")
		assertBizError(t, err, bizerr.CodeNotFound)
	})

	t.Run("returns resource on success", func(t *testing.T) {
		clusterRepo := newMockClusterRepo()
		clusterRepo.addCluster(makeTestCluster(1, "prod"))
		k8sRepo := newMockK8sRepo()
		k8sRepo.getResult = &repository.Resource{Name: "my-pod", Namespace: "default", Kind: "Pod"}
		svc := newResourceSvc(k8sRepo, clusterRepo)

		res, err := svc.GetResource(context.Background(), 1, "pods", "default", "my-pod")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.Name != "my-pod" {
			t.Errorf("Name = %q, want %q", res.Name, "my-pod")
		}
	})
}

// ---------------------------------------------------------------------------
// Tests: CreateResource
// ---------------------------------------------------------------------------

func TestResourceService_CreateResource(t *testing.T) {
	t.Run("returns not-found for unknown cluster", func(t *testing.T) {
		svc := newResourceSvc(newMockK8sRepo(), newMockClusterRepo())

		_, err := svc.CreateResource(context.Background(), 999, "pods", "default", []byte(validBody))
		assertBizError(t, err, bizerr.CodeNotFound)
	})

	t.Run("returns param-invalid for unknown resource type", func(t *testing.T) {
		clusterRepo := newMockClusterRepo()
		clusterRepo.addCluster(makeTestCluster(1, "prod"))
		svc := newResourceSvc(newMockK8sRepo(), clusterRepo)

		_, err := svc.CreateResource(context.Background(), 1, "widgets", "default", []byte(validBody))
		assertBizError(t, err, bizerr.CodeParamInvalid)
	})

	t.Run("returns param-invalid for invalid JSON body", func(t *testing.T) {
		clusterRepo := newMockClusterRepo()
		clusterRepo.addCluster(makeTestCluster(1, "prod"))
		svc := newResourceSvc(newMockK8sRepo(), clusterRepo)

		_, err := svc.CreateResource(context.Background(), 1, "pods", "default", []byte(invalidBody))
		assertBizError(t, err, bizerr.CodeParamInvalid)
	})

	t.Run("returns internal error when k8s repo Create fails", func(t *testing.T) {
		clusterRepo := newMockClusterRepo()
		clusterRepo.addCluster(makeTestCluster(1, "prod"))
		k8sRepo := newMockK8sRepo()
		k8sRepo.createErr = errors.New("create failed")
		svc := newResourceSvc(k8sRepo, clusterRepo)

		_, err := svc.CreateResource(context.Background(), 1, "pods", "default", []byte(validBody))
		assertBizError(t, err, bizerr.CodeInternal)
	})

	t.Run("returns created resource on success", func(t *testing.T) {
		clusterRepo := newMockClusterRepo()
		clusterRepo.addCluster(makeTestCluster(1, "prod"))
		k8sRepo := newMockK8sRepo()
		k8sRepo.createResult = &repository.Resource{Name: "test-pod", Namespace: "default", Kind: "Pod"}
		svc := newResourceSvc(k8sRepo, clusterRepo)

		res, err := svc.CreateResource(context.Background(), 1, "pods", "default", []byte(validBody))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.Name != "test-pod" {
			t.Errorf("Name = %q, want %q", res.Name, "test-pod")
		}
	})
}

// ---------------------------------------------------------------------------
// Tests: UpdateResource
// ---------------------------------------------------------------------------

func TestResourceService_UpdateResource(t *testing.T) {
	t.Run("returns not-found for unknown cluster", func(t *testing.T) {
		svc := newResourceSvc(newMockK8sRepo(), newMockClusterRepo())

		_, err := svc.UpdateResource(context.Background(), 999, "pods", "default", "my-pod", []byte(validBody))
		assertBizError(t, err, bizerr.CodeNotFound)
	})

	t.Run("returns param-invalid for unknown resource type", func(t *testing.T) {
		clusterRepo := newMockClusterRepo()
		clusterRepo.addCluster(makeTestCluster(1, "prod"))
		svc := newResourceSvc(newMockK8sRepo(), clusterRepo)

		_, err := svc.UpdateResource(context.Background(), 1, "gadgets", "default", "x", []byte(validBody))
		assertBizError(t, err, bizerr.CodeParamInvalid)
	})

	t.Run("returns param-invalid for invalid JSON body", func(t *testing.T) {
		clusterRepo := newMockClusterRepo()
		clusterRepo.addCluster(makeTestCluster(1, "prod"))
		svc := newResourceSvc(newMockK8sRepo(), clusterRepo)

		_, err := svc.UpdateResource(context.Background(), 1, "pods", "default", "my-pod", []byte(invalidBody))
		assertBizError(t, err, bizerr.CodeParamInvalid)
	})

	t.Run("returns internal error when k8s repo Update fails", func(t *testing.T) {
		clusterRepo := newMockClusterRepo()
		clusterRepo.addCluster(makeTestCluster(1, "prod"))
		k8sRepo := newMockK8sRepo()
		k8sRepo.updateErr = errors.New("update failed")
		svc := newResourceSvc(k8sRepo, clusterRepo)

		_, err := svc.UpdateResource(context.Background(), 1, "pods", "default", "my-pod", []byte(validBody))
		assertBizError(t, err, bizerr.CodeInternal)
	})

	t.Run("returns updated resource on success", func(t *testing.T) {
		clusterRepo := newMockClusterRepo()
		clusterRepo.addCluster(makeTestCluster(1, "prod"))
		k8sRepo := newMockK8sRepo()
		k8sRepo.updateResult = &repository.Resource{Name: "my-pod", Namespace: "default", Kind: "Pod"}
		svc := newResourceSvc(k8sRepo, clusterRepo)

		res, err := svc.UpdateResource(context.Background(), 1, "pods", "default", "my-pod", []byte(validBody))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.Name != "my-pod" {
			t.Errorf("Name = %q, want %q", res.Name, "my-pod")
		}
	})
}

// ---------------------------------------------------------------------------
// Tests: DeleteResource
// ---------------------------------------------------------------------------

func TestResourceService_DeleteResource(t *testing.T) {
	t.Run("returns not-found for unknown cluster", func(t *testing.T) {
		svc := newResourceSvc(newMockK8sRepo(), newMockClusterRepo())

		err := svc.DeleteResource(context.Background(), 999, "pods", "default", "my-pod")
		assertBizError(t, err, bizerr.CodeNotFound)
	})

	t.Run("returns param-invalid for unknown resource type", func(t *testing.T) {
		clusterRepo := newMockClusterRepo()
		clusterRepo.addCluster(makeTestCluster(1, "prod"))
		svc := newResourceSvc(newMockK8sRepo(), clusterRepo)

		err := svc.DeleteResource(context.Background(), 1, "unknowns", "default", "x")
		assertBizError(t, err, bizerr.CodeParamInvalid)
	})

	t.Run("returns internal error when k8s repo Delete fails", func(t *testing.T) {
		clusterRepo := newMockClusterRepo()
		clusterRepo.addCluster(makeTestCluster(1, "prod"))
		k8sRepo := newMockK8sRepo()
		k8sRepo.deleteErr = errors.New("delete failed")
		svc := newResourceSvc(k8sRepo, clusterRepo)

		err := svc.DeleteResource(context.Background(), 1, "pods", "default", "my-pod")
		assertBizError(t, err, bizerr.CodeInternal)
	})

	t.Run("returns nil on success", func(t *testing.T) {
		clusterRepo := newMockClusterRepo()
		clusterRepo.addCluster(makeTestCluster(1, "prod"))
		svc := newResourceSvc(newMockK8sRepo(), clusterRepo)

		err := svc.DeleteResource(context.Background(), 1, "pods", "default", "my-pod")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
}

// ---------------------------------------------------------------------------
// Tests: PatchResource
// ---------------------------------------------------------------------------

func TestResourceService_PatchResource(t *testing.T) {
	const validPatch = `{"metadata":{"labels":{"env":"prod"}}}`
	const badPatch = `not-json`

	t.Run("returns not-found for unknown cluster", func(t *testing.T) {
		svc := newResourceSvc(newMockK8sRepo(), newMockClusterRepo())

		_, err := svc.PatchResource(context.Background(), 999, "pods", "default", "my-pod", []byte(validPatch))
		assertBizError(t, err, bizerr.CodeNotFound)
	})

	t.Run("returns param-invalid for unknown resource type", func(t *testing.T) {
		clusterRepo := newMockClusterRepo()
		clusterRepo.addCluster(makeTestCluster(1, "prod"))
		svc := newResourceSvc(newMockK8sRepo(), clusterRepo)

		_, err := svc.PatchResource(context.Background(), 1, "thingies", "default", "x", []byte(validPatch))
		assertBizError(t, err, bizerr.CodeParamInvalid)
	})

	t.Run("returns param-invalid for malformed patch JSON", func(t *testing.T) {
		clusterRepo := newMockClusterRepo()
		clusterRepo.addCluster(makeTestCluster(1, "prod"))
		svc := newResourceSvc(newMockK8sRepo(), clusterRepo)

		_, err := svc.PatchResource(context.Background(), 1, "pods", "default", "my-pod", []byte(badPatch))
		assertBizError(t, err, bizerr.CodeParamInvalid)
	})

	t.Run("returns internal error when k8s repo Patch fails", func(t *testing.T) {
		clusterRepo := newMockClusterRepo()
		clusterRepo.addCluster(makeTestCluster(1, "prod"))
		k8sRepo := newMockK8sRepo()
		k8sRepo.patchErr = errors.New("patch failed")
		svc := newResourceSvc(k8sRepo, clusterRepo)

		_, err := svc.PatchResource(context.Background(), 1, "pods", "default", "my-pod", []byte(validPatch))
		assertBizError(t, err, bizerr.CodeInternal)
	})

	t.Run("returns patched resource on success", func(t *testing.T) {
		clusterRepo := newMockClusterRepo()
		clusterRepo.addCluster(makeTestCluster(1, "prod"))
		k8sRepo := newMockK8sRepo()
		k8sRepo.patchResult = &repository.Resource{Name: "my-pod", Namespace: "default", Kind: "Pod"}
		svc := newResourceSvc(k8sRepo, clusterRepo)

		res, err := svc.PatchResource(context.Background(), 1, "pods", "default", "my-pod", []byte(validPatch))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.Name != "my-pod" {
			t.Errorf("Name = %q, want %q", res.Name, "my-pod")
		}
	})
}

// ---------------------------------------------------------------------------
// Tests: DryRunCreateResource (new method)
// ---------------------------------------------------------------------------

func TestResourceService_DryRunCreateResource(t *testing.T) {
	t.Run("returns not-found biz error when cluster does not exist", func(t *testing.T) {
		svc := newResourceSvc(newMockK8sRepo(), newMockClusterRepo())

		_, err := svc.DryRunCreateResource(context.Background(), 999, "pods", "default", []byte(validBody))
		assertBizError(t, err, bizerr.CodeNotFound)
	})

	t.Run("returns param-invalid for unknown resource type", func(t *testing.T) {
		clusterRepo := newMockClusterRepo()
		clusterRepo.addCluster(makeTestCluster(1, "prod"))
		svc := newResourceSvc(newMockK8sRepo(), clusterRepo)

		_, err := svc.DryRunCreateResource(context.Background(), 1, "thingies", "default", []byte(validBody))
		assertBizError(t, err, bizerr.CodeParamInvalid)
	})

	t.Run("returns param-invalid for invalid JSON body", func(t *testing.T) {
		clusterRepo := newMockClusterRepo()
		clusterRepo.addCluster(makeTestCluster(1, "prod"))
		svc := newResourceSvc(newMockK8sRepo(), clusterRepo)

		_, err := svc.DryRunCreateResource(context.Background(), 1, "pods", "default", []byte(invalidBody))
		assertBizError(t, err, bizerr.CodeParamInvalid)
	})

	t.Run("returns valid=true result with proposed resource on success", func(t *testing.T) {
		clusterRepo := newMockClusterRepo()
		clusterRepo.addCluster(makeTestCluster(1, "prod"))
		k8sRepo := newMockK8sRepo()
		k8sRepo.dryRunCreateResult = &repository.Resource{
			Name:      "test-pod",
			Namespace: "default",
			Kind:      "Pod",
		}
		svc := newResourceSvc(k8sRepo, clusterRepo)

		result, err := svc.DryRunCreateResource(context.Background(), 1, "pods", "default", []byte(validBody))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result == nil {
			t.Fatal("expected non-nil DryRunResult")
		}
		if !result.Valid {
			t.Errorf("expected Valid=true, got false; errors: %v", result.Errors)
		}
		if result.Proposed == nil {
			t.Fatal("expected non-nil Proposed resource")
		}
		if result.Proposed.Name != "test-pod" {
			t.Errorf("Proposed.Name = %q, want %q", result.Proposed.Name, "test-pod")
		}
		if result.Current != nil {
			t.Error("expected nil Current resource for dry-run create")
		}
	})

	t.Run("returns valid=false result with error message when k8s repo fails", func(t *testing.T) {
		clusterRepo := newMockClusterRepo()
		clusterRepo.addCluster(makeTestCluster(1, "prod"))
		k8sRepo := newMockK8sRepo()
		k8sRepo.dryRunCreateErr = errors.New("validation: cpu limit too high")
		svc := newResourceSvc(k8sRepo, clusterRepo)

		result, err := svc.DryRunCreateResource(context.Background(), 1, "pods", "default", []byte(validBody))
		// The service MUST NOT return a hard error here; it should surface the
		// validation failure in the DryRunResult.
		if err != nil {
			t.Fatalf("expected no hard error, got: %v", err)
		}
		if result == nil {
			t.Fatal("expected non-nil DryRunResult")
		}
		if result.Valid {
			t.Error("expected Valid=false when k8s repo dry-run fails")
		}
		if len(result.Errors) == 0 {
			t.Error("expected at least one error message in result.Errors")
		}
		if result.Proposed != nil {
			t.Error("expected nil Proposed when dry-run fails")
		}
	})

	t.Run("dry-run failure error message is propagated to result", func(t *testing.T) {
		clusterRepo := newMockClusterRepo()
		clusterRepo.addCluster(makeTestCluster(1, "prod"))
		k8sRepo := newMockK8sRepo()
		k8sRepo.dryRunCreateErr = errors.New("spec.containers[0].resources: Required value")
		svc := newResourceSvc(k8sRepo, clusterRepo)

		result, err := svc.DryRunCreateResource(context.Background(), 1, "pods", "default", []byte(validBody))
		if err != nil {
			t.Fatalf("unexpected hard error: %v", err)
		}
		if result.Errors[0] != "spec.containers[0].resources: Required value" {
			t.Errorf("error message = %q, want %q", result.Errors[0], "spec.containers[0].resources: Required value")
		}
	})
}

// ---------------------------------------------------------------------------
// Tests: DryRunUpdateResource (new method)
// ---------------------------------------------------------------------------

func TestResourceService_DryRunUpdateResource(t *testing.T) {
	t.Run("returns not-found biz error when cluster does not exist", func(t *testing.T) {
		svc := newResourceSvc(newMockK8sRepo(), newMockClusterRepo())

		_, err := svc.DryRunUpdateResource(context.Background(), 999, "pods", "default", "my-pod", []byte(validBody))
		assertBizError(t, err, bizerr.CodeNotFound)
	})

	t.Run("returns param-invalid for unknown resource type", func(t *testing.T) {
		clusterRepo := newMockClusterRepo()
		clusterRepo.addCluster(makeTestCluster(1, "prod"))
		svc := newResourceSvc(newMockK8sRepo(), clusterRepo)

		_, err := svc.DryRunUpdateResource(context.Background(), 1, "thingies", "default", "x", []byte(validBody))
		assertBizError(t, err, bizerr.CodeParamInvalid)
	})

	t.Run("returns param-invalid for invalid JSON body", func(t *testing.T) {
		clusterRepo := newMockClusterRepo()
		clusterRepo.addCluster(makeTestCluster(1, "prod"))
		svc := newResourceSvc(newMockK8sRepo(), clusterRepo)

		_, err := svc.DryRunUpdateResource(context.Background(), 1, "pods", "default", "my-pod", []byte(invalidBody))
		assertBizError(t, err, bizerr.CodeParamInvalid)
	})

	t.Run("returns valid=true result with current and proposed resources on success", func(t *testing.T) {
		clusterRepo := newMockClusterRepo()
		clusterRepo.addCluster(makeTestCluster(1, "prod"))
		k8sRepo := newMockK8sRepo()
		k8sRepo.dryRunUpdateCurrent = &repository.Resource{Name: "my-pod", Namespace: "default", Kind: "Pod"}
		k8sRepo.dryRunUpdateProposed = &repository.Resource{Name: "my-pod", Namespace: "default", Kind: "Pod",
			Raw: map[string]interface{}{"updated": true}}
		svc := newResourceSvc(k8sRepo, clusterRepo)

		result, err := svc.DryRunUpdateResource(context.Background(), 1, "pods", "default", "my-pod", []byte(validBody))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result == nil {
			t.Fatal("expected non-nil DryRunResult")
		}
		if !result.Valid {
			t.Errorf("expected Valid=true, errors: %v", result.Errors)
		}
		if result.Current == nil {
			t.Error("expected non-nil Current resource")
		}
		if result.Proposed == nil {
			t.Error("expected non-nil Proposed resource")
		}
		if result.Current.Name != "my-pod" {
			t.Errorf("Current.Name = %q, want %q", result.Current.Name, "my-pod")
		}
	})

	t.Run("returns valid=false result when k8s repo dry-run update fails", func(t *testing.T) {
		clusterRepo := newMockClusterRepo()
		clusterRepo.addCluster(makeTestCluster(1, "prod"))
		k8sRepo := newMockK8sRepo()
		k8sRepo.dryRunUpdateErr = errors.New("immutable field changed: spec.selector")
		svc := newResourceSvc(k8sRepo, clusterRepo)

		result, err := svc.DryRunUpdateResource(context.Background(), 1, "pods", "default", "my-pod", []byte(validBody))
		// Must be a soft validation failure, not a hard error.
		if err != nil {
			t.Fatalf("expected no hard error, got: %v", err)
		}
		if result == nil {
			t.Fatal("expected non-nil DryRunResult")
		}
		if result.Valid {
			t.Error("expected Valid=false when update dry-run fails")
		}
		if len(result.Errors) == 0 {
			t.Error("expected at least one error in result.Errors")
		}
		if result.Current != nil {
			t.Error("expected nil Current when dry-run fails")
		}
		if result.Proposed != nil {
			t.Error("expected nil Proposed when dry-run fails")
		}
	})

	t.Run("dry-run update failure error message is propagated", func(t *testing.T) {
		clusterRepo := newMockClusterRepo()
		clusterRepo.addCluster(makeTestCluster(1, "prod"))
		k8sRepo := newMockK8sRepo()
		k8sRepo.dryRunUpdateErr = errors.New("immutable field changed: spec.selector")
		svc := newResourceSvc(k8sRepo, clusterRepo)

		result, _ := svc.DryRunUpdateResource(context.Background(), 1, "pods", "default", "my-pod", []byte(validBody))
		if result.Errors[0] != "immutable field changed: spec.selector" {
			t.Errorf("error = %q, want %q", result.Errors[0], "immutable field changed: spec.selector")
		}
	})
}

// ---------------------------------------------------------------------------
// Helper: assert biz error code
// ---------------------------------------------------------------------------

func assertBizError(t *testing.T, err error, wantCode int) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected BizError with code %d, got nil", wantCode)
	}
	var bizErr *bizerr.BizError
	if !errors.As(err, &bizErr) {
		t.Fatalf("expected BizError, got %T: %v", err, err)
	}
	if bizErr.Code != wantCode {
		t.Errorf("BizError code = %d, want %d", bizErr.Code, wantCode)
	}
}
