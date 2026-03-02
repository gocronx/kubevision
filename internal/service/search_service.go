package service

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/kubevision/kubevision/internal/kubernetes/cluster"
	"github.com/kubevision/kubevision/internal/kubernetes/informer"
	"github.com/kubevision/kubevision/internal/kubernetes/resource"
	bizerr "github.com/kubevision/kubevision/internal/pkg/errors"
	"github.com/kubevision/kubevision/internal/repository"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

const (
	defaultSearchLimit = 10
	maxSearchLimit     = 100

	// relevance score constants — higher is better
	scoreExact     = 3
	scorePrefix    = 2
	scoreSubstring = 1
)

// SearchResultItem is a single matched Kubernetes resource.
type SearchResultItem struct {
	Name         string            `json:"name"`
	Namespace    string            `json:"namespace,omitempty"`
	Kind         string            `json:"kind"`
	ResourceType string            `json:"resourceType"`
	APIVersion   string            `json:"apiVersion"`
	Labels       map[string]string `json:"labels,omitempty"`
	score        int               // internal only, not serialised
}

// SearchResultGroup holds matched items for one resource type.
type SearchResultGroup struct {
	ResourceType string             `json:"resource_type"`
	Items        []SearchResultItem `json:"items"`
	Total        int                `json:"total"`
}

// SearchResponse is the full payload returned by the search endpoint.
type SearchResponse struct {
	Results []SearchResultGroup `json:"results"`
	Total   int                 `json:"total"`
}

// SearchOptions carries the parsed query parameters for a search request.
type SearchOptions struct {
	// Query is the raw search string supplied by the user.
	Query string
	// Namespace restricts results to a single namespace when non-empty.
	Namespace string
	// Types is an optional allowlist of resource type names (e.g. "pods",
	// "deployments").  An empty slice means "search all types".
	Types []string
	// Limit is the maximum number of items returned per resource type.
	Limit int
	// Offset is the number of matching items to skip per resource type.
	Offset int
}

// SearchService performs global search across Kubernetes resource types for a
// given cluster.  It reads from the informer cache for cached resource types
// and falls back to the dynamic API client for uncached types.
type SearchService struct {
	informerMgr *informer.Manager
	clusterMgr  *cluster.Manager
	registry    *resource.Registry
	clusterRepo repository.ClusterRepo
}

// NewSearchService creates a new SearchService.
func NewSearchService(
	informerMgr *informer.Manager,
	clusterMgr *cluster.Manager,
	registry *resource.Registry,
	clusterRepo repository.ClusterRepo,
) *SearchService {
	return &SearchService{
		informerMgr: informerMgr,
		clusterMgr:  clusterMgr,
		registry:    registry,
		clusterRepo: clusterRepo,
	}
}

// Search runs a global search across resource types registered in the registry.
// It returns results grouped by resource type, each sorted by relevance.
func (s *SearchService) Search(
	ctx context.Context,
	clusterID uint,
	opts SearchOptions,
) (*SearchResponse, error) {
	q := strings.TrimSpace(opts.Query)
	if q == "" {
		return &SearchResponse{Results: []SearchResultGroup{}, Total: 0}, nil
	}

	// Resolve the cluster record to obtain the cluster name used as the key by
	// the cluster manager and informer manager.
	dbCluster, err := s.clusterRepo.GetByID(ctx, clusterID)
	if err != nil {
		return nil, bizerr.New(bizerr.CodeNotFound, fmt.Sprintf("cluster %d not found", clusterID))
	}
	clusterKey := dbCluster.Name

	// Normalise pagination options.
	limit := opts.Limit
	if limit <= 0 {
		limit = defaultSearchLimit
	}
	if limit > maxSearchLimit {
		limit = maxSearchLimit
	}
	offset := opts.Offset
	if offset < 0 {
		offset = 0
	}

	// Build an allowlist of resource type names when the caller supplies a filter.
	typeFilter := make(map[string]bool, len(opts.Types))
	for _, t := range opts.Types {
		if t != "" {
			typeFilter[strings.ToLower(t)] = true
		}
	}

	// Collect all registered resource metadata and sort by name for deterministic
	// output ordering regardless of map iteration order.
	allMeta := s.registry.All()
	resourceNames := make([]string, 0, len(allMeta))
	for name := range allMeta {
		resourceNames = append(resourceNames, name)
	}
	sort.Strings(resourceNames)

	var groups []SearchResultGroup
	grandTotal := 0

	for _, resName := range resourceNames {
		meta := allMeta[resName]

		// Skip resource types that are not in the caller-supplied filter.
		if len(typeFilter) > 0 && !typeFilter[strings.ToLower(resName)] {
			continue
		}

		matched, err := s.searchResource(ctx, clusterKey, meta, q, opts.Namespace)
		if err != nil {
			// A single failing resource type must not abort the whole search; the
			// cluster may not have that API group installed.
			continue
		}

		total := len(matched)
		grandTotal += total

		if total == 0 {
			continue
		}

		// Apply per-type pagination.
		start := offset
		if start > total {
			start = total
		}
		end := start + limit
		if end > total {
			end = total
		}

		groups = append(groups, SearchResultGroup{
			ResourceType: resName,
			Items:        matched[start:end],
			Total:        total,
		})
	}

	if groups == nil {
		groups = []SearchResultGroup{}
	}

	return &SearchResponse{Results: groups, Total: grandTotal}, nil
}

