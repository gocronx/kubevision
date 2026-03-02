package service

import (
	"context"
	"errors"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/kubevision/kubevision/internal/kubernetes/cluster"
	bizerr "github.com/kubevision/kubevision/internal/pkg/errors"
)

// ---------------------------------------------------------------------------
// Helpers: build ResourceActionService with no live cluster
// ---------------------------------------------------------------------------

// newActionSvc returns a service backed by a cluster.Manager with no real
// cluster connections registered.  This lets us exercise the pure validation
// logic (kind allow-lists, replicas range) that runs BEFORE typedClient() is
// called, as well as the cluster-not-found error path.
//
// Tests that need a live Kubernetes API server belong in integration tests.
func newActionSvc(clusterRepo *mockClusterRepo) *ResourceActionService {
	mgr := cluster.NewManager() // empty — no cluster registered
	return NewResourceActionService(clusterRepo, mgr)
}

// ---------------------------------------------------------------------------
// Helpers: lightweight k8s API type builders
// ---------------------------------------------------------------------------

func makeDeploymentWithUID(uid string) *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			UID: types.UID(uid),
		},
	}
}

type ownerRefSpec struct {
	kind string
	uid  string
}

func ownerRef(kind, uid string) ownerRefSpec { return ownerRefSpec{kind, uid} }

func makeReplicaSetWithOwners(refs ...ownerRefSpec) *appsv1.ReplicaSet {
	rs := &appsv1.ReplicaSet{}
	for _, r := range refs {
		rs.OwnerReferences = append(rs.OwnerReferences, metav1.OwnerReference{
			Kind: r.kind,
			UID:  types.UID(r.uid),
		})
	}
	return rs
}

// ---------------------------------------------------------------------------
// Tests: Scale input validation
// ---------------------------------------------------------------------------

func TestResourceActionService_Scale_Validation(t *testing.T) {
	t.Run("unsupported kind returns param-invalid error", func(t *testing.T) {
		svc := newActionSvc(newMockClusterRepo())

		err := svc.Scale(context.Background(), 1, "pods", "default", "my-pod", ScaleRequest{Replicas: 1})
		if err == nil {
			t.Fatal("expected error for non-scalable kind, got nil")
		}
		var bizErr *bizerr.BizError
		if !errors.As(err, &bizErr) {
			t.Fatalf("expected BizError, got %T: %v", err, err)
		}
		if bizErr.Code != bizerr.CodeParamInvalid {
			t.Errorf("code = %d, want %d", bizErr.Code, bizerr.CodeParamInvalid)
		}
	})

	t.Run("daemonsets is not scalable — returns param-invalid", func(t *testing.T) {
		svc := newActionSvc(newMockClusterRepo())

		err := svc.Scale(context.Background(), 1, "daemonsets", "default", "my-ds", ScaleRequest{Replicas: 3})
		assertBizError(t, err, bizerr.CodeParamInvalid)
	})

	t.Run("services is not scalable — returns param-invalid", func(t *testing.T) {
		svc := newActionSvc(newMockClusterRepo())

		err := svc.Scale(context.Background(), 1, "services", "default", "my-svc", ScaleRequest{Replicas: 1})
		assertBizError(t, err, bizerr.CodeParamInvalid)
	})

	t.Run("negative replicas returns param-invalid error", func(t *testing.T) {
		svc := newActionSvc(newMockClusterRepo())

		err := svc.Scale(context.Background(), 1, "deployments", "default", "my-deploy", ScaleRequest{Replicas: -1})
		if err == nil {
			t.Fatal("expected error for negative replicas, got nil")
		}
		var bizErr *bizerr.BizError
		if !errors.As(err, &bizErr) {
			t.Fatalf("expected BizError, got %T: %v", err, err)
		}
		if bizErr.Code != bizerr.CodeParamInvalid {
			t.Errorf("code = %d, want %d", bizErr.Code, bizerr.CodeParamInvalid)
		}
	})

	t.Run("zero replicas is valid — proceeds to cluster lookup, not param error", func(t *testing.T) {
		// Zero replicas should NOT be rejected by the validation guard.
		// The service passes validation and then fails at cluster resolution.
		svc := newActionSvc(newMockClusterRepo())

		err := svc.Scale(context.Background(), 999, "deployments", "default", "my-deploy", ScaleRequest{Replicas: 0})
		if err == nil {
			t.Fatal("expected cluster-not-found error, got nil")
		}
		var bizErr *bizerr.BizError
		if !errors.As(err, &bizErr) {
			t.Fatalf("expected BizError, got %T", err)
		}
		if bizErr.Code == bizerr.CodeParamInvalid {
			t.Error("replicas=0 should not produce a param-invalid error")
		}
	})

	t.Run("scalable kinds pass validation — fail only at cluster lookup", func(t *testing.T) {
		svc := newActionSvc(newMockClusterRepo())
		validKinds := []string{"deployments", "statefulsets", "replicasets"}

		for _, kind := range validKinds {
			kind := kind
			t.Run(kind, func(t *testing.T) {
				err := svc.Scale(context.Background(), 999, kind, "default", "x", ScaleRequest{Replicas: 2})
				if err == nil {
					t.Fatal("expected cluster-not-found error, got nil")
				}
				var bizErr *bizerr.BizError
				if !errors.As(err, &bizErr) {
					t.Fatalf("expected BizError, got %T", err)
				}
				// Must NOT be param-invalid — the kind validation passed.
				if bizErr.Code == bizerr.CodeParamInvalid {
					t.Errorf("kind=%q passed validation but returned param-invalid", kind)
				}
			})
		}
	})

	t.Run("cluster not found returns not-found biz error", func(t *testing.T) {
		clusterRepo := newMockClusterRepo()
		// Cluster 42 does not exist.
		svc := newActionSvc(clusterRepo)

		err := svc.Scale(context.Background(), 42, "deployments", "default", "my-deploy", ScaleRequest{Replicas: 1})
		assertBizError(t, err, bizerr.CodeNotFound)
	})
}

