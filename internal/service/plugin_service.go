package service

import (
	"context"
	"encoding/json"
	"fmt"

	"go.uber.org/zap"

	"github.com/gocronx/kubevision/internal/model"
	bizerr "github.com/gocronx/kubevision/internal/pkg/errors"
	"github.com/gocronx/kubevision/internal/plugin"
	"github.com/gocronx/kubevision/internal/plugin/argocd"
	"github.com/gocronx/kubevision/internal/plugin/grafana"
	"github.com/gocronx/kubevision/internal/plugin/prometheus"
	"github.com/gocronx/kubevision/internal/repository"
)

// PluginInfo is the public summary of a registered plugin.
type PluginInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Version     string `json:"version"`
	Type        string `json:"type"`
	Enabled     bool   `json:"enabled"`
	Healthy     bool   `json:"healthy"`
}

// PluginConfigRequest is the payload for enabling/configuring a plugin.
type PluginConfigRequest struct {
	Enabled bool              `json:"enabled"`
	Config  map[string]string `json:"config"`
}

// PluginService manages plugin lifecycle and configuration.
type PluginService struct {
	registry *plugin.Registry
	repo     repository.PluginConfigRepo
	logger   *zap.Logger
}

// NewPluginService creates a new PluginService.
func NewPluginService(repo repository.PluginConfigRepo, logger *zap.Logger) *PluginService {
	svc := &PluginService{
		registry: plugin.NewRegistry(),
		repo:     repo,
		logger:   logger,
	}

	// Register built-in plugins.
	svc.registry.Register(prometheus.New())
	svc.registry.Register(grafana.New())
	svc.registry.Register(argocd.New())

	return svc
}

// Registry returns the underlying plugin registry.
func (s *PluginService) Registry() *plugin.Registry {
	return s.registry
}

// List returns info about all registered plugins, including their enabled/health state.
func (s *PluginService) List(ctx context.Context) ([]PluginInfo, error) {
	all := s.registry.All()
	configs, _ := s.repo.List(ctx) // ignore error; treat as empty

	configMap := make(map[string]*model.PluginConfig, len(configs))
	for i := range configs {
		configMap[configs[i].Name] = &configs[i]
	}

	var infos []PluginInfo
	for _, p := range all {
		info := PluginInfo{
			Name:        p.Name(),
			Description: p.Description(),
			Version:     p.Version(),
			Type:        p.Type(),
		}
		if cfg, ok := configMap[p.Name()]; ok {
			info.Enabled = cfg.Enabled
		}
		if info.Enabled {
			if err := p.HealthCheck(ctx); err == nil {
				info.Healthy = true
			}
		}
		infos = append(infos, info)
	}
	return infos, nil
}

// GetConfig returns the stored configuration for a plugin.
func (s *PluginService) GetConfig(ctx context.Context, name string) (*model.PluginConfig, error) {
	_, ok := s.registry.Get(name)
	if !ok {
		return nil, bizerr.New(bizerr.CodeNotFound, fmt.Sprintf("plugin %q not registered", name))
	}

	pc, err := s.repo.GetByName(ctx, name)
	if err != nil {
		return nil, bizerr.New(bizerr.CodeNotFound, fmt.Sprintf("plugin %q not configured", name))
	}
	return pc, nil
}

// Configure updates the configuration and optionally enables/disables a plugin.
func (s *PluginService) Configure(ctx context.Context, name string, req *PluginConfigRequest) (*model.PluginConfig, error) {
	p, ok := s.registry.Get(name)
	if !ok {
		return nil, bizerr.New(bizerr.CodeNotFound, fmt.Sprintf("plugin %q not registered", name))
	}

	// Validate by trying to init with the provided config.
	if req.Enabled && req.Config != nil {
		if err := p.Init(req.Config); err != nil {
			return nil, bizerr.New(bizerr.CodeParamInvalid, fmt.Sprintf("invalid config: %v", err))
		}
	}

	configJSON, err := json.Marshal(req.Config)
	if err != nil {
		return nil, bizerr.New(bizerr.CodeInternal, "failed to marshal config")
	}

	// Upsert configuration.
	pc, err := s.repo.GetByName(ctx, name)
	if err != nil {
		// Create new.
		pc = &model.PluginConfig{
			Name:       name,
			PluginType: p.Type(),
			Enabled:    req.Enabled,
			Config:     string(configJSON),
		}
		if err := s.repo.Create(ctx, pc); err != nil {
			return nil, bizerr.New(bizerr.CodeInternal, "failed to save plugin config")
		}
		return pc, nil
	}

	// Update existing.
	pc.Enabled = req.Enabled
	pc.Config = string(configJSON)
	if err := s.repo.Update(ctx, pc); err != nil {
		return nil, bizerr.New(bizerr.CodeInternal, "failed to update plugin config")
	}
	return pc, nil
}

// HealthCheck tests connectivity for a specific plugin.
func (s *PluginService) HealthCheck(ctx context.Context, name string) error {
	p, ok := s.registry.Get(name)
	if !ok {
		return bizerr.New(bizerr.CodeNotFound, fmt.Sprintf("plugin %q not registered", name))
	}
	return p.HealthCheck(ctx)
}

// InitFromDB loads stored plugin configurations and initializes enabled plugins.
func (s *PluginService) InitFromDB(ctx context.Context) {
	configs, err := s.repo.List(ctx)
	if err != nil {
		s.logger.Warn("failed to load plugin configs from DB", zap.Error(err))
		return
	}

	for _, cfg := range configs {
		if !cfg.Enabled {
			continue
		}
		p, ok := s.registry.Get(cfg.Name)
		if !ok {
			continue
		}
		var configMap map[string]string
		if err := json.Unmarshal([]byte(cfg.Config), &configMap); err != nil {
			s.logger.Warn("invalid plugin config JSON", zap.String("plugin", cfg.Name), zap.Error(err))
			continue
		}
		if err := p.Init(configMap); err != nil {
			s.logger.Warn("plugin init failed", zap.String("plugin", cfg.Name), zap.Error(err))
		} else {
			s.logger.Info("plugin initialized", zap.String("plugin", cfg.Name))
		}
	}
}

// GetPrometheus returns the Prometheus plugin if registered.
func (s *PluginService) GetPrometheus() (*prometheus.Plugin, bool) {
	p, ok := s.registry.Get("prometheus")
	if !ok {
		return nil, false
	}
	pp, ok := p.(*prometheus.Plugin)
	return pp, ok
}

// GetGrafana returns the Grafana plugin if registered.
func (s *PluginService) GetGrafana() (*grafana.Plugin, bool) {
	p, ok := s.registry.Get("grafana")
	if !ok {
		return nil, false
	}
	gp, ok := p.(*grafana.Plugin)
	return gp, ok
}

// GetArgoCD returns the ArgoCD plugin if registered.
func (s *PluginService) GetArgoCD() (*argocd.Plugin, bool) {
	p, ok := s.registry.Get("argocd")
	if !ok {
		return nil, false
	}
	ap, ok := p.(*argocd.Plugin)
	return ap, ok
}