// searchResource collects and scores matching resources for a single type.
// For cached types it reads from the in-process informer cache; for uncached
// types it issues a live List call to the API server.
func (s *SearchService) searchResource(
	ctx context.Context,
	clusterKey string,
	meta resource.Meta,
	query string,
	namespace string,
) ([]SearchResultItem, error) {
	lowerQuery := strings.ToLower(query)

	var items []SearchResultItem
	var err error

	if meta.Cached {
		items, err = s.searchFromCache(clusterKey, meta, lowerQuery, namespace)
	} else {
		items, err = s.searchFromAPIServer(ctx, clusterKey, meta, lowerQuery, namespace)
	}
	if err != nil {
		return nil, err
	}

	sortByScore(items)
	return items, nil
}

// searchFromCache reads unstructured objects from the informer cache and
// applies text matching.
func (s *SearchService) searchFromCache(
	clusterKey string,
	meta resource.Meta,
	lowerQuery string,
	namespace string,
) ([]SearchResultItem, error) {
	objs, _, err := s.informerMgr.List(clusterKey, meta.GVR, namespace)
	if err != nil {
		return nil, err
	}

	apiVersion := buildAPIVersion(meta)
	var matched []SearchResultItem

	for i := range objs {
		u := &objs[i]
		sc := matchScore(lowerQuery, u.GetName(), u.GetNamespace(), u.GetLabels())
		if sc == 0 {
			continue
		}
		matched = append(matched, SearchResultItem{
			Name:         u.GetName(),
			Namespace:    u.GetNamespace(),
			Kind:         meta.GVK.Kind,
			ResourceType: meta.Name,
			APIVersion:   apiVersion,
			Labels:       u.GetLabels(),
			score:        sc,
		})
	}

	return matched, nil
}

// searchFromAPIServer issues a live List to the K8s API server for uncached
// resource types and applies text matching to the returned objects.
func (s *SearchService) searchFromAPIServer(
	ctx context.Context,
	clusterKey string,
	meta resource.Meta,
	lowerQuery string,
	namespace string,
) ([]SearchResultItem, error) {
	dynClient, err := s.clusterMgr.DynamicClient(clusterKey)
	if err != nil {
		return nil, fmt.Errorf("get dynamic client for cluster %s: %w", clusterKey, err)
	}

	listOpts := metav1.ListOptions{}
	var result *unstructured.UnstructuredList

	if namespace != "" {
		result, err = dynClient.Resource(meta.GVR).Namespace(namespace).List(ctx, listOpts)
	} else {
		result, err = dynClient.Resource(meta.GVR).List(ctx, listOpts)
	}
	if err != nil {
		return nil, fmt.Errorf("list %s from API server: %w", meta.Name, err)
	}

	apiVersion := buildAPIVersion(meta)
	var matched []SearchResultItem

	for i := range result.Items {
		u := &result.Items[i]
		sc := matchScore(lowerQuery, u.GetName(), u.GetNamespace(), u.GetLabels())
		if sc == 0 {
			continue
		}
		matched = append(matched, SearchResultItem{
			Name:         u.GetName(),
			Namespace:    u.GetNamespace(),
			Kind:         meta.GVK.Kind,
			ResourceType: meta.Name,
			APIVersion:   apiVersion,
			Labels:       u.GetLabels(),
			score:        sc,
		})
	}

	return matched, nil
}

// matchScore returns a relevance score (0 = no match) for a resource against
// the lower-cased query string.  Matching is evaluated against:
//   - resource name  (exact match → 3, prefix match → 2, substring → 1)
//   - resource namespace (substring → 1)
//   - label key and value strings (substring → 1)
func matchScore(lowerQuery, name, namespace string, labels map[string]string) int {
	lowerName := strings.ToLower(name)

	switch {
	case lowerName == lowerQuery:
		return scoreExact
	case strings.HasPrefix(lowerName, lowerQuery):
		return scorePrefix
	case strings.Contains(lowerName, lowerQuery):
		return scoreSubstring
	}

	if namespace != "" && strings.Contains(strings.ToLower(namespace), lowerQuery) {
		return scoreSubstring
	}

	for k, v := range labels {
		if strings.Contains(strings.ToLower(k), lowerQuery) ||
			strings.Contains(strings.ToLower(v), lowerQuery) {
			return scoreSubstring
		}
	}

	return 0
}

// sortByScore sorts items descending by score with a secondary stable sort by
// name so that output is deterministic for items of equal relevance.
func sortByScore(items []SearchResultItem) {
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].score != items[j].score {
			return items[i].score > items[j].score
		}
		return items[i].Name < items[j].Name
	})
}

// buildAPIVersion composes the apiVersion string ("group/version" or just
// "version" for core resources) from a resource.Meta.
func buildAPIVersion(meta resource.Meta) string {
	if meta.GVK.Group == "" {
		return meta.GVK.Version
	}
	return meta.GVK.Group + "/" + meta.GVK.Version
}
