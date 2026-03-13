package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gocronx/kubevision/internal/config"
	"github.com/gocronx/kubevision/internal/repository"
	"github.com/gocronx/kubevision/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// ---------------------------------------------------------------------------
// Test setup
// ---------------------------------------------------------------------------

// setupFavoriteTestDB creates an isolated in-memory SQLite DB for favorite tests.
func setupFavoriteTestDB(t *testing.T) *gorm.DB {
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

// setupFavoriteHandler creates a FavoriteHandler backed by a real SQLite DB.
// It returns the handler and the database so individual tests can seed rows.
func setupFavoriteHandler(t *testing.T) (*FavoriteHandler, *gorm.DB) {
	t.Helper()
	db := setupFavoriteTestDB(t)
	favRepo := repository.NewFavoriteRepo(db)
	favService := service.NewFavoriteService(favRepo)
	handler := NewFavoriteHandler(favService)
	return handler, db
}

// injectUserID returns a Gin middleware that sets the userID context key so
// that FavoriteHandler.List/Create/etc. see a valid authenticated user.
func injectUserID(userID uint) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("userID", userID)
		c.Next()
	}
}

// ---------------------------------------------------------------------------
// Helpers for response shapes
// ---------------------------------------------------------------------------

type favoriteListData []service.FavoriteResponse
type favoriteData service.FavoriteResponse
type toggleData service.ToggleFavoriteResponse
type checkData service.CheckFavoriteResponse

// ---------------------------------------------------------------------------
// List tests
// ---------------------------------------------------------------------------

func TestFavoriteHandler_List_Unauthorized(t *testing.T) {
	handler, _ := setupFavoriteHandler(t)

	router := gin.New()
	// No middleware injecting userID → GetUserID returns 0.
	router.GET("/api/v1/favorites", handler.List)

	w := performRequest(router, "GET", "/api/v1/favorites", nil)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp authAPIResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, 40100, resp.Code, "should return CodeUnauthorized when no userID in context")
}

func TestFavoriteHandler_List_EmptyForNewUser(t *testing.T) {
	handler, _ := setupFavoriteHandler(t)

	router := gin.New()
	router.Use(injectUserID(42))
	router.GET("/api/v1/favorites", handler.List)

	w := performRequest(router, "GET", "/api/v1/favorites", nil)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp authAPIResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, 0, resp.Code)

	var data favoriteListData
	err = json.Unmarshal(resp.Data, &data)
	require.NoError(t, err)
	assert.Empty(t, data, "new user should have no favorites")
}

func TestFavoriteHandler_List_ReturnsFavoritesForUser(t *testing.T) {
	handler, _ := setupFavoriteHandler(t)
	const testUserID uint = 1

	router := gin.New()
	router.Use(injectUserID(testUserID))
	router.GET("/api/v1/favorites", handler.List)
	router.POST("/api/v1/favorites", handler.Create)

	// Create two favorites.
	addBody1 := map[string]string{
		"clusterId":    "cluster-1",
		"resourceType": "deployments",
		"resourceName": "nginx",
		"namespace":    "default",
	}
	addBody2 := map[string]string{
		"clusterId":    "cluster-1",
		"resourceType": "pods",
		"resourceName": "my-pod",
		"namespace":    "kube-system",
	}
	w := performRequest(router, "POST", "/api/v1/favorites", addBody1)
	require.Equal(t, http.StatusOK, w.Code)
	w = performRequest(router, "POST", "/api/v1/favorites", addBody2)
	require.Equal(t, http.StatusOK, w.Code)

	// List should return both favorites.
	w = performRequest(router, "GET", "/api/v1/favorites", nil)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp authAPIResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, 0, resp.Code)

	var data favoriteListData
	err = json.Unmarshal(resp.Data, &data)
	require.NoError(t, err)
	assert.Len(t, data, 2, "should return 2 favorites")
}

// ---------------------------------------------------------------------------
// Create tests
// ---------------------------------------------------------------------------

func TestFavoriteHandler_Create_Unauthorized(t *testing.T) {
	handler, _ := setupFavoriteHandler(t)

	router := gin.New()
	router.POST("/api/v1/favorites", handler.Create)

	body := map[string]string{
		"clusterId":    "cluster-1",
		"resourceType": "deployments",
		"resourceName": "nginx",
	}
	w := performRequest(router, "POST", "/api/v1/favorites", body)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp authAPIResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, 40100, resp.Code)
}

