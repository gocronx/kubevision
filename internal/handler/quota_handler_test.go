package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/kubevision/kubevision/internal/model"
	"github.com/kubevision/kubevision/internal/repository"
	"github.com/kubevision/kubevision/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Stub K8s resource repo for quota tests
// ---------------------------------------------------------------------------

// stubK8sResourceRepo implements repository.K8sResourceRepo for test use.
// List returns an empty ResourceList by default; callers can override
// listFn to inject specific results or errors.
type stubK8sResourceRepo struct {
	listFn func(ctx context.Context, clusterID, kind, namespace string, opts repository.ListOptions) (*repository.ResourceList, error)
}

func (s *stubK8sResourceRepo) List(ctx context.Context, clusterID, kind, namespace string, opts repository.ListOptions) (*repository.ResourceList, error) {
	if s.listFn != nil {
		return s.listFn(ctx, clusterID, kind, namespace, opts)
	}
	return &repository.ResourceList{Items: []repository.Resource{}, Total: 0}, nil
}

func (s *stubK8sResourceRepo) Get(_ context.Context, _, _, _, _ string) (*repository.Resource, error) {
	return nil, nil
}

func (s *stubK8sResourceRepo) Create(_ context.Context, _, _, _ string, _ map[string]interface{}) (*repository.Resource, error) {
	return nil, nil
}

func (s *stubK8sResourceRepo) Update(_ context.Context, _, _, _, _ string, _ map[string]interface{}) (*repository.Resource, error) {
	return nil, nil
}

func (s *stubK8sResourceRepo) Delete(_ context.Context, _, _, _, _ string) error {
	return nil
}

func (s *stubK8sResourceRepo) Patch(_ context.Context, _, _, _, _ string, _ []byte) (*repository.Resource, error) {
	return nil, nil
}

func (s *stubK8sResourceRepo) DryRunCreate(_ context.Context, _, _, _ string, _ map[string]interface{}) (*repository.Resource, error) {
	return nil, nil
}

func (s *stubK8sResourceRepo) DryRunUpdate(_ context.Context, _, _, _, _ string, _ map[string]interface{}) (*repository.Resource, *repository.Resource, error) {
	return nil, nil, nil
}

// Compile-time check: stubK8sResourceRepo must implement K8sResourceRepo.
var _ repository.K8sResourceRepo = (*stubK8sResourceRepo)(nil)

// ---------------------------------------------------------------------------
// Test setup
// ---------------------------------------------------------------------------

// setupQuotaHandler creates a QuotaHandler backed by a stubK8sResourceRepo and
// a stubClusterRepo. The caller supplies the clusters to seed and the optional
// listFn to control what the K8s repo returns for list calls.
func setupQuotaHandler(
	t *testing.T,
	clusters []*model.Cluster,
	listFn func(ctx context.Context, clusterID, kind, namespace string, opts repository.ListOptions) (*repository.ResourceList, error),
) *QuotaHandler {
	t.Helper()

	clusterRepo := newStubClusterRepo(clusters...)
	k8sRepo := &stubK8sResourceRepo{listFn: listFn}
	quotaService := service.NewQuotaService(k8sRepo, clusterRepo)
	return NewQuotaHandler(quotaService)
}

// ---------------------------------------------------------------------------
// GetQuotaSummary tests
// ---------------------------------------------------------------------------

func TestQuotaHandler_GetQuotaSummary_InvalidClusterID(t *testing.T) {
	handler := setupQuotaHandler(t, nil, nil)

	router := gin.New()
	router.GET("/api/v1/clusters/:id/quota-summary", handler.GetQuotaSummary)

	w := performRequest(router, "GET", "/api/v1/clusters/abc/quota-summary", nil)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp authAPIResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, 40002, resp.Code, "non-numeric cluster id should return CodeParamInvalid")
}

func TestQuotaHandler_GetQuotaSummary_ClusterNotFound(t *testing.T) {
	// Empty cluster repo — no clusters seeded.
	handler := setupQuotaHandler(t, nil, nil)

	router := gin.New()
	router.GET("/api/v1/clusters/:id/quota-summary", handler.GetQuotaSummary)

	w := performRequest(router, "GET", "/api/v1/clusters/9999/quota-summary", nil)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp authAPIResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, 40400, resp.Code, "non-existent cluster should return CodeNotFound")
}

func TestQuotaHandler_GetQuotaSummary_EmptyQuotas(t *testing.T) {
	testCluster := &model.Cluster{ID: 1, Name: "test-cluster"}
	// listFn returns empty list — cluster exists but has no ResourceQuotas.
	handler := setupQuotaHandler(t, []*model.Cluster{testCluster}, nil)

	router := gin.New()
	router.GET("/api/v1/clusters/:id/quota-summary", handler.GetQuotaSummary)

	w := performRequest(router, "GET", "/api/v1/clusters/1/quota-summary", nil)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp authAPIResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, 0, resp.Code, "empty quota list should still return success")

	var data service.QuotaSummaryResponse
	err = json.Unmarshal(resp.Data, &data)
	require.NoError(t, err)
	assert.Empty(t, data.Namespaces, "no quotas means no namespace summaries")
}

