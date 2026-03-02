package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/kubevision/kubevision/internal/kubernetes/cluster"
	"github.com/kubevision/kubevision/internal/model"
	"github.com/kubevision/kubevision/internal/repository"
	"github.com/kubevision/kubevision/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Stub ClusterRepo for ResourceActionService
// ---------------------------------------------------------------------------

// stubClusterRepo implements repository.ClusterRepo for test use.
// It holds a map of clusters keyed by ID and by name.
type stubClusterRepo struct {
	byID   map[uint]*model.Cluster
	byName map[string]*model.Cluster
}

func newStubClusterRepo(clusters ...*model.Cluster) *stubClusterRepo {
	r := &stubClusterRepo{
		byID:   make(map[uint]*model.Cluster),
		byName: make(map[string]*model.Cluster),
	}
	for _, c := range clusters {
		r.byID[c.ID] = c
		r.byName[c.Name] = c
	}
	return r
}

func (s *stubClusterRepo) Create(_ context.Context, cluster *model.Cluster) error {
	s.byID[cluster.ID] = cluster
	s.byName[cluster.Name] = cluster
	return nil
}

func (s *stubClusterRepo) GetByID(_ context.Context, id uint) (*model.Cluster, error) {
	c, ok := s.byID[id]
	if !ok {
		return nil, fmt.Errorf("cluster %d not found", id)
	}
	return c, nil
}

func (s *stubClusterRepo) GetByName(_ context.Context, name string) (*model.Cluster, error) {
	c, ok := s.byName[name]
	if !ok {
		return nil, fmt.Errorf("cluster %q not found", name)
	}
	return c, nil
}

func (s *stubClusterRepo) Update(_ context.Context, cluster *model.Cluster) error {
	s.byID[cluster.ID] = cluster
	s.byName[cluster.Name] = cluster
	return nil
}

func (s *stubClusterRepo) Delete(_ context.Context, id uint) error {
	c, ok := s.byID[id]
	if !ok {
		return fmt.Errorf("cluster %d not found", id)
	}
	delete(s.byName, c.Name)
	delete(s.byID, id)
	return nil
}

func (s *stubClusterRepo) List(_ context.Context) ([]model.Cluster, error) {
	out := make([]model.Cluster, 0, len(s.byID))
	for _, c := range s.byID {
		out = append(out, *c)
	}
	return out, nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// setupActionHandler creates a ResourceActionHandler backed by a stub repo
// with no real Kubernetes cluster (the cluster manager is empty — no REST
// configs are loaded — so any call that requires a real K8s client will
// return an error before reaching the API server).
func setupActionHandler(t *testing.T) *ResourceActionHandler {
	t.Helper()

	testCluster := &model.Cluster{
		ID:       1,
		Name:     "test-cluster",
		AuthType: "kubeconfig",
		Status:   "healthy",
	}
	clusterRepo := newStubClusterRepo(testCluster)

	// cluster.NewManager() returns an empty manager — no REST configs loaded.
	// RESTConfig("test-cluster") will return "cluster test-cluster not found",
	// producing CodeK8sUnavailable in typedClient().
	clusterMgr := cluster.NewManager()

	actionService := service.NewResourceActionService(clusterRepo, clusterMgr)
	return NewResourceActionHandler(actionService)
}

// ---------------------------------------------------------------------------
// Scale tests
// ---------------------------------------------------------------------------

func TestResourceActionHandler_Scale_InvalidClusterID(t *testing.T) {
	handler := setupActionHandler(t)

	router := gin.New()
	router.PUT("/api/v1/clusters/:id/namespaces/:namespace/:kind/:name/scale", handler.Scale)

	w := performRequest(router, "PUT",
		"/api/v1/clusters/abc/namespaces/default/deployments/my-app/scale",
		map[string]interface{}{"replicas": 3},
	)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp authAPIResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, 40002, resp.Code, "should return CodeParamInvalid for non-numeric cluster id")
}

func TestResourceActionHandler_Scale_MissingReplicasField(t *testing.T) {
	handler := setupActionHandler(t)

	router := gin.New()
	router.PUT("/api/v1/clusters/:id/namespaces/:namespace/:kind/:name/scale", handler.Scale)

	// Empty JSON body — replicas field missing, ShouldBindJSON will succeed
	// (replicas is not tagged binding:"required"), but the service call should
	// still exercise the path down to the service.
	// We verify we get a non-success response (cluster unreachable).
	w := performRequest(router, "PUT",
		"/api/v1/clusters/1/namespaces/default/deployments/my-app/scale",
		nil,
	)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp authAPIResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	// body is nil so ShouldBindJSON fails → CodeParamInvalid
	assert.Equal(t, 40002, resp.Code)
}

