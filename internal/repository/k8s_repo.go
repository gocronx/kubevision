package repository

import (
	"context"
	"fmt"

	"github.com/gocronx/kubevision/internal/kubernetes/cluster"
	"github.com/gocronx/kubevision/internal/kubernetes/informer"
	"github.com/gocronx/kubevision/internal/kubernetes/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
)

// k8sResourceRepo is the implementation of K8sResourceRepo that reads from
// the informer cache for cached resources and falls back to the dynamic client
// for uncached resources.
type k8sResourceRepo struct {
	informerMgr *informer.Manager
	clusterMgr  *cluster.Manager
	registry    *resource.Registry
}

// NewK8sResourceRepo creates a new K8sResourceRepo with the given dependencies.
func NewK8sResourceRepo(
	informerMgr *informer.Manager,
	clusterMgr *cluster.Manager,
	registry *resource.Registry,
) K8sResourceRepo {
	return &k8sResourceRepo{
		informerMgr: informerMgr,
		clusterMgr:  clusterMgr,
		registry:    registry,
	}
}

// List returns resources of the given kind in the specified namespace. For
// cached resources it reads from the informer cache; for uncached resources
// it queries the API server directly via the dynamic client.
func (r *k8sResourceRepo) List(
	ctx context.Context,
	clusterID, kind, namespace string,
	opts ListOptions,
) (*ResourceList, error) {
	meta, ok := r.registry.Get(kind)
	if !ok {
		return nil, fmt.Errorf("unknown resource kind: %s", kind)
	}

	// Cached path: read from informer cache.
	if meta.Cached {
		items, stale, err := r.informerMgr.List(clusterID, meta.GVR, namespace)
		if err != nil {
			return nil, fmt.Errorf("list %s from cache: %w", kind, err)
		}

		resources := make([]Resource, 0, len(items))
		for _, u := range items {
			resources = append(resources, toResource(&u))
		}

		return &ResourceList{
			Items: resources,
			Total: int64(len(resources)),
			Stale: stale,
		}, nil
	}

	// Uncached path: query the API server directly.
	dynClient, err := r.clusterMgr.DynamicClient(clusterID)
	if err != nil {
		return nil, fmt.Errorf("get dynamic client for cluster %s: %w", clusterID, err)
	}

	listOpts := metav1.ListOptions{}
	if opts.LabelSelector != "" {
		listOpts.LabelSelector = opts.LabelSelector
	}
	if opts.FieldSelector != "" {
		listOpts.FieldSelector = opts.FieldSelector
	}
	if opts.Limit > 0 {
		listOpts.Limit = opts.Limit
	}
	if opts.Continue != "" {
		listOpts.Continue = opts.Continue
	}

	var result *unstructured.UnstructuredList
	if namespace != "" {
		result, err = dynClient.Resource(meta.GVR).Namespace(namespace).List(ctx, listOpts)
	} else {
		result, err = dynClient.Resource(meta.GVR).List(ctx, listOpts)
	}
	if err != nil {
		return nil, fmt.Errorf("list %s from API server: %w", kind, err)
	}

	resources := make([]Resource, 0, len(result.Items))
	for i := range result.Items {
		resources = append(resources, toResource(&result.Items[i]))
	}

	return &ResourceList{
		Items:    resources,
		Total:    int64(len(resources)),
		Continue: result.GetContinue(),
	}, nil
}

// Get retrieves a single resource by name. For cached resources it reads from
// the informer cache; for uncached resources it queries the API server.
func (r *k8sResourceRepo) Get(
	ctx context.Context,
	clusterID, kind, namespace, name string,
) (*Resource, error) {
	meta, ok := r.registry.Get(kind)
	if !ok {
		return nil, fmt.Errorf("unknown resource kind: %s", kind)
	}

	// Cached path: read from informer cache.
	if meta.Cached {
		u, err := r.informerMgr.Get(clusterID, meta.GVR, namespace, name)
		if err != nil {
			return nil, fmt.Errorf("get %s/%s from cache: %w", kind, name, err)
		}
		res := toResource(u)
		return &res, nil
	}

	// Uncached path: query the API server.
	dynClient, err := r.clusterMgr.DynamicClient(clusterID)
	if err != nil {
		return nil, fmt.Errorf("get dynamic client for cluster %s: %w", clusterID, err)
	}

	var u *unstructured.Unstructured
	if namespace != "" {
		u, err = dynClient.Resource(meta.GVR).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	} else {
		u, err = dynClient.Resource(meta.GVR).Get(ctx, name, metav1.GetOptions{})
	}
	if err != nil {
		return nil, fmt.Errorf("get %s/%s from API server: %w", kind, name, err)
	}

	res := toResource(u)
	return &res, nil
}

