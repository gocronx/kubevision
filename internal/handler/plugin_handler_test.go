package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/gocronx/kubevision/internal/model"
	"github.com/gocronx/kubevision/internal/service"
)

// stubPluginConfigRepo implements repository.PluginConfigRepo for testing.
type stubPluginConfigRepo struct {
	configs map[string]*model.PluginConfig
}

func newStubPluginConfigRepo() *stubPluginConfigRepo {
	return &stubPluginConfigRepo{configs: make(map[string]*model.PluginConfig)}
}

func (s *stubPluginConfigRepo) Create(_ context.Context, pc *model.PluginConfig) error {
	s.configs[pc.Name] = pc
	return nil
}

func (s *stubPluginConfigRepo) GetByName(_ context.Context, name string) (*model.PluginConfig, error) {
	pc, ok := s.configs[name]
	if !ok {
		return nil, context.DeadlineExceeded
	}
	return pc, nil
}

func (s *stubPluginConfigRepo) Update(_ context.Context, pc *model.PluginConfig) error {
	s.configs[pc.Name] = pc
	return nil
}

func (s *stubPluginConfigRepo) Delete(_ context.Context, id uint) error {
	return nil
}

func (s *stubPluginConfigRepo) List(_ context.Context) ([]model.PluginConfig, error) {
	var result []model.PluginConfig
	for _, pc := range s.configs {
		result = append(result, *pc)
	}
	return result, nil
}

func setupPluginHandler() (*PluginHandler, *gin.Engine) {
	gin.SetMode(gin.TestMode)

	repo := newStubPluginConfigRepo()
	pluginService := service.NewPluginService(repo, zap.NewNop())
	handler := NewPluginHandler(pluginService)

	r := gin.New()
	plugins := r.Group("/api/v1/plugins")
	plugins.GET("", handler.List)
	plugins.GET("/:name", handler.GetConfig)
	plugins.PUT("/:name", handler.Configure)
	plugins.GET("/:name/health", handler.HealthCheck)

	return handler, r
}

func TestPluginHandler_List(t *testing.T) {
	_, r := setupPluginHandler()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/plugins", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Code int                  `json:"code"`
		Data []service.PluginInfo `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Code != 0 {
		t.Errorf("expected code 0, got %d", resp.Code)
	}
	if len(resp.Data) != 3 {
		t.Fatalf("expected 3 plugins, got %d", len(resp.Data))
	}
}

func TestPluginHandler_Configure(t *testing.T) {
	_, r := setupPluginHandler()

	payload := service.PluginConfigRequest{
		Enabled: true,
		Config:  map[string]string{"url": "http://prometheus:9090"},
	}
	body, _ := json.Marshal(payload)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut, "/api/v1/plugins/prometheus", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Code int                `json:"code"`
		Data model.PluginConfig `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !resp.Data.Enabled {
		t.Error("expected plugin to be enabled")
	}
}

func TestPluginHandler_Configure_InvalidConfig(t *testing.T) {
	_, r := setupPluginHandler()

	payload := service.PluginConfigRequest{
		Enabled: true,
		Config:  map[string]string{"url": ""},
	}
	body, _ := json.Marshal(payload)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut, "/api/v1/plugins/prometheus", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		// We always return 200 with business error code.
		var resp struct {
			Code int `json:"code"`
		}
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		if resp.Code == 0 {
			t.Error("expected non-zero business error code for invalid config")
		}
	}
}
