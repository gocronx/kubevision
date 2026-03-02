package resource

import (
	"sync"

	"k8s.io/apimachinery/pkg/runtime/schema"
)

// Scope indicates whether a resource is namespace-scoped or cluster-scoped.
type Scope string

const (
	NamespaceScoped Scope = "namespace"
	ClusterScoped   Scope = "cluster"
)

// Meta holds metadata about a Kubernetes resource type.
type Meta struct {
	Name   string
	GVR    schema.GroupVersionResource
	GVK    schema.GroupVersionKind
	Scope  Scope
	Cached bool // whether the resource is watched via Informer cache
}

// Registry tracks known Kubernetes resource types and provides lookup
// capabilities for dynamic resource operations.
type Registry struct {
	mu        sync.RWMutex
	resources map[string]Meta
}

// NewRegistry creates a new Registry pre-populated with built-in resources.
func NewRegistry() *Registry {
	r := &Registry{
		resources: make(map[string]Meta),
	}
	r.registerBuiltins()
	return r
}

// Get returns the resource metadata for the given resource name.
func (r *Registry) Get(name string) (Meta, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	m, ok := r.resources[name]
	return m, ok
}

// All returns a copy of all registered resource metadata.
func (r *Registry) All() map[string]Meta {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make(map[string]Meta, len(r.resources))
	for k, v := range r.resources {
		result[k] = v
	}
	return result
}

// CachedGVRs returns the list of GVRs for resources that should be watched
// via the Informer cache.
func (r *Registry) CachedGVRs() []schema.GroupVersionResource {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var gvrs []schema.GroupVersionResource
	for _, m := range r.resources {
		if m.Cached {
			gvrs = append(gvrs, m.GVR)
		}
	}
	return gvrs
}