// ---------------------------------------------------------------------------
// Tests: Restart input validation
// ---------------------------------------------------------------------------

func TestResourceActionService_Restart_Validation(t *testing.T) {
	t.Run("unsupported kind returns param-invalid error", func(t *testing.T) {
		svc := newActionSvc(newMockClusterRepo())

		err := svc.Restart(context.Background(), 1, "pods", "default", "my-pod")
		assertBizError(t, err, bizerr.CodeParamInvalid)
	})

	t.Run("replicasets is not restartable — returns param-invalid", func(t *testing.T) {
		svc := newActionSvc(newMockClusterRepo())

		err := svc.Restart(context.Background(), 1, "replicasets", "default", "my-rs")
		assertBizError(t, err, bizerr.CodeParamInvalid)
	})

	t.Run("services is not restartable — returns param-invalid", func(t *testing.T) {
		svc := newActionSvc(newMockClusterRepo())

		err := svc.Restart(context.Background(), 1, "services", "default", "my-svc")
		assertBizError(t, err, bizerr.CodeParamInvalid)
	})

	t.Run("jobs is not restartable — returns param-invalid", func(t *testing.T) {
		svc := newActionSvc(newMockClusterRepo())

		err := svc.Restart(context.Background(), 1, "jobs", "default", "my-job")
		assertBizError(t, err, bizerr.CodeParamInvalid)
	})

	t.Run("restartable kinds pass validation — fail only at cluster lookup", func(t *testing.T) {
		svc := newActionSvc(newMockClusterRepo())
		validKinds := []string{"deployments", "statefulsets", "daemonsets"}

		for _, kind := range validKinds {
			kind := kind
			t.Run(kind, func(t *testing.T) {
				err := svc.Restart(context.Background(), 999, kind, "default", "x")
				if err == nil {
					t.Fatal("expected cluster error, got nil")
				}
				var bizErr *bizerr.BizError
				if !errors.As(err, &bizErr) {
					t.Fatalf("expected BizError, got %T", err)
				}
				if bizErr.Code == bizerr.CodeParamInvalid {
					t.Errorf("kind=%q should pass restart validation but returned param-invalid", kind)
				}
			})
		}
	})

	t.Run("cluster not found returns not-found error after validation passes", func(t *testing.T) {
		svc := newActionSvc(newMockClusterRepo())

		err := svc.Restart(context.Background(), 777, "deployments", "default", "my-deploy")
		assertBizError(t, err, bizerr.CodeNotFound)
	})
}

// ---------------------------------------------------------------------------
// Tests: RolloutHistory — cluster lookup error paths
// ---------------------------------------------------------------------------