// Create creates a new resource via the dynamic client.
func (r *k8sResourceRepo) Create(
	ctx context.Context,
	clusterID, kind, namespace string,
	obj map[string]interface{},
) (*Resource, error) {
	meta, ok := r.registry.Get(kind)
	if !ok {
		return nil, fmt.Errorf("unknown resource kind: %s", kind)
	}

	dynClient, err := r.clusterMgr.DynamicClient(clusterID)
	if err != nil {
		return nil, fmt.Errorf("get dynamic client for cluster %s: %w", clusterID, err)
	}

	u := &unstructured.Unstructured{Object: obj}

	var created *unstructured.Unstructured
	if namespace != "" {
		created, err = dynClient.Resource(meta.GVR).Namespace(namespace).Create(ctx, u, metav1.CreateOptions{})
	} else {
		created, err = dynClient.Resource(meta.GVR).Create(ctx, u, metav1.CreateOptions{})
	}
	if err != nil {
		return nil, fmt.Errorf("create %s: %w", kind, err)
	}

	res := toResource(created)
	return &res, nil
}

// Update replaces an existing resource via the dynamic client.
func (r *k8sResourceRepo) Update(
	ctx context.Context,
	clusterID, kind, namespace, name string,
	obj map[string]interface{},
) (*Resource, error) {
	meta, ok := r.registry.Get(kind)
	if !ok {
		return nil, fmt.Errorf("unknown resource kind: %s", kind)
	}

	dynClient, err := r.clusterMgr.DynamicClient(clusterID)
	if err != nil {
		return nil, fmt.Errorf("get dynamic client for cluster %s: %w", clusterID, err)
	}

	u := &unstructured.Unstructured{Object: obj}
	u.SetName(name)

	var updated *unstructured.Unstructured
	if namespace != "" {
		updated, err = dynClient.Resource(meta.GVR).Namespace(namespace).Update(ctx, u, metav1.UpdateOptions{})
	} else {
		updated, err = dynClient.Resource(meta.GVR).Update(ctx, u, metav1.UpdateOptions{})
	}
	if err != nil {
		return nil, fmt.Errorf("update %s/%s: %w", kind, name, err)
	}

	res := toResource(updated)
	return &res, nil
}

