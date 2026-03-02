package service

import (
	"context"
	"fmt"

	bizerr "github.com/kubevision/kubevision/internal/pkg/errors"
	"github.com/kubevision/kubevision/internal/repository"
)

// QuotaResourceEntry holds the hard limits and current usage for a single
// ResourceQuota object.
type QuotaResourceEntry struct {
	Name string            `json:"name"`
	Hard map[string]string `json:"hard"`
	Used map[string]string `json:"used"`
}

// NamespaceQuotaSummary aggregates all ResourceQuota entries for one namespace.
type NamespaceQuotaSummary struct {
	Namespace string               `json:"namespace"`
	Quotas    []QuotaResourceEntry `json:"quotas"`
}

// QuotaSummaryResponse is the top-level response returned by the quota-summary
// endpoint.
type QuotaSummaryResponse struct {
	Namespaces []NamespaceQuotaSummary `json:"namespaces"`
}

// QuotaService encapsulates business logic for ResourceQuota aggregation.
type QuotaService struct {
	k8sRepo     repository.K8sResourceRepo
	clusterRepo repository.ClusterRepo
}

// NewQuotaService creates a new QuotaService.
func NewQuotaService(
	k8sRepo repository.K8sResourceRepo,
	clusterRepo repository.ClusterRepo,
) *QuotaService {
	return &QuotaService{
		k8sRepo:     k8sRepo,
		clusterRepo: clusterRepo,
	}
}

// GetQuotaSummary fetches ResourceQuota objects for the given cluster and
// (optionally) namespace, then aggregates them into a structured summary.
// When namespace is empty all namespaces are queried.
func (s *QuotaService) GetQuotaSummary(
	ctx context.Context,
	clusterID uint,
	namespace string,
) (*QuotaSummaryResponse, error) {
	// Validate cluster exists.
	cluster, err := s.clusterRepo.GetByID(ctx, clusterID)
	if err != nil {
		return nil, bizerr.New(bizerr.CodeNotFound, fmt.Sprintf("cluster %d not found", clusterID))
	}

	clusterKey := cluster.Name

	result, err := s.k8sRepo.List(ctx, clusterKey, "resourcequotas", namespace, repository.ListOptions{})
	if err != nil {
		return nil, bizerr.New(bizerr.CodeInternal, fmt.Sprintf("failed to list resourcequotas: %s", err.Error()))
	}

	// Group quotas by namespace.
	nsMap := make(map[string][]QuotaResourceEntry)
	for _, item := range result.Items {
		ns := item.Namespace
		if ns == "" {
			ns = "default"
		}

		entry := quotaEntryFromRaw(item.Name, item.Raw)
		nsMap[ns] = append(nsMap[ns], entry)
	}

	// Build ordered response. If a specific namespace was requested and has no
	// quotas, return it with an empty list so the UI can show the empty state.
	var summaries []NamespaceQuotaSummary
	if namespace != "" {
		quotas := nsMap[namespace]
		if quotas == nil {
			quotas = []QuotaResourceEntry{}
		}
		summaries = []NamespaceQuotaSummary{
			{Namespace: namespace, Quotas: quotas},
		}
	} else {
		for ns, quotas := range nsMap {
			summaries = append(summaries, NamespaceQuotaSummary{
				Namespace: ns,
				Quotas:    quotas,
			})
		}
	}

	return &QuotaSummaryResponse{Namespaces: summaries}, nil
}

// quotaEntryFromRaw converts the unstructured resource map returned by the
// generic repository into a typed QuotaResourceEntry.
func quotaEntryFromRaw(name string, raw map[string]interface{}) QuotaResourceEntry {
	entry := QuotaResourceEntry{
		Name: name,
		Hard: make(map[string]string),
		Used: make(map[string]string),
	}

	status, ok := raw["status"].(map[string]interface{})
	if !ok {
		return entry
	}

	if hard, ok := status["hard"].(map[string]interface{}); ok {
		for k, v := range hard {
			entry.Hard[k] = fmt.Sprintf("%v", v)
		}
	}

	if used, ok := status["used"].(map[string]interface{}); ok {
		for k, v := range used {
			entry.Used[k] = fmt.Sprintf("%v", v)
		}
	}

	return entry
}