func TestResourceActionService_RolloutHistory_ClusterLookup(t *testing.T) {
	t.Run("unknown cluster ID returns not-found error", func(t *testing.T) {
		svc := newActionSvc(newMockClusterRepo())

		_, err := svc.RolloutHistory(context.Background(), 999, "default", "my-deploy")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		assertBizError(t, err, bizerr.CodeNotFound)
	})

	t.Run("known cluster not connected to manager returns k8s-unavailable error", func(t *testing.T) {
		// The cluster record exists in the DB repo but the cluster.Manager
		// has no connection for it (no Add() was called), so RESTConfig fails.
		clusterRepo := newMockClusterRepo()
		clusterRepo.addCluster(makeTestCluster(1, "prod"))
		svc := newActionSvc(clusterRepo)

		_, err := svc.RolloutHistory(context.Background(), 1, "default", "my-deploy")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		assertBizError(t, err, bizerr.CodeK8sUnavailable)
	})
}

// ---------------------------------------------------------------------------
// Tests: Rollback — cluster lookup error paths
// ---------------------------------------------------------------------------

func TestResourceActionService_Rollback_ClusterLookup(t *testing.T) {
	t.Run("unknown cluster ID returns not-found error", func(t *testing.T) {
		svc := newActionSvc(newMockClusterRepo())

		err := svc.Rollback(context.Background(), 999, "default", "my-deploy", RollbackRequest{Revision: 1})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		assertBizError(t, err, bizerr.CodeNotFound)
	})

	t.Run("known cluster not in manager returns k8s-unavailable error", func(t *testing.T) {
		clusterRepo := newMockClusterRepo()
		clusterRepo.addCluster(makeTestCluster(1, "prod"))
		svc := newActionSvc(clusterRepo)

		err := svc.Rollback(context.Background(), 1, "default", "my-deploy", RollbackRequest{Revision: 2})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		assertBizError(t, err, bizerr.CodeK8sUnavailable)
	})
}

// ---------------------------------------------------------------------------
// Tests: isOwnedByDeployment (package-level helper, same package access)
// ---------------------------------------------------------------------------

func TestIsOwnedByDeployment(t *testing.T) {
	t.Run("returns false for replica set with no owner references", func(t *testing.T) {
		rs := makeReplicaSetWithOwners()
		d := makeDeploymentWithUID("uid-abc")
		if isOwnedByDeployment(rs, d) {
			t.Error("expected false for RS with no owner refs, got true")
		}
	})

	t.Run("returns false when owner kind does not match Deployment", func(t *testing.T) {
		rs := makeReplicaSetWithOwners(ownerRef("StatefulSet", "uid-abc"))
		d := makeDeploymentWithUID("uid-abc")
		if isOwnedByDeployment(rs, d) {
			t.Error("expected false when owner kind is StatefulSet, not Deployment")
		}
	})

	t.Run("returns false when kind matches but UID differs", func(t *testing.T) {
		rs := makeReplicaSetWithOwners(ownerRef("Deployment", "uid-xyz"))
		d := makeDeploymentWithUID("uid-abc")
		if isOwnedByDeployment(rs, d) {
			t.Error("expected false when UID does not match")
		}
	})

	t.Run("returns true when kind is Deployment and UID matches", func(t *testing.T) {
		rs := makeReplicaSetWithOwners(ownerRef("Deployment", "uid-abc"))
		d := makeDeploymentWithUID("uid-abc")
		if !isOwnedByDeployment(rs, d) {
			t.Error("expected true when kind=Deployment and UID matches")
		}
	})

	t.Run("returns true when matching owner is among multiple owner refs", func(t *testing.T) {
		rs := makeReplicaSetWithOwners(
			ownerRef("ReplicaSet", "uid-other"),
			ownerRef("Deployment", "uid-abc"),
		)
		d := makeDeploymentWithUID("uid-abc")
		if !isOwnedByDeployment(rs, d) {
			t.Error("expected true when matching Deployment ref is not the first entry")
		}
	})

	t.Run("returns false when multiple owners present but none match", func(t *testing.T) {
		rs := makeReplicaSetWithOwners(
			ownerRef("Deployment", "uid-111"),
			ownerRef("Deployment", "uid-222"),
		)
		d := makeDeploymentWithUID("uid-abc")
		if isOwnedByDeployment(rs, d) {
			t.Error("expected false when no owner UID matches the deployment")
		}
	})
}
