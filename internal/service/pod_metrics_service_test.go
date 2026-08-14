package service

import (
	"context"
	"testing"

	"github.com/gocronx/kubevision/internal/model"
	"github.com/gocronx/kubevision/internal/repository"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/fake"
)

type fakeDynamicClientProvider struct{ client dynamic.Interface }

func (f fakeDynamicClientProvider) DynamicClient(string) (dynamic.Interface, error) {
	return f.client, nil
}

func TestPodMetricsServiceList(t *testing.T) {
	metric := &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "metrics.k8s.io/v1beta1",
		// Use Pod so the dynamic fake maps the object to the metrics pods resource.
		"kind": "Pod",
		"metadata": map[string]interface{}{
			"name": "demo", "namespace": "default",
		},
		"timestamp": "2026-08-14T09:00:00Z",
		"window":    "15s",
		"containers": []interface{}{
			map[string]interface{}{
				"name":  "app",
				"usage": map[string]interface{}{"cpu": "2500000n", "memory": "10Mi"},
			},
			map[string]interface{}{
				"name":  "sidecar",
				"usage": map[string]interface{}{"cpu": "3m", "memory": "2048Ki"},
			},
		},
	}}
	// The dynamic fake needs the metrics GVR's list kind registered explicitly.
	metricsClient := fake.NewSimpleDynamicClientWithCustomListKinds(
		runtime.NewScheme(),
		map[schema.GroupVersionResource]string{podMetricsGVR: "PodMetricsList"},
		metric,
	)
	clusters := newMockClusterRepo()
	clusters.addCluster(&model.Cluster{ID: 1, Name: "local"})
	service := NewPodMetricsService(clusters, fakeDynamicClientProvider{client: metricsClient})

	items, err := service.List(context.Background(), 1, "default")
	require.NoError(t, err)
	usage := items["default/demo"]
	require.NotNil(t, usage)
	require.Equal(t, int64(6), usage.CPUMilli)
	require.Equal(t, int64(12*1024*1024), usage.MemoryBytes)
	require.Len(t, usage.Containers, 2)
	require.Equal(t, "15s", usage.Window)
}

func TestApplyPodResourceAllocations(t *testing.T) {
	metrics := &repository.PodMetrics{Containers: []repository.ContainerMetrics{
		{Name: "app"}, {Name: "sidecar"},
	}}
	pod := map[string]interface{}{
		"spec": map[string]interface{}{
			"containers": []interface{}{
				map[string]interface{}{
					"name": "app",
					"resources": map[string]interface{}{
						"requests": map[string]interface{}{"cpu": "100m", "memory": "64Mi"},
						"limits":   map[string]interface{}{"cpu": "500m", "memory": "256Mi"},
					},
				},
				map[string]interface{}{
					"name": "sidecar",
					"resources": map[string]interface{}{
						"requests": map[string]interface{}{"cpu": "50m", "memory": "32Mi"},
						"limits":   map[string]interface{}{"cpu": "200m", "memory": "128Mi"},
					},
				},
			},
		},
	}

	ApplyPodResourceAllocations(metrics, pod)

	require.Equal(t, int64(150), metrics.CPURequestMilli)
	require.Equal(t, int64(700), metrics.CPULimitMilli)
	require.Equal(t, int64(96*1024*1024), metrics.MemoryRequestBytes)
	require.Equal(t, int64(384*1024*1024), metrics.MemoryLimitBytes)
}