func TestFavoriteHandler_Create_MissingRequiredFields(t *testing.T) {
	handler, _ := setupFavoriteHandler(t)

	router := gin.New()
	router.Use(injectUserID(1))
	router.POST("/api/v1/favorites", handler.Create)

	// Missing clusterId and resourceType
	body := map[string]string{
		"resourceName": "nginx",
	}
	w := performRequest(router, "POST", "/api/v1/favorites", body)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp authAPIResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, 40002, resp.Code, "should return CodeParamInvalid for missing required fields")
}

func TestFavoriteHandler_Create_Success(t *testing.T) {
	handler, _ := setupFavoriteHandler(t)
	const testUserID uint = 1

	router := gin.New()
	router.Use(injectUserID(testUserID))
	router.POST("/api/v1/favorites", handler.Create)

	body := map[string]string{
		"clusterId":    "cluster-1",
		"resourceType": "deployments",
		"resourceName": "nginx",
		"namespace":    "default",
		"displayName":  "Nginx Deployment",
	}
	w := performRequest(router, "POST", "/api/v1/favorites", body)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp authAPIResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, 0, resp.Code)

	var data favoriteData
	err = json.Unmarshal(resp.Data, &data)
	require.NoError(t, err)
	assert.NotZero(t, data.ID, "created favorite should have a non-zero ID")
	assert.Equal(t, "cluster-1", data.ClusterID)
	assert.Equal(t, "deployments", data.ResourceType)
	assert.Equal(t, "nginx", data.ResourceName)
	assert.Equal(t, "Nginx Deployment", data.DisplayName)
}

func TestFavoriteHandler_Create_DefaultsDisplayNameToResourceName(t *testing.T) {
	handler, _ := setupFavoriteHandler(t)

	router := gin.New()
	router.Use(injectUserID(1))
	router.POST("/api/v1/favorites", handler.Create)

	body := map[string]string{
		"clusterId":    "cluster-1",
		"resourceType": "pods",
		"resourceName": "my-pod",
	}
	w := performRequest(router, "POST", "/api/v1/favorites", body)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp authAPIResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, 0, resp.Code)

	var data favoriteData
	err = json.Unmarshal(resp.Data, &data)
	require.NoError(t, err)
	assert.Equal(t, "my-pod", data.DisplayName, "displayName should default to resourceName")
}

func TestFavoriteHandler_Create_Duplicate_ReturnsConflict(t *testing.T) {
	handler, _ := setupFavoriteHandler(t)

	router := gin.New()
	router.Use(injectUserID(1))
	router.POST("/api/v1/favorites", handler.Create)

	body := map[string]string{
		"clusterId":    "cluster-1",
		"resourceType": "deployments",
		"resourceName": "nginx",
		"namespace":    "default",
	}

	// First create should succeed.
	w := performRequest(router, "POST", "/api/v1/favorites", body)
	require.Equal(t, http.StatusOK, w.Code)
	var resp1 authAPIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp1))
	assert.Equal(t, 0, resp1.Code)

	// Second create of same resource should return conflict.
	w = performRequest(router, "POST", "/api/v1/favorites", body)
	assert.Equal(t, http.StatusOK, w.Code)
	var resp2 authAPIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp2))
	assert.Equal(t, 40900, resp2.Code, "duplicate favorite should return CodeConflict")
	assert.Contains(t, resp2.Message, "already favorited")
}

func TestFavoriteHandler_Create_EmptyBody_ReturnsParamInvalid(t *testing.T) {
	handler, _ := setupFavoriteHandler(t)

	router := gin.New()
	router.Use(injectUserID(1))
	router.POST("/api/v1/favorites", handler.Create)

	w := performRequest(router, "POST", "/api/v1/favorites", map[string]string{})

	assert.Equal(t, http.StatusOK, w.Code)

	var resp authAPIResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, 40002, resp.Code)
}

// ---------------------------------------------------------------------------
// Delete tests
// ---------------------------------------------------------------------------

