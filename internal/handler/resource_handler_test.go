package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gocronx/kubevision/internal/kubernetes/resource"
	"github.com/gocronx/kubevision/internal/model"
	"github.com/gocronx/kubevision/internal/repository"
	"github.com/gocronx/kubevision/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Configurable stub K8s repo for resource handler tests
// ---------------------------------------------------------------------------

// fullStubK8sRepo is a stub that allows per-method overrides for all K8s
// operations used by ResourceService.
type fullStubK8sRepo struct {
	listFn         func(ctx context.Context, clusterID, kind, namespace string, opts repository.ListOptions) (*repository.ResourceList, error)
	getFn          func(ctx context.Context, clusterID, kind, namespace, name string) (*repository.Resource, error)
	createFn       func(ctx context.Context, clusterID, kind, namespace string, obj map[string]interface{}) (*repository.Resource, error)
	updateFn       func(ctx context.Context, clusterID, kind, namespace, name string, obj map[string]interface{}) (*repository.Resource, error)
	deleteFn       func(ctx context.Context, clusterID, kind, namespace, name string) error
	patchFn        func(ctx context.Context, clusterID, kind, namespace, name string, patchData []byte) (*repository.Resource, error)
	dryRunCreateFn func(ctx context.Context, clusterID, kind, namespace string, obj map[string]interface{}) (*repository.Resource, error)
	dryRunUpdateFn func(ctx context.Context, clusterID, kind, namespace, name string, obj map[string]interface{}) (*repository.Resource, *repository.Resource, error)
}

func (s *fullStubK8sRepo) List(ctx context.Context, clusterID, kind, namespace string, opts repository.ListOptions) (*repository.ResourceList, error) {
	if s.listFn != nil {
		return s.listFn(ctx, clusterID, kind, namespace, opts)
	}
	return &repository.ResourceList{Items: []repository.Resource{}, Total: 0}, nil
}

func (s *fullStubK8sRepo) Get(ctx context.Context, clusterID, kind, namespace, name string) (*repository.Resource, error) {
	if s.getFn != nil {
		return s.getFn(ctx, clusterID, kind, namespace, name)
	}
	return &repository.Resource{Name: name, Namespace: namespace, Kind: kind}, nil
}

func (s *fullStubK8sRepo) Create(ctx context.Context, clusterID, kind, namespace string, obj map[string]interface{}) (*repository.Resource, error) {
	if s.createFn != nil {
		return s.createFn(ctx, clusterID, kind, namespace, obj)
	}
	return &repository.Resource{Kind: kind, Namespace: namespace}, nil
}

func (s *fullStubK8sRepo) Update(ctx context.Context, clusterID, kind, namespace, name string, obj map[string]interface{}) (*repository.Resource, error) {
	if s.updateFn != nil {
		return s.updateFn(ctx, clusterID, kind, namespace, name, obj)
	}
	return &repository.Resource{Name: name, Kind: kind, Namespace: namespace}, nil
}

func (s *fullStubK8sRepo) Delete(ctx context.Context, clusterID, kind, namespace, name string) error {
	if s.deleteFn != nil {
		return s.deleteFn(ctx, clusterID, kind, namespace, name)
	}
	return nil
}

func (s *fullStubK8sRepo) Patch(ctx context.Context, clusterID, kind, namespace, name string, patchData []byte) (*repository.Resource, error) {
	if s.patchFn != nil {
		return s.patchFn(ctx, clusterID, kind, namespace, name, patchData)
	}
	return &repository.Resource{Name: name, Kind: kind, Namespace: namespace}, nil
}

func (s *fullStubK8sRepo) DryRunCreate(ctx context.Context, clusterID, kind, namespace string, obj map[string]interface{}) (*repository.Resource, error) {
	if s.dryRunCreateFn != nil {
		return s.dryRunCreateFn(ctx, clusterID, kind, namespace, obj)
	}
	return &repository.Resource{Kind: kind, Namespace: namespace}, nil
}

