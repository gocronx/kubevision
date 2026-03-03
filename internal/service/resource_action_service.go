package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"

	"github.com/kubevision/kubevision/internal/kubernetes/cluster"
	bizerr "github.com/kubevision/kubevision/internal/pkg/errors"
	"github.com/kubevision/kubevision/internal/repository"
)

// scalableKinds enumerates workload resource types that expose a scale subresource.
var scalableKinds = map[string]bool{
	"deployments":  true,
	"statefulsets": true,
	"replicasets":  true,
}

// restartableKinds enumerates workload resource types that support rolling restart.
var restartableKinds = map[string]bool{
	"deployments":  true,
	"statefulsets": true,
	"daemonsets":   true,
	"pods":         true,
}

// ScaleRequest is the request body for scaling a workload.
type ScaleRequest struct {
	Replicas int32 `json:"replicas"`
}

// RollbackRequest is the optional body for rollback.
// A Revision of 0 means "rollback to the previous revision".
type RollbackRequest struct {
	Revision int64 `json:"revision"`
}

// RolloutRevision is a single entry in a Deployment's rollout history.
type RolloutRevision struct {
	Revision    int64  `json:"revision"`
	ChangeCause string `json:"changeCause"`
}

// ResourceActionService implements scale, restart, rollback, and rollout-history
// operations for Kubernetes workload resources.
type ResourceActionService struct {
	clusterRepo    repository.ClusterRepo
	clusterManager *cluster.Manager
}

// NewResourceActionService creates a new ResourceActionService.
func NewResourceActionService(
	clusterRepo repository.ClusterRepo,
	clusterManager *cluster.Manager,
) *ResourceActionService {
	return &ResourceActionService{
		clusterRepo:    clusterRepo,
		clusterManager: clusterManager,
	}
}

// Scale sets the desired replica count on a scalable workload resource.
// Supported kinds: deployments, statefulsets, replicasets.
func (s *ResourceActionService) Scale(
	ctx context.Context,
	clusterID uint,
	kind, namespace, name string,
	req ScaleRequest,
) error {
	if !scalableKinds[kind] {
		return bizerr.New(bizerr.CodeParamInvalid,
			fmt.Sprintf("resource type %q does not support scaling; supported: deployments, statefulsets, replicasets", kind))
	}
	if req.Replicas < 0 {
		return bizerr.New(bizerr.CodeParamInvalid, "replicas must be >= 0")
	}

	clientset, err := s.typedClient(ctx, clusterID)
	if err != nil {
		return err
	}

	// Use a merge patch against the scale subresource so we do not need to
	// fetch and re-submit the full resource.
	patchBody := fmt.Sprintf(`{"spec":{"replicas":%d}}`, req.Replicas)
	pt := types.MergePatchType

	switch kind {
	case "deployments":
		_, err = clientset.AppsV1().Deployments(namespace).Patch(
			ctx, name, pt, []byte(patchBody), metav1.PatchOptions{}, "scale",
		)
	case "statefulsets":
		_, err = clientset.AppsV1().StatefulSets(namespace).Patch(
			ctx, name, pt, []byte(patchBody), metav1.PatchOptions{}, "scale",
		)
	case "replicasets":
		_, err = clientset.AppsV1().ReplicaSets(namespace).Patch(
			ctx, name, pt, []byte(patchBody), metav1.PatchOptions{}, "scale",
		)
	}

	if err != nil {
		return bizerr.New(bizerr.CodeInternal,
			fmt.Sprintf("failed to scale %s %q: %s", kind, name, err.Error()))
	}
	return nil
}