func TestQuotaHandler_GetQuotaSummary_WithNamespaceFilter(t *testing.T) {
	testCluster := &model.Cluster{ID: 1, Name: "test-cluster"}
	handler := setupQuotaHandler(t, []*model.Cluster{testCluster}, nil)

	router := gin.New()
	router.GET("/api/v1/clusters/:id/quota-summary", handler.GetQuotaSummary)

	// With namespace filter, the service always returns a summary for that
	// namespace even when there are no quotas (empty Quotas slice).
	w := performRequest(router, "GET", "/api/v1/clusters/1/quota-summary?namespace=default", nil)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp authAPIResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, 0, resp.Code)

	var data service.QuotaSummaryResponse
	err = json.Unmarshal(resp.Data, &data)
	require.NoError(t, err)
	require.Len(t, data.Namespaces, 1, "should return exactly one namespace entry when filtered")
	assert.Equal(t, "default", data.Namespaces[0].Namespace)
	assert.Empty(t, data.Namespaces[0].Quotas, "no quotas exist, so quota list should be empty")
}

func TestQuotaHandler_GetQuotaSummary_WithQuotaData(t *testing.T) {
	testCluster := &model.Cluster{ID: 1, Name: "test-cluster"}

	// Simulate a resourcequota returned by the K8s repo.
	quotaItem := repository.Resource{
		APIVersion: "v1",
		Kind:       "ResourceQuota",
		Name:       "default-quota",
		Namespace:  "production",
		Raw: map[string]interface{}{
			"status": map[string]interface{}{
				"hard": map[string]interface{}{
					"cpu":    "10",
					"memory": "20Gi",
				},
				"used": map[string]interface{}{
					"cpu":    "5",
					"memory": "10Gi",
				},
			},
		},
	}

	listFn := func(_ context.Context, _, _, _ string, _ repository.ListOptions) (*repository.ResourceList, error) {
		return &repository.ResourceList{
			Items: []repository.Resource{quotaItem},
			Total: 1,
		}, nil
	}

	handler := setupQuotaHandler(t, []*model.Cluster{testCluster}, listFn)

	router := gin.New()
	router.GET("/api/v1/clusters/:id/quota-summary", handler.GetQuotaSummary)

	w := performRequest(router, "GET", "/api/v1/clusters/1/quota-summary", nil)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp authAPIResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, 0, resp.Code)

	var data service.QuotaSummaryResponse
	err = json.Unmarshal(resp.Data, &data)
	require.NoError(t, err)
	assert.Len(t, data.Namespaces, 1, "one namespace should be returned")
	assert.Equal(t, "production", data.Namespaces[0].Namespace)
	require.Len(t, data.Namespaces[0].Quotas, 1)
	assert.Equal(t, "default-quota", data.Namespaces[0].Quotas[0].Name)
	assert.Equal(t, "10", data.Namespaces[0].Quotas[0].Hard["cpu"])
	assert.Equal(t, "5", data.Namespaces[0].Quotas[0].Used["cpu"])
}

func TestQuotaHandler_GetQuotaSummary_ResponseBodyStructure(t *testing.T) {
	testCluster := &model.Cluster{ID: 1, Name: "test-cluster"}
	handler := setupQuotaHandler(t, []*model.Cluster{testCluster}, nil)

	router := gin.New()
	router.GET("/api/v1/clusters/:id/quota-summary", handler.GetQuotaSummary)

	w := performRequest(router, "GET", "/api/v1/clusters/1/quota-summary", nil)

	assert.Equal(t, http.StatusOK, w.Code)

	// Verify the envelope structure has the expected keys.
	var raw map[string]json.RawMessage
	err := json.Unmarshal(w.Body.Bytes(), &raw)
	require.NoError(t, err)
	assert.Contains(t, raw, "code")
	assert.Contains(t, raw, "message")
	assert.Contains(t, raw, "data")

	// Verify the data payload contains the "namespaces" key.
	var dataPayload map[string]json.RawMessage
	err = json.Unmarshal(raw["data"], &dataPayload)
	require.NoError(t, err)
	assert.Contains(t, dataPayload, "namespaces")
}

func TestQuotaHandler_GetQuotaSummary_MultipleNamespaces(t *testing.T) {
	testCluster := &model.Cluster{ID: 1, Name: "test-cluster"}

	// Return two quotas in different namespaces.
	listFn := func(_ context.Context, _, _, _ string, _ repository.ListOptions) (*repository.ResourceList, error) {
		return &repository.ResourceList{
			Items: []repository.Resource{
				{
					Name:      "quota-a",
					Namespace: "ns-a",
					Raw: map[string]interface{}{
						"status": map[string]interface{}{
							"hard": map[string]interface{}{"cpu": "4"},
							"used": map[string]interface{}{"cpu": "1"},
						},
					},
				},
				{
					Name:      "quota-b",
					Namespace: "ns-b",
					Raw: map[string]interface{}{
						"status": map[string]interface{}{
							"hard": map[string]interface{}{"memory": "8Gi"},
							"used": map[string]interface{}{"memory": "2Gi"},
						},
					},
				},
			},
			Total: 2,
		}, nil
	}

	handler := setupQuotaHandler(t, []*model.Cluster{testCluster}, listFn)

	router := gin.New()
	router.GET("/api/v1/clusters/:id/quota-summary", handler.GetQuotaSummary)

	w := performRequest(router, "GET", "/api/v1/clusters/1/quota-summary", nil)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp authAPIResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, 0, resp.Code)

	var data service.QuotaSummaryResponse
	err = json.Unmarshal(resp.Data, &data)
	require.NoError(t, err)
	assert.Len(t, data.Namespaces, 2, "should have one entry per namespace")

	nsNames := map[string]bool{}
	for _, ns := range data.Namespaces {
		nsNames[ns.Namespace] = true
	}
	assert.True(t, nsNames["ns-a"])
	assert.True(t, nsNames["ns-b"])
}