func TestFavoriteHandler_Delete_Unauthorized(t *testing.T) {
	handler, _ := setupFavoriteHandler(t)

	router := gin.New()
	router.DELETE("/api/v1/favorites/:id", handler.Delete)

	w := performRequest(router, "DELETE", "/api/v1/favorites/1", nil)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp authAPIResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, 40100, resp.Code)
}

func TestFavoriteHandler_Delete_InvalidID(t *testing.T) {
	handler, _ := setupFavoriteHandler(t)

	router := gin.New()
	router.Use(injectUserID(1))
	router.DELETE("/api/v1/favorites/:id", handler.Delete)

	w := performRequest(router, "DELETE", "/api/v1/favorites/not-a-number", nil)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp authAPIResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, 40002, resp.Code, "non-numeric ID should return CodeParamInvalid")
}

func TestFavoriteHandler_Delete_NotFound(t *testing.T) {
	handler, _ := setupFavoriteHandler(t)

	router := gin.New()
	router.Use(injectUserID(1))
	router.DELETE("/api/v1/favorites/:id", handler.Delete)

	// Favorite ID 99999 does not exist.
	w := performRequest(router, "DELETE", "/api/v1/favorites/99999", nil)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp authAPIResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, 40400, resp.Code, "non-existent favorite should return CodeNotFound")
}

func TestFavoriteHandler_Delete_Success(t *testing.T) {
	handler, _ := setupFavoriteHandler(t)
	const testUserID uint = 1

	router := gin.New()
	router.Use(injectUserID(testUserID))
	router.POST("/api/v1/favorites", handler.Create)
	router.DELETE("/api/v1/favorites/:id", handler.Delete)
	router.GET("/api/v1/favorites", handler.List)

	// Create a favorite first.
	createBody := map[string]string{
		"clusterId":    "cluster-1",
		"resourceType": "deployments",
		"resourceName": "to-delete",
		"namespace":    "default",
	}
	w := performRequest(router, "POST", "/api/v1/favorites", createBody)
	require.Equal(t, http.StatusOK, w.Code)
	var createResp authAPIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &createResp))
	require.Equal(t, 0, createResp.Code)

	var created favoriteData
	require.NoError(t, json.Unmarshal(createResp.Data, &created))

	// Delete it.
	w = performRequest(router, "DELETE", fmt.Sprintf("/api/v1/favorites/%d", created.ID), nil)
	assert.Equal(t, http.StatusOK, w.Code)

	var deleteResp authAPIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &deleteResp))
	assert.Equal(t, 0, deleteResp.Code, "delete should succeed")

	// List should be empty now.
	w = performRequest(router, "GET", "/api/v1/favorites", nil)
	require.Equal(t, http.StatusOK, w.Code)
	var listResp authAPIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &listResp))
	var favs favoriteListData
	require.NoError(t, json.Unmarshal(listResp.Data, &favs))
	assert.Empty(t, favs, "list should be empty after delete")
}

// ---------------------------------------------------------------------------
// Toggle tests
// ---------------------------------------------------------------------------

func TestFavoriteHandler_Toggle_Unauthorized(t *testing.T) {
	handler, _ := setupFavoriteHandler(t)

	router := gin.New()
	router.POST("/api/v1/favorites/toggle", handler.Toggle)

	body := map[string]string{
		"clusterId":    "cluster-1",
		"resourceType": "deployments",
		"resourceName": "nginx",
	}
	w := performRequest(router, "POST", "/api/v1/favorites/toggle", body)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp authAPIResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, 40100, resp.Code)
}

func TestFavoriteHandler_Toggle_AddsWhenNotFavorited(t *testing.T) {
	handler, _ := setupFavoriteHandler(t)

	router := gin.New()
	router.Use(injectUserID(1))
	router.POST("/api/v1/favorites/toggle", handler.Toggle)

	body := map[string]string{
		"clusterId":    "cluster-1",
		"resourceType": "deployments",
		"resourceName": "nginx",
		"namespace":    "default",
	}
	w := performRequest(router, "POST", "/api/v1/favorites/toggle", body)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp authAPIResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, 0, resp.Code)

	var data toggleData
	err = json.Unmarshal(resp.Data, &data)
	require.NoError(t, err)
	assert.True(t, data.Favorited, "toggle should set favorited=true when not previously favorited")
	assert.NotNil(t, data.Favorite, "toggle should return the created favorite")
}