// Restart triggers a rolling restart by patching the pod template annotation
// kubectl.kubernetes.io/restartedAt with the current UTC timestamp.
// Supported kinds: deployments, statefulsets, daemonsets.
func (s *ResourceActionService) Restart(
	ctx context.Context,
	clusterID uint,
	kind, namespace, name string,
) error {
	if !restartableKinds[kind] {
		return bizerr.New(bizerr.CodeParamInvalid,
			fmt.Sprintf("resource type %q does not support restart; supported: deployments, statefulsets, daemonsets", kind))
	}

	clientset, err := s.typedClient(ctx, clusterID)
	if err != nil {
		return err
	}

	now := time.Now().UTC().Format(time.RFC3339)
	patchBody := fmt.Sprintf(
		`{"spec":{"template":{"metadata":{"annotations":{"kubectl.kubernetes.io/restartedAt":%q}}}}}`,
		now,
	)

	switch kind {
	case "pods":
		// For pods: delete the pod so its controller (ReplicaSet/StatefulSet/DaemonSet) recreates it.
		// Check ownerReferences first to prevent accidental deletion of standalone pods.
		pod, getErr := clientset.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
		if getErr != nil {
			return bizerr.New(bizerr.CodeNotFound,
				fmt.Sprintf("pod %q not found in namespace %q: %s", name, namespace, getErr.Error()))
		}
		managed := false
		for _, ref := range pod.OwnerReferences {
			if ref.Controller != nil && *ref.Controller {
				managed = true
				break
			}
		}
		if !managed {
			return bizerr.New(bizerr.CodeParamInvalid,
				fmt.Sprintf("pod %q has no controller; restarting would permanently delete it", name))
		}
		err = clientset.CoreV1().Pods(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	case "deployments":
		_, err = clientset.AppsV1().Deployments(namespace).Patch(
			ctx, name, types.StrategicMergePatchType, []byte(patchBody), metav1.PatchOptions{},
		)
	case "statefulsets":
		_, err = clientset.AppsV1().StatefulSets(namespace).Patch(
			ctx, name, types.StrategicMergePatchType, []byte(patchBody), metav1.PatchOptions{},
		)
	case "daemonsets":
		_, err = clientset.AppsV1().DaemonSets(namespace).Patch(
			ctx, name, types.StrategicMergePatchType, []byte(patchBody), metav1.PatchOptions{},
		)
	}

	if err != nil {
		return bizerr.New(bizerr.CodeInternal,
			fmt.Sprintf("failed to restart %s %q: %s", kind, name, err.Error()))
	}
	return nil
}

// RolloutHistory returns the rollout history for a Deployment, derived from
// the owned ReplicaSets' deployment.kubernetes.io/revision annotations.
func (s *ResourceActionService) RolloutHistory(
	ctx context.Context,
	clusterID uint,
	namespace, name string,
) ([]RolloutRevision, error) {
	clientset, err := s.typedClient(ctx, clusterID)
	if err != nil {
		return nil, err
	}

	deployment, err := clientset.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, bizerr.New(bizerr.CodeNotFound,
			fmt.Sprintf("deployment %q not found in namespace %q: %s", name, namespace, err.Error()))
	}

	rsList, err := clientset.AppsV1().ReplicaSets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, bizerr.New(bizerr.CodeInternal,
			fmt.Sprintf("failed to list ReplicaSets: %s", err.Error()))
	}

	var revisions []RolloutRevision
	for i := range rsList.Items {
		rs := &rsList.Items[i]
		if !isOwnedByDeployment(rs, deployment) {
			continue
		}
		revStr, ok := rs.Annotations["deployment.kubernetes.io/revision"]
		if !ok {
			continue
		}
		rev, parseErr := strconv.ParseInt(revStr, 10, 64)
		if parseErr != nil {
			continue
		}
		revisions = append(revisions, RolloutRevision{
			Revision:    rev,
			ChangeCause: rs.Annotations["kubernetes.io/change-cause"],
		})
	}

	sort.Slice(revisions, func(i, j int) bool {
		return revisions[i].Revision < revisions[j].Revision
	})

	return revisions, nil
}

