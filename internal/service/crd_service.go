package service

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/kubevision/kubevision/internal/repository"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/discovery"
)

// CRDInfo describes a discovered Custom Resource Definition.
type CRDInfo struct {
	Group      string `json:"group"`
	Version    string `json:"version"`
	Kind       string `json:"kind"`
	Plural     string `json:"plural"`
	Namespaced bool   `json:"namespaced"`
	Categories []string `json:"categories,omitempty"`
}

// DiscoveryClientProvider is satisfied by cluster.Manager.
type DiscoveryClientProvider interface {
	DiscoveryClient(id string) (discovery.DiscoveryInterface, error)
}

// CRDService discovers custom resource definitions in clusters.
type CRDService struct {
	provider    DiscoveryClientProvider
	clusterRepo repository.ClusterRepo
	logger      *zap.Logger

	mu    sync.RWMutex
	cache map[string][]CRDInfo // clusterID -> CRDs
}

// NewCRDService creates a new CRDService.
func NewCRDService(provider DiscoveryClientProvider, clusterRepo repository.ClusterRepo, logger *zap.Logger) *CRDService {
	return &CRDService{
		provider:    provider,
		clusterRepo: clusterRepo,
		logger:      logger,
		cache:       make(map[string][]CRDInfo),
	}
}

// resolveClusterName converts a numeric DB ID to the cluster name used by the cluster manager.
func (s *CRDService) resolveClusterName(ctx context.Context, idStr string) (string, error) {
	// Try as numeric DB ID first.
	var id uint
	if _, err := fmt.Sscanf(idStr, "%d", &id); err == nil && s.clusterRepo != nil {
		cluster, err := s.clusterRepo.GetByID(ctx, id)
		if err == nil {
			return cluster.Name, nil
		}
	}
	// Fall back to treating it as the cluster name directly.
	return idStr, nil
}

// builtinGroups contains core Kubernetes API groups that are NOT CRDs.
var builtinGroups = map[string]bool{
	"":                             true,
	"apps":                         true,
	"batch":                        true,
	"autoscaling":                  true,
	"networking.k8s.io":            true,
	"storage.k8s.io":               true,
	"policy":                       true,
	"rbac.authorization.k8s.io":    true,
	"admissionregistration.k8s.io": true,
	"apiregistration.k8s.io":       true,
	"authentication.k8s.io":        true,
	"authorization.k8s.io":         true,
	"certificates.k8s.io":          true,
	"coordination.k8s.io":          true,
	"discovery.k8s.io":             true,
	"events.k8s.io":                true,
	"flowcontrol.apiserver.k8s.io": true,
	"node.k8s.io":                  true,
	"scheduling.k8s.io":            true,
	"apiextensions.k8s.io":         true,
	"metrics.k8s.io":               true,
}

// Discover fetches CRDs from the specified cluster's API server.
// clusterID can be a numeric DB ID or a cluster name.
func (s *CRDService) Discover(ctx context.Context, clusterID string) ([]CRDInfo, error) {
	name, err := s.resolveClusterName(ctx, clusterID)
	if err != nil {
		return nil, fmt.Errorf("resolve cluster: %w", err)
	}
	disco, err := s.provider.DiscoveryClient(name)
	if err != nil {
		return nil, fmt.Errorf("get discovery client: %w", err)
	}

	_, apiResourceLists, err := disco.ServerGroupsAndResources()
	if err != nil {
		// ServerGroupsAndResources may return partial results with non-nil error
		if apiResourceLists == nil {
			return nil, fmt.Errorf("discover API resources: %w", err)
		}
		s.logger.Warn("partial discovery error", zap.String("cluster", clusterID), zap.Error(err))
	}

	var crds []CRDInfo
	for _, list := range apiResourceLists {
		gv := list.GroupVersion
		parts := strings.SplitN(gv, "/", 2)
		group := ""
		version := ""
		if len(parts) == 2 {
			group = parts[0]
			version = parts[1]
		} else {
			version = parts[0]
		}

		if builtinGroups[group] {
			continue
		}

		for _, r := range list.APIResources {
			// Skip sub-resources (e.g. pods/status)
			if strings.Contains(r.Name, "/") {
				continue
			}
			crds = append(crds, CRDInfo{
				Group:      group,
				Version:    version,
				Kind:       r.Kind,
				Plural:     r.Name,
				Namespaced: r.Namespaced,
				Categories: r.Categories,
			})
		}
	}

	sort.Slice(crds, func(i, j int) bool {
		if crds[i].Group != crds[j].Group {
			return crds[i].Group < crds[j].Group
		}
		return crds[i].Kind < crds[j].Kind
	})

	// Update cache
	s.mu.Lock()
	s.cache[clusterID] = crds
	s.mu.Unlock()

	return crds, nil
}

// ListCached returns cached CRDs for a cluster, or discovers them if not cached.
func (s *CRDService) ListCached(ctx context.Context, clusterID string) ([]CRDInfo, error) {
	s.mu.RLock()
	cached, ok := s.cache[clusterID]
	s.mu.RUnlock()

	if ok {
		return cached, nil
	}
	return s.Discover(ctx, clusterID)
}

// StartPeriodicDiscovery runs background CRD discovery for all known clusters.
func (s *CRDService) StartPeriodicDiscovery(ctx context.Context, clusterIDs func() []string, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Initial discovery
	for _, id := range clusterIDs() {
		if _, err := s.Discover(ctx, id); err != nil {
			s.logger.Warn("initial CRD discovery failed", zap.String("cluster", id), zap.Error(err))
		}
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			for _, id := range clusterIDs() {
				if _, err := s.Discover(context.Background(), id); err != nil {
					s.logger.Warn("periodic CRD discovery failed", zap.String("cluster", id), zap.Error(err))
				}
			}
		}
	}
}

// InvalidateCache removes cached data for a cluster.
func (s *CRDService) InvalidateCache(clusterID string) {
	s.mu.Lock()
	delete(s.cache, clusterID)
	s.mu.Unlock()
}

// ServerGroupResource returns the GVR string usable in API calls for the given CRD info.
func (c *CRDInfo) GroupVersionResource() metav1.GroupVersionResource {
	return metav1.GroupVersionResource{
		Group:    c.Group,
		Version:  c.Version,
		Resource: c.Plural,
	}
}