// registerBuiltins populates the registry with all known Kubernetes resource types.
// Resources are divided into two tiers:
//   - Cached: frequently accessed resources watched via shared informers
//   - Uncached: less frequently accessed resources fetched directly from the API server
func (r *Registry) registerBuiltins() {
	// ---- Cached resources (watched via Informer) ----

	r.resources["pods"] = Meta{
		Name:   "pods",
		GVR:    schema.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"},
		GVK:    schema.GroupVersionKind{Group: "", Version: "v1", Kind: "Pod"},
		Scope:  NamespaceScoped,
		Cached: true,
	}
	r.resources["deployments"] = Meta{
		Name:   "deployments",
		GVR:    schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"},
		GVK:    schema.GroupVersionKind{Group: "apps", Version: "v1", Kind: "Deployment"},
		Scope:  NamespaceScoped,
		Cached: true,
	}
	r.resources["statefulsets"] = Meta{
		Name:   "statefulsets",
		GVR:    schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "statefulsets"},
		GVK:    schema.GroupVersionKind{Group: "apps", Version: "v1", Kind: "StatefulSet"},
		Scope:  NamespaceScoped,
		Cached: true,
	}
	r.resources["daemonsets"] = Meta{
		Name:   "daemonsets",
		GVR:    schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "daemonsets"},
		GVK:    schema.GroupVersionKind{Group: "apps", Version: "v1", Kind: "DaemonSet"},
		Scope:  NamespaceScoped,
		Cached: true,
	}
	r.resources["replicasets"] = Meta{
		Name:   "replicasets",
		GVR:    schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "replicasets"},
		GVK:    schema.GroupVersionKind{Group: "apps", Version: "v1", Kind: "ReplicaSet"},
		Scope:  NamespaceScoped,
		Cached: true,
	}
	r.resources["services"] = Meta{
		Name:   "services",
		GVR:    schema.GroupVersionResource{Group: "", Version: "v1", Resource: "services"},
		GVK:    schema.GroupVersionKind{Group: "", Version: "v1", Kind: "Service"},
		Scope:  NamespaceScoped,
		Cached: true,
	}
	r.resources["nodes"] = Meta{
		Name:   "nodes",
		GVR:    schema.GroupVersionResource{Group: "", Version: "v1", Resource: "nodes"},
		GVK:    schema.GroupVersionKind{Group: "", Version: "v1", Kind: "Node"},
		Scope:  ClusterScoped,
		Cached: true,
	}
	r.resources["namespaces"] = Meta{
		Name:   "namespaces",
		GVR:    schema.GroupVersionResource{Group: "", Version: "v1", Resource: "namespaces"},
		GVK:    schema.GroupVersionKind{Group: "", Version: "v1", Kind: "Namespace"},
		Scope:  ClusterScoped,
		Cached: true,
	}

	// ---- Uncached resources (fetched directly from API server) ----

	r.resources["jobs"] = Meta{
		Name:   "jobs",
		GVR:    schema.GroupVersionResource{Group: "batch", Version: "v1", Resource: "jobs"},
		GVK:    schema.GroupVersionKind{Group: "batch", Version: "v1", Kind: "Job"},
		Scope:  NamespaceScoped,
		Cached: false,
	}
	r.resources["cronjobs"] = Meta{
		Name:   "cronjobs",
		GVR:    schema.GroupVersionResource{Group: "batch", Version: "v1", Resource: "cronjobs"},
		GVK:    schema.GroupVersionKind{Group: "batch", Version: "v1", Kind: "CronJob"},
		Scope:  NamespaceScoped,
		Cached: false,
	}
	r.resources["ingresses"] = Meta{
		Name:   "ingresses",
		GVR:    schema.GroupVersionResource{Group: "networking.k8s.io", Version: "v1", Resource: "ingresses"},
		GVK:    schema.GroupVersionKind{Group: "networking.k8s.io", Version: "v1", Kind: "Ingress"},
		Scope:  NamespaceScoped,
		Cached: false,
	}
	r.resources["configmaps"] = Meta{
		Name:   "configmaps",
		GVR:    schema.GroupVersionResource{Group: "", Version: "v1", Resource: "configmaps"},
		GVK:    schema.GroupVersionKind{Group: "", Version: "v1", Kind: "ConfigMap"},
		Scope:  NamespaceScoped,
		Cached: false,
	}
	r.resources["secrets"] = Meta{
		Name:   "secrets",
		GVR:    schema.GroupVersionResource{Group: "", Version: "v1", Resource: "secrets"},
		GVK:    schema.GroupVersionKind{Group: "", Version: "v1", Kind: "Secret"},
		Scope:  NamespaceScoped,
		Cached: false,
	}
	r.resources["events"] = Meta{
		Name:   "events",
		GVR:    schema.GroupVersionResource{Group: "", Version: "v1", Resource: "events"},
		GVK:    schema.GroupVersionKind{Group: "", Version: "v1", Kind: "Event"},
		Scope:  NamespaceScoped,
		Cached: false,
	}
	r.resources["persistentvolumes"] = Meta{
		Name:   "persistentvolumes",
		GVR:    schema.GroupVersionResource{Group: "", Version: "v1", Resource: "persistentvolumes"},
		GVK:    schema.GroupVersionKind{Group: "", Version: "v1", Kind: "PersistentVolume"},
		Scope:  ClusterScoped,
		Cached: false,
	}
	r.resources["persistentvolumeclaims"] = Meta{
		Name:   "persistentvolumeclaims",
		GVR:    schema.GroupVersionResource{Group: "", Version: "v1", Resource: "persistentvolumeclaims"},
		GVK:    schema.GroupVersionKind{Group: "", Version: "v1", Kind: "PersistentVolumeClaim"},
		Scope:  NamespaceScoped,
		Cached: false,
	}
	r.resources["storageclasses"] = Meta{
		Name:   "storageclasses",
		GVR:    schema.GroupVersionResource{Group: "storage.k8s.io", Version: "v1", Resource: "storageclasses"},
		GVK:    schema.GroupVersionKind{Group: "storage.k8s.io", Version: "v1", Kind: "StorageClass"},
		Scope:  ClusterScoped,
		Cached: false,
	}
	r.resources["serviceaccounts"] = Meta{
		Name:   "serviceaccounts",
		GVR:    schema.GroupVersionResource{Group: "", Version: "v1", Resource: "serviceaccounts"},
		GVK:    schema.GroupVersionKind{Group: "", Version: "v1", Kind: "ServiceAccount"},
		Scope:  NamespaceScoped,
		Cached: false,
	}
	r.resources["roles"] = Meta{
		Name:   "roles",
		GVR:    schema.GroupVersionResource{Group: "rbac.authorization.k8s.io", Version: "v1", Resource: "roles"},
		GVK:    schema.GroupVersionKind{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "Role"},
		Scope:  NamespaceScoped,
		Cached: false,
	}
	r.resources["clusterroles"] = Meta{
		Name:   "clusterroles",
		GVR:    schema.GroupVersionResource{Group: "rbac.authorization.k8s.io", Version: "v1", Resource: "clusterroles"},
		GVK:    schema.GroupVersionKind{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "ClusterRole"},
		Scope:  ClusterScoped,
		Cached: false,
	}
	r.resources["rolebindings"] = Meta{
		Name:   "rolebindings",
		GVR:    schema.GroupVersionResource{Group: "rbac.authorization.k8s.io", Version: "v1", Resource: "rolebindings"},
		GVK:    schema.GroupVersionKind{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "RoleBinding"},
		Scope:  NamespaceScoped,
		Cached: false,
	}
	r.resources["clusterrolebindings"] = Meta{
		Name:   "clusterrolebindings",
		GVR:    schema.GroupVersionResource{Group: "rbac.authorization.k8s.io", Version: "v1", Resource: "clusterrolebindings"},
		GVK:    schema.GroupVersionKind{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "ClusterRoleBinding"},
		Scope:  ClusterScoped,
		Cached: false,
	}
	r.resources["horizontalpodautoscalers"] = Meta{
		Name:   "horizontalpodautoscalers",
		GVR:    schema.GroupVersionResource{Group: "autoscaling", Version: "v2", Resource: "horizontalpodautoscalers"},
		GVK:    schema.GroupVersionKind{Group: "autoscaling", Version: "v2", Kind: "HorizontalPodAutoscaler"},
		Scope:  NamespaceScoped,
		Cached: false,
	}
	r.resources["endpoints"] = Meta{
		Name:   "endpoints",
		GVR:    schema.GroupVersionResource{Group: "", Version: "v1", Resource: "endpoints"},
		GVK:    schema.GroupVersionKind{Group: "", Version: "v1", Kind: "Endpoints"},
		Scope:  NamespaceScoped,
		Cached: false,
	}
	r.resources["networkpolicies"] = Meta{
		Name:   "networkpolicies",
		GVR:    schema.GroupVersionResource{Group: "networking.k8s.io", Version: "v1", Resource: "networkpolicies"},
		GVK:    schema.GroupVersionKind{Group: "networking.k8s.io", Version: "v1", Kind: "NetworkPolicy"},
		Scope:  NamespaceScoped,
		Cached: false,
	}
}
