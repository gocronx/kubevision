package service

import (
	"context"
	"fmt"
	"sort"

	bizerr "github.com/gocronx/kubevision/internal/pkg/errors"
	"github.com/gocronx/kubevision/internal/repository"
	"k8s.io/apimachinery/pkg/api/resource"
)

// ResourceMetric holds current usage and configured allocation for one resource dimension.
type ResourceMetric struct {
	Allocatable int64 `json:"allocatable"` // millicores for CPU, bytes for memory
	Usage       int64 `json:"usage"`
	Requests    int64 `json:"requests"`
	Limits      int64 `json:"limits"`
}

// ResourceUsage holds CPU and memory usage and allocation summaries.
type ResourceUsage struct {
	CPU              ResourceMetric `json:"cpu"`
	Memory           ResourceMetric `json:"memory"`
	MetricsAvailable bool           `json:"metricsAvailable"`
}

// EventSummary is a condensed representation of a Kubernetes event.
type EventSummary struct {
	Type       string `json:"type"` // Normal, Warning
	Reason     string `json:"reason"`
	Message    string `json:"message"`
	ObjectKind string `json:"objectKind"`
	ObjectName string `json:"objectName"`
	Namespace  string `json:"namespace,omitempty"`
	Timestamp  string `json:"timestamp"`
}

// PodStatusDist holds the distribution of pods across phases.
type PodStatusDist struct {
	Running   int `json:"running"`
	Pending   int `json:"pending"`
	Succeeded int `json:"succeeded"`
	Failed    int `json:"failed"`
	Unknown   int `json:"unknown"`
}

// OverviewResponse holds the aggregated resource counts for a cluster overview.
type OverviewResponse struct {
	Pods             int            `json:"pods"`
	RunningPods      int            `json:"runningPods"`
	Deployments      int            `json:"deployments"`
	ReadyDeployments int            `json:"readyDeployments"`
	Services         int            `json:"services"`
	Nodes            int            `json:"nodes"`
	ReadyNodes       int            `json:"readyNodes"`
	Namespaces       int            `json:"namespaces"`
	ActiveNamespaces int            `json:"activeNamespaces"`
	Resources        ResourceUsage  `json:"resources"`
	RecentEvents     []EventSummary `json:"recentEvents"`

	// Workload stats
	StatefulSets      int `json:"statefulSets"`
	ReadyStatefulSets int `json:"readyStatefulSets"`
	DaemonSets        int `json:"daemonSets"`
	ReadyDaemonSets   int `json:"readyDaemonSets"`
	Jobs              int `json:"jobs"`
	SucceededJobs     int `json:"succeededJobs"`
	FailedJobs        int `json:"failedJobs"`
	CronJobs          int `json:"cronJobs"`
	ActiveCronJobs    int `json:"activeCronJobs"`
	Ingresses         int `json:"ingresses"`

	// Storage stats
	PersistentVolumes      int   `json:"persistentVolumes"`
	BoundPVs               int   `json:"boundPVs"`
	AvailablePVs           int   `json:"availablePVs"`
	ReleasedPVs            int   `json:"releasedPVs"`
	PersistentVolumeClaims int   `json:"persistentVolumeClaims"`
	BoundPVCs              int   `json:"boundPVCs"`
	PendingPVCs            int   `json:"pendingPVCs"`
	TotalStorageBytes      int64 `json:"totalStorageBytes"`
	AllocatedStorageBytes  int64 `json:"allocatedStorageBytes"`
	// UsedStorageBytes is retained for API compatibility. It represents bound PV
	// capacity, not filesystem usage; use AllocatedStorageBytes in new clients.
	UsedStorageBytes int64 `json:"usedStorageBytes"`

	// Pod status distribution
	PodStatusDistribution PodStatusDist `json:"podStatusDistribution"`
}

// OverviewService aggregates cluster-level resource counts.
type OverviewService struct {
	k8sRepo     repository.K8sResourceRepo
	clusterRepo repository.ClusterRepo
	podMetrics  podMetricsReader
}

