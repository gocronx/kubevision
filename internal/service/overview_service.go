package service

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	bizerr "github.com/kubevision/kubevision/internal/pkg/errors"
	"github.com/kubevision/kubevision/internal/repository"
)

// ResourceMetric holds allocatable vs requested vs limited for a single resource dimension.
type ResourceMetric struct {
	Allocatable int64 `json:"allocatable"` // millicores for CPU, bytes for memory
	Requests    int64 `json:"requests"`
	Limits      int64 `json:"limits"`
}

// ResourceUsage holds CPU and Memory allocation summaries.
type ResourceUsage struct {
	CPU    ResourceMetric `json:"cpu"`
	Memory ResourceMetric `json:"memory"`
}

// EventSummary is a condensed representation of a Kubernetes event.
type EventSummary struct {
	Type       string `json:"type"`                 // Normal, Warning
	Reason     string `json:"reason"`
	Message    string `json:"message"`
	ObjectKind string `json:"objectKind"`
	ObjectName string `json:"objectName"`
	Namespace  string `json:"namespace,omitempty"`
	Timestamp  string `json:"timestamp"`
}

// OverviewResponse holds the aggregated resource counts for a cluster overview.
type OverviewResponse struct {
	Pods         int            `json:"pods"`
	RunningPods  int            `json:"runningPods"`
	Deployments  int            `json:"deployments"`
	Services     int            `json:"services"`
	Nodes        int            `json:"nodes"`
	ReadyNodes   int            `json:"readyNodes"`
	Namespaces   int            `json:"namespaces"`
	Resources    ResourceUsage  `json:"resources"`
	RecentEvents []EventSummary `json:"recentEvents"`
}

// OverviewService aggregates cluster-level resource counts.
type OverviewService struct {
	k8sRepo     repository.K8sResourceRepo
	clusterRepo repository.ClusterRepo
}

// NewOverviewService creates a new OverviewService.
func NewOverviewService(
	k8sRepo repository.K8sResourceRepo,
	clusterRepo repository.ClusterRepo,
) *OverviewService {
	return &OverviewService{
		k8sRepo:     k8sRepo,
		clusterRepo: clusterRepo,
	}
}

// GetOverview fetches counts for pods, deployments, services, nodes, and
// namespaces for the given cluster, along with resource usage metrics and
// recent events. An empty namespace string is intentional: pods, deployments,
// and services are queried across all namespaces, and nodes/namespaces are
// cluster-scoped.
func (s *OverviewService) GetOverview(
	ctx context.Context,
	clusterID uint,
) (*OverviewResponse, error) {
	cluster, err := s.clusterRepo.GetByID(ctx, clusterID)
	if err != nil {
		return nil, bizerr.New(bizerr.CodeNotFound, fmt.Sprintf("cluster %d not found", clusterID))
	}

	clusterKey := cluster.Name

	// Fetch resource lists
	resourceTypes := []string{"pods", "deployments", "services", "nodes", "namespaces"}
	lists := make(map[string]*repository.ResourceList, len(resourceTypes))

	for _, rt := range resourceTypes {
		list, err := s.k8sRepo.List(ctx, clusterKey, rt, "", repository.ListOptions{})
		if err != nil {
			return nil, bizerr.New(
				bizerr.CodeInternal,
				fmt.Sprintf("failed to list %s: %s", rt, err.Error()),
			)
		}
		lists[rt] = list
	}

	// Fetch recent events (limit to 20 for efficiency)
	eventList, err := s.k8sRepo.List(ctx, clusterKey, "events", "", repository.ListOptions{Limit: 20})
	if err != nil {
		// Events are optional — don't fail the whole overview.
		eventList = &repository.ResourceList{}
	}

	// Count running pods
	runningPods := 0
	for _, pod := range lists["pods"].Items {
		phase := getNestedString(pod.Raw, "status", "phase")
		if phase == "Running" {
			runningPods++
		}
	}

	// Count ready nodes and aggregate resource metrics
	readyNodes := 0
	var resources ResourceUsage
	for _, node := range lists["nodes"].Items {
		// Check ready status
		conditions := getNestedSlice(node.Raw, "status", "conditions")
		for _, c := range conditions {
			cond, ok := c.(map[string]any)
			if !ok {
				continue
			}
			if cond["type"] == "Ready" && cond["status"] == "True" {
				readyNodes++
				break
			}
		}
		// Aggregate allocatable resources
		allocatable := getNestedMap(node.Raw, "status", "allocatable")
		if allocatable != nil {
			if cpu, ok := allocatable["cpu"].(string); ok {
				resources.CPU.Allocatable += parseCPUQuantity(cpu)
			}
			if mem, ok := allocatable["memory"].(string); ok {
				resources.Memory.Allocatable += parseMemoryQuantity(mem)
			}
		}
	}

	// Aggregate pod resource requests and limits
	for _, pod := range lists["pods"].Items {
		containers := getNestedSlice(pod.Raw, "spec", "containers")
		for _, c := range containers {
			container, ok := c.(map[string]any)
			if !ok {
				continue
			}
			res := getNestedMap(container, "resources")
			if res == nil {
				continue
			}
			if requests, ok := res["requests"].(map[string]any); ok {
				if cpu, ok := requests["cpu"].(string); ok {
					resources.CPU.Requests += parseCPUQuantity(cpu)
				}
				if mem, ok := requests["memory"].(string); ok {
					resources.Memory.Requests += parseMemoryQuantity(mem)
				}
			}
			if limits, ok := res["limits"].(map[string]any); ok {
				if cpu, ok := limits["cpu"].(string); ok {
					resources.CPU.Limits += parseCPUQuantity(cpu)
				}
				if mem, ok := limits["memory"].(string); ok {
					resources.Memory.Limits += parseMemoryQuantity(mem)
				}
			}
		}
	}

	// Parse recent events
	recentEvents := make([]EventSummary, 0, len(eventList.Items))
	for _, ev := range eventList.Items {
		evType := getNestedString(ev.Raw, "type")
		if evType == "" {
			evType = "Normal"
		}
		reason := getNestedString(ev.Raw, "reason")
		message := getNestedString(ev.Raw, "message")

		// Truncate long messages
		if len(message) > 200 {
			message = message[:200] + "..."
		}

		objectKind := getNestedString(ev.Raw, "involvedObject", "kind")
		objectName := getNestedString(ev.Raw, "involvedObject", "name")
		namespace := getNestedString(ev.Raw, "involvedObject", "namespace")

		// Use lastTimestamp, fall back to metadata.creationTimestamp
		timestamp := getNestedString(ev.Raw, "lastTimestamp")
		if timestamp == "" {
			timestamp = getNestedString(ev.Raw, "metadata", "creationTimestamp")
		}

		recentEvents = append(recentEvents, EventSummary{
			Type:       evType,
			Reason:     reason,
			Message:    message,
			ObjectKind: objectKind,
			ObjectName: objectName,
			Namespace:  namespace,
			Timestamp:  timestamp,
		})
	}

	// Sort events by timestamp descending and take top 10
	sort.Slice(recentEvents, func(i, j int) bool {
		return recentEvents[i].Timestamp > recentEvents[j].Timestamp
	})
	if len(recentEvents) > 10 {
		recentEvents = recentEvents[:10]
	}

	return &OverviewResponse{
		Pods:         int(lists["pods"].Total),
		RunningPods:  runningPods,
		Deployments:  int(lists["deployments"].Total),
		Services:     int(lists["services"].Total),
		Nodes:        int(lists["nodes"].Total),
		ReadyNodes:   readyNodes,
		Namespaces:   int(lists["namespaces"].Total),
		Resources:    resources,
		RecentEvents: recentEvents,
	}, nil
}

