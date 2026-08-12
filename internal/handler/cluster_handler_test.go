package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gocronx/kubevision/internal/config"
	"github.com/gocronx/kubevision/internal/kubernetes/cluster"
	"github.com/gocronx/kubevision/internal/kubernetes/informer"
	"github.com/gocronx/kubevision/internal/kubernetes/resource"
	"github.com/gocronx/kubevision/internal/model"
	"github.com/gocronx/kubevision/internal/repository"
	"github.com/gocronx/kubevision/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// setupClusterTestDB creates an in-memory SQLite DB for cluster handler tests.
// Each call creates an isolated database using a unique DSN.
func setupClusterTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	cfg := &config.Config{
		Database: config.DatabaseConfig{
			Driver: "sqlite",
			DSN:    dsn,
		},
	}
	logger, _ := zap.NewDevelopment()
	db, err := repository.NewDB(cfg, logger)
	require.NoError(t, err)
	return db
}

// setupClusterHandler creates a ClusterHandler backed by a real DB and
// minimal (but real) K8s manager stubs.
func setupClusterHandler(t *testing.T) (*ClusterHandler, *gorm.DB) {
	t.Helper()
	db := setupClusterTestDB(t)
	clusterRepo := repository.NewClusterRepo(db)
	clusterManager := cluster.NewManager()
	logger, _ := zap.NewDevelopment()
	informerMgr := informer.NewManager(logger)
	registry := resource.NewRegistry()

	clusterService := service.NewClusterService(
		clusterRepo,
		clusterManager,
		informerMgr,
		registry,
		logger,
		"test-encrypt-key",
	)
	handler := NewClusterHandler(clusterService)
	return handler, db
}

// seedCluster inserts a cluster directly into the database for testing.
func seedCluster(t *testing.T, db *gorm.DB, name string) *model.Cluster {
	t.Helper()
	c := &model.Cluster{
		Name:      name,
		AuthType:  "kubeconfig",
		Status:    "healthy",
		APIServer: "https://test:6443",
	}
	err := db.Create(c).Error
	require.NoError(t, err)
	return c
}

// uintToStr converts a uint to its string representation for URL path segments.
func uintToStr(id uint) string {
	return fmt.Sprintf("%d", id)
}

func TestClusterHandler_List_ReturnsEmptyList(t *testing.T) {
	handler, _ := setupClusterHandler(t)

	router := gin.New()
	router.GET("/api/v1/clusters", handler.List)

	w := performRequest(router, "GET", "/api/v1/clusters", nil)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp authAPIResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, 0, resp.Code)
	assert.Equal(t, "success", resp.Message)

	// Data should be an empty JSON array.
	var clusters []json.RawMessage
	err = json.Unmarshal(resp.Data, &clusters)
	require.NoError(t, err)
	assert.Empty(t, clusters, "should return empty list when no clusters exist")
}

func TestClusterHandler_List_ReturnsClusters(t *testing.T) {
	handler, db := setupClusterHandler(t)

	// Seed some clusters directly in the DB.
	seedCluster(t, db, "prod")
	seedCluster(t, db, "staging")

	router := gin.New()
	router.GET("/api/v1/clusters", handler.List)

	w := performRequest(router, "GET", "/api/v1/clusters", nil)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp authAPIResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, 0, resp.Code)

	var clusters []service.ClusterResponse
	err = json.Unmarshal(resp.Data, &clusters)
	require.NoError(t, err)
	assert.Len(t, clusters, 2, "should return 2 seeded clusters")

	names := map[string]bool{}
	for _, c := range clusters {
		names[c.Name] = true
	}
	assert.True(t, names["prod"], "prod cluster should be in list")
	assert.True(t, names["staging"], "staging cluster should be in list")
}

func TestClusterHandler_Create_MissingName(t *testing.T) {
	handler, _ := setupClusterHandler(t)

	router := gin.New()
	router.POST("/api/v1/clusters", handler.Create)

	// Missing required "name" field.
	body := map[string]string{
		"authType":   "kubeconfig",
		"kubeconfig": "fake-kubeconfig-data",
	}
	w := performRequest(router, "POST", "/api/v1/clusters", body)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp authAPIResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, 40002, resp.Code, "should return CodeParamInvalid when name is missing")
	assert.Contains(t, resp.Message, "invalid")
}

func TestClusterHandler_Create_MissingAuthType(t *testing.T) {
	handler, _ := setupClusterHandler(t)

	router := gin.New()
	router.POST("/api/v1/clusters", handler.Create)

	// Missing required "authType" field.
	body := map[string]string{
		"name": "test-cluster",
	}
	w := performRequest(router, "POST", "/api/v1/clusters", body)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp authAPIResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, 40002, resp.Code, "should return CodeParamInvalid when authType is missing")
}

func TestClusterHandler_Create_EmptyBody(t *testing.T) {
	handler, _ := setupClusterHandler(t)

	router := gin.New()
	router.POST("/api/v1/clusters", handler.Create)

	w := performRequest(router, "POST", "/api/v1/clusters", map[string]string{})

	assert.Equal(t, http.StatusOK, w.Code)

	var resp authAPIResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, 40002, resp.Code, "should return CodeParamInvalid for empty body")
}