// Rollback rolls a Deployment back to the specified revision by restoring the
// pod template from the matching ReplicaSet. A revision of 0 means "previous".
func (s *ResourceActionService) Rollback(
	ctx context.Context,
	clusterID uint,
	namespace, name string,
	req RollbackRequest,
) error {
	clientset, err := s.typedClient(ctx, clusterID)
	if err != nil {
		return err
	}

	deployment, err := clientset.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return bizerr.New(bizerr.CodeNotFound,
			fmt.Sprintf("deployment %q not found in namespace %q: %s", name, namespace, err.Error()))
	}

	rsList, err := clientset.AppsV1().ReplicaSets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return bizerr.New(bizerr.CodeInternal,
			fmt.Sprintf("failed to list ReplicaSets: %s", err.Error()))
	}

	// Resolve "previous revision" when the caller passes 0.
	targetRevision := req.Revision
	if targetRevision == 0 {
		currentRevStr := deployment.Annotations["deployment.kubernetes.io/revision"]
		currentRev, _ := strconv.ParseInt(currentRevStr, 10, 64)
		targetRevision = currentRev - 1
	}
	if targetRevision <= 0 {
		return bizerr.New(bizerr.CodeParamInvalid,
			"no previous revision is available to roll back to")
	}

	var targetRS *appsv1.ReplicaSet
	for i := range rsList.Items {
		rs := &rsList.Items[i]
		if !isOwnedByDeployment(rs, deployment) {
			continue
		}
		revStr := rs.Annotations["deployment.kubernetes.io/revision"]
		rev, _ := strconv.ParseInt(revStr, 10, 64)
		if rev == targetRevision {
			targetRS = rs
			break
		}
	}
	if targetRS == nil {
		return bizerr.New(bizerr.CodeNotFound,
			fmt.Sprintf("revision %d not found in rollout history", targetRevision))
	}

	// Patch the Deployment with the pod template from the target ReplicaSet.
	// This triggers a new rollout without needing the deprecated rollback API.
	patchObj := map[string]interface{}{
		"spec": map[string]interface{}{
			"template": targetRS.Spec.Template,
		},
	}
	patchBytes, err := json.Marshal(patchObj)
	if err != nil {
		return bizerr.New(bizerr.CodeInternal, "failed to build rollback patch")
	}

	_, err = clientset.AppsV1().Deployments(namespace).Patch(
		ctx, name, types.StrategicMergePatchType, patchBytes, metav1.PatchOptions{},
	)
	if err != nil {
		return bizerr.New(bizerr.CodeInternal,
			fmt.Sprintf("failed to apply rollback patch: %s", err.Error()))
	}

	return nil
}

// typedClient resolves the cluster DB record and returns a typed Kubernetes
// client built from the cluster manager's cached REST config.
func (s *ResourceActionService) typedClient(ctx context.Context, clusterID uint) (*kubernetes.Clientset, error) {
	record, err := s.clusterRepo.GetByID(ctx, clusterID)
	if err != nil {
		return nil, bizerr.New(bizerr.CodeNotFound,
			fmt.Sprintf("cluster %d not found", clusterID))
	}

	restCfg, err := s.clusterManager.RESTConfig(record.Name)
	if err != nil {
		return nil, bizerr.New(bizerr.CodeK8sUnavailable,
			fmt.Sprintf("cluster %q is not connected: %s", record.Name, err.Error()))
	}

	clientset, err := kubernetes.NewForConfig(restCfg)
	if err != nil {
		return nil, bizerr.New(bizerr.CodeInternal,
			fmt.Sprintf("failed to build Kubernetes client: %s", err.Error()))
	}

	return clientset, nil
}

// isOwnedByDeployment reports whether rs has an OwnerReference pointing to the
// given Deployment.
func isOwnedByDeployment(rs *appsv1.ReplicaSet, d *appsv1.Deployment) bool {
	for _, ref := range rs.OwnerReferences {
		if ref.Kind == "Deployment" && ref.UID == d.UID {
			return true
		}
	}
	return false
}