func TestFavoriteHandler_Toggle_RemovesWhenAlreadyFavorited(t *testing.T) {
	handler, _ := setupFavoriteHandler(t)

	router := gin.New()
	router.Use(injectUserID(1))
	router.POST("/api/v1/favorites/toggle", handler.Toggle)

	body := map[string]string{
		"clusterId":    "cluster-1",
		"resourceType": "deployments",
		"resourceName": "nginx",
		"namespace":    "default",
	}

	// First toggle: adds
	w := performRequest(router, "POST", "/api/v1/favorites/toggle", body)
	require.Equal(t, http.StatusOK, w.Code)
	var r1 authAPIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &r1))
	var d1 toggleData
	require.NoError(t, json.Unmarshal(r1.Data, &d1))
	assert.True(t, d1.Favorited)

	// Second toggle: removes
	w = performRequest(router, "POST", "/api/v1/favorites/toggle", body)
	assert.Equal(t, http.StatusOK, w.Code)
	var r2 authAPIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &r2))
	assert.Equal(t, 0, r2.Code)
	var d2 toggleData
	require.NoError(t, json.Unmarshal(r2.Data, &d2))
	assert.False(t, d2.Favorited, "second toggle should set favorited=false")
	assert.Nil(t, d2.Favorite, "second toggle should return nil favorite")
}

func TestFavoriteHandler_Toggle_MissingBody_ReturnsParamInvalid(t *testing.T) {
	handler, _ := setupFavoriteHandler(t)

	router := gin.New()
	router.Use(injectUserID(1))
	router.POST("/api/v1/favorites/toggle", handler.Toggle)

	w := performRequest(router, "POST", "/api/v1/favorites/toggle", map[string]string{})

	assert.Equal(t, http.StatusOK, w.Code)

	var resp authAPIResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, 40002, resp.Code)
}

// ---------------------------------------------------------------------------
// Reorder tests
// ---------------------------------------------------------------------------

func TestFavoriteHandler_Reorder_Unauthorized(t *testing.T) {
	handler, _ := setupFavoriteHandler(t)

	router := gin.New()
	router.PUT("/api/v1/favorites/reorder", handler.Reorder)

	body := map[string]interface{}{
		"orderedIds": []uint{1, 2, 3},
	}
	w := performRequest(router, "PUT", "/api/v1/favorites/reorder", body)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp authAPIResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, 40100, resp.Code)
}

func TestFavoriteHandler_Reorder_MissingBody_ReturnsParamInvalid(t *testing.T) {
	handler, _ := setupFavoriteHandler(t)

	router := gin.New()
	router.Use(injectUserID(1))
	router.PUT("/api/v1/favorites/reorder", handler.Reorder)

	w := performRequest(router, "PUT", "/api/v1/favorites/reorder", map[string]interface{}{})

	assert.Equal(t, http.StatusOK, w.Code)

	var resp authAPIResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, 40002, resp.Code, "missing orderedIds should return CodeParamInvalid")
}

func TestFavoriteHandler_Reorder_NonExistentID_ReturnsNotFound(t *testing.T) {
	handler, _ := setupFavoriteHandler(t)

	router := gin.New()
	router.Use(injectUserID(1))
	router.PUT("/api/v1/favorites/reorder", handler.Reorder)

	body := map[string]interface{}{
		"orderedIds": []uint{99999},
	}
	w := performRequest(router, "PUT", "/api/v1/favorites/reorder", body)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp authAPIResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, 40400, resp.Code, "reorder with unknown ID should return CodeNotFound")
}

