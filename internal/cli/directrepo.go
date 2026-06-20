package cli

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"

	"github.com/gocronx/kubevision/internal/kubernetes/cluster"
	"github.com/gocronx/kubevision/internal/kubernetes/resource"
	"github.com/gocronx/kubevision/internal/repository"
)

// directRepo is a repository.K8sResourceRepo that always talks to the API server
// via the dynamic client, bypassing the informer cache. The server reads from a
// shared informer cache, but a short-lived CLI process should not pay the cost
// of starting and syncing informers, so it reads directly.
type directRepo struct {
	mgr      *cluster.Manager
	registry *resource.Registry
}

func newDirectRepo(mgr *cluster.Manager, registry *resource.Registry) repository.K8sResourceRepo {
	return &directRepo{mgr: mgr, registry: registry}
}

func (r *directRepo) List(ctx context.Context, clusterID, kind, namespace string, opts repository.ListOptions) (*repository.ResourceList, error) {
	meta, ok := r.registry.Get(kind)
	if !ok {
		return nil, fmt.Errorf("unknown resource kind: %s", kind)
	}
	dyn, err := r.mgr.DynamicClient(clusterID)
	if err != nil {
		return nil, err
	}
	listOpts := metav1.ListOptions{
		LabelSelector: opts.LabelSelector,
		FieldSelector: opts.FieldSelector,
		Limit:         opts.Limit,
		Continue:      opts.Continue,
	}
	var result *unstructured.UnstructuredList
	if namespace != "" {
		result, err = dyn.Resource(meta.GVR).Namespace(namespace).List(ctx, listOpts)
	} else {
		result, err = dyn.Resource(meta.GVR).List(ctx, listOpts)
	}
	if err != nil {
		return nil, fmt.Errorf("list %s: %w", kind, err)
	}
	items := make([]repository.Resource, 0, len(result.Items))
	for i := range result.Items {
		items = append(items, toRes(&result.Items[i]))
	}
	return &repository.ResourceList{Items: items, Total: int64(len(items)), Continue: result.GetContinue()}, nil
}

func (r *directRepo) Get(ctx context.Context, clusterID, kind, namespace, name string) (*repository.Resource, error) {
	meta, ok := r.registry.Get(kind)
	if !ok {
		return nil, fmt.Errorf("unknown resource kind: %s", kind)
	}
	dyn, err := r.mgr.DynamicClient(clusterID)
	if err != nil {
		return nil, err
	}
	var u *unstructured.Unstructured
	if namespace != "" {
		u, err = dyn.Resource(meta.GVR).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	} else {
		u, err = dyn.Resource(meta.GVR).Get(ctx, name, metav1.GetOptions{})
	}
	if err != nil {
		return nil, fmt.Errorf("get %s/%s: %w", kind, name, err)
	}
	res := toRes(u)
	return &res, nil
}

func (r *directRepo) Create(ctx context.Context, clusterID, kind, namespace string, obj map[string]any) (*repository.Resource, error) {
	meta, ok := r.registry.Get(kind)
	if !ok {
		return nil, fmt.Errorf("unknown resource kind: %s", kind)
	}
	dyn, err := r.mgr.DynamicClient(clusterID)
	if err != nil {
		return nil, err
	}
	u := &unstructured.Unstructured{Object: obj}
	var created *unstructured.Unstructured
	if namespace != "" {
		created, err = dyn.Resource(meta.GVR).Namespace(namespace).Create(ctx, u, metav1.CreateOptions{})
	} else {
		created, err = dyn.Resource(meta.GVR).Create(ctx, u, metav1.CreateOptions{})
	}
	if err != nil {
		return nil, fmt.Errorf("create %s: %w", kind, err)
	}
	res := toRes(created)
	return &res, nil
}

func (r *directRepo) Update(ctx context.Context, clusterID, kind, namespace, name string, obj map[string]any) (*repository.Resource, error) {
	meta, ok := r.registry.Get(kind)
	if !ok {
		return nil, fmt.Errorf("unknown resource kind: %s", kind)
	}
	dyn, err := r.mgr.DynamicClient(clusterID)
	if err != nil {
		return nil, err
	}
	u := &unstructured.Unstructured{Object: obj}
	u.SetName(name)
	var updated *unstructured.Unstructured
	if namespace != "" {
		updated, err = dyn.Resource(meta.GVR).Namespace(namespace).Update(ctx, u, metav1.UpdateOptions{})
	} else {
		updated, err = dyn.Resource(meta.GVR).Update(ctx, u, metav1.UpdateOptions{})
	}
	if err != nil {
		return nil, fmt.Errorf("update %s/%s: %w", kind, name, err)
	}
	res := toRes(updated)
	return &res, nil
}

func (r *directRepo) Delete(ctx context.Context, clusterID, kind, namespace, name string) error {
	meta, ok := r.registry.Get(kind)
	if !ok {
		return fmt.Errorf("unknown resource kind: %s", kind)
	}
	dyn, err := r.mgr.DynamicClient(clusterID)
	if err != nil {
		return err
	}
	if namespace != "" {
		err = dyn.Resource(meta.GVR).Namespace(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	} else {
		err = dyn.Resource(meta.GVR).Delete(ctx, name, metav1.DeleteOptions{})
	}
	if err != nil {
		return fmt.Errorf("delete %s/%s: %w", kind, name, err)
	}
	return nil
}

func (r *directRepo) Patch(ctx context.Context, clusterID, kind, namespace, name string, patchData []byte) (*repository.Resource, error) {
	meta, ok := r.registry.Get(kind)
	if !ok {
		return nil, fmt.Errorf("unknown resource kind: %s", kind)
	}
	dyn, err := r.mgr.DynamicClient(clusterID)
	if err != nil {
		return nil, err
	}
	var patched *unstructured.Unstructured
	if namespace != "" {
		patched, err = dyn.Resource(meta.GVR).Namespace(namespace).Patch(ctx, name, types.StrategicMergePatchType, patchData, metav1.PatchOptions{})
	} else {
		patched, err = dyn.Resource(meta.GVR).Patch(ctx, name, types.StrategicMergePatchType, patchData, metav1.PatchOptions{})
	}
	if err != nil {
		return nil, fmt.Errorf("patch %s/%s: %w", kind, name, err)
	}
	res := toRes(patched)
	return &res, nil
}

// DryRun operations are unused by the AI tools and unsupported by the CLI repo.
func (r *directRepo) DryRunCreate(context.Context, string, string, string, map[string]any) (*repository.Resource, error) {
	return nil, fmt.Errorf("dry-run not supported in CLI")
}

func (r *directRepo) DryRunUpdate(context.Context, string, string, string, string, map[string]any) (*repository.Resource, *repository.Resource, error) {
	return nil, nil, fmt.Errorf("dry-run not supported in CLI")
}

func toRes(u *unstructured.Unstructured) repository.Resource {
	return repository.Resource{
		APIVersion: u.GetAPIVersion(),
		Kind:       u.GetKind(),
		Name:       u.GetName(),
		Namespace:  u.GetNamespace(),
		Raw:        u.Object,
	}
}