func (s *fullStubK8sRepo) DryRunUpdate(ctx context.Context, clusterID, kind, namespace, name string, obj map[string]interface{}) (*repository.Resource, *repository.Resource, error) {
	if s.dryRunUpdateFn != nil {
		return s.dryRunUpdateFn(ctx, clusterID, kind, namespace, name, obj)
	}
	current := &repository.Resource{Name: name, Kind: kind, Namespace: namespace}
	proposed := &repository.Resource{Name: name, Kind: kind, Namespace: namespace}
	return current, proposed, nil
}

// Compile-time interface check.
var _ repository.K8sResourceRepo = (*fullStubK8sRepo)(nil)

// ---------------------------------------------------------------------------
// Setup helpers
// ---------------------------------------------------------------------------

// setupResourceHandler creates a ResourceHandler with a real registry (so
// resource type validation passes for known types), the given K8s stub repo,
// and cluster ID 1 pre-seeded in the stub cluster repo.
func setupResourceHandler(t *testing.T, k8sRepo repository.K8sResourceRepo) *ResourceHandler {
	t.Helper()
	testCluster := &model.Cluster{ID: 1, Name: "test-cluster"}
	clusterRepo := newStubClusterRepo(testCluster)
	registry := resource.NewRegistry()
	resourceService := service.NewResourceService(k8sRepo, registry, clusterRepo)
	return NewResourceHandler(resourceService)
}

// setupResourceHandlerNoCluster creates a ResourceHandler where no clusters
// are registered, so every request fails with CodeNotFound.
func setupResourceHandlerNoCluster(t *testing.T) *ResourceHandler {
	t.Helper()
	clusterRepo := newStubClusterRepo()
	registry := resource.NewRegistry()
	k8sRepo := &fullStubK8sRepo{}
	resourceService := service.NewResourceService(k8sRepo, registry, clusterRepo)
	return NewResourceHandler(resourceService)
}

// performRequestRawBody sends an HTTP request with a raw byte body (used for
// tests that need to send exact JSON without going through json.Marshal).
func performRequestRawBody(r http.Handler, method, path string, body []byte) *httptest.ResponseRecorder {
	var buf *bytes.Buffer
	if body != nil {
		buf = bytes.NewBuffer(body)
	} else {
		buf = bytes.NewBuffer(nil)
	}
	req := httptest.NewRequest(method, path, buf)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// sampleDeploymentJSON is a minimal Deployment manifest for test requests.
const sampleDeploymentJSON = `{
  "apiVersion": "apps/v1",
  "kind": "Deployment",
  "metadata": {"name": "my-deploy", "namespace": "default"},
  "spec": {
    "replicas": 1,
    "selector": {"matchLabels": {"app": "my-deploy"}},
    "template": {
      "metadata": {"labels": {"app": "my-deploy"}},
      "spec": {"containers": [{"name": "c", "image": "nginx:latest"}]}
    }
  }
}`

// ---------------------------------------------------------------------------
// List tests
// ---------------------------------------------------------------------------

func TestResourceHandler_List_InvalidClusterID(t *testing.T) {
	handler := setupResourceHandler(t, &fullStubK8sRepo{})

	router := gin.New()
	router.GET("/api/v1/clusters/:id/resources/:resource", handler.List)

	w := performRequest(router, "GET", "/api/v1/clusters/abc/resources/pods", nil)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp authAPIResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, 40002, resp.Code, "non-numeric cluster id should return CodeParamInvalid")
}

func TestResourceHandler_List_ClusterNotFound(t *testing.T) {
	handler := setupResourceHandlerNoCluster(t)

	router := gin.New()
	router.GET("/api/v1/clusters/:id/resources/:resource", handler.List)

	w := performRequest(router, "GET", "/api/v1/clusters/9999/resources/pods", nil)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp authAPIResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, 40400, resp.Code, "unknown cluster should return CodeNotFound")
}

func TestResourceHandler_List_UnknownResourceType(t *testing.T) {
	handler := setupResourceHandler(t, &fullStubK8sRepo{})

	router := gin.New()
	router.GET("/api/v1/clusters/:id/resources/:resource", handler.List)

	w := performRequest(router, "GET", "/api/v1/clusters/1/resources/unknowntype", nil)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp authAPIResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, 40002, resp.Code, "unknown resource type should return CodeParamInvalid")
}

