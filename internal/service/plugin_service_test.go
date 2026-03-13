package service

import (
	"context"
	"testing"

	"go.uber.org/zap"

	"github.com/gocronx/kubevision/internal/model"
)

// mockPluginConfigRepo implements repository.PluginConfigRepo for testing.
type mockPluginConfigRepo struct {
	configs map[string]*model.PluginConfig
}

func newMockPluginConfigRepo() *mockPluginConfigRepo {
	return &mockPluginConfigRepo{configs: make(map[string]*model.PluginConfig)}
}

func (m *mockPluginConfigRepo) Create(_ context.Context, pc *model.PluginConfig) error {
	m.configs[pc.Name] = pc
	return nil
}

func (m *mockPluginConfigRepo) GetByName(_ context.Context, name string) (*model.PluginConfig, error) {
	pc, ok := m.configs[name]
	if !ok {
		return nil, context.DeadlineExceeded // simulate not found
	}
	return pc, nil
}

func (m *mockPluginConfigRepo) Update(_ context.Context, pc *model.PluginConfig) error {
	m.configs[pc.Name] = pc
	return nil
}

func (m *mockPluginConfigRepo) Delete(_ context.Context, id uint) error {
	for k, v := range m.configs {
		if v.ID == id {
			delete(m.configs, k)
			return nil
		}
	}
	return nil
}

func (m *mockPluginConfigRepo) List(_ context.Context) ([]model.PluginConfig, error) {
	var result []model.PluginConfig
	for _, pc := range m.configs {
		result = append(result, *pc)
	}
	return result, nil
}

func TestPluginService_List(t *testing.T) {
	repo := newMockPluginConfigRepo()
	svc := NewPluginService(repo, zap.NewNop())

	plugins, err := svc.List(context.Background())
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	// Should have 3 built-in plugins: prometheus, grafana, argocd.
	if len(plugins) != 3 {
		t.Fatalf("expected 3 plugins, got %d", len(plugins))
	}

	names := make(map[string]bool)
	for _, p := range plugins {
		names[p.Name] = true
	}
	for _, expected := range []string{"prometheus", "grafana", "argocd"} {
		if !names[expected] {
			t.Errorf("expected plugin %q not found", expected)
		}
	}
}

func TestPluginService_Configure(t *testing.T) {
	repo := newMockPluginConfigRepo()
	svc := NewPluginService(repo, zap.NewNop())

	// Configure prometheus with valid URL.
	cfg, err := svc.Configure(context.Background(), "prometheus", &PluginConfigRequest{
		Enabled: true,
		Config:  map[string]string{"url": "http://prometheus:9090"},
	})
	if err != nil {
		t.Fatalf("Configure failed: %v", err)
	}
	if !cfg.Enabled {
		t.Error("expected plugin to be enabled")
	}
	if cfg.Name != "prometheus" {
		t.Errorf("expected name prometheus, got %s", cfg.Name)
	}

	// Update the same plugin.
	cfg2, err := svc.Configure(context.Background(), "prometheus", &PluginConfigRequest{
		Enabled: false,
		Config:  map[string]string{"url": "http://prometheus:9090"},
	})
	if err != nil {
		t.Fatalf("Configure update failed: %v", err)
	}
	if cfg2.Enabled {
		t.Error("expected plugin to be disabled after update")
	}
}

func TestPluginService_Configure_InvalidURL(t *testing.T) {
	repo := newMockPluginConfigRepo()
	svc := NewPluginService(repo, zap.NewNop())

	// Enabled but with invalid URL — should fail validation.
	_, err := svc.Configure(context.Background(), "prometheus", &PluginConfigRequest{
		Enabled: true,
		Config:  map[string]string{"url": ""},
	})
	if err == nil {
		t.Fatal("expected error for invalid config")
	}
}

func TestPluginService_Configure_UnknownPlugin(t *testing.T) {
	repo := newMockPluginConfigRepo()
	svc := NewPluginService(repo, zap.NewNop())

	_, err := svc.Configure(context.Background(), "nonexistent", &PluginConfigRequest{
		Enabled: true,
		Config:  map[string]string{},
	})
	if err == nil {
		t.Fatal("expected error for unknown plugin")
	}
}

func TestPluginService_HealthCheck_NotRegistered(t *testing.T) {
	repo := newMockPluginConfigRepo()
	svc := NewPluginService(repo, zap.NewNop())

	err := svc.HealthCheck(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error for unregistered plugin")
	}
}

func TestPluginService_GetConfig(t *testing.T) {
	repo := newMockPluginConfigRepo()
	svc := NewPluginService(repo, zap.NewNop())

	// Not configured yet.
	_, err := svc.GetConfig(context.Background(), "prometheus")
	if err == nil {
		t.Fatal("expected error for unconfigured plugin")
	}

	// Configure it.
	_, _ = svc.Configure(context.Background(), "prometheus", &PluginConfigRequest{
		Enabled: true,
		Config:  map[string]string{"url": "http://prometheus:9090"},
	})

	cfg, err := svc.GetConfig(context.Background(), "prometheus")
	if err != nil {
		t.Fatalf("GetConfig failed: %v", err)
	}
	if cfg.Name != "prometheus" {
		t.Errorf("expected name prometheus, got %s", cfg.Name)
	}
}

func TestPluginService_GetTypedPlugins(t *testing.T) {
	repo := newMockPluginConfigRepo()
	svc := NewPluginService(repo, zap.NewNop())

	if _, ok := svc.GetPrometheus(); !ok {
		t.Error("expected prometheus plugin to be available")
	}
	if _, ok := svc.GetGrafana(); !ok {
		t.Error("expected grafana plugin to be available")
	}
	if _, ok := svc.GetArgoCD(); !ok {
		t.Error("expected argocd plugin to be available")
	}
}