func TestResourceActionHandler_Scale_ClusterNotFound(t *testing.T) {
	handler := setupActionHandler(t)

	router := gin.New()
	router.PUT("/api/v1/clusters/:id/namespaces/:namespace/:kind/:name/scale", handler.Scale)

	// Cluster ID 9999 does not exist in the stub repo.
	w := performRequest(router, "PUT",
		"/api/v1/clusters/9999/namespaces/default/deployments/my-app/scale",
		map[string]interface{}{"replicas": 2},
	)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp authAPIResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, 40400, resp.Code, "should return CodeNotFound for missing cluster")
}

func TestResourceActionHandler_Scale_UnsupportedKind(t *testing.T) {
	// Use a cluster repo with cluster ID 1, but send an unsupported kind.
	// The service validates the kind before hitting the K8s API.
	testCluster := &model.Cluster{ID: 1, Name: "test-cluster"}
	clusterRepo := newStubClusterRepo(testCluster)
	actionService := service.NewResourceActionService(clusterRepo, cluster.NewManager())
	handler := NewResourceActionHandler(actionService)

	router := gin.New()
	router.PUT("/api/v1/clusters/:id/namespaces/:namespace/:kind/:name/scale", handler.Scale)

	w := performRequest(router, "PUT",
		"/api/v1/clusters/1/namespaces/default/pods/my-pod/scale",
		map[string]interface{}{"replicas": 2},
	)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp authAPIResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, 40002, resp.Code, "should return CodeParamInvalid for unsupported kind")
	assert.Contains(t, resp.Message, "does not support scaling")
}

func TestResourceActionHandler_Scale_NegativeReplicas(t *testing.T) {
	testCluster := &model.Cluster{ID: 1, Name: "test-cluster"}
	clusterRepo := newStubClusterRepo(testCluster)
	actionService := service.NewResourceActionService(clusterRepo, cluster.NewManager())
	handler := NewResourceActionHandler(actionService)

	router := gin.New()
	router.PUT("/api/v1/clusters/:id/namespaces/:namespace/:kind/:name/scale", handler.Scale)

	w := performRequest(router, "PUT",
		"/api/v1/clusters/1/namespaces/default/deployments/my-app/scale",
		map[string]interface{}{"replicas": -1},
	)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp authAPIResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, 40002, resp.Code, "should return CodeParamInvalid for negative replicas")
	assert.Contains(t, resp.Message, "replicas must be >= 0")
}

// ---------------------------------------------------------------------------
// Restart tests
// ---------------------------------------------------------------------------

func TestResourceActionHandler_Restart_InvalidClusterID(t *testing.T) {
	handler := setupActionHandler(t)

	router := gin.New()
	router.POST("/api/v1/clusters/:id/namespaces/:namespace/:kind/:name/restart", handler.Restart)

	w := performRequest(router, "POST",
		"/api/v1/clusters/xyz/namespaces/default/deployments/my-app/restart",
		nil,
	)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp authAPIResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, 40002, resp.Code, "should return CodeParamInvalid for non-numeric cluster id")
}

func TestResourceActionHandler_Restart_ClusterNotFound(t *testing.T) {
	handler := setupActionHandler(t)

	router := gin.New()
	router.POST("/api/v1/clusters/:id/namespaces/:namespace/:kind/:name/restart", handler.Restart)

	w := performRequest(router, "POST",
		"/api/v1/clusters/9999/namespaces/default/deployments/my-app/restart",
		nil,
	)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp authAPIResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, 40400, resp.Code, "should return CodeNotFound for missing cluster")
}

func TestResourceActionHandler_Restart_UnsupportedKind(t *testing.T) {
	testCluster := &model.Cluster{ID: 1, Name: "test-cluster"}
	clusterRepo := newStubClusterRepo(testCluster)
	actionService := service.NewResourceActionService(clusterRepo, cluster.NewManager())
	handler := NewResourceActionHandler(actionService)

	router := gin.New()
	router.POST("/api/v1/clusters/:id/namespaces/:namespace/:kind/:name/restart", handler.Restart)

	w := performRequest(router, "POST",
		"/api/v1/clusters/1/namespaces/default/replicasets/my-rs/restart",
		nil,
	)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp authAPIResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, 40002, resp.Code, "should return CodeParamInvalid for unsupported kind")
	assert.Contains(t, resp.Message, "does not support restart")
}