func TestResourceHandler_List_Success(t *testing.T) {
	k8sRepo := &fullStubK8sRepo{
		listFn: func(_ context.Context, _, _, _ string, _ repository.ListOptions) (*repository.ResourceList, error) {
			return &repository.ResourceList{
				Items: []repository.Resource{
					{Name: "pod-1", Namespace: "default", Kind: "Pod"},
					{Name: "pod-2", Namespace: "default", Kind: "Pod"},
				},
				Total: 2,
			}, nil
		},
	}
	handler := setupResourceHandler(t, k8sRepo)

	router := gin.New()
	router.GET("/api/v1/clusters/:id/resources/:resource", handler.List)

	w := performRequest(router, "GET", "/api/v1/clusters/1/resources/pods?namespace=default", nil)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp authAPIResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, 0, resp.Code, "list should succeed")
}

// ---------------------------------------------------------------------------
// Get tests
// ---------------------------------------------------------------------------

func TestResourceHandler_Get_InvalidClusterID(t *testing.T) {
	handler := setupResourceHandler(t, &fullStubK8sRepo{})

	router := gin.New()
	router.GET("/api/v1/clusters/:id/resources/:resource/:name", handler.Get)

	w := performRequest(router, "GET", "/api/v1/clusters/xyz/resources/pods/my-pod", nil)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp authAPIResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, 40002, resp.Code)
}

func TestResourceHandler_Get_ClusterNotFound(t *testing.T) {
	handler := setupResourceHandlerNoCluster(t)

	router := gin.New()
	router.GET("/api/v1/clusters/:id/resources/:resource/:name", handler.Get)

	w := performRequest(router, "GET", "/api/v1/clusters/9999/resources/pods/my-pod", nil)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp authAPIResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, 40400, resp.Code)
}

func TestResourceHandler_Get_UnknownResourceType(t *testing.T) {
	handler := setupResourceHandler(t, &fullStubK8sRepo{})

	router := gin.New()
	router.GET("/api/v1/clusters/:id/resources/:resource/:name", handler.Get)

	w := performRequest(router, "GET", "/api/v1/clusters/1/resources/widgets/my-widget", nil)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp authAPIResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, 40002, resp.Code)
}

func TestResourceHandler_Get_Success(t *testing.T) {
	k8sRepo := &fullStubK8sRepo{
		getFn: func(_ context.Context, _, _, ns, nm string) (*repository.Resource, error) {
			return &repository.Resource{Name: nm, Namespace: ns, Kind: "Pod"}, nil
		},
	}
	handler := setupResourceHandler(t, k8sRepo)

	router := gin.New()
	router.GET("/api/v1/clusters/:id/resources/:resource/:name", handler.Get)

	w := performRequest(router, "GET", "/api/v1/clusters/1/resources/pods/my-pod?namespace=default", nil)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp authAPIResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, 0, resp.Code)
}

// ---------------------------------------------------------------------------
// Create tests
// ---------------------------------------------------------------------------

func TestResourceHandler_Create_InvalidClusterID(t *testing.T) {
	handler := setupResourceHandler(t, &fullStubK8sRepo{})

	router := gin.New()
	router.POST("/api/v1/clusters/:id/resources/:resource", handler.Create)

	w := performRequestRawBody(router, "POST", "/api/v1/clusters/bad/resources/deployments", []byte(sampleDeploymentJSON))

	assert.Equal(t, http.StatusOK, w.Code)

	var resp authAPIResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, 40002, resp.Code)
}

func TestResourceHandler_Create_EmptyBody(t *testing.T) {
	handler := setupResourceHandler(t, &fullStubK8sRepo{})

	router := gin.New()
	router.POST("/api/v1/clusters/:id/resources/:resource", handler.Create)

	// Nil body — should return CodeParamMissing.
	w := performRequestRawBody(router, "POST", "/api/v1/clusters/1/resources/deployments", nil)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp authAPIResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, 40001, resp.Code, "empty body should return CodeParamMissing")
}

func TestResourceHandler_Create_ClusterNotFound(t *testing.T) {
	handler := setupResourceHandlerNoCluster(t)

	router := gin.New()
	router.POST("/api/v1/clusters/:id/resources/:resource", handler.Create)

	w := performRequestRawBody(router, "POST", "/api/v1/clusters/9999/resources/deployments", []byte(sampleDeploymentJSON))

	assert.Equal(t, http.StatusOK, w.Code)

	var resp authAPIResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, 40400, resp.Code)
}

