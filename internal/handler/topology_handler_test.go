package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/kubevision/kubevision/internal/model"
	"github.com/kubevision/kubevision/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTopologyHandler(t *testing.T) (*gin.Engine, *TopologyHandler) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	k8sRepo := &fullStubK8sRepo{}
	clusterRepo := newStubClusterRepo(&model.Cluster{ID: 1, Name: "prod", AuthType: "kubeconfig", Status: "healthy"})
	topologySvc := service.NewTopologyService(k8sRepo, clusterRepo)
	handler := NewTopologyHandler(topologySvc)

	router := gin.New()
	router.GET("/api/v1/clusters/:id/namespaces/:namespace/topology", handler.GetTopology)
	return router, handler
}

func TestTopologyHandler_GetTopology(t *testing.T) {
	router, _ := setupTopologyHandler(t)

	req := httptest.NewRequest("GET", "/api/v1/clusters/1/namespaces/default/topology", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp authAPIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Code)
}

func TestTopologyHandler_InvalidClusterID(t *testing.T) {
	router, _ := setupTopologyHandler(t)

	req := httptest.NewRequest("GET", "/api/v1/clusters/abc/namespaces/default/topology", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var resp authAPIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.NotEqual(t, 0, resp.Code)
}

func TestTopologyHandler_ClusterNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	k8sRepo := &fullStubK8sRepo{}
	clusterRepo := newStubClusterRepo() // empty
	topologySvc := service.NewTopologyService(k8sRepo, clusterRepo)
	handler := NewTopologyHandler(topologySvc)

	router := gin.New()
	router.GET("/api/v1/clusters/:id/namespaces/:namespace/topology", handler.GetTopology)

	req := httptest.NewRequest("GET", "/api/v1/clusters/999/namespaces/default/topology", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var resp authAPIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.NotEqual(t, 0, resp.Code)
}