func TestClusterHandler_Create_DuplicateName(t *testing.T) {
	handler, db := setupClusterHandler(t)

	// Seed a cluster with the name we will try to create.
	seedCluster(t, db, "existing-cluster")

	router := gin.New()
	router.POST("/api/v1/clusters", handler.Create)

	body := map[string]string{
		"name":       "existing-cluster",
		"authType":   "kubeconfig",
		"kubeconfig": "fake-kubeconfig",
	}
	w := performRequest(router, "POST", "/api/v1/clusters", body)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp authAPIResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, 40900, resp.Code, "should return CodeConflict for duplicate cluster name")
	assert.Contains(t, resp.Message, "already exists")
}

func TestClusterHandler_Delete_ValidCluster(t *testing.T) {
	handler, db := setupClusterHandler(t)

	c := seedCluster(t, db, "delete-me")

	router := gin.New()
	router.DELETE("/api/v1/clusters/:id", handler.Delete)

	w := performRequest(router, "DELETE", "/api/v1/clusters/"+uintToStr(c.ID), nil)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp authAPIResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, 0, resp.Code, "delete should succeed")

	// Verify the cluster is soft-deleted from the DB.
	var count int64
	db.Model(&model.Cluster{}).Where("id = ?", c.ID).Count(&count)
	assert.Equal(t, int64(0), count, "cluster should be soft-deleted from DB")
}

func TestClusterHandler_Delete_NonExistentCluster(t *testing.T) {
	handler, _ := setupClusterHandler(t)

	router := gin.New()
	router.DELETE("/api/v1/clusters/:id", handler.Delete)

	w := performRequest(router, "DELETE", "/api/v1/clusters/99999", nil)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp authAPIResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, 40400, resp.Code, "should return CodeNotFound for non-existent cluster")
}

func TestClusterHandler_Delete_InvalidID(t *testing.T) {
	handler, _ := setupClusterHandler(t)

	router := gin.New()
	router.DELETE("/api/v1/clusters/:id", handler.Delete)

	w := performRequest(router, "DELETE", "/api/v1/clusters/abc", nil)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp authAPIResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, 40002, resp.Code, "should return CodeParamInvalid for non-numeric ID")
}

func TestClusterHandler_Get_ValidCluster(t *testing.T) {
	handler, db := setupClusterHandler(t)

	c := seedCluster(t, db, "get-me")

	router := gin.New()
	router.GET("/api/v1/clusters/:id", handler.Get)

	w := performRequest(router, "GET", "/api/v1/clusters/"+uintToStr(c.ID), nil)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp authAPIResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, 0, resp.Code)

	var clusterResp service.ClusterResponse
	err = json.Unmarshal(resp.Data, &clusterResp)
	require.NoError(t, err)
	assert.Equal(t, "get-me", clusterResp.Name)
	assert.Equal(t, "healthy", clusterResp.Status)
}

func TestClusterHandler_Get_NonExistent(t *testing.T) {
	handler, _ := setupClusterHandler(t)

	router := gin.New()
	router.GET("/api/v1/clusters/:id", handler.Get)

	w := performRequest(router, "GET", "/api/v1/clusters/99999", nil)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp authAPIResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, 40400, resp.Code, "should return CodeNotFound")
}

func TestClusterHandler_Delete_ThenListExcludes(t *testing.T) {
	handler, db := setupClusterHandler(t)

	c1 := seedCluster(t, db, "keep")
	seedCluster(t, db, "remove")

	router := gin.New()
	router.GET("/api/v1/clusters", handler.List)
	router.DELETE("/api/v1/clusters/:id", handler.Delete)

	// List should show 2 clusters.
	w := performRequest(router, "GET", "/api/v1/clusters", nil)
	var resp authAPIResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	var beforeClusters []service.ClusterResponse
	err = json.Unmarshal(resp.Data, &beforeClusters)
	require.NoError(t, err)
	assert.Len(t, beforeClusters, 2)

	// Delete the second cluster (look it up by name from beforeClusters).
	var removeID uint
	for _, c := range beforeClusters {
		if c.Name == "remove" {
			removeID = c.ID
		}
	}
	require.NotZero(t, removeID, "should find the 'remove' cluster")

	w = performRequest(router, "DELETE", "/api/v1/clusters/"+uintToStr(removeID), nil)
	require.Equal(t, http.StatusOK, w.Code)

	// List again should only show 1 cluster.
	w = performRequest(router, "GET", "/api/v1/clusters", nil)
	require.Equal(t, http.StatusOK, w.Code)

	err = json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, 0, resp.Code)

	var afterClusters []service.ClusterResponse
	err = json.Unmarshal(resp.Data, &afterClusters)
	require.NoError(t, err)
	assert.Len(t, afterClusters, 1)
	assert.Equal(t, c1.Name, afterClusters[0].Name)
}