func TestResourceHandler_Create_Success(t *testing.T) {
	k8sRepo := &fullStubK8sRepo{
		createFn: func(_ context.Context, _, kind, ns string, _ map[string]interface{}) (*repository.Resource, error) {
			return &repository.Resource{Kind: kind, Namespace: ns, Name: "my-deploy"}, nil
		},
	}
	handler := setupResourceHandler(t, k8sRepo)

	router := gin.New()
	router.POST("/api/v1/clusters/:id/resources/:resource", handler.Create)

	w := performRequestRawBody(router, "POST", "/api/v1/clusters/1/resources/deployments?namespace=default", []byte(sampleDeploymentJSON))

	assert.Equal(t, http.StatusOK, w.Code)

	var resp authAPIResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, 0, resp.Code, "create should succeed")
}

// ---------------------------------------------------------------------------
// Update tests
// ---------------------------------------------------------------------------

func TestResourceHandler_Update_InvalidClusterID(t *testing.T) {
	handler := setupResourceHandler(t, &fullStubK8sRepo{})

	router := gin.New()
	router.PUT("/api/v1/clusters/:id/resources/:resource/:name", handler.Update)

	w := performRequestRawBody(router, "PUT", "/api/v1/clusters/bad/resources/deployments/my-deploy", []byte(sampleDeploymentJSON))

	assert.Equal(t, http.StatusOK, w.Code)

	var resp authAPIResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, 40002, resp.Code)
}

func TestResourceHandler_Update_EmptyBody(t *testing.T) {
	handler := setupResourceHandler(t, &fullStubK8sRepo{})

	router := gin.New()
	router.PUT("/api/v1/clusters/:id/resources/:resource/:name", handler.Update)

	w := performRequestRawBody(router, "PUT", "/api/v1/clusters/1/resources/deployments/my-deploy", nil)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp authAPIResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, 40001, resp.Code, "empty body should return CodeParamMissing")
}

func TestResourceHandler_Update_Success(t *testing.T) {
	k8sRepo := &fullStubK8sRepo{
		updateFn: func(_ context.Context, _, kind, ns, name string, _ map[string]interface{}) (*repository.Resource, error) {
			return &repository.Resource{Kind: kind, Namespace: ns, Name: name}, nil
		},
	}
	handler := setupResourceHandler(t, k8sRepo)

	router := gin.New()
	router.PUT("/api/v1/clusters/:id/resources/:resource/:name", handler.Update)

	w := performRequestRawBody(router, "PUT", "/api/v1/clusters/1/resources/deployments/my-deploy?namespace=default", []byte(sampleDeploymentJSON))

	assert.Equal(t, http.StatusOK, w.Code)

	var resp authAPIResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, 0, resp.Code, "update should succeed")
}

// ---------------------------------------------------------------------------
// Delete tests
// ---------------------------------------------------------------------------

func TestResourceHandler_Delete_InvalidClusterID(t *testing.T) {
	handler := setupResourceHandler(t, &fullStubK8sRepo{})

	router := gin.New()
	router.DELETE("/api/v1/clusters/:id/resources/:resource/:name", handler.Delete)

	w := performRequest(router, "DELETE", "/api/v1/clusters/xyz/resources/pods/my-pod", nil)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp authAPIResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, 40002, resp.Code)
}

func TestResourceHandler_Delete_ClusterNotFound(t *testing.T) {
	handler := setupResourceHandlerNoCluster(t)

	router := gin.New()
	router.DELETE("/api/v1/clusters/:id/resources/:resource/:name", handler.Delete)

	w := performRequest(router, "DELETE", "/api/v1/clusters/9999/resources/pods/my-pod", nil)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp authAPIResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, 40400, resp.Code)
}

func TestResourceHandler_Delete_Success(t *testing.T) {
	handler := setupResourceHandler(t, &fullStubK8sRepo{})

	router := gin.New()
	router.DELETE("/api/v1/clusters/:id/resources/:resource/:name", handler.Delete)

	w := performRequest(router, "DELETE", "/api/v1/clusters/1/resources/pods/my-pod?namespace=default", nil)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp authAPIResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, 0, resp.Code, "delete should succeed")
}