// Delete removes a resource via the dynamic client.
func (r *k8sResourceRepo) Delete(
	ctx context.Context,
	clusterID, kind, namespace, name string,
) error {
	meta, ok := r.registry.Get(kind)
	if !ok {
		return fmt.Errorf("unknown resource kind: %s", kind)
	}

	dynClient, err := r.clusterMgr.DynamicClient(clusterID)
	if err != nil {
		return fmt.Errorf("get dynamic client for cluster %s: %w", clusterID, err)
	}

	if namespace != "" {
		err = dynClient.Resource(meta.GVR).Namespace(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	} else {
		err = dynClient.Resource(meta.GVR).Delete(ctx, name, metav1.DeleteOptions{})
	}
	if err != nil {
		return fmt.Errorf("delete %s/%s: %w", kind, name, err)
	}

	return nil
}

// Patch applies a strategic merge patch to a resource via the dynamic client.
func (r *k8sResourceRepo) Patch(
	ctx context.Context,
	clusterID, kind, namespace, name string,
	patchData []byte,
) (*Resource, error) {
	meta, ok := r.registry.Get(kind)
	if !ok {
		return nil, fmt.Errorf("unknown resource kind: %s", kind)
	}

	dynClient, err := r.clusterMgr.DynamicClient(clusterID)
	if err != nil {
		return nil, fmt.Errorf("get dynamic client for cluster %s: %w", clusterID, err)
	}

	var patched *unstructured.Unstructured
	if namespace != "" {
		patched, err = dynClient.Resource(meta.GVR).Namespace(namespace).Patch(
			ctx, name, types.StrategicMergePatchType, patchData, metav1.PatchOptions{},
		)
	} else {
		patched, err = dynClient.Resource(meta.GVR).Patch(
			ctx, name, types.StrategicMergePatchType, patchData, metav1.PatchOptions{},
		)
	}
	if err != nil {
		return nil, fmt.Errorf("patch %s/%s: %w", kind, name, err)
	}

	res := toResource(patched)
	return &res, nil
}

// DryRunCreate performs a server-side dry-run create via the dynamic client.
// The Kubernetes API server validates the resource and fills in defaulted fields
// but does not persist anything.
func (r *k8sResourceRepo) DryRunCreate(
	ctx context.Context,
	clusterID, kind, namespace string,
	obj map[string]interface{},
) (*Resource, error) {
	meta, ok := r.registry.Get(kind)
	if !ok {
		return nil, fmt.Errorf("unknown resource kind: %s", kind)
	}

	dynClient, err := r.clusterMgr.DynamicClient(clusterID)
	if err != nil {
		return nil, fmt.Errorf("get dynamic client for cluster %s: %w", clusterID, err)
	}

	u := &unstructured.Unstructured{Object: obj}
	createOpts := metav1.CreateOptions{
		DryRun: []string{metav1.DryRunAll},
	}

	var result *unstructured.Unstructured
	if namespace != "" {
		result, err = dynClient.Resource(meta.GVR).Namespace(namespace).Create(ctx, u, createOpts)
	} else {
		result, err = dynClient.Resource(meta.GVR).Create(ctx, u, createOpts)
	}
	if err != nil {
		return nil, fmt.Errorf("dry-run create %s: %w", kind, err)
	}

	res := toResource(result)
	return &res, nil
}

// DryRunUpdate performs a server-side dry-run update via the dynamic client.
// It fetches the current live resource first so callers can compare before/after.
// The proposed update is validated by the API server but not persisted.
func (r *k8sResourceRepo) DryRunUpdate(
	ctx context.Context,
	clusterID, kind, namespace, name string,
	obj map[string]interface{},
) (*Resource, *Resource, error) {
	meta, ok := r.registry.Get(kind)
	if !ok {
		return nil, nil, fmt.Errorf("unknown resource kind: %s", kind)
	}

	dynClient, err := r.clusterMgr.DynamicClient(clusterID)
	if err != nil {
		return nil, nil, fmt.Errorf("get dynamic client for cluster %s: %w", clusterID, err)
	}

	// Fetch the current live resource for diff comparison.
	var current *unstructured.Unstructured
	if namespace != "" {
		current, err = dynClient.Resource(meta.GVR).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	} else {
		current, err = dynClient.Resource(meta.GVR).Get(ctx, name, metav1.GetOptions{})
	}
	if err != nil {
		return nil, nil, fmt.Errorf("get current %s/%s: %w", kind, name, err)
	}

	// Perform the dry-run update.
	u := &unstructured.Unstructured{Object: obj}
	u.SetName(name)
	updateOpts := metav1.UpdateOptions{
		DryRun: []string{metav1.DryRunAll},
	}

	var proposed *unstructured.Unstructured
	if namespace != "" {
		proposed, err = dynClient.Resource(meta.GVR).Namespace(namespace).Update(ctx, u, updateOpts)
	} else {
		proposed, err = dynClient.Resource(meta.GVR).Update(ctx, u, updateOpts)
	}
	if err != nil {
		return nil, nil, fmt.Errorf("dry-run update %s/%s: %w", kind, name, err)
	}

	currentRes := toResource(current)
	proposedRes := toResource(proposed)
	return &currentRes, &proposedRes, nil
}

// toResource converts an unstructured Kubernetes object into the generic
// Resource holder.
func toResource(u *unstructured.Unstructured) Resource {
	return Resource{
		APIVersion: u.GetAPIVersion(),
		Kind:       u.GetKind(),
		Name:       u.GetName(),
		Namespace:  u.GetNamespace(),
		Raw:        u.Object,
	}
}