// parseCPUQuantity parses a Kubernetes CPU quantity string into millicores.
// Examples: "500m" -> 500, "1" -> 1000, "2.5" -> 2500
func parseCPUQuantity(val string) int64 {
	if val == "" || val == "0" {
		return 0
	}
	if strings.HasSuffix(val, "m") {
		v, _ := strconv.ParseFloat(strings.TrimSuffix(val, "m"), 64)
		return int64(v)
	}
	v, _ := strconv.ParseFloat(val, 64)
	return int64(v * 1000)
}

// parseMemoryQuantity parses a Kubernetes memory quantity string into bytes.
func parseMemoryQuantity(val string) int64 {
	if val == "" || val == "0" {
		return 0
	}
	suffixes := []struct {
		suffix     string
		multiplier int64
	}{
		{"Ti", 1024 * 1024 * 1024 * 1024},
		{"Gi", 1024 * 1024 * 1024},
		{"Mi", 1024 * 1024},
		{"Ki", 1024},
		{"T", 1000 * 1000 * 1000 * 1000},
		{"G", 1000 * 1000 * 1000},
		{"M", 1000 * 1000},
		{"K", 1000},
	}
	for _, s := range suffixes {
		if strings.HasSuffix(val, s.suffix) {
			v, _ := strconv.ParseFloat(strings.TrimSuffix(val, s.suffix), 64)
			return int64(float64(s.multiplier) * v)
		}
	}
	v, _ := strconv.ParseFloat(val, 64)
	return int64(v)
}

// getNestedString safely extracts a string value from a nested map path.
func getNestedString(obj map[string]any, keys ...string) string {
	current := obj
	for i, key := range keys {
		v, ok := current[key]
		if !ok {
			return ""
		}
		if i == len(keys)-1 {
			s, _ := v.(string)
			return s
		}
		next, ok := v.(map[string]any)
		if !ok {
			return ""
		}
		current = next
	}
	return ""
}

// getNestedSlice safely extracts a slice from a nested map path.
func getNestedSlice(obj map[string]any, keys ...string) []any {
	current := obj
	for i, key := range keys {
		v, ok := current[key]
		if !ok {
			return nil
		}
		if i == len(keys)-1 {
			s, _ := v.([]any)
			return s
		}
		next, ok := v.(map[string]any)
		if !ok {
			return nil
		}
		current = next
	}
	return nil
}

// getNestedMap safely extracts a map from a nested path.
func getNestedMap(obj map[string]any, keys ...string) map[string]any {
	current := obj
	for _, key := range keys {
		v, ok := current[key]
		if !ok {
			return nil
		}
		next, ok := v.(map[string]any)
		if !ok {
			return nil
		}
		current = next
	}
	return current
}