// ---------------------------------------------------------------------------
// Patch tests
// ---------------------------------------------------------------------------

func TestResourceHandler_Patch_InvalidClusterID(t *testing.T) {
	handler := setupResourceHandler(t, &fullStubK8sRepo{})

	router := gin.New()
	router.PATCH("/api/v1/clusters/:id/resources/:resource/:name", handler.Patch)

	w := performRequestRawBody(router, "PATCH", "/api/v1/clusters/bad/resources/deployments/my-deploy", []byte(`{"spec":{"replicas":2}}`))

	assert.Equal(t, http.StatusOK, w.Code)

	var resp authAPIResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, 40002, resp.Code)
}

func TestResourceHandler_Patch_EmptyBody(t *testing.T) {
	handler := setupResourceHandler(t, &fullStubK8sRepo{})

	router := gin.New()
	router.PATCH("/api/v1/clusters/:id/resources/:resource/:name", handler.Patch)

	w := performRequestRawBody(router, "PATCH", "/api/v1/clusters/1/resources/deployments/my-deploy", nil)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp authAPIResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, 40001, resp.Code, "empty body should return CodeParamMissing")
}

func TestResourceHandler_Patch_Success(t *testing.T) {
	k8sRepo := &fullStubK8sRepo{
		patchFn: func(_ context.Context, _, kind, ns, name string, _ []byte) (*repository.Resource, error) {
			return &repository.Resource{Kind: kind, Namespace: ns, Name: name}, nil
		},
	}
	handler := setupResourceHandler(t, k8sRepo)

	router := gin.New()
	router.PATCH("/api/v1/clusters/:id/resources/:resource/:name", handler.Patch)

	w := performRequestRawBody(router, "PATCH", "/api/v1/clusters/1/resources/deployments/my-deploy?namespace=default", []byte(`{"spec":{"replicas":3}}`))

	assert.Equal(t, http.StatusOK, w.Code)

	var resp authAPIResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, 0, resp.Code, "patch should succeed")
}

// ---------------------------------------------------------------------------
// DryRunCreate tests
// ---------------------------------------------------------------------------

func TestResourceHandler_DryRunCreate_InvalidClusterID(t *testing.T) {
	handler := setupResourceHandler(t, &fullStubK8sRepo{})

	router := gin.New()
	router.POST("/api/v1/clusters/:id/resources/:resource/dry-run", handler.DryRunCreate)

	w := performRequestRawBody(router, "POST", "/api/v1/clusters/abc/resources/deployments/dry-run", []byte(sampleDeploymentJSON))

	assert.Equal(t, http.StatusOK, w.Code)

	var resp authAPIResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, 40002, resp.Code, "non-numeric cluster id should return CodeParamInvalid")
}

func TestResourceHandler_DryRunCreate_EmptyBody(t *testing.T) {
	handler := setupResourceHandler(t, &fullStubK8sRepo{})

	router := gin.New()
	router.POST("/api/v1/clusters/:id/resources/:resource/dry-run", handler.DryRunCreate)

	w := performRequestRawBody(router, "POST", "/api/v1/clusters/1/resources/deployments/dry-run", nil)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp authAPIResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, 40001, resp.Code, "empty body should return CodeParamMissing")
}

func TestResourceHandler_DryRunCreate_ClusterNotFound(t *testing.T) {
	handler := setupResourceHandlerNoCluster(t)

	router := gin.New()
	router.POST("/api/v1/clusters/:id/resources/:resource/dry-run", handler.DryRunCreate)

	w := performRequestRawBody(router, "POST", "/api/v1/clusters/9999/resources/deployments/dry-run", []byte(sampleDeploymentJSON))

	assert.Equal(t, http.StatusOK, w.Code)

	var resp authAPIResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, 40400, resp.Code, "non-existent cluster should return CodeNotFound")
}

