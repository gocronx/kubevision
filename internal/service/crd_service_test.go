package service

import (
	"context"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/discovery"
	discoveryfake "k8s.io/client-go/discovery/fake"
	"k8s.io/client-go/kubernetes/fake"

	"go.uber.org/zap"
)

type mockDiscoveryProvider struct {
	client discovery.DiscoveryInterface
	err    error
}

func (m *mockDiscoveryProvider) DiscoveryClient(id string) (discovery.DiscoveryInterface, error) {
	return m.client, m.err
}

func TestCRDService_Discover(t *testing.T) {
	fakeClient := fake.NewSimpleClientset()
	fakeDiscovery := fakeClient.Discovery().(*discoveryfake.FakeDiscovery)

	// Add a CRD-like API group.
	fakeDiscovery.Resources = []*metav1.APIResourceList{
		{
			GroupVersion: "stable.example.com/v1",
			APIResources: []metav1.APIResource{
				{Name: "crontabs", Kind: "CronTab", Namespaced: true},
				{Name: "crontabs/status", Kind: "CronTab", Namespaced: true}, // sub-resource — should be skipped
			},
		},
		// Built-in group — should be skipped.
		{
			GroupVersion: "apps/v1",
			APIResources: []metav1.APIResource{
				{Name: "deployments", Kind: "Deployment", Namespaced: true},
			},
		},
	}

	provider := &mockDiscoveryProvider{client: fakeDiscovery}
	svc := NewCRDService(provider, zap.NewNop())

	crds, err := svc.Discover(context.Background(), "test-cluster")
	if err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	if len(crds) != 1 {
		t.Fatalf("expected 1 CRD, got %d", len(crds))
	}

	crd := crds[0]
	if crd.Group != "stable.example.com" {
		t.Errorf("expected group stable.example.com, got %s", crd.Group)
	}
	if crd.Kind != "CronTab" {
		t.Errorf("expected kind CronTab, got %s", crd.Kind)
	}
	if crd.Plural != "crontabs" {
		t.Errorf("expected plural crontabs, got %s", crd.Plural)
	}
	if !crd.Namespaced {
		t.Errorf("expected namespaced=true")
	}
}

func TestCRDService_ListCached(t *testing.T) {
	provider := &mockDiscoveryProvider{
		client: fake.NewSimpleClientset().Discovery().(*discoveryfake.FakeDiscovery),
	}
	svc := NewCRDService(provider, zap.NewNop())

	// Pre-populate cache.
	svc.mu.Lock()
	svc.cache["cached-cluster"] = []CRDInfo{
		{Group: "test.io", Version: "v1", Kind: "Foo", Plural: "foos"},
	}
	svc.mu.Unlock()

	crds, err := svc.ListCached(context.Background(), "cached-cluster")
	if err != nil {
		t.Fatalf("ListCached failed: %v", err)
	}
	if len(crds) != 1 {
		t.Fatalf("expected 1 cached CRD, got %d", len(crds))
	}
	if crds[0].Kind != "Foo" {
		t.Errorf("expected cached kind Foo, got %s", crds[0].Kind)
	}
}

func TestCRDService_InvalidateCache(t *testing.T) {
	provider := &mockDiscoveryProvider{
		client: fake.NewSimpleClientset().Discovery().(*discoveryfake.FakeDiscovery),
	}
	svc := NewCRDService(provider, zap.NewNop())

	svc.mu.Lock()
	svc.cache["test-cluster"] = []CRDInfo{{Kind: "Foo"}}
	svc.mu.Unlock()

	svc.InvalidateCache("test-cluster")

	svc.mu.RLock()
	_, ok := svc.cache["test-cluster"]
	svc.mu.RUnlock()

	if ok {
		t.Error("expected cache to be invalidated")
	}
}

func TestCRDInfo_GroupVersionResource(t *testing.T) {
	info := CRDInfo{
		Group:   "stable.example.com",
		Version: "v1",
		Plural:  "crontabs",
	}
	gvr := info.GroupVersionResource()
	if gvr.Group != "stable.example.com" {
		t.Errorf("expected group stable.example.com, got %s", gvr.Group)
	}
	if gvr.Version != "v1" {
		t.Errorf("expected version v1, got %s", gvr.Version)
	}
	if gvr.Resource != "crontabs" {
		t.Errorf("expected resource crontabs, got %s", gvr.Resource)
	}
}
