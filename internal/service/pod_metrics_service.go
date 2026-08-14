package service

import (
	"context"
	"fmt"
	"time"

	"github.com/gocronx/kubevision/internal/repository"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

var podMetricsGVR = schema.GroupVersionResource{Group: "metrics.k8s.io", Version: "v1beta1", Resource: "pods"}

const podMetricsTimeout = 5 * time.Second

type dynamicClientProvider interface {
	DynamicClient(string) (dynamic.Interface, error)
}

// PodMetricsService reads current Pod usage without requiring node proxy access.
type PodMetricsService struct {
	clusters repository.ClusterRepo
	clients  dynamicClientProvider
}

func NewPodMetricsService(clusters repository.ClusterRepo, clients dynamicClientProvider) *PodMetricsService {
	return &PodMetricsService{clusters: clusters, clients: clients}
}

func (s *PodMetricsService) List(ctx context.Context, clusterID uint, namespace string) (map[string]*repository.PodMetrics, error) {
	metricsCtx, cancel := context.WithTimeout(ctx, podMetricsTimeout)
	defer cancel()

	cluster, err := s.clusters.GetByID(metricsCtx, clusterID)
	if err != nil {
		return nil, fmt.Errorf("get cluster %d: %w", clusterID, err)
	}
	client, err := s.clients.DynamicClient(clusterKey(cluster))
	if err != nil {
		return nil, fmt.Errorf("get metrics client: %w", err)
	}

	var listItems []map[string]interface{}
	if namespace == "" {
		list, listErr := client.Resource(podMetricsGVR).List(metricsCtx, metav1.ListOptions{})
		if listErr != nil {
			return nil, fmt.Errorf("list pod metrics: %w", listErr)
		}
		listItems = make([]map[string]interface{}, 0, len(list.Items))
		for i := range list.Items {
			listItems = append(listItems, list.Items[i].Object)
		}
	} else {
		list, listErr := client.Resource(podMetricsGVR).Namespace(namespace).List(metricsCtx, metav1.ListOptions{})
		if listErr != nil {
			return nil, fmt.Errorf("list pod metrics: %w", listErr)
		}
		listItems = make([]map[string]interface{}, 0, len(list.Items))
		for i := range list.Items {
			listItems = append(listItems, list.Items[i].Object)
		}
	}

	result := make(map[string]*repository.PodMetrics, len(listItems))
	for _, item := range listItems {
		metrics, name, ns, parseErr := parsePodMetrics(item)
		if parseErr != nil {
			continue
		}
		result[podMetricsKey(ns, name)] = metrics
	}
	return result, nil
}

func parsePodMetrics(obj map[string]interface{}) (*repository.PodMetrics, string, string, error) {
	metadata, _ := obj["metadata"].(map[string]interface{})
	name, _ := metadata["name"].(string)
	namespace, _ := metadata["namespace"].(string)
	if name == "" {
		return nil, "", "", fmt.Errorf("pod metric has no name")
	}

	metrics := &repository.PodMetrics{Window: stringValue(obj["window"])}
	if rawTimestamp := stringValue(obj["timestamp"]); rawTimestamp != "" {
		metrics.Timestamp, _ = time.Parse(time.RFC3339, rawTimestamp)
	}
	containers, _ := obj["containers"].([]interface{})
	metrics.Containers = make([]repository.ContainerMetrics, 0, len(containers))
	for _, raw := range containers {
		container, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		usage, _ := container["usage"].(map[string]interface{})
		entry := repository.ContainerMetrics{
			Name:        stringValue(container["name"]),
			CPUMilli:    cpuMilli(stringValue(usage["cpu"])),
			MemoryBytes: quantityValue(stringValue(usage["memory"])),
		}
		metrics.CPUMilli += entry.CPUMilli
		metrics.MemoryBytes += entry.MemoryBytes
		metrics.Containers = append(metrics.Containers, entry)
	}
	return metrics, name, namespace, nil
}

// ApplyPodResourceAllocations adds requests and limits from the Pod spec.
func ApplyPodResourceAllocations(metrics *repository.PodMetrics, raw map[string]interface{}) {
	if metrics == nil {
		return
	}
	spec, _ := raw["spec"].(map[string]interface{})
	containers, _ := spec["containers"].([]interface{})
	byName := make(map[string]map[string]interface{}, len(containers))
	for _, rawContainer := range containers {
		container, ok := rawContainer.(map[string]interface{})
		if ok {
			byName[stringValue(container["name"])] = container
		}
	}
	for i := range metrics.Containers {
		container := byName[metrics.Containers[i].Name]
		resources, _ := container["resources"].(map[string]interface{})
		requests, _ := resources["requests"].(map[string]interface{})
		limits, _ := resources["limits"].(map[string]interface{})
		entry := &metrics.Containers[i]
		entry.CPURequestMilli = cpuMilli(stringValue(requests["cpu"]))
		entry.CPULimitMilli = cpuMilli(stringValue(limits["cpu"]))
		entry.MemoryRequestBytes = quantityValue(stringValue(requests["memory"]))
		entry.MemoryLimitBytes = quantityValue(stringValue(limits["memory"]))
		metrics.CPURequestMilli += entry.CPURequestMilli
		metrics.CPULimitMilli += entry.CPULimitMilli
		metrics.MemoryRequestBytes += entry.MemoryRequestBytes
		metrics.MemoryLimitBytes += entry.MemoryLimitBytes
	}
}

func podMetricsKey(namespace, name string) string { return namespace + "/" + name }

func stringValue(value interface{}) string {
	text, _ := value.(string)
	return text
}

func cpuMilli(value string) int64 {
	if value == "" {
		return 0
	}
	quantity, err := resource.ParseQuantity(value)
	if err != nil {
		return 0
	}
	return quantity.MilliValue()
}

func quantityValue(value string) int64 {
	if value == "" {
		return 0
	}
	quantity, err := resource.ParseQuantity(value)
	if err != nil {
		return 0
	}
	return quantity.Value()
}