func TestResourceHandler_DryRunCreate_UnknownResourceType(t *testing.T) {
	handler := setupResourceHandler(t, &fullStubK8sRepo{})

	router := gin.New()
	router.POST("/api/v1/clusters/:id/resources/:resource/dry-run", handler.DryRunCreate)

	w := performRequestRawBody(router, "POST", "/api/v1/clusters/1/resources/unknownkind/dry-run", []byte(sampleDeploymentJSON))

	assert.Equal(t, http.StatusOK, w.Code)

	var resp authAPIResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, 40002, resp.Code, "unknown resource type should return CodeParamInvalid")
}

func TestResourceHandler_DryRunCreate_InvalidJSON(t *testing.T) {
	handler := setupResourceHandler(t, &fullStubK8sRepo{})

	router := gin.New()
	router.POST("/api/v1/clusters/:id/resources/:resource/dry-run", handler.DryRunCreate)

	// Malformed JSON body
	w := performRequestRawBody(router, "POST", "/api/v1/clusters/1/resources/deployments/dry-run?namespace=default", []byte(`{not valid json`))

	assert.Equal(t, http.StatusOK, w.Code)

	var resp authAPIResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, 40002, resp.Code, "malformed JSON body should return CodeParamInvalid")
}

func TestResourceHandler_DryRunCreate_Success_ValidResult(t *testing.T) {
	proposedResource := &repository.Resource{
		APIVersion: "apps/v1",
		Kind:       "Deployment",
		Name:       "my-deploy",
		Namespace:  "default",
		Raw:        map[string]interface{}{"spec": map[string]interface{}{"replicas": float64(1)}},
	}

	k8sRepo := &fullStubK8sRepo{
		dryRunCreateFn: func(_ context.Context, _, _, _ string, _ map[string]interface{}) (*repository.Resource, error) {
			return proposedResource, nil
		},
	}
	handler := setupResourceHandler(t, k8sRepo)

	router := gin.New()
	router.POST("/api/v1/clusters/:id/resources/:resource/dry-run", handler.DryRunCreate)

	w := performRequestRawBody(router, "POST", "/api/v1/clusters/1/resources/deployments/dry-run?namespace=default", []byte(sampleDeploymentJSON))

	assert.Equal(t, http.StatusOK, w.Code)

	var resp authAPIResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, 0, resp.Code, "successful dry-run create should return code 0")

	var result service.DryRunResult
	err = json.Unmarshal(resp.Data, &result)
	require.NoError(t, err)
	assert.True(t, result.Valid, "dry-run create result should be valid")
	assert.Nil(t, result.Current, "dry-run create should have nil Current (nothing existed before)")
	require.NotNil(t, result.Proposed, "dry-run create should return a Proposed resource")
	assert.Equal(t, "my-deploy", result.Proposed.Name)
}

func TestResourceHandler_DryRunCreate_ValidationFailure(t *testing.T) {
	// When the K8s dry-run returns an error, the service surfaces it as
	// Valid=false with an error message, not as an HTTP error code.
	k8sRepo := &fullStubK8sRepo{
		dryRunCreateFn: func(_ context.Context, _, _, _ string, _ map[string]interface{}) (*repository.Resource, error) {
			return nil, fmt.Errorf("spec.replicas: Invalid value: -1: must be greater than or equal to 0")
		},
	}
	handler := setupResourceHandler(t, k8sRepo)

	router := gin.New()
	router.POST("/api/v1/clusters/:id/resources/:resource/dry-run", handler.DryRunCreate)

	w := performRequestRawBody(router, "POST", "/api/v1/clusters/1/resources/deployments/dry-run?namespace=default", []byte(sampleDeploymentJSON))

	assert.Equal(t, http.StatusOK, w.Code)

	var resp authAPIResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	// The service returns code 0 but with Valid=false in the payload.
	assert.Equal(t, 0, resp.Code)

	var result service.DryRunResult
	err = json.Unmarshal(resp.Data, &result)
	require.NoError(t, err)
	assert.False(t, result.Valid, "K8s validation failure should produce Valid=false")
	assert.NotEmpty(t, result.Errors, "validation failure should include at least one error message")
}

