package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/kubevision/kubevision/internal/repository"
	"github.com/kubevision/kubevision/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Stub K8s repo for compare handler tests — returns a fixed resource.
// ---------------------------------------------------------------------------

type compareStubK8sRepo struct {
	fullStubK8sRepo
}

func newCompareStubK8sRepo() *compareStubK8sRepo {
	r := &compareStubK8sRepo{}
	r.getFn = func(_ context.Context, clusterID, kind, ns, name string) (*repository.Resource, error) {
		return &repository.Resource{
			Name:      name,
			Namespace: ns,
			Kind:      kind,
			Raw: map[string]interface{}{
				"apiVersion": "v1",
				"kind":       kind,
				"metadata":   map[string]interface{}{"name": name, "namespace": ns},
			},
		}, nil
	}
	return r
}

func setupCompareHandler(t *testing.T) (*gin.Engine, *CompareHandler) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	k8sRepo := newCompareStubK8sRepo()
	compareSvc := service.NewCompareService(k8sRepo)
	handler := NewCompareHandler(compareSvc)

	router := gin.New()
	router.POST("/api/v1/compare", handler.Compare)
	return router, handler
}

func TestCompareHandler_Compare_Success(t *testing.T) {
	router, _ := setupCompareHandler(t)

	body := service.CompareRequest{
		Source: service.CompareTarget{
			Cluster:   "cluster-a",
			Namespace: "default",
			Resource:  "pods",
			Name:      "nginx",
		},
		Target: service.CompareTarget{
			Cluster:   "cluster-b",
			Namespace: "default",
			Resource:  "pods",
			Name:      "nginx",
		},
	}
	w := performRequest(router, "POST", "/api/v1/compare", body)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp authAPIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Code)

	var result service.CompareResult
	require.NoError(t, json.Unmarshal(resp.Data, &result))
	assert.NotEmpty(t, result.SourceYAML)
	assert.NotEmpty(t, result.TargetYAML)
}

func TestCompareHandler_Compare_InvalidBody(t *testing.T) {
	router, _ := setupCompareHandler(t)

	// Missing required fields.
	w := performRequest(router, "POST", "/api/v1/compare", map[string]string{})

	var resp authAPIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 40002, resp.Code)
}

func TestCompareHandler_Compare_EmptyBody(t *testing.T) {
	router, _ := setupCompareHandler(t)

	w := performRequestRawBody(router, "POST", "/api/v1/compare", nil)

	var resp authAPIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.NotEqual(t, 0, resp.Code)
}
