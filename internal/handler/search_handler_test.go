package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gocronx/kubevision/internal/kubernetes/cluster"
	"github.com/gocronx/kubevision/internal/kubernetes/informer"
	"github.com/gocronx/kubevision/internal/kubernetes/resource"
	"github.com/gocronx/kubevision/internal/model"
	"github.com/gocronx/kubevision/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// searchClusterRepo reuses stubClusterRepo defined in resource_action_handler_test.go.

// newSearchHandler creates a SearchHandler backed by a SearchService with an
// empty cluster manager and an empty informer manager. Cluster ID 1 is seeded
// so that the service's cluster lookup succeeds; subsequent K8s calls will
// simply return no results because neither the informer nor the dynamic client
// is set up.
func newSearchHandler(t *testing.T) *SearchHandler {
	t.Helper()

	testCluster := &model.Cluster{
		ID:   1,
		Name: "test-cluster",
	}
	clusterRepo := newStubClusterRepo(testCluster)
	logger, _ := zap.NewDevelopment()
	informerMgr := informer.NewManager(logger)
	clusterMgr := cluster.NewManager()
	registry := resource.NewRegistry()

	searchService := service.NewSearchService(informerMgr, clusterMgr, registry, clusterRepo)
	return NewSearchHandler(searchService)
}

// searchClusterNotFoundHandler creates a SearchHandler whose cluster repo has
// no seeded clusters so that every request returns CodeNotFound.
func newSearchHandlerNoCluster(t *testing.T) *SearchHandler {
	t.Helper()

	clusterRepo := newStubClusterRepo() // empty
	logger, _ := zap.NewDevelopment()
	informerMgr := informer.NewManager(logger)
	clusterMgr := cluster.NewManager()
	registry := resource.NewRegistry()

	searchService := service.NewSearchService(informerMgr, clusterMgr, registry, clusterRepo)
	return NewSearchHandler(searchService)
}

// Suppress unused import warning – context is used in stubClusterRepoForSearch.
var _ context.Context

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestSearchHandler_Search_InvalidClusterID(t *testing.T) {
	handler := newSearchHandler(t)

	router := gin.New()
	router.GET("/api/v1/clusters/:id/search", handler.Search)

	w := performRequest(router, "GET", "/api/v1/clusters/abc/search?q=nginx", nil)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp authAPIResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, 40002, resp.Code, "should return CodeParamInvalid for non-numeric cluster id")
}

func TestSearchHandler_Search_MissingQueryParam(t *testing.T) {
	handler := newSearchHandler(t)

	router := gin.New()
	router.GET("/api/v1/clusters/:id/search", handler.Search)

	// 'q' parameter is absent
	w := performRequest(router, "GET", "/api/v1/clusters/1/search", nil)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp authAPIResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, 40001, resp.Code, "should return CodeParamMissing when 'q' is absent")
	assert.Contains(t, resp.Message, "'q'")
}

func TestSearchHandler_Search_EmptyQuery(t *testing.T) {
	handler := newSearchHandler(t)

	router := gin.New()
	router.GET("/api/v1/clusters/:id/search", handler.Search)

	// 'q' is a whitespace-only string (percent-encoded spaces).
	w := performRequest(router, "GET", "/api/v1/clusters/1/search?q=%20%20%20", nil)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp authAPIResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, 40001, resp.Code, "should return CodeParamMissing for blank query")
}

func TestSearchHandler_Search_ClusterNotFound(t *testing.T) {
	handler := newSearchHandlerNoCluster(t)

	router := gin.New()
	router.GET("/api/v1/clusters/:id/search", handler.Search)

	w := performRequest(router, "GET", "/api/v1/clusters/9999/search?q=nginx", nil)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp authAPIResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, 40400, resp.Code, "should return CodeNotFound for unknown cluster")
}

func TestSearchHandler_Search_ValidQuery_EmptyRegistry(t *testing.T) {
	// When the registry has no resource types, Search should return an empty
	// result set without error.
	handler := newSearchHandler(t)

	router := gin.New()
	router.GET("/api/v1/clusters/:id/search", handler.Search)

	w := performRequest(router, "GET", "/api/v1/clusters/1/search?q=nginx", nil)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp authAPIResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, 0, resp.Code, "should return success code when registry is empty")
	assert.Equal(t, "success", resp.Message)

	// Data should decode to a SearchResponse with empty results.
	var data service.SearchResponse
	err = json.Unmarshal(resp.Data, &data)
	require.NoError(t, err)
	assert.Equal(t, 0, data.Total)
	assert.Empty(t, data.Results)
}

func TestSearchHandler_Search_TypesFilterParsed(t *testing.T) {
	// Verify that the handler accepts and passes the 'types' query param.
	// With an empty registry no resources match; we only check the response code.
	handler := newSearchHandler(t)

	router := gin.New()
	router.GET("/api/v1/clusters/:id/search", handler.Search)

	w := performRequest(router, "GET", "/api/v1/clusters/1/search?q=nginx&types=pods,deployments", nil)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp authAPIResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, 0, resp.Code, "should return success with valid types filter")
}

func TestSearchHandler_Search_WithNamespaceFilter(t *testing.T) {
	handler := newSearchHandler(t)

	router := gin.New()
	router.GET("/api/v1/clusters/:id/search", handler.Search)

	w := performRequest(router, "GET", "/api/v1/clusters/1/search?q=nginx&namespace=kube-system", nil)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp authAPIResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, 0, resp.Code, "should return success with namespace filter")
}

func TestSearchHandler_Search_WithPaginationParams(t *testing.T) {
	handler := newSearchHandler(t)

	router := gin.New()
	router.GET("/api/v1/clusters/:id/search", handler.Search)

	w := performRequest(router, "GET", "/api/v1/clusters/1/search?q=pod&limit=5&offset=10", nil)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp authAPIResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, 0, resp.Code)
}

func TestSearchHandler_Search_ResponseHasMeta(t *testing.T) {
	// Verify that the response structure contains a "meta" field with the
	// total count (which will be 0 with an empty registry).
	handler := newSearchHandler(t)

	router := gin.New()
	router.GET("/api/v1/clusters/:id/search", handler.Search)

	w := performRequest(router, "GET", "/api/v1/clusters/1/search?q=nginx", nil)

	assert.Equal(t, http.StatusOK, w.Code)

	// Unmarshal into a raw map so we can inspect the "meta" key.
	var raw map[string]json.RawMessage
	err := json.Unmarshal(w.Body.Bytes(), &raw)
	require.NoError(t, err)
	assert.Contains(t, raw, "meta", "response should contain a 'meta' field")
}