func TestResourceHandler_DryRunCreate_ResponseShape(t *testing.T) {
	k8sRepo := &fullStubK8sRepo{
		dryRunCreateFn: func(_ context.Context, _, _, _ string, _ map[string]interface{}) (*repository.Resource, error) {
			return &repository.Resource{Name: "my-deploy", Kind: "Deployment"}, nil
		},
	}
	handler := setupResourceHandler(t, k8sRepo)

	router := gin.New()
	router.POST("/api/v1/clusters/:id/resources/:resource/dry-run", handler.DryRunCreate)

	w := performRequestRawBody(router, "POST", "/api/v1/clusters/1/resources/deployments/dry-run?namespace=default", []byte(sampleDeploymentJSON))

	require.Equal(t, http.StatusOK, w.Code)

	// Inspect raw keys in the data payload.
	var outer map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &outer))
	require.Contains(t, outer, "data")

	var data map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(outer["data"], &data))
	assert.Contains(t, data, "valid", "data should have 'valid' field")
	assert.Contains(t, data, "proposed", "data should have 'proposed' field")
}

// ---------------------------------------------------------------------------
// DryRunUpdate tests
// ---------------------------------------------------------------------------

func TestResourceHandler_DryRunUpdate_InvalidClusterID(t *testing.T) {
	handler := setupResourceHandler(t, &fullStubK8sRepo{})

	router := gin.New()
	router.PUT("/api/v1/clusters/:id/resources/:resource/:name/dry-run", handler.DryRunUpdate)

	w := performRequestRawBody(router, "PUT", "/api/v1/clusters/bad/resources/deployments/my-deploy/dry-run", []byte(sampleDeploymentJSON))

	assert.Equal(t, http.StatusOK, w.Code)

	var resp authAPIResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, 40002, resp.Code, "non-numeric cluster id should return CodeParamInvalid")
}

func TestResourceHandler_DryRunUpdate_EmptyBody(t *testing.T) {
	handler := setupResourceHandler(t, &fullStubK8sRepo{})

	router := gin.New()
	router.PUT("/api/v1/clusters/:id/resources/:resource/:name/dry-run", handler.DryRunUpdate)

	w := performRequestRawBody(router, "PUT", "/api/v1/clusters/1/resources/deployments/my-deploy/dry-run", nil)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp authAPIResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, 40001, resp.Code, "empty body should return CodeParamMissing")
}

func TestResourceHandler_DryRunUpdate_ClusterNotFound(t *testing.T) {
	handler := setupResourceHandlerNoCluster(t)

	router := gin.New()
	router.PUT("/api/v1/clusters/:id/resources/:resource/:name/dry-run", handler.DryRunUpdate)

	w := performRequestRawBody(router, "PUT", "/api/v1/clusters/9999/resources/deployments/my-deploy/dry-run", []byte(sampleDeploymentJSON))

	assert.Equal(t, http.StatusOK, w.Code)

	var resp authAPIResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, 40400, resp.Code, "non-existent cluster should return CodeNotFound")
}

func TestResourceHandler_DryRunUpdate_UnknownResourceType(t *testing.T) {
	handler := setupResourceHandler(t, &fullStubK8sRepo{})

	router := gin.New()
	router.PUT("/api/v1/clusters/:id/resources/:resource/:name/dry-run", handler.DryRunUpdate)

	w := performRequestRawBody(router, "PUT", "/api/v1/clusters/1/resources/unknownkind/my-thing/dry-run", []byte(sampleDeploymentJSON))

	assert.Equal(t, http.StatusOK, w.Code)

	var resp authAPIResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, 40002, resp.Code, "unknown resource type should return CodeParamInvalid")
}

func TestResourceHandler_DryRunUpdate_InvalidJSON(t *testing.T) {
	handler := setupResourceHandler(t, &fullStubK8sRepo{})

	router := gin.New()
	router.PUT("/api/v1/clusters/:id/resources/:resource/:name/dry-run", handler.DryRunUpdate)

	w := performRequestRawBody(router, "PUT", "/api/v1/clusters/1/resources/deployments/my-deploy/dry-run?namespace=default", []byte(`{bad json`))

	assert.Equal(t, http.StatusOK, w.Code)

	var resp authAPIResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, 40002, resp.Code, "malformed JSON body should return CodeParamInvalid")
}

