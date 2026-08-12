package cluster

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

func testKubeconfig(t *testing.T, serverURL string) []byte {
	t.Helper()
	config := clientcmdapi.Config{
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
	}
	data, err := clientcmd.Write(config)
	if err != nil {
		t.Fatalf("write kubeconfig: %v", err)
	}
	return data
}

func TestProbeReturnsServerMetadata(t *testing.T) {
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

	manager := NewManager()
	if err := manager.Add("local", testKubeconfig(t, server.URL)); err != nil {
		t.Fatalf("add cluster: %v", err)
	}

	info, err := manager.Probe(context.Background(), "local")
	if err != nil {
		t.Fatalf("probe cluster: %v", err)
	}
	if info.APIServer != server.URL {
		t.Fatalf("API server = %q, want %q", info.APIServer, server.URL)
	}
	if info.Version != "v1.30.2+k3s1" {
		t.Fatalf("version = %q", info.Version)
	}
}

func TestProbeRejectsInvalidCredentials(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
	}))
	defer server.Close()

	manager := NewManager()
	if err := manager.Add("local", testKubeconfig(t, server.URL)); err != nil {
		t.Fatalf("add cluster: %v", err)
	}

	_, err := manager.Probe(context.Background(), "local")
	if err == nil || !strings.Contains(err.Error(), "authenticate to Kubernetes API server") {
		t.Fatalf("expected authentication failure, got %v", err)
	}
}
