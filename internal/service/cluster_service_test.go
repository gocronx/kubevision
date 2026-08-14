package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gocronx/kubevision/internal/kubernetes/cluster"
	"github.com/gocronx/kubevision/internal/kubernetes/informer"
	"github.com/gocronx/kubevision/internal/kubernetes/resource"
	"github.com/gocronx/kubevision/internal/pkg/errors"
	"go.uber.org/zap"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

func serviceTestKubeconfig(t *testing.T, serverURL string) string {
	t.Helper()
	data, err := clientcmd.Write(clientcmdapi.Config{
		CurrentContext: "test",
		Clusters: map[string]*clientcmdapi.Cluster{
			"test": {Server: serverURL, InsecureSkipTLSVerify: true},
		},
		AuthInfos: map[string]*clientcmdapi.AuthInfo{
			"test": {Token: "test-token"},
		},
		Contexts: map[string]*clientcmdapi.Context{
			"test": {Cluster: "test", AuthInfo: "test"},
		},
	})
	if err != nil {
		t.Fatalf("write kubeconfig: %v", err)
	}
	return string(data)
}

func newClusterServiceForAddTest(repo *mockClusterRepo) *ClusterService {
	logger := zap.NewNop()
	return NewClusterService(
		repo,
		cluster.NewManager(),
		informer.NewManager(logger),
		resource.NewRegistry(),
		logger,
		"test-encrypt-key",
	)
}

func TestClusterService_AddRejectsInvalidCredentials(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
	}))
	defer server.Close()
	repo := newMockClusterRepo()
	svc := newClusterServiceForAddTest(repo)

	_, err := svc.Add(context.Background(), &AddClusterRequest{
		Name:       "local",
		AuthType:   "kubeconfig",
		Kubeconfig: serviceTestKubeconfig(t, server.URL),
	})
	if !errors.Is(err, errors.CodeK8sUnavailable) {
		t.Fatalf("expected Kubernetes unavailable error, got %v", err)
	}
	if len(repo.clusters) != 0 {
		t.Fatalf("failed import persisted %d clusters", len(repo.clusters))
	}
}

func TestClusterService_AddRejectsInClusterAuthOutsideKubernetes(t *testing.T) {
	t.Setenv("KUBERNETES_SERVICE_HOST", "")
	t.Setenv("KUBERNETES_SERVICE_PORT", "")
	svc := newClusterServiceForAddTest(newMockClusterRepo())

	_, err := svc.Add(context.Background(), &AddClusterRequest{
		Name:     "local",
		AuthType: "in-cluster",
	})
	bizErr, ok := err.(*errors.BizError)
	if !ok || bizErr.Code != errors.CodeParamInvalid {
		t.Fatalf("expected invalid parameter error, got %v", err)
	}
	if !strings.Contains(bizErr.Message, "use a kubeconfig for local development") {
		t.Fatalf("unexpected error message: %q", bizErr.Message)
	}
}

func TestClusterService_AddPersistsProbeMetadata(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"kind":"APIVersions","versions":["v1"]}`))
		case "/version":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"gitVersion":"v1.30.2+k3s1"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	repo := newMockClusterRepo()
	svc := newClusterServiceForAddTest(repo)

	response, err := svc.Add(context.Background(), &AddClusterRequest{
		Name:       "local",
		AuthType:   "kubeconfig",
		Kubeconfig: serviceTestKubeconfig(t, server.URL),
	})
	if err != nil {
		t.Fatalf("add cluster: %v", err)
	}
	if response.Status != "healthy" || response.APIServer != server.URL || response.Version != "v1.30.2+k3s1" {
		t.Fatalf("unexpected cluster response: %#v", response)
	}
}

func TestClusterService_RefreshHealthReconnectsMissingClient(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"kind":"APIVersions","versions":["v1"]}`))
		case "/version":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"gitVersion":"v1.30.2+k3s1"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	repo := newMockClusterRepo()
	manager := cluster.NewManager()
	logger := zap.NewNop()
	svc := NewClusterService(
		repo,
		manager,
		informer.NewManager(logger),
		resource.NewRegistry(),
		logger,
		"test-encrypt-key",
	)

	_, err := svc.Add(context.Background(), &AddClusterRequest{
		Name:       "local",
		AuthType:   "kubeconfig",
		Kubeconfig: serviceTestKubeconfig(t, server.URL),
	})
	if err != nil {
		t.Fatalf("add cluster: %v", err)
	}

	// Simulate another process persisting this cluster, or a lost in-memory
	// client after a development reload.
	manager.Remove("local")
	svc.RefreshHealth(context.Background())

	if _, err := manager.RESTConfig("local"); err != nil {
		t.Fatalf("health refresh did not reconnect missing client: %v", err)
	}
	stored, err := repo.GetByName(context.Background(), "local")
	if err != nil {
		t.Fatalf("get cluster: %v", err)
	}
	if stored.Status != "healthy" {
		t.Fatalf("cluster status = %q, want healthy", stored.Status)
	}
}