func TestFavoriteHandler_Reorder_Success(t *testing.T) {
	handler, _ := setupFavoriteHandler(t)
	const testUserID uint = 1

	router := gin.New()
	router.Use(injectUserID(testUserID))
	router.POST("/api/v1/favorites", handler.Create)
	router.PUT("/api/v1/favorites/reorder", handler.Reorder)
	router.GET("/api/v1/favorites", handler.List)

	// Create two favorites.
	bodies := []map[string]string{
		{"clusterId": "cluster-1", "resourceType": "deployments", "resourceName": "alpha", "namespace": "default"},
		{"clusterId": "cluster-1", "resourceType": "deployments", "resourceName": "beta", "namespace": "default"},
	}
	var ids []uint
	for _, b := range bodies {
		w := performRequest(router, "POST", "/api/v1/favorites", b)
		require.Equal(t, http.StatusOK, w.Code)
		var resp authAPIResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		var d favoriteData
		require.NoError(t, json.Unmarshal(resp.Data, &d))
		ids = append(ids, d.ID)
	}

	// Reverse the order.
	reorderBody := map[string]interface{}{
		"orderedIds": []uint{ids[1], ids[0]},
	}
	w := performRequest(router, "PUT", "/api/v1/favorites/reorder", reorderBody)
	assert.Equal(t, http.StatusOK, w.Code)

	var reorderResp authAPIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &reorderResp))
	assert.Equal(t, 0, reorderResp.Code, "reorder should succeed")
}

// ---------------------------------------------------------------------------
// Check tests
// ---------------------------------------------------------------------------

func TestFavoriteHandler_Check_Unauthorized(t *testing.T) {
	handler, _ := setupFavoriteHandler(t)

	router := gin.New()
	router.GET("/api/v1/favorites/check", handler.Check)

	w := performRequest(router, "GET",
		"/api/v1/favorites/check?cluster_id=cluster-1&resource_type=deployments&name=nginx",
		nil,
	)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp authAPIResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, 40100, resp.Code)
}

func TestFavoriteHandler_Check_MissingRequiredParams(t *testing.T) {
	handler, _ := setupFavoriteHandler(t)

	router := gin.New()
	router.Use(injectUserID(1))
	router.GET("/api/v1/favorites/check", handler.Check)

	// Missing 'name'
	w := performRequest(router, "GET",
		"/api/v1/favorites/check?cluster_id=cluster-1&resource_type=deployments",
		nil,
	)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp authAPIResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, 40001, resp.Code, "missing name param should return CodeParamMissing")
}

func TestFavoriteHandler_Check_NotFavorited(t *testing.T) {
	handler, _ := setupFavoriteHandler(t)

	router := gin.New()
	router.Use(injectUserID(1))
	router.GET("/api/v1/favorites/check", handler.Check)

	w := performRequest(router, "GET",
		"/api/v1/favorites/check?cluster_id=cluster-1&resource_type=deployments&name=nginx&namespace=default",
		nil,
	)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp authAPIResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, 0, resp.Code)

	var data checkData
	err = json.Unmarshal(resp.Data, &data)
	require.NoError(t, err)
	assert.False(t, data.Favorited, "resource that is not favorited should return favorited=false")
	assert.Nil(t, data.Favorite)
}

func TestFavoriteHandler_Check_IsFavorited(t *testing.T) {
	handler, _ := setupFavoriteHandler(t)
	const testUserID uint = 1

	router := gin.New()
	router.Use(injectUserID(testUserID))
	router.POST("/api/v1/favorites", handler.Create)
	router.GET("/api/v1/favorites/check", handler.Check)

	// Add the favorite first.
	createBody := map[string]string{
		"clusterId":    "cluster-1",
		"resourceType": "deployments",
		"resourceName": "nginx",
		"namespace":    "default",
	}
	w := performRequest(router, "POST", "/api/v1/favorites", createBody)
	require.Equal(t, http.StatusOK, w.Code)

	// Now check it.
	w = performRequest(router, "GET",
		"/api/v1/favorites/check?cluster_id=cluster-1&resource_type=deployments&name=nginx&namespace=default",
		nil,
	)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp authAPIResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, 0, resp.Code)

	var data checkData
	err = json.Unmarshal(resp.Data, &data)
	require.NoError(t, err)
	assert.True(t, data.Favorited, "resource that is favorited should return favorited=true")
	assert.NotNil(t, data.Favorite, "favorited resource should return the favorite object")
}

func TestFavoriteHandler_Check_MissingClusterID(t *testing.T) {
	handler, _ := setupFavoriteHandler(t)

	router := gin.New()
	router.Use(injectUserID(1))
	router.GET("/api/v1/favorites/check", handler.Check)

	w := performRequest(router, "GET",
		"/api/v1/favorites/check?resource_type=deployments&name=nginx",
		nil,
	)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp authAPIResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, 40001, resp.Code, "missing cluster_id should return CodeParamMissing")
}
