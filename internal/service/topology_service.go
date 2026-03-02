package service

import (
	"context"
	"fmt"

	bizerr "github.com/kubevision/kubevision/internal/pkg/errors"
	"github.com/kubevision/kubevision/internal/repository"
)

// TopologyNode represents a node in the resource topology graph.
type TopologyNode struct {
	ID        string            `json:"id"`
	Kind      string            `json:"kind"`
	Name      string            `json:"name"`
	Namespace string            `json:"namespace,omitempty"`
	Status    string            `json:"status,omitempty"`
	Labels    map[string]string `json:"labels,omitempty"`
}

// TopologyEdge represents a relationship between two nodes.
type TopologyEdge struct {
	Source   string `json:"source"`
	Target   string `json:"target"`
	Relation string `json:"relation"` // "owns", "selects", "exposes"
}

// TopologyResult holds the full topology graph.
type TopologyResult struct {
	Nodes []TopologyNode `json:"nodes"`
	Edges []TopologyEdge `json:"edges"`
}

// TopologyService builds resource relationship graphs.
type TopologyService struct {
	k8sRepo     repository.K8sResourceRepo
	clusterRepo repository.ClusterRepo
}

// NewTopologyService creates a new TopologyService.
func NewTopologyService(
	k8sRepo repository.K8sResourceRepo,
	clusterRepo repository.ClusterRepo,
) *TopologyService {
	return &TopologyService{
		k8sRepo:     k8sRepo,
		clusterRepo: clusterRepo,
	}
}

// GetNamespaceTopology returns the topology graph for all resources in a namespace.
func (s *TopologyService) GetNamespaceTopology(
	ctx context.Context,
	clusterID uint,
	namespace string,
) (*TopologyResult, error) {
	cluster, err := s.clusterRepo.GetByID(ctx, clusterID)
	if err != nil {
		return nil, bizerr.New(bizerr.CodeNotFound, fmt.Sprintf("cluster %d not found", clusterID))
	}
	clusterName := cluster.Name

	result := &TopologyResult{
		Nodes: []TopologyNode{},
		Edges: []TopologyEdge{},
	}

	nodeMap := make(map[string]bool)

	addNode := func(kind, name, namespace, status string, labels map[string]string) string {
		id := fmt.Sprintf("%s/%s", kind, name)
		if !nodeMap[id] {
			result.Nodes = append(result.Nodes, TopologyNode{
				ID:        id,
				Kind:      kind,
				Name:      name,
				Namespace: namespace,
				Status:    status,
				Labels:    labels,
			})
			nodeMap[id] = true
		}
		return id
	}

	// Fetch core resources
	resTypes := []string{"deployments", "statefulsets", "daemonsets", "replicasets", "pods", "services"}
	resources := make(map[string][]repository.Resource)

	for _, rt := range resTypes {
		list, err := s.k8sRepo.List(ctx, clusterName, rt, namespace, repository.ListOptions{})
		if err != nil {
			continue // Skip resources we can't fetch
		}
		resources[rt] = list.Items
	}

	// Build nodes and ownership edges from ownerReferences
	for _, rt := range resTypes {
		for _, res := range resources[rt] {
			raw := res.Raw
			if raw == nil {
				continue
			}

			meta := extractMap(raw, "metadata")
			name := extractString(meta, "name")
			ns := extractString(meta, "namespace")
			labels := extractStringMap(meta, "labels")
			status := extractResourceStatus(rt, raw)

			nodeID := addNode(rt, name, ns, status, labels)

			// Process ownerReferences
			ownerRefs := extractSlice(meta, "ownerReferences")
			for _, ownerRef := range ownerRefs {
				ref, ok := ownerRef.(map[string]interface{})
				if !ok {
					continue
				}
				ownerKind := extractString(ref, "kind")
				ownerName := extractString(ref, "name")
				if ownerKind != "" && ownerName != "" {
					// Normalize kind to plural lowercase
					ownerType := kindToResourceType(ownerKind)
					ownerID := addNode(ownerType, ownerName, ns, "", nil)
					result.Edges = append(result.Edges, TopologyEdge{
						Source:   ownerID,
						Target:   nodeID,
						Relation: "owns",
					})
				}
			}
		}
	}

	// Build service → pod selector edges
	for _, svc := range resources["services"] {
		raw := svc.Raw
		if raw == nil {
			continue
		}
		meta := extractMap(raw, "metadata")
		svcName := extractString(meta, "name")
		spec := extractMap(raw, "spec")
		selector := extractStringMap(spec, "selector")
		if len(selector) == 0 {
			continue
		}

		svcID := fmt.Sprintf("services/%s", svcName)

		// Match pods by selector
		for _, pod := range resources["pods"] {
			podRaw := pod.Raw
			if podRaw == nil {
				continue
			}
			podMeta := extractMap(podRaw, "metadata")
			podLabels := extractStringMap(podMeta, "labels")
			if matchLabels(selector, podLabels) {
				podName := extractString(podMeta, "name")
				podID := fmt.Sprintf("pods/%s", podName)
				result.Edges = append(result.Edges, TopologyEdge{
					Source:   svcID,
					Target:   podID,
					Relation: "selects",
				})
			}
		}
	}

	return result, nil
}

// Helper functions

func extractMap(obj map[string]interface{}, key string) map[string]interface{} {
	if v, ok := obj[key]; ok {
		if m, ok := v.(map[string]interface{}); ok {
			return m
		}
	}
	return nil
}

func extractString(obj map[string]interface{}, key string) string {
	if obj == nil {
		return ""
	}
	if v, ok := obj[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func extractStringMap(obj map[string]interface{}, key string) map[string]string {
	if obj == nil {
		return nil
	}
	v, ok := obj[key]
	if !ok {
		return nil
	}
	m, ok := v.(map[string]interface{})
	if !ok {
		return nil
	}
	result := make(map[string]string, len(m))
	for k, val := range m {
		if s, ok := val.(string); ok {
			result[k] = s
		}
	}
	return result
}

func extractSlice(obj map[string]interface{}, key string) []interface{} {
	if obj == nil {
		return nil
	}
	if v, ok := obj[key]; ok {
		if s, ok := v.([]interface{}); ok {
			return s
		}
	}
	return nil
}

func extractResourceStatus(resourceType string, raw map[string]interface{}) string {
	status := extractMap(raw, "status")
	if status == nil {
		return ""
	}

	switch resourceType {
	case "pods":
		return extractString(status, "phase")
	case "deployments", "statefulsets", "daemonsets":
		conditions := extractSlice(status, "conditions")
		for _, c := range conditions {
			cond, ok := c.(map[string]interface{})
			if !ok {
				continue
			}
			if extractString(cond, "type") == "Available" && extractString(cond, "status") == "True" {
				return "Available"
			}
		}
		return "Progressing"
	default:
		return ""
	}
}

func matchLabels(selector, labels map[string]string) bool {
	if len(selector) == 0 {
		return false
	}
	for k, v := range selector {
		if labels[k] != v {
			return false
		}
	}
	return true
}

func kindToResourceType(kind string) string {
	switch kind {
	case "Deployment":
		return "deployments"
	case "ReplicaSet":
		return "replicasets"
	case "StatefulSet":
		return "statefulsets"
	case "DaemonSet":
		return "daemonsets"
	case "Job":
		return "jobs"
	case "CronJob":
		return "cronjobs"
	case "Node":
		return "nodes"
	case "Service":
		return "services"
	default:
		return kind
	}
}