func TestClusterService_List_Empty(t *testing.T) {
	clusterRepo := newMockClusterRepo()
	k8sRepo := newMockK8sRepo()
	_ = k8sRepo // ClusterService doesn't directly use k8sRepo

	// ClusterService requires concrete types from kubernetes packages, so we
	// test the parts that only interact with ClusterRepo.
	svc := &ClusterService{
		clusterRepo: clusterRepo,
	}

	ctx := context.Background()
	list, err := svc.List(ctx)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(list) != 0 {
		t.Errorf("expected empty list, got %d", len(list))
	}
}

func TestClusterService_List_WithClusters(t *testing.T) {
	clusterRepo := newMockClusterRepo()
	clusterRepo.addCluster(makeTestCluster(1, "prod"))
	clusterRepo.addCluster(makeTestCluster(2, "staging"))

	svc := &ClusterService{
		clusterRepo: clusterRepo,
	}

	ctx := context.Background()
	list, err := svc.List(ctx)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(list) != 2 {
		t.Errorf("expected 2 clusters, got %d", len(list))
	}
}

func TestClusterService_Get_Success(t *testing.T) {
	clusterRepo := newMockClusterRepo()
	clusterRepo.addCluster(makeTestCluster(1, "prod"))

	svc := &ClusterService{
		clusterRepo: clusterRepo,
	}

	ctx := context.Background()
	resp, err := svc.Get(ctx, 1)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if resp.Name != "prod" {
		t.Errorf("expected name 'prod', got %q", resp.Name)
	}
}

func TestClusterService_Get_NotFound(t *testing.T) {
	clusterRepo := newMockClusterRepo()

	svc := &ClusterService{
		clusterRepo: clusterRepo,
	}

	ctx := context.Background()
	_, err := svc.Get(ctx, 999)
	if err == nil {
		t.Fatal("expected error for non-existent cluster")
	}
}

func TestClusterService_ResolveClusterID_ByNumericID(t *testing.T) {
	clusterRepo := newMockClusterRepo()
	clusterRepo.addCluster(makeTestCluster(1, "prod"))

	svc := &ClusterService{
		clusterRepo: clusterRepo,
	}

	ctx := context.Background()
	name, err := svc.ResolveClusterID(ctx, "1")
	if err != nil {
		t.Fatalf("ResolveClusterID failed: %v", err)
	}
	if name != "prod" {
		t.Errorf("expected 'prod', got %q", name)
	}
}

func TestClusterService_ResolveClusterID_ByName(t *testing.T) {
	clusterRepo := newMockClusterRepo()
	clusterRepo.addCluster(makeTestCluster(1, "prod"))

	svc := &ClusterService{
		clusterRepo: clusterRepo,
	}

	ctx := context.Background()
	name, err := svc.ResolveClusterID(ctx, "prod")
	if err != nil {
		t.Fatalf("ResolveClusterID failed: %v", err)
	}
	if name != "prod" {
		t.Errorf("expected 'prod', got %q", name)
	}
}

func TestClusterService_ResolveClusterID_NotFound(t *testing.T) {
	clusterRepo := newMockClusterRepo()

	svc := &ClusterService{
		clusterRepo: clusterRepo,
	}

	ctx := context.Background()
	_, err := svc.ResolveClusterID(ctx, "unknown")
	if err == nil {
		t.Fatal("expected error for non-existent cluster")
	}
}

func TestToClusterResponse(t *testing.T) {
	c := makeTestCluster(1, "prod")
	resp := toClusterResponse(c)

	if resp.ID != 1 {
		t.Errorf("expected ID 1, got %d", resp.ID)
	}
	if resp.Name != "prod" {
		t.Errorf("expected name 'prod', got %q", resp.Name)
	}
	if resp.Status != "healthy" {
		t.Errorf("expected status 'healthy', got %q", resp.Status)
	}
}
