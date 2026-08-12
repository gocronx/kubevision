package packages

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// Run explicitly against a disposable kind cluster after creating a release:
// KUBEVISION_KIND_KUBECONFIG=/path/to/kubeconfig KUBEVISION_KIND_RELEASE=name
func TestHelmAdapterKindReleaseInventory(t *testing.T) {
	kubeconfig := os.Getenv("KUBEVISION_KIND_KUBECONFIG")
	releaseName := os.Getenv("KUBEVISION_KIND_RELEASE")
	if kubeconfig == "" || releaseName == "" {
		t.Skip("kind kubeconfig and fixture release are not configured")
	}
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	require.NoError(t, err)
	adapter := NewHelmAdapter(staticConfigProvider{config: config})
	items, err := adapter.List(t.Context(), "kind", ListOptions{Namespace: "default", Limit: 20})
	require.NoError(t, err)
	require.Contains(t, releaseNames(items), releaseName)
}

type staticConfigProvider struct{ config *rest.Config }

func (p staticConfigProvider) RESTConfig(string) (*rest.Config, error) { return p.config, nil }
func releaseNames(items []Release) []string {
	names := make([]string, len(items))
	for i := range items {
		names[i] = items[i].Name
	}
	return names
}
