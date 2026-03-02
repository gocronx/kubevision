package resource

import (
	"testing"

	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestNewRegistry_ReturnsNonNil(t *testing.T) {
	r := NewRegistry()
	if r == nil {
		t.Fatal("NewRegistry() returned nil")
	}
}

func TestNewRegistry_HasBuiltinResources(t *testing.T) {
	r := NewRegistry()
	all := r.All()
	if len(all) == 0 {
		t.Fatal("NewRegistry() should have pre-populated builtin resources, got 0")
	}
}

func TestRegistry_Get_KnownResources(t *testing.T) {
	r := NewRegistry()

	tests := []struct {
		name       string
		wantGVR    schema.GroupVersionResource
		wantGVK    schema.GroupVersionKind
		wantScope  Scope
		wantCached bool
	}{
		{
			name:       "pods",
			wantGVR:    schema.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"},
			wantGVK:    schema.GroupVersionKind{Group: "", Version: "v1", Kind: "Pod"},
			wantScope:  NamespaceScoped,
			wantCached: true,
		},
		{
			name:       "deployments",
			wantGVR:    schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"},
			wantGVK:    schema.GroupVersionKind{Group: "apps", Version: "v1", Kind: "Deployment"},
			wantScope:  NamespaceScoped,
			wantCached: true,
		},
		{
			name:       "statefulsets",
			wantGVR:    schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "statefulsets"},
			wantGVK:    schema.GroupVersionKind{Group: "apps", Version: "v1", Kind: "StatefulSet"},
			wantScope:  NamespaceScoped,
			wantCached: true,
		},
		{
			name:       "daemonsets",
			wantGVR:    schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "daemonsets"},
			wantGVK:    schema.GroupVersionKind{Group: "apps", Version: "v1", Kind: "DaemonSet"},
			wantScope:  NamespaceScoped,
			wantCached: true,
		},
		{
			name:       "replicasets",
			wantGVR:    schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "replicasets"},
			wantGVK:    schema.GroupVersionKind{Group: "apps", Version: "v1", Kind: "ReplicaSet"},
			wantScope:  NamespaceScoped,
			wantCached: true,
		},
		{
			name:       "services",
			wantGVR:    schema.GroupVersionResource{Group: "", Version: "v1", Resource: "services"},
			wantGVK:    schema.GroupVersionKind{Group: "", Version: "v1", Kind: "Service"},
			wantScope:  NamespaceScoped,
			wantCached: true,
		},
		{
			name:       "nodes",
			wantGVR:    schema.GroupVersionResource{Group: "", Version: "v1", Resource: "nodes"},
			wantGVK:    schema.GroupVersionKind{Group: "", Version: "v1", Kind: "Node"},
			wantScope:  ClusterScoped,
			wantCached: true,
		},
		{
			name:       "namespaces",
			wantGVR:    schema.GroupVersionResource{Group: "", Version: "v1", Resource: "namespaces"},
			wantGVK:    schema.GroupVersionKind{Group: "", Version: "v1", Kind: "Namespace"},
			wantScope:  ClusterScoped,
			wantCached: true,
		},
		{
			name:       "jobs",
			wantGVR:    schema.GroupVersionResource{Group: "batch", Version: "v1", Resource: "jobs"},
			wantGVK:    schema.GroupVersionKind{Group: "batch", Version: "v1", Kind: "Job"},
			wantScope:  NamespaceScoped,
			wantCached: false,
		},
		{
			name:       "cronjobs",
			wantGVR:    schema.GroupVersionResource{Group: "batch", Version: "v1", Resource: "cronjobs"},
			wantGVK:    schema.GroupVersionKind{Group: "batch", Version: "v1", Kind: "CronJob"},
			wantScope:  NamespaceScoped,
			wantCached: false,
		},
		{
			name:       "ingresses",
			wantGVR:    schema.GroupVersionResource{Group: "networking.k8s.io", Version: "v1", Resource: "ingresses"},
			wantGVK:    schema.GroupVersionKind{Group: "networking.k8s.io", Version: "v1", Kind: "Ingress"},
			wantScope:  NamespaceScoped,
			wantCached: false,
		},
		{
			name:       "configmaps",
			wantGVR:    schema.GroupVersionResource{Group: "", Version: "v1", Resource: "configmaps"},
			wantGVK:    schema.GroupVersionKind{Group: "", Version: "v1", Kind: "ConfigMap"},
			wantScope:  NamespaceScoped,
			wantCached: false,
		},
		{
			name:       "secrets",
			wantGVR:    schema.GroupVersionResource{Group: "", Version: "v1", Resource: "secrets"},
			wantGVK:    schema.GroupVersionKind{Group: "", Version: "v1", Kind: "Secret"},
			wantScope:  NamespaceScoped,
			wantCached: false,
		},
		{
			name:       "persistentvolumes",
			wantGVR:    schema.GroupVersionResource{Group: "", Version: "v1", Resource: "persistentvolumes"},
			wantGVK:    schema.GroupVersionKind{Group: "", Version: "v1", Kind: "PersistentVolume"},
			wantScope:  ClusterScoped,
			wantCached: false,
		},
		{
			name:       "clusterroles",
			wantGVR:    schema.GroupVersionResource{Group: "rbac.authorization.k8s.io", Version: "v1", Resource: "clusterroles"},
			wantGVK:    schema.GroupVersionKind{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "ClusterRole"},
			wantScope:  ClusterScoped,
			wantCached: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			meta, ok := r.Get(tc.name)
			if !ok {
				t.Fatalf("Get(%q) returned ok=false, expected the resource to be registered", tc.name)
			}

			if meta.Name != tc.name {
				t.Errorf("Name = %q, want %q", meta.Name, tc.name)
			}
			if meta.GVR != tc.wantGVR {
				t.Errorf("GVR = %v, want %v", meta.GVR, tc.wantGVR)
			}
			if meta.GVK != tc.wantGVK {
				t.Errorf("GVK = %v, want %v", meta.GVK, tc.wantGVK)
			}
			if meta.Scope != tc.wantScope {
				t.Errorf("Scope = %q, want %q", meta.Scope, tc.wantScope)
			}
			if meta.Cached != tc.wantCached {
				t.Errorf("Cached = %v, want %v", meta.Cached, tc.wantCached)
			}
		})
	}
}

