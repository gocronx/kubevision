package ai

import (
	"strings"
	"testing"
)

func TestArgHelpers(t *testing.T) {
	args := map[string]any{
		"kind":      "pods",
		"tailLines": float64(50), // JSON numbers decode to float64
		"previous":  true,
	}
	if argString(args, "kind") != "pods" {
		t.Fatal("argString")
	}
	if argString(args, "missing") != "" {
		t.Fatal("argString missing should be empty")
	}
	if argInt(args, "tailLines", 200) != 50 {
		t.Fatal("argInt from float64")
	}
	if argInt(args, "missing", 200) != 200 {
		t.Fatal("argInt default")
	}
	if !argBool(args, "previous") {
		t.Fatal("argBool true")
	}
	if argBool(args, "missing") {
		t.Fatal("argBool missing should be false")
	}
}

func TestParseManifest(t *testing.T) {
	obj, err := parseManifest("apiVersion: v1\nkind: Pod\nmetadata:\n  name: web\n  namespace: prod\n")
	if err != nil {
		t.Fatalf("parseManifest: %v", err)
	}
	if obj["kind"] != "Pod" {
		t.Fatalf("kind = %v", obj["kind"])
	}
	if ns := manifestNamespace(obj); ns != "prod" {
		t.Fatalf("namespace = %q", ns)
	}
	if _, err := parseManifest(""); err == nil {
		t.Fatal("empty manifest should error")
	}
}

func TestPodPhase(t *testing.T) {
	raw := map[string]any{"status": map[string]any{"phase": "Running"}}
	if podPhase(raw) != "Running" {
		t.Fatal("podPhase")
	}
	if podPhase(map[string]any{}) != "" {
		t.Fatal("podPhase missing should be empty")
	}
}

func TestToolDefinitionsCoverAllTools(t *testing.T) {
	defs := toolDefinitions()
	names := map[string]bool{}
	for _, d := range defs {
		names[d.Function.Name] = true
		if d.Type != "function" {
			t.Fatalf("tool %s has wrong type %q", d.Function.Name, d.Type)
		}
	}
	for _, want := range []string{
		"get_resource", "list_resources", "get_pod_logs", "get_cluster_overview",
		"query_prometheus", "create_resource", "update_resource", "patch_resource", "delete_resource",
	} {
		if !names[want] {
			t.Errorf("missing tool definition: %s", want)
		}
	}
}

func TestBuildSystemPrompt(t *testing.T) {
	p := buildSystemPrompt(promptContext{
		clusterName:  "prod",
		userRole:     "admin",
		namespace:    "default",
		resourceKind: "Pod",
		resourceName: "web",
	})
	for _, want := range []string{"prod", "admin", "default", "Pod/web"} {
		if !strings.Contains(p, want) {
			t.Errorf("system prompt missing %q", want)
		}
	}
}
