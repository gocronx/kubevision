package ai

import (
	"context"
	"testing"

	"github.com/gocronx/kubevision/internal/model"
)

// stubSettingRepo is an in-memory SettingRepo for config tests.
type stubSettingRepo struct {
	store map[string]*model.Setting
}

func newStubSettingRepo() *stubSettingRepo {
	return &stubSettingRepo{store: map[string]*model.Setting{}}
}

func (s *stubSettingRepo) Get(_ context.Context, key string) (*model.Setting, error) {
	return s.store[key], nil
}
func (s *stubSettingRepo) Set(_ context.Context, setting *model.Setting) error {
	s.store[setting.Key] = setting
	return nil
}
func (s *stubSettingRepo) List(context.Context, string) ([]model.Setting, error) { return nil, nil }
func (s *stubSettingRepo) Delete(_ context.Context, key string) error {
	delete(s.store, key)
	return nil
}

func TestConfigWithDefaults(t *testing.T) {
	c := Config{}.withDefaults()
	if c.BaseURL != defaultBaseURL || c.Model != defaultModel || c.MaxTokens != defaultMaxTokens {
		t.Fatalf("defaults not applied: %+v", c)
	}
}

func TestConfigReady(t *testing.T) {
	if (Config{Enabled: true}).Ready() {
		t.Fatal("config without API key should not be ready")
	}
	if !(Config{Enabled: true, APIKey: "k"}).Ready() {
		t.Fatal("enabled config with key should be ready")
	}
	if (Config{Enabled: false, APIKey: "k"}).Ready() {
		t.Fatal("disabled config should not be ready")
	}
}

func TestConfigStoreRoundTrip(t *testing.T) {
	store := NewConfigStore(newStubSettingRepo())
	ctx := context.Background()

	// Loading an empty store yields a disabled config.
	got, err := store.Load(ctx)
	if err != nil {
		t.Fatalf("load empty: %v", err)
	}
	if got.Enabled {
		t.Fatal("empty config should be disabled")
	}

	in := Config{Enabled: true, APIKey: "secret", Model: "gpt-4o", BaseURL: "https://x/v1", MaxTokens: 1000}
	if err := store.Save(ctx, in); err != nil {
		t.Fatalf("save: %v", err)
	}
	out, err := store.Load(ctx)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if out != in {
		t.Fatalf("round trip mismatch: got %+v want %+v", out, in)
	}
}
