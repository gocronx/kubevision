// Package ai implements the AI assistant: an OpenAI-compatible agent that can
// inspect and (with explicit user approval) mutate Kubernetes resources through
// a fixed set of tools, with per-tool RBAC enforcement.
package ai

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/gocronx/kubevision/internal/auth"
	"github.com/gocronx/kubevision/internal/model"
	"github.com/gocronx/kubevision/internal/repository"
)

// settingKey is the Setting table primary key under which the AI configuration
// JSON blob is stored.
const settingKey = "ai.config"

// settingCategory groups the AI setting in the Setting table.
const settingCategory = "ai"

// Default model and token budget applied when not configured.
const (
	defaultModel     = "gpt-4o-mini"
	defaultBaseURL   = "https://api.openai.com/v1"
	defaultMaxTokens = 4096
	maximumMaxTokens = 32768
)

// Config holds the runtime configuration for the AI assistant. It is persisted
// as a JSON blob in the Setting table so administrators can manage it from the
// UI without restarting the server.
type Config struct {
	Enabled   bool   `json:"enabled"`
	BaseURL   string `json:"baseURL"`
	APIKey    string `json:"apiKey"`
	Model     string `json:"model"`
	MaxTokens int    `json:"maxTokens"`
}

// withDefaults returns a copy of the config with empty fields filled in.
func (c Config) withDefaults() Config {
	if c.BaseURL == "" {
		c.BaseURL = defaultBaseURL
	}
	if c.Model == "" {
		c.Model = defaultModel
	}
	if c.MaxTokens <= 0 {
		c.MaxTokens = defaultMaxTokens
	} else if c.MaxTokens > maximumMaxTokens {
		c.MaxTokens = maximumMaxTokens
	}
	return c
}

// Ready reports whether the assistant has enough configuration to serve chat
// requests (enabled and holding an API key).
func (c Config) Ready() bool {
	return c.Enabled && c.APIKey != ""
}

// ConfigStore loads and persists the AI configuration via the Setting table.
type ConfigStore struct {
	settings   repository.SettingRepo
	encryptKey string
}

// NewConfigStore creates a ConfigStore backed by the given SettingRepo.
func NewConfigStore(settings repository.SettingRepo, encryptKey ...string) *ConfigStore {
	store := &ConfigStore{settings: settings}
	if len(encryptKey) > 0 {
		store.encryptKey = encryptKey[0]
	}
	return store
}

type persistedConfig struct {
	Enabled   bool   `json:"enabled"`
	BaseURL   string `json:"baseURL"`
	APIKey    string `json:"apiKey,omitempty"`
	APIKeyEnc string `json:"apiKeyEncrypted,omitempty"`
	Model     string `json:"model"`
	MaxTokens int    `json:"maxTokens"`
}

// Load reads the persisted configuration, returning a zero-value (disabled)
// config when nothing has been saved yet.
func (s *ConfigStore) Load(ctx context.Context) (Config, error) {
	row, err := s.settings.Get(ctx, settingKey)
	if err != nil {
		return Config{}, err
	}
	if row == nil || row.Value == "" {
		return Config{}, nil
	}
	var stored persistedConfig
	if err := json.Unmarshal([]byte(row.Value), &stored); err != nil {
		return Config{}, err
	}
	apiKey := stored.APIKey
	if stored.APIKeyEnc != "" {
		if s.encryptKey == "" {
			return Config{}, fmt.Errorf("AI API key is encrypted but no encryption key is configured")
		}
		apiKey, err = auth.Decrypt(stored.APIKeyEnc, s.encryptKey)
		if err != nil {
			return Config{}, fmt.Errorf("decrypt AI API key: %w", err)
		}
	}
	cfg := Config{Enabled: stored.Enabled, BaseURL: stored.BaseURL, APIKey: apiKey, Model: stored.Model, MaxTokens: stored.MaxTokens}
	return cfg.withDefaults(), nil
}

// Save persists the configuration as a JSON blob.
func (s *ConfigStore) Save(ctx context.Context, cfg Config) error {
	stored := persistedConfig{Enabled: cfg.Enabled, BaseURL: cfg.BaseURL, Model: cfg.Model, MaxTokens: cfg.MaxTokens}
	if cfg.APIKey != "" {
		if s.encryptKey == "" {
			return fmt.Errorf("cannot persist AI API key without an encryption key")
		}
		ciphertext, err := auth.Encrypt(cfg.APIKey, s.encryptKey)
		if err != nil {
			return fmt.Errorf("encrypt AI API key: %w", err)
		}
		stored.APIKeyEnc = ciphertext
	}
	data, err := json.Marshal(stored)
	if err != nil {
		return err
	}
	return s.settings.Set(ctx, &model.Setting{
		Key:      settingKey,
		Value:    string(data),
		Category: settingCategory,
	})
}
