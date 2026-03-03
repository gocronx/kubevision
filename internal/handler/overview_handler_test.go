package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/kubevision/kubevision/internal/model"
	"github.com/kubevision/kubevision/internal/repository"
	"github.com/kubevision/kubevision/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Setup helpers
// ---------------------------------------------------------------------------

// setupOverviewHandler creates an OverviewHandler backed by a stubClusterRepo
// and a fullStubK8sRepo (both defined elsewhere in this package).
//
// listFn optionally controls what the K8s repo returns for each List call so
// tests can inject specific counts or errors.
func setupOverviewHandler(
	t *testing.T,
	clusters []*model.Cluster,
	listFn func(ctx context.Context, clusterID, kind, namespace string, opts repository.ListOptions) (*repository.ResourceList, error),
) *OverviewHandler {
	t.Helper()

	clusterRepo := newStubClusterRepo(clusters...)
	k8sRepo := &fullStubK8sRepo{listFn: listFn}
	overviewService := service.NewOverviewService(k8sRepo, clusterRepo)
	return NewOverviewHandler(overviewService)
}

// overviewRoute returns a Gin engine with the overview route registered.
func overviewRoute(h *OverviewHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/v1/clusters/:id/overview", h.GetOverview)
	return r
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestOverviewHandler_GetOverview_InvalidClusterID(t *testing.T) {
	h := setupOverviewHandler(t, nil, nil)
	router := overviewRoute(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/clusters/abc/overview", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp authAPIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 40002, resp.Code, "non-numeric cluster id should return CodeParamInvalid")
}

func TestOverviewHandler_GetOverview_ClusterNotFound(t *testing.T) {
	// Empty cluster repo — no clusters seeded.
	h := setupOverviewHandler(t, nil, nil)
	router := overviewRoute(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/clusters/9999/overview", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp authAPIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 40400, resp.Code, "non-existent cluster should return CodeNotFound")
}

func TestOverviewHandler_GetOverview_EmptyCluster(t *testing.T) {
	testCluster := &model.Cluster{ID: 1, Name: "test-cluster"}
	// listFn returns empty list — no resources exist.
	h := setupOverviewHandler(t, []*model.Cluster{testCluster}, nil)
	router := overviewRoute(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/clusters/1/overview", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp authAPIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Code, "empty cluster should still return success")

	var data service.OverviewResponse
	require.NoError(t, json.Unmarshal(resp.Data, &data))
	assert.Equal(t, 0, data.Pods, "pods should be 0")
	assert.Equal(t, 0, data.RunningPods, "runningPods should be 0")
	assert.Equal(t, 0, data.Deployments, "deployments should be 0")
	assert.Equal(t, 0, data.Services, "services should be 0")
	assert.Equal(t, 0, data.Nodes, "nodes should be 0")
	assert.Equal(t, 0, data.ReadyNodes, "readyNodes should be 0")
	assert.Equal(t, 0, data.Namespaces, "namespaces should be 0")
	assert.Empty(t, data.RecentEvents, "recentEvents should be empty")
}

func TestOverviewHandler_GetOverview_WithCounts(t *testing.T) {
	testCluster := &model.Cluster{ID: 1, Name: "test-cluster"}

	// Return different totals per resource kind. The service now queries:
	// pods, deployments, services, nodes, namespaces, then events.
	totals := map[string]int64{
		"pods":        10,
		"deployments": 3,
		"services":    5,
		"nodes":       2,
		"namespaces":  4,
		"events":      0,
	}
	listFn := func(_ context.Context, _, kind, _ string, _ repository.ListOptions) (*repository.ResourceList, error) {
		total := totals[kind]
		return &repository.ResourceList{
			Items: make([]repository.Resource, total),
			Total: total,
		}, nil
	}

	h := setupOverviewHandler(t, []*model.Cluster{testCluster}, listFn)
	router := overviewRoute(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/clusters/1/overview", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp authAPIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Code)

	var data service.OverviewResponse
	require.NoError(t, json.Unmarshal(resp.Data, &data))
	assert.Equal(t, 10, data.Pods)
	assert.Equal(t, 3, data.Deployments)
	assert.Equal(t, 5, data.Services)
	assert.Equal(t, 2, data.Nodes)
	assert.Equal(t, 4, data.Namespaces)
}

func TestOverviewHandler_GetOverview_ResponseBodyStructure(t *testing.T) {
	testCluster := &model.Cluster{ID: 1, Name: "test-cluster"}
	h := setupOverviewHandler(t, []*model.Cluster{testCluster}, nil)
	router := overviewRoute(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/clusters/1/overview", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Verify the standard response envelope.
	var raw map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &raw))
	assert.Contains(t, raw, "code")
	assert.Contains(t, raw, "message")
	assert.Contains(t, raw, "data")

	// Verify the data payload has all new fields from the enhanced OverviewResponse.
	var dataPayload map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(raw["data"], &dataPayload))
	assert.Contains(t, dataPayload, "pods")
	assert.Contains(t, dataPayload, "runningPods")
	assert.Contains(t, dataPayload, "deployments")
	assert.Contains(t, dataPayload, "services")
	assert.Contains(t, dataPayload, "nodes")
	assert.Contains(t, dataPayload, "readyNodes")
	assert.Contains(t, dataPayload, "namespaces")
	assert.Contains(t, dataPayload, "resources")
	assert.Contains(t, dataPayload, "recentEvents")
}

func TestOverviewHandler_GetOverview_K8sError(t *testing.T) {
	testCluster := &model.Cluster{ID: 1, Name: "test-cluster"}

	// Simulate a k8s repo failure on the first List call.
	listFn := func(_ context.Context, _, _, _ string, _ repository.ListOptions) (*repository.ResourceList, error) {
		return nil, assert.AnError
	}

	h := setupOverviewHandler(t, []*model.Cluster{testCluster}, listFn)
	router := overviewRoute(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/clusters/1/overview", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp authAPIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 50000, resp.Code, "k8s error should return CodeInternal")
}