// ---------------------------------------------------------------------------
// RolloutHistory tests
// ---------------------------------------------------------------------------

func TestResourceActionHandler_RolloutHistory_InvalidClusterID(t *testing.T) {
	handler := setupActionHandler(t)

	router := gin.New()
	router.GET("/api/v1/clusters/:id/namespaces/:namespace/deployments/:name/history", handler.RolloutHistory)

	w := performRequest(router, "GET",
		"/api/v1/clusters/not-a-number/namespaces/default/deployments/my-app/history",
		nil,
	)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp authAPIResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, 40002, resp.Code)
}

func TestResourceActionHandler_RolloutHistory_ClusterNotFound(t *testing.T) {
	handler := setupActionHandler(t)

	router := gin.New()
	router.GET("/api/v1/clusters/:id/namespaces/:namespace/deployments/:name/history", handler.RolloutHistory)

	w := performRequest(router, "GET",
		"/api/v1/clusters/9999/namespaces/default/deployments/my-app/history",
		nil,
	)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp authAPIResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, 40400, resp.Code, "should return CodeNotFound for missing cluster")
}

// ---------------------------------------------------------------------------
// Rollback tests
// ---------------------------------------------------------------------------

func TestResourceActionHandler_Rollback_InvalidClusterID(t *testing.T) {
	handler := setupActionHandler(t)

	router := gin.New()
	router.POST("/api/v1/clusters/:id/namespaces/:namespace/deployments/:name/rollback", handler.Rollback)

	w := performRequest(router, "POST",
		"/api/v1/clusters/bad/namespaces/default/deployments/my-app/rollback",
		nil,
	)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp authAPIResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, 40002, resp.Code)
}

func TestResourceActionHandler_Rollback_ClusterNotFound(t *testing.T) {
	handler := setupActionHandler(t)

	router := gin.New()
	router.POST("/api/v1/clusters/:id/namespaces/:namespace/deployments/:name/rollback", handler.Rollback)

	w := performRequest(router, "POST",
		"/api/v1/clusters/9999/namespaces/default/deployments/my-app/rollback",
		nil,
	)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp authAPIResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, 40400, resp.Code, "should return CodeNotFound for missing cluster")
}

func TestResourceActionHandler_Rollback_OptionalBody_NoBody(t *testing.T) {
	// Rollback body is optional; passing nil body should not cause an error
	// at the handler level (it ignores bind errors for the optional body).
	// The cluster manager is empty so typedClient returns CodeK8sUnavailable,
	// but we confirm the handler does not return CodeParamInvalid for no body.
	testCluster := &model.Cluster{ID: 1, Name: "test-cluster"}
	clusterRepo := newStubClusterRepo(testCluster)
	actionService := service.NewResourceActionService(clusterRepo, cluster.NewManager())
	handler := NewResourceActionHandler(actionService)

	router := gin.New()
	router.POST("/api/v1/clusters/:id/namespaces/:namespace/deployments/:name/rollback", handler.Rollback)

	w := performRequest(router, "POST",
		"/api/v1/clusters/1/namespaces/default/deployments/my-app/rollback",
		nil,
	)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp authAPIResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	// Should NOT be 40002 (param invalid) — body is optional
	assert.NotEqual(t, 40002, resp.Code, "no body should not return CodeParamInvalid")
}

func TestResourceActionHandler_Rollback_WithRevisionBody(t *testing.T) {
	testCluster := &model.Cluster{ID: 1, Name: "test-cluster"}
	clusterRepo := newStubClusterRepo(testCluster)
	actionService := service.NewResourceActionService(clusterRepo, cluster.NewManager())
	handler := NewResourceActionHandler(actionService)

	router := gin.New()
	router.POST("/api/v1/clusters/:id/namespaces/:namespace/deployments/:name/rollback", handler.Rollback)

	w := performRequest(router, "POST",
		"/api/v1/clusters/1/namespaces/default/deployments/my-app/rollback",
		map[string]interface{}{"revision": 2},
	)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp authAPIResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	// The cluster manager has no config → CodeK8sUnavailable (50200)
	assert.NotEqual(t, 40002, resp.Code, "valid body should not return CodeParamInvalid")
}

// Compile-time check: ensure the stubClusterRepo satisfies the interface.
var _ repository.ClusterRepo = (*stubClusterRepo)(nil)
