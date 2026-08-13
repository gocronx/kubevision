package ai

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/gocronx/kubevision/internal/model"
)

// stubRoleRepo is a minimal RoleRepo for authorization tests.
type stubRoleRepo struct {
	roles map[string]*model.Role
}

func (s *stubRoleRepo) GetByName(_ context.Context, name string) (*model.Role, error) {
	return s.roles[name], nil
}
func (s *stubRoleRepo) Create(context.Context, *model.Role) error          { return nil }
func (s *stubRoleRepo) GetByID(context.Context, uint) (*model.Role, error) { return nil, nil }
func (s *stubRoleRepo) Update(context.Context, *model.Role) error          { return nil }
func (s *stubRoleRepo) Delete(context.Context, uint) error                 { return nil }
func (s *stubRoleRepo) List(context.Context) ([]model.Role, error)         { return nil, nil }

func roleWith(perms ...string) *model.Role {
	b, _ := json.Marshal(perms)
	return &model.Role{Permissions: string(b)}
}

func TestGrants(t *testing.T) {
	cases := []struct {
		perms          []string
		resource, verb string
		want           bool
	}{
		{[]string{"*:*"}, "pods", "delete", true},
		{[]string{"pods:*"}, "pods", "delete", true},
		{[]string{"*:get"}, "deployments", "get", true},
		{[]string{"pods:get"}, "pods", "get", true},
		{[]string{"pods:get"}, "pods", "delete", false},
		{[]string{"deployments:update"}, "pods", "update", false},
		{[]string{"bad-perm"}, "pods", "get", false},
	}
	for _, tc := range cases {
		if got := grants(tc.perms, tc.resource, tc.verb); got != tc.want {
			t.Errorf("grants(%v, %s, %s) = %v, want %v", tc.perms, tc.resource, tc.verb, got, tc.want)
		}
	}
}

func TestRequiredPermission(t *testing.T) {
	p, ok := requiredPermission("delete_resource", map[string]any{"kind": "Pods"})
	if !ok || p.resource != "pods" || p.action != "delete" {
		t.Fatalf("got %+v ok=%v", p, ok)
	}
	if p, ok := requiredPermission("get_pod_logs", nil); !ok || p.resource != "pods" || p.action != "list" {
		t.Fatalf("logs perm = %+v ok=%v", p, ok)
	}
	if p, ok := requiredPermission("get_cluster_overview", nil); !ok || p.resource != "overview" || p.action != "list" {
		t.Fatalf("overview perm = %+v ok=%v", p, ok)
	}
	if _, ok := requiredPermission("unknown_tool", nil); ok {
		t.Fatalf("unknown tool should need no permission")
	}
}

func TestAuthorize(t *testing.T) {
	repo := &stubRoleRepo{roles: map[string]*model.Role{
		"viewer": roleWith("pods:get", "pods:list"),
		"editor": roleWith("*:*"),
	}}
	a := newAuthorizer(repo)
	ctx := context.Background()

	// Admin bypass: no role lookup needed.
	if msg := a.authorize(ctx, "admin", "delete_resource", map[string]any{"kind": "pods"}); msg != "" {
		t.Fatalf("admin should bypass, got %q", msg)
	}
	// Viewer may read pods.
	if msg := a.authorize(ctx, "viewer", "get_resource", map[string]any{"kind": "pods"}); msg != "" {
		t.Fatalf("viewer get pods should pass, got %q", msg)
	}
	// Viewer may not delete pods.
	if msg := a.authorize(ctx, "viewer", "delete_resource", map[string]any{"kind": "pods"}); msg == "" {
		t.Fatalf("viewer delete pods should be denied")
	}
	// Editor with wildcard may delete.
	if msg := a.authorize(ctx, "editor", "delete_resource", map[string]any{"kind": "pods"}); msg != "" {
		t.Fatalf("editor delete should pass, got %q", msg)
	}
	// Missing kind on a kind-scoped tool is rejected.
	if msg := a.authorize(ctx, "viewer", "get_resource", map[string]any{}); msg == "" {
		t.Fatalf("missing kind should be rejected")
	}
}

func TestAuthorizeSupportsGenericResourcePermissions(t *testing.T) {
	a := newAuthorizer(&stubRoleRepo{roles: map[string]*model.Role{
		"editor": roleWith("resources:get", "resources:list", "resources:create", "resources:update", "resources:delete"),
	}})
	ctx := context.Background()
	for _, tc := range []struct {
		tool, action string
	}{
		{"get_resource", "get"}, {"list_resources", "list"}, {"create_resource", "create"},
		{"update_resource", "update"}, {"patch_resource", "update"}, {"delete_resource", "delete"},
	} {
		args := map[string]any{"kind": "deployments", "name": "web", "yaml": "kind: Deployment", "patch": `{}`}
		if msg := a.authorize(ctx, "editor", tc.tool, args); msg != "" {
			t.Errorf("%s denied despite resources:%s: %s", tc.tool, tc.action, msg)
		}
	}
}

func TestAuthorizeBlocksHighRiskMutationsForNonAdmins(t *testing.T) {
	a := newAuthorizer(&stubRoleRepo{roles: map[string]*model.Role{"editor": roleWith("resources:*")}})
	ctx := context.Background()
	cases := []map[string]any{
		{"kind": "clusterrolebindings", "name": "escalate"},
		{"kind": "deployments", "yaml": "kind: Deployment\nspec:\n  template:\n    spec:\n      hostNetwork: true"},
		{"kind": "pods", "yaml": "kind: Pod\nspec:\n  containers:\n  - name: shell\n    securityContext:\n      privileged: true"},
	}
	for _, args := range cases {
		if msg := a.authorize(ctx, "editor", "create_resource", args); msg == "" {
			t.Errorf("high-risk mutation was allowed: %#v", args)
		}
	}
	if msg := a.authorize(ctx, "admin", "create_resource", cases[0]); msg != "" {
		t.Fatalf("admin mutation should pass: %s", msg)
	}
	if msg := a.authorize(ctx, "editor", "query_prometheus", map[string]any{"query": "up"}); msg == "" {
		t.Fatal("non-admin Prometheus query should be denied")
	}
}