func TestRegistry_Get_UnknownResource(t *testing.T) {
	r := NewRegistry()

	unknownNames := []string{
		"nonexistent",
		"foobar",
		"",
		"Pods",  // case-sensitive: "Pods" != "pods"
		"NODES", // case-sensitive
	}

	for _, name := range unknownNames {
		t.Run("unknown_"+name, func(t *testing.T) {
			_, ok := r.Get(name)
			if ok {
				t.Errorf("Get(%q) returned ok=true, expected ok=false for unknown resource", name)
			}
		})
	}
}

func TestRegistry_CachedGVRs(t *testing.T) {
	r := NewRegistry()
	cachedGVRs := r.CachedGVRs()

	if len(cachedGVRs) == 0 {
		t.Fatal("CachedGVRs() returned empty slice, expected cached resources")
	}

	// Build a set of all returned GVRs for lookup.
	gvrSet := make(map[schema.GroupVersionResource]bool, len(cachedGVRs))
	for _, gvr := range cachedGVRs {
		gvrSet[gvr] = true
	}

	// Verify that all resources marked as Cached appear in the result.
	all := r.All()
	expectedCachedCount := 0
	for name, meta := range all {
		if meta.Cached {
			expectedCachedCount++
			if !gvrSet[meta.GVR] {
				t.Errorf("resource %q is marked Cached=true but its GVR %v was not returned by CachedGVRs()", name, meta.GVR)
			}
		}
	}

	if len(cachedGVRs) != expectedCachedCount {
		t.Errorf("CachedGVRs() returned %d items, but %d resources are marked Cached=true", len(cachedGVRs), expectedCachedCount)
	}

	// Verify that no uncached resource GVRs appear in the result.
	for name, meta := range all {
		if !meta.Cached && gvrSet[meta.GVR] {
			t.Errorf("resource %q is marked Cached=false but its GVR %v appeared in CachedGVRs()", name, meta.GVR)
		}
	}
}

func TestRegistry_All(t *testing.T) {
	r := NewRegistry()
	all := r.All()

	// The registry has a known set of builtin resources. Verify we have a reasonable count.
	// Based on the source code, there are 8 cached + 19 uncached = 27 total resources.
	expectedCount := 27
	if len(all) != expectedCount {
		t.Errorf("All() returned %d resources, expected %d", len(all), expectedCount)
	}

	// Verify each entry has consistent Name field.
	for name, meta := range all {
		if meta.Name != name {
			t.Errorf("resource map key %q has Name field %q, expected them to match", name, meta.Name)
		}
		if meta.GVR.Resource == "" {
			t.Errorf("resource %q has empty GVR.Resource", name)
		}
		if meta.GVK.Kind == "" {
			t.Errorf("resource %q has empty GVK.Kind", name)
		}
		if meta.Scope != NamespaceScoped && meta.Scope != ClusterScoped {
			t.Errorf("resource %q has invalid Scope %q", name, meta.Scope)
		}
	}
}

func TestRegistry_All_ReturnsCopy(t *testing.T) {
	r := NewRegistry()
	all1 := r.All()
	all2 := r.All()

	// Modifying one copy should not affect the other.
	all1["test-mutation"] = Meta{Name: "test-mutation"}

	if _, exists := all2["test-mutation"]; exists {
		t.Error("All() did not return a copy; modifying one map affected another")
	}

	// Also verify the original registry is not affected.
	if _, ok := r.Get("test-mutation"); ok {
		t.Error("All() did not return a copy; modifying the returned map affected the registry")
	}
}

func TestRegistry_CachedGVRs_ContainsExpectedResources(t *testing.T) {
	r := NewRegistry()
	cachedGVRs := r.CachedGVRs()

	// Build lookup set.
	gvrSet := make(map[schema.GroupVersionResource]bool, len(cachedGVRs))
	for _, gvr := range cachedGVRs {
		gvrSet[gvr] = true
	}

	expectedCached := []struct {
		name string
		gvr  schema.GroupVersionResource
	}{
		{"pods", schema.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"}},
		{"deployments", schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}},
		{"services", schema.GroupVersionResource{Group: "", Version: "v1", Resource: "services"}},
		{"nodes", schema.GroupVersionResource{Group: "", Version: "v1", Resource: "nodes"}},
		{"namespaces", schema.GroupVersionResource{Group: "", Version: "v1", Resource: "namespaces"}},
	}

	for _, ec := range expectedCached {
		if !gvrSet[ec.gvr] {
			t.Errorf("expected %q (GVR %v) to be in CachedGVRs(), but it was missing", ec.name, ec.gvr)
		}
	}
}
