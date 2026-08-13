package cli

import (
	"testing"

	"github.com/gocronx/kubevision/internal/ai"
)

func TestFirstNonEmpty(t *testing.T) {
	if got := firstNonEmpty("", "", "x", "y"); got != "x" {
		t.Fatalf("got %q, want x", got)
	}
	if got := firstNonEmpty("", ""); got != "" {
		t.Fatalf("got %q, want empty", got)
	}
}

func TestApplyAIOverrides(t *testing.T) {
	t.Setenv("API_KEY", "env-key")
	t.Setenv("API_BASE_URL", "https://env.example/v1")
	t.Setenv("MODEL_ID", "env-model")

	cfg := applyAIOverrides(ai.Config{Enabled: false}, "flag-key", "", "flag-model")
	if !cfg.Enabled || cfg.APIKey != "flag-key" {
		t.Fatalf("API key override was not applied: %+v", cfg)
	}
	if cfg.BaseURL != "https://env.example/v1" || cfg.Model != "flag-model" {
		t.Fatalf("provider overrides were not applied: %+v", cfg)
	}
}

func TestApplyAIOverridesKeepsSavedSettings(t *testing.T) {
	t.Setenv("API_KEY", "")
	t.Setenv("API_BASE_URL", "")
	t.Setenv("MODEL_ID", "")

	want := ai.Config{Enabled: true, APIKey: "saved-key", BaseURL: "https://saved.example/v1", Model: "saved-model"}
	got := applyAIOverrides(want, "", "", "")
	if got != want {
		t.Fatalf("saved settings changed: got %+v, want %+v", got, want)
	}
}

func TestCompactArgs(t *testing.T) {
	// Large blobs are dropped; scalar fields are kept.
	got := compactArgs(map[string]any{"kind": "pods", "namespace": "default", "yaml": "big...", "patch": "..."})
	if got != `{"kind":"pods","namespace":"default"}` {
		t.Fatalf("compactArgs = %s", got)
	}
	if compactArgs(map[string]any{}) != "" {
		t.Fatal("empty args should render empty")
	}
	if compactArgs("not a map") != "" {
		t.Fatal("non-map should render empty")
	}
}

func TestCurrentContext(t *testing.T) {
	kube := []byte("apiVersion: v1\nkind: Config\ncurrent-context: prod\nclusters: []\ncontexts: []\nusers: []\n")
	if got := currentContext(kube); got != "prod" {
		t.Fatalf("currentContext = %q, want prod", got)
	}
	if got := currentContext([]byte("not yaml: [")); got != "default" {
		t.Fatalf("invalid kubeconfig should fall back to default, got %q", got)
	}
}

func TestIndent(t *testing.T) {
	if got := indent("a\nb"); got != "    a\n    b" {
		t.Fatalf("indent = %q", got)
	}
}

func TestStr(t *testing.T) {
	m := map[string]any{"a": "x", "n": 1}
	if str(m, "a") != "x" || str(m, "n") != "" || str(nil, "a") != "" {
		t.Fatal("str helper misbehaved")
	}
}
