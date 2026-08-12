package cluster

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/version"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const healthCheckTimeout = 5 * time.Second

// Info holds metadata for a registered cluster.
type Info struct {
	ID        string
	APIServer string
	Version   string
}

// clientSet holds the Kubernetes clients for a single cluster.
type clientSet struct {
	dynamicClient dynamic.Interface
	restConfig    *rest.Config
}

// Manager manages connections to multiple Kubernetes clusters.
type Manager struct {
	mu       sync.RWMutex
	clusters map[string]*clientSet
}

// NewManager creates a new cluster Manager.
func NewManager() *Manager {
	return &Manager{
		clusters: make(map[string]*clientSet),
	}
}

// Add registers a cluster using kubeconfig bytes. The kubeconfig data is parsed
// to create a REST config and a dynamic client.
func (m *Manager) Add(id string, kubeconfigData []byte) error {
	restCfg, err := clientcmd.RESTConfigFromKubeConfig(kubeconfigData)
	if err != nil {
		return fmt.Errorf("parse kubeconfig for cluster %s: %w", id, err)
	}

	dynClient, err := dynamic.NewForConfig(restCfg)
	if err != nil {
		return fmt.Errorf("create dynamic client for cluster %s: %w", id, err)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.clusters[id] = &clientSet{
		dynamicClient: dynClient,
		restConfig:    restCfg,
	}
	return nil
}

// AddInCluster registers a cluster using in-cluster service account credentials.
// This is used when the application is running inside a Kubernetes pod.
func (m *Manager) AddInCluster(id string) error {
	restCfg, err := rest.InClusterConfig()
	if err != nil {
		return fmt.Errorf("in-cluster config for cluster %s: %w", id, err)
	}

	dynClient, err := dynamic.NewForConfig(restCfg)
	if err != nil {
		return fmt.Errorf("create dynamic client for cluster %s: %w", id, err)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.clusters[id] = &clientSet{
		dynamicClient: dynClient,
		restConfig:    restCfg,
	}
	return nil
}

// Remove unregisters a cluster and discards its clients.
func (m *Manager) Remove(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.clusters, id)
}

// DynamicClient returns the dynamic.Interface for the specified cluster.
func (m *Manager) DynamicClient(id string) (dynamic.Interface, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	cs, ok := m.clusters[id]
	if !ok {
		return nil, fmt.Errorf("cluster %s not found", id)
	}
	return cs.dynamicClient, nil
}

// RESTConfig returns the rest.Config for the specified cluster.
func (m *Manager) RESTConfig(id string) (*rest.Config, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	cs, ok := m.clusters[id]
	if !ok {
		return nil, fmt.Errorf("cluster %s not found", id)
	}
	return cs.restConfig, nil
}

// DiscoveryClient returns a discovery.DiscoveryInterface for the specified cluster.
func (m *Manager) DiscoveryClient(id string) (discovery.DiscoveryInterface, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	cs, ok := m.clusters[id]
	if !ok {
		return nil, fmt.Errorf("cluster %s not found", id)
	}
	return discovery.NewDiscoveryClientForConfig(cs.restConfig)
}

// Probe verifies that the API server is reachable with the configured
// credentials and returns server metadata. It does not access cluster
// resources, so it is safe for read-only credentials.
func (m *Manager) Probe(ctx context.Context, id string) (*Info, error) {
	m.mu.RLock()
	cs, ok := m.clusters[id]
	if !ok {
		m.mu.RUnlock()
		return nil, fmt.Errorf("cluster %s not found", id)
	}
	restCfg := rest.CopyConfig(cs.restConfig)
	m.mu.RUnlock()

	restCfg.Timeout = healthCheckTimeout
	discoveryClient, err := discovery.NewDiscoveryClientForConfig(restCfg)
	if err != nil {
		return nil, fmt.Errorf("create discovery client for cluster %s: %w", id, err)
	}

	probeCtx, cancel := context.WithTimeout(ctx, healthCheckTimeout)
	defer cancel()
	if _, err := discoveryClient.RESTClient().Get().AbsPath("/api").DoRaw(probeCtx); err != nil {
		if apierrors.IsUnauthorized(err) {
			return nil, fmt.Errorf("authenticate to Kubernetes API server %s: %w", restCfg.Host, err)
		}
		return nil, fmt.Errorf("connect to Kubernetes API server %s: %w", restCfg.Host, err)
	}

	raw, err := discoveryClient.RESTClient().Get().AbsPath("/version").DoRaw(probeCtx)
	if err != nil {
		if apierrors.IsUnauthorized(err) {
			return nil, fmt.Errorf("authenticate to Kubernetes API server %s: %w", restCfg.Host, err)
		}
		return nil, fmt.Errorf("connect to Kubernetes API server %s: %w", restCfg.Host, err)
	}

	var serverVersion version.Info
	if err := json.Unmarshal(raw, &serverVersion); err != nil {
		return nil, fmt.Errorf("decode Kubernetes API server version: %w", err)
	}

	return &Info{
		ID:        id,
		APIServer: restCfg.Host,
		Version:   serverVersion.GitVersion,
	}, nil
}

// ListIDs returns the IDs of all registered clusters.
func (m *Manager) ListIDs() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	ids := make([]string, 0, len(m.clusters))
	for id := range m.clusters {
		ids = append(ids, id)
	}
	return ids
}