func TestResourceHandler_DryRunUpdate_Success_BothCurrentAndProposed(t *testing.T) {
	currentResource := &repository.Resource{
		APIVersion: "apps/v1",
		Kind:       "Deployment",
		Name:       "my-deploy",
		Namespace:  "default",
		Raw:        map[string]interface{}{"spec": map[string]interface{}{"replicas": float64(1)}},
	}
	proposedResource := &repository.Resource{
		APIVersion: "apps/v1",
		Kind:       "Deployment",
		Name:       "my-deploy",
		Namespace:  "default",
		Raw:        map[string]interface{}{"spec": map[string]interface{}{"replicas": float64(3)}},
	}

	k8sRepo := &fullStubK8sRepo{
		dryRunUpdateFn: func(_ context.Context, _, _, _, _ string, _ map[string]interface{}) (*repository.Resource, *repository.Resource, error) {
			return currentResource, proposedResource, nil
		},
	}
	handler := setupResourceHandler(t, k8sRepo)

	router := gin.New()
	router.PUT("/api/v1/clusters/:id/resources/:resource/:name/dry-run", handler.DryRunUpdate)

	w := performRequestRawBody(router, "PUT", "/api/v1/clusters/1/resources/deployments/my-deploy/dry-run?namespace=default", []byte(sampleDeploymentJSON))

	assert.Equal(t, http.StatusOK, w.Code)

	var resp authAPIResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, 0, resp.Code, "successful dry-run update should return code 0")

	var result service.DryRunResult
	err = json.Unmarshal(resp.Data, &result)
	require.NoError(t, err)
	assert.True(t, result.Valid)
	require.NotNil(t, result.Current, "dry-run update should return the Current (live) resource")
	require.NotNil(t, result.Proposed, "dry-run update should return the Proposed resource")
	assert.Equal(t, "my-deploy", result.Current.Name)
	assert.Equal(t, "my-deploy", result.Proposed.Name)
}

func TestResourceHandler_DryRunUpdate_ValidationFailure(t *testing.T) {
	k8sRepo := &fullStubK8sRepo{
		dryRunUpdateFn: func(_ context.Context, _, _, _, _ string, _ map[string]interface{}) (*repository.Resource, *repository.Resource, error) {
			return nil, nil, fmt.Errorf("metadata.resourceVersion: Required value: must be specified for an update")
		},
	}
	handler := setupResourceHandler(t, k8sRepo)

	router := gin.New()
	router.PUT("/api/v1/clusters/:id/resources/:resource/:name/dry-run", handler.DryRunUpdate)

	w := performRequestRawBody(router, "PUT", "/api/v1/clusters/1/resources/deployments/my-deploy/dry-run?namespace=default", []byte(sampleDeploymentJSON))

	assert.Equal(t, http.StatusOK, w.Code)

	var resp authAPIResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	// Service surfaces validation errors as code=0 / Valid=false.
	assert.Equal(t, 0, resp.Code)

	var result service.DryRunResult
	err = json.Unmarshal(resp.Data, &result)
	require.NoError(t, err)
	assert.False(t, result.Valid, "K8s validation failure should produce Valid=false")
	assert.NotEmpty(t, result.Errors, "validation failure should include error messages")
}

func TestResourceHandler_DryRunUpdate_ResponseShape(t *testing.T) {
	k8sRepo := &fullStubK8sRepo{
		dryRunUpdateFn: func(_ context.Context, _, _, _, _ string, _ map[string]interface{}) (*repository.Resource, *repository.Resource, error) {
			current := &repository.Resource{Name: "my-deploy"}
			proposed := &repository.Resource{Name: "my-deploy"}
			return current, proposed, nil
		},
	}
	handler := setupResourceHandler(t, k8sRepo)

	router := gin.New()
	router.PUT("/api/v1/clusters/:id/resources/:resource/:name/dry-run", handler.DryRunUpdate)

	w := performRequestRawBody(router, "PUT", "/api/v1/clusters/1/resources/deployments/my-deploy/dry-run?namespace=default", []byte(sampleDeploymentJSON))

	require.Equal(t, http.StatusOK, w.Code)

	var outer map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &outer))
	require.Contains(t, outer, "data")

	var data map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(outer["data"], &data))
	assert.Contains(t, data, "valid", "response data should have 'valid' field")
	assert.Contains(t, data, "proposed", "response data should have 'proposed' field")
	assert.Contains(t, data, "current", "response data should have 'current' field")
}