type podMetricsReader interface {
	List(context.Context, uint, string) (map[string]*repository.PodMetrics, error)
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

// WithPodMetrics enables best-effort live CPU and memory usage in the overview.
func (s *OverviewService) WithPodMetrics(metrics podMetricsReader) *OverviewService {
	s.podMetrics = metrics
	return s
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
	resourceTypes := []string{
		"pods", "deployments", "services", "nodes", "namespaces",
		"statefulsets", "daemonsets", "jobs", "cronjobs", "ingresses",
		"persistentvolumes", "persistentvolumeclaims",
	}
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

	// Fetch events without a server-side limit. The Kubernetes List API returns
	// items in etcd insertion order (oldest first), not by time, so a limit would
	// drop the most recent events. Events have a default TTL of 1 hour in K8s,
	// so the total count is bounded and safe to fetch in full.
	eventList, err := s.k8sRepo.List(ctx, clusterKey, "events", "", repository.ListOptions{})
	if err != nil {
		// Events are optional — don't fail the whole overview.
		eventList = &repository.ResourceList{}
	}

	// Count running pods and pod status distribution
	runningPods := 0
	var podStatusDist PodStatusDist
	for _, pod := range lists["pods"].Items {
		phase := getNestedString(pod.Raw, "status", "phase")
		switch phase {
		case "Running":
			runningPods++
			podStatusDist.Running++
		case "Pending":
			podStatusDist.Pending++
		case "Succeeded":
			podStatusDist.Succeeded++
		case "Failed":
			podStatusDist.Failed++
		default:
			podStatusDist.Unknown++
		}
	}

	// Count ready deployments (availableReplicas >= desired replicas)
	readyDeployments := 0
	for _, dep := range lists["deployments"].Items {
		desired := getNestedFloat(dep.Raw, "spec", "replicas")
		available := getNestedFloat(dep.Raw, "status", "availableReplicas")
		if desired >= 0 && available >= desired {
			readyDeployments++
		}
	}

	// Count active namespaces
	activeNamespaces := 0
	for _, ns := range lists["namespaces"].Items {
		phase := getNestedString(ns.Raw, "status", "phase")
		if phase == "Active" {
			activeNamespaces++
		}
	}

	// Count ready statefulsets
	readyStatefulSets := 0
	for _, ss := range lists["statefulsets"].Items {
		desired := getNestedFloat(ss.Raw, "spec", "replicas")
		ready := getNestedFloat(ss.Raw, "status", "readyReplicas")
		if desired >= 0 && ready >= desired {
			readyStatefulSets++
		}
	}

	// Count ready daemonsets
	readyDaemonSets := 0
	for _, ds := range lists["daemonsets"].Items {
		desired := getNestedFloat(ds.Raw, "status", "desiredNumberScheduled")
		ready := getNestedFloat(ds.Raw, "status", "numberReady")
		if desired >= 0 && ready >= desired {
			readyDaemonSets++
		}
	}

	// Count job statuses
	succeededJobs := 0
	failedJobs := 0
	for _, job := range lists["jobs"].Items {
		succeeded := getNestedFloat(job.Raw, "status", "succeeded")
		failed := getNestedFloat(job.Raw, "status", "failed")
		conditions := getNestedSlice(job.Raw, "status", "conditions")
		isComplete := false
		isFailed := false
		for _, c := range conditions {
			cond, ok := c.(map[string]any)
			if !ok {
				continue
			}
			if cond["type"] == "Complete" && cond["status"] == "True" {
				isComplete = true
			}
			if cond["type"] == "Failed" && cond["status"] == "True" {
				isFailed = true
			}
		}
		if isComplete || succeeded > 0 {
			succeededJobs++
		} else if isFailed || failed > 0 {
			failedJobs++
		}
	}

	// Count active cronjobs (those with active jobs)
	activeCronJobs := 0
	for _, cj := range lists["cronjobs"].Items {
		active := getNestedSlice(cj.Raw, "status", "active")
		if len(active) > 0 {
			activeCronJobs++
		}
	}

	// Storage: PV stats
	boundPVs := 0
	availablePVs := 0
	releasedPVs := 0
	var totalStorageBytes int64
	var allocatedStorageBytes int64
	for _, pv := range lists["persistentvolumes"].Items {
		phase := getNestedString(pv.Raw, "status", "phase")
		capacity := getNestedMap(pv.Raw, "spec", "capacity")
		var pvBytes int64
		if capacity != nil {
			if storage, ok := capacity["storage"].(string); ok {
				pvBytes = parseMemoryQuantity(storage)
				totalStorageBytes += pvBytes
			}
		}
		switch phase {
		case "Bound":
			boundPVs++
			allocatedStorageBytes += pvBytes
		case "Available":
			availablePVs++
		case "Released":
			releasedPVs++
		}
	}

	// Storage: PVC stats
	boundPVCs := 0
	pendingPVCs := 0
	for _, pvc := range lists["persistentvolumeclaims"].Items {
		phase := getNestedString(pvc.Raw, "status", "phase")
		switch phase {
		case "Bound":
			boundPVCs++
		case "Pending":
			pendingPVCs++
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
		if isTerminalPod(pod) {
			continue
		}
		allocation := effectivePodAllocation(pod.Raw)
		resources.CPU.Requests += allocation.cpuRequests
		resources.CPU.Limits += allocation.cpuLimits
		resources.Memory.Requests += allocation.memoryRequests
		resources.Memory.Limits += allocation.memoryLimits
	}

	// Metrics Server is optional. Allocation data remains useful when live
	// metrics are unavailable, so a metrics error must not fail the overview.
	if s.podMetrics != nil {
		if metrics, metricsErr := s.podMetrics.List(ctx, clusterID, ""); metricsErr == nil {
			resources.MetricsAvailable = true
			for _, podMetrics := range metrics {
				resources.CPU.Usage += podMetrics.CPUMilli
				resources.Memory.Usage += podMetrics.MemoryBytes
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
		Pods:             int(lists["pods"].Total),
		RunningPods:      runningPods,
		Deployments:      int(lists["deployments"].Total),
		ReadyDeployments: readyDeployments,
		Services:         int(lists["services"].Total),
		Nodes:            int(lists["nodes"].Total),
		ReadyNodes:       readyNodes,
		Namespaces:       int(lists["namespaces"].Total),
		ActiveNamespaces: activeNamespaces,
		Resources:        resources,
		RecentEvents:     recentEvents,

		StatefulSets:      int(lists["statefulsets"].Total),
		ReadyStatefulSets: readyStatefulSets,
		DaemonSets:        int(lists["daemonsets"].Total),
		ReadyDaemonSets:   readyDaemonSets,
		Jobs:              int(lists["jobs"].Total),
		SucceededJobs:     succeededJobs,
		FailedJobs:        failedJobs,
		CronJobs:          int(lists["cronjobs"].Total),
		ActiveCronJobs:    activeCronJobs,
		Ingresses:         int(lists["ingresses"].Total),

		PersistentVolumes:      int(lists["persistentvolumes"].Total),
		BoundPVs:               boundPVs,
		AvailablePVs:           availablePVs,
		ReleasedPVs:            releasedPVs,
		PersistentVolumeClaims: int(lists["persistentvolumeclaims"].Total),
		BoundPVCs:              boundPVCs,
		PendingPVCs:            pendingPVCs,
		TotalStorageBytes:      totalStorageBytes,
		AllocatedStorageBytes:  allocatedStorageBytes,
		UsedStorageBytes:       allocatedStorageBytes,

		PodStatusDistribution: podStatusDist,
	}, nil
}

func isTerminalPod(pod repository.Resource) bool {
	phase := getNestedString(pod.Raw, "status", "phase")
	return phase == "Succeeded" || phase == "Failed"
}

type podAllocation struct {
	cpuRequests    int64
	cpuLimits      int64
	memoryRequests int64
	memoryLimits   int64
}

func (a *podAllocation) add(other podAllocation) {
	a.cpuRequests += other.cpuRequests
	a.cpuLimits += other.cpuLimits
	a.memoryRequests += other.memoryRequests
	a.memoryLimits += other.memoryLimits
}

func (a *podAllocation) max(other podAllocation) {
	a.cpuRequests = max(a.cpuRequests, other.cpuRequests)
	a.cpuLimits = max(a.cpuLimits, other.cpuLimits)
	a.memoryRequests = max(a.memoryRequests, other.memoryRequests)
	a.memoryLimits = max(a.memoryLimits, other.memoryLimits)
}

func effectivePodAllocation(pod map[string]any) podAllocation {
	var steadyState podAllocation
	for _, rawContainer := range getNestedSlice(pod, "spec", "containers") {
		if container, ok := rawContainer.(map[string]any); ok {
			steadyState.add(containerAllocation(container))
		}
	}

	// Restartable init containers (sidecars) remain active after startup. A
	// regular init container runs alongside all restartable init containers
	// declared before it, while ordinary init containers run sequentially.
	var sidecars podAllocation
	var initPeak podAllocation
	for _, rawContainer := range getNestedSlice(pod, "spec", "initContainers") {
		container, ok := rawContainer.(map[string]any)
		if !ok {
			continue
		}
		current := containerAllocation(container)
		if getNestedString(container, "restartPolicy") == "Always" {
			sidecars.add(current)
			initPeak.max(sidecars)
			continue
		}
		current.add(sidecars)
		initPeak.max(current)
	}

	steadyState.add(sidecars)
	steadyState.max(initPeak)
	if overhead := getNestedMap(pod, "spec", "overhead"); overhead != nil {
		values := resourceListValues(overhead)
		steadyState.add(podAllocation{
			cpuRequests:    values.cpu,
			cpuLimits:      values.cpu,
			memoryRequests: values.memory,
			memoryLimits:   values.memory,
		})
	}
	return steadyState
}

func containerAllocation(container map[string]any) podAllocation {
	resources := getNestedMap(container, "resources")
	if resources == nil {
		return podAllocation{}
	}
	requests, _ := resources["requests"].(map[string]any)
	limits, _ := resources["limits"].(map[string]any)
	requestValues := resourceListValues(requests)
	limitValues := resourceListValues(limits)
	return podAllocation{
		cpuRequests:    requestValues.cpu,
		cpuLimits:      limitValues.cpu,
		memoryRequests: requestValues.memory,
		memoryLimits:   limitValues.memory,
	}
}

type resourceValues struct {
	cpu    int64
	memory int64
}

func resourceListValues(resources map[string]any) resourceValues {
	cpu, _ := resources["cpu"].(string)
	memory, _ := resources["memory"].(string)
	return resourceValues{
		cpu:    parseCPUQuantity(cpu),
		memory: parseMemoryQuantity(memory),
	}
}

// parseCPUQuantity parses a Kubernetes CPU quantity into millicores.
func parseCPUQuantity(val string) int64 {
	quantity, err := resource.ParseQuantity(val)
	if err != nil {
		return 0
	}
	return quantity.MilliValue()
}

// parseMemoryQuantity parses a Kubernetes memory quantity string into bytes.
func parseMemoryQuantity(val string) int64 {
	quantity, err := resource.ParseQuantity(val)
	if err != nil {
		return 0
	}
	return quantity.Value()
}

// getNestedFloat safely extracts a numeric value from a nested map path.
// JSON numbers from unstructured K8s resources are typically float64.
func getNestedFloat(obj map[string]any, keys ...string) float64 {
	current := obj
	for i, key := range keys {
		v, ok := current[key]
		if !ok {
			return -1
		}
		if i == len(keys)-1 {
			switch n := v.(type) {
			case float64:
				return n
			case int64:
				return float64(n)
			case int:
				return float64(n)
			default:
				return -1
			}
		}
		next, ok := v.(map[string]any)
		if !ok {
			return -1
		}
		current = next
	}
	return -1
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
