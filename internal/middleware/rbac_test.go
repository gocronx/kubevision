package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/gocronx/kubevision/internal/model"
	bizerr "github.com/gocronx/kubevision/internal/pkg/errors"
)

// ---------------------------------------------------------------------------
// Mock RoleRepo
// ---------------------------------------------------------------------------

type mockRoleRepo struct {
	roles map[string]*model.Role
	err   error
}

func newMockRoleRepo() *mockRoleRepo {
	return &mockRoleRepo{roles: make(map[string]*model.Role)}
}

func (m *mockRoleRepo) addRole(r *model.Role) {
	m.roles[r.Name] = r
}

func (m *mockRoleRepo) Create(_ context.Context, role *model.Role) error { return nil }
func (m *mockRoleRepo) GetByID(_ context.Context, id uint) (*model.Role, error) {
	return nil, errors.New("not implemented")
}
func (m *mockRoleRepo) GetByName(_ context.Context, name string) (*model.Role, error) {
	if m.err != nil {
		return nil, m.err
	}
	r, ok := m.roles[name]
	if !ok {
		return nil, errors.New("role not found")
	}
	return r, nil
}
func (m *mockRoleRepo) Update(_ context.Context, role *model.Role) error { return nil }
func (m *mockRoleRepo) Delete(_ context.Context, id uint) error          { return nil }
func (m *mockRoleRepo) List(_ context.Context) ([]model.Role, error)     { return nil, nil }

// ---------------------------------------------------------------------------
// Tests: extractResource
// ---------------------------------------------------------------------------

func TestExtractResource(t *testing.T) {
	tests := []struct {
		fullPath string
		want     string
	}{
		{"/api/v1/clusters", "clusters"},
		{"/api/v1/clusters/:id", "clusters"},
		{"/api/v1/clusters/:id/resources/:resource", "resources"},
		{"/api/v1/audit-logs", "audit-logs"},
		{"/api/v1/api-keys", "api-keys"},
		{"/api/v1/favorites", "favorites"},
		{"/api/v1/clusters/:id/namespaces/:namespace/pods/:name/exec", "exec"},
		{"/healthz", ""},
		{"", ""},
	}

	for _, tc := range tests {
		t.Run(tc.fullPath, func(t *testing.T) {
			got := extractResource(tc.fullPath)
			if got != tc.want {
				t.Errorf("extractResource(%q) = %q, want %q", tc.fullPath, got, tc.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Tests: extractAction
// ---------------------------------------------------------------------------

func TestExtractAction(t *testing.T) {
	tests := []struct {
		method   string
		fullPath string
		want     string
	}{
		{"GET", "/api/v1/clusters", "list"},
		{"GET", "/api/v1/clusters/:id", "get"},
		{"POST", "/api/v1/clusters", "create"},
		{"PUT", "/api/v1/clusters/:id", "update"},
		{"PATCH", "/api/v1/clusters/:id", "update"},
		{"DELETE", "/api/v1/clusters/:id", "delete"},
		{"GET", "/api/v1/clusters/:id/namespaces/:namespace/pods/:name/exec", "exec"},
		{"GET", "/api/v1/clusters/:id/namespaces/:namespace/pods/:name/logs", "logs"},
	}

	for _, tc := range tests {
		t.Run(tc.method+" "+tc.fullPath, func(t *testing.T) {
			got := extractAction(tc.method, tc.fullPath)
			if got != tc.want {
				t.Errorf("extractAction(%q, %q) = %q, want %q", tc.method, tc.fullPath, got, tc.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Tests: hasPermission
// ---------------------------------------------------------------------------

func TestHasPermission(t *testing.T) {
	tests := []struct {
		name     string
		perms    []string
		resource string
		action   string
		want     bool
	}{
		{"wildcard all", []string{"*:*"}, "pods", "list", true},
		{"wildcard resource", []string{"*:list"}, "pods", "list", true},
		{"wildcard action", []string{"pods:*"}, "pods", "delete", true},
		{"exact match", []string{"pods:list"}, "pods", "list", true},
		{"no match", []string{"pods:list"}, "services", "delete", false},
		{"multiple perms", []string{"pods:list", "services:get"}, "services", "get", true},
		{"empty perms", []string{}, "pods", "list", false},
		{"malformed perm", []string{"invalid"}, "pods", "list", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := hasPermission(tc.perms, tc.resource, tc.action)
			if got != tc.want {
				t.Errorf("hasPermission(%v, %q, %q) = %v, want %v", tc.perms, tc.resource, tc.action, got, tc.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Tests: RBACMiddleware
// ---------------------------------------------------------------------------

func performRBACRequest(roleRepo *mockRoleRepo, role, method, path, routePattern string) (*httptest.ResponseRecorder, bool) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	handlerCalled := false

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("userRole", role)
		c.Next()
	})
	router.Use(RBACMiddleware(roleRepo))
	router.Handle(method, routePattern, func(c *gin.Context) {
		handlerCalled = true
		c.JSON(200, gin.H{"ok": true})
	})

	req := httptest.NewRequest(method, path, nil)
	router.ServeHTTP(w, req)
	return w, handlerCalled
}

func TestRBACMiddleware_AdminBypass(t *testing.T) {
	repo := newMockRoleRepo()
	_, called := performRBACRequest(repo, "admin", "DELETE", "/api/v1/clusters/1", "/api/v1/clusters/:id")
	if !called {
		t.Error("admin should bypass RBAC and reach handler")
	}
}

func TestRBACMiddleware_AllowedPermission(t *testing.T) {
	repo := newMockRoleRepo()
	repo.addRole(&model.Role{Name: "viewer", Permissions: `["clusters:list"]`})

	_, called := performRBACRequest(repo, "viewer", "GET", "/api/v1/clusters", "/api/v1/clusters")
	if !called {
		t.Error("viewer with clusters:list should be allowed to list clusters")
	}
}

func TestRBACMiddleware_DeniedPermission(t *testing.T) {
	repo := newMockRoleRepo()
	repo.addRole(&model.Role{Name: "viewer", Permissions: `["clusters:list"]`})

	w, called := performRBACRequest(repo, "viewer", "DELETE", "/api/v1/clusters/1", "/api/v1/clusters/:id")
	if called {
		t.Error("viewer without delete permission should be denied")
	}
	var resp apiResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Code != bizerr.CodeForbidden {
		t.Errorf("expected code %d, got %d", bizerr.CodeForbidden, resp.Code)
	}
}

func TestRBACMiddleware_RoleNotFound(t *testing.T) {
	repo := newMockRoleRepo() // no roles added

	w, called := performRBACRequest(repo, "unknown-role", "GET", "/api/v1/clusters", "/api/v1/clusters")
	if called {
		t.Error("unknown role should be denied")
	}
	var resp apiResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Code != bizerr.CodeForbidden {
		t.Errorf("expected code %d, got %d", bizerr.CodeForbidden, resp.Code)
	}
}

func TestRBAC_NoopMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	handlerCalled := false

	router := gin.New()
	router.Use(RBAC("some:perm"))
	router.GET("/test", func(c *gin.Context) {
		handlerCalled = true
		c.Status(200)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	if !handlerCalled {
		t.Error("RBAC noop middleware should pass through")
	}
}
