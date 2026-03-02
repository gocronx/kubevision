package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/kubevision/kubevision/internal/service"
	discoveryfake "k8s.io/client-go/discovery/fake"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/kubernetes/fake"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type stubDiscoveryProvider struct {
	client discovery.DiscoveryInterface
}

func (s *stubDiscoveryProvider) DiscoveryClient(id string) (discovery.DiscoveryInterface, error) {
	return s.client, nil
}

func setupCRDHandler() (*CRDHandler, *gin.Engine) {
	gin.SetMode(gin.TestMode)

	fakeClient := fake.NewSimpleClientset()
	fakeDiscovery := fakeClient.Discovery().(*discoveryfake.FakeDiscovery)
	fakeDiscovery.Resources = []*metav1.APIResourceList{
		{
			GroupVersion: "example.com/v1",
			APIResources: []metav1.APIResource{
				{Name: "widgets", Kind: "Widget", Namespaced: true},
			},
		},
	}

	provider := &stubDiscoveryProvider{client: fakeDiscovery}
	crdService := service.NewCRDService(provider, zap.NewNop())
	handler := NewCRDHandler(crdService)

	r := gin.New()
	r.GET("/api/v1/clusters/:id/crds", handler.List)
	r.POST("/api/v1/clusters/:id/crds/refresh", handler.Refresh)

	return handler, r
}

func TestCRDHandler_List(t *testing.T) {
	_, r := setupCRDHandler()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/clusters/test/crds", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Code int                `json:"code"`
		Data []service.CRDInfo `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.Code != 0 {
		t.Errorf("expected code 0, got %d", resp.Code)
	}
	if len(resp.Data) != 1 {
		t.Fatalf("expected 1 CRD, got %d", len(resp.Data))
	}
	if resp.Data[0].Kind != "Widget" {
		t.Errorf("expected kind Widget, got %s", resp.Data[0].Kind)
	}
}

func TestCRDHandler_Refresh(t *testing.T) {
	_, r := setupCRDHandler()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/clusters/test/crds/refresh", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}
