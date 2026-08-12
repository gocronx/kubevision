package packages

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/gocronx/kubevision/internal/model"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type fakeAdapter struct {
	releases []Release
	rollback func()
	remove   func()
}

func (f *fakeAdapter) List(context.Context, string, ListOptions) ([]Release, error) {
	return f.releases, nil
}
func (f *fakeAdapter) Get(_ context.Context, _, _, _ string, revision int) (*Release, error) {
	for i := range f.releases {
		if revision == 0 || f.releases[i].Revision == revision {
			copy := f.releases[i]
			return &copy, nil
		}
	}
	return nil, errors.New("not found")
}
func (f *fakeAdapter) History(context.Context, string, string, string) ([]Release, error) {
	return f.releases, nil
}
func (f *fakeAdapter) Rollback(context.Context, string, string, string, RollbackOptions) error {
	if f.rollback != nil {
		f.rollback()
	}
	return nil
}
func (f *fakeAdapter) Remove(context.Context, string, string, string, RemoveOptions) error {
	if f.remove != nil {
		f.remove()
	}
	return nil
}

type fakeAuth struct{ permissions map[string]bool }

func (a fakeAuth) Allowed(_ context.Context, _ Actor, permission, _, _ string) bool {
	return a.permissions[permission]
}

type fakeAudit struct {
	mu     sync.Mutex
	events []AuditEvent
}

func (a *fakeAudit) RecordPackageAudit(event AuditEvent) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.events = append(a.events, event)
}

func TestServiceDistinctPermissions(t *testing.T) {
	adapter := &fakeAdapter{releases: []Release{{Name: "demo", Namespace: "team-a", Revision: 1}}}
	service := NewService(adapter, fakeAuth{permissions: map[string]bool{PermissionRead: true}}, &fakeAudit{})
	_, err := service.List(context.Background(), Actor{}, "cluster", ListOptions{Namespace: "team-a"})
	require.NoError(t, err)
	err = service.Rollback(context.Background(), Actor{}, "cluster", "team-a", "demo", RollbackOptions{Revision: 1})
	require.Error(t, err)
	err = service.Remove(context.Background(), Actor{}, "cluster", "team-a", "demo", RemoveOptions{Confirmation: "demo"})
	require.Error(t, err)
}

func TestServiceRedactsSecretShapedValues(t *testing.T) {
	adapter := &fakeAdapter{releases: []Release{{Name: "demo", Revision: 1, Values: map[string]interface{}{"image": "ok", "apiToken": "hidden", "nested": map[string]interface{}{"password": "hidden"}}, Resources: []ResourceRef{{Kind: "Secret", Name: "credential"}}}}}
	service := NewService(adapter, fakeAuth{permissions: map[string]bool{PermissionRead: true}}, nil)
	items, err := service.List(context.Background(), Actor{}, "cluster", ListOptions{})
	require.NoError(t, err)
	require.Equal(t, "[REDACTED]", items[0].Values["apiToken"])
	require.Equal(t, "[REDACTED]", items[0].Values["nested"].(map[string]interface{})["password"])
	require.Equal(t, "[REDACTED]", items[0].Resources[0].Name)
}

func TestServiceTypedRemovalConfirmationAndAudit(t *testing.T) {
	audit := &fakeAudit{}
	service := NewService(&fakeAdapter{}, fakeAuth{permissions: map[string]bool{PermissionRemove: true}}, audit)
	err := service.Remove(context.Background(), Actor{Username: "alice"}, "cluster", "team-a", "demo", RemoveOptions{Confirmation: "other"})
	require.Error(t, err)
	require.Empty(t, audit.events)
	err = service.Remove(context.Background(), Actor{Username: "alice"}, "cluster", "team-a", "demo", RemoveOptions{Confirmation: "demo"})
	require.NoError(t, err)
	require.Len(t, audit.events, 1)
	require.Equal(t, "remove", audit.events[0].Action)
	require.Equal(t, "succeeded", audit.events[0].Outcome)
}

func TestServiceSerializesMutationsPerRelease(t *testing.T) {
	entered := make(chan struct{})
	release := make(chan struct{})
	adapter := &fakeAdapter{releases: []Release{{Name: "demo", Revision: 1}}, rollback: func() { close(entered); <-release }}
	service := NewService(adapter, fakeAuth{permissions: map[string]bool{PermissionRollback: true}}, &fakeAudit{})
	done := make(chan error, 1)
	go func() {
		done <- service.Rollback(context.Background(), Actor{}, "cluster", "team-a", "demo", RollbackOptions{Revision: 1, Timeout: time.Second})
	}()
	<-entered
	err := service.Rollback(context.Background(), Actor{}, "cluster", "team-a", "demo", RollbackOptions{Revision: 1, Timeout: time.Second})
	require.Error(t, err)
	close(release)
	require.NoError(t, <-done)
}

type roleRepo struct{ role *model.Role }

func (r roleRepo) Create(context.Context, *model.Role) error              { return nil }
func (r roleRepo) GetByID(context.Context, uint) (*model.Role, error)     { return r.role, nil }
func (r roleRepo) GetByName(context.Context, string) (*model.Role, error) { return r.role, nil }
func (r roleRepo) Update(context.Context, *model.Role) error              { return nil }
func (r roleRepo) Delete(context.Context, uint) error                     { return nil }
func (r roleRepo) List(context.Context) ([]model.Role, error)             { return nil, nil }

func TestRoleAuthorizerEnforcesClusterNamespaceMapping(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:package-authz?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.Cluster{}, &model.Role{}, &model.UserClusterRole{}))
	cluster := model.Cluster{Name: "cluster-a"}
	role := model.Role{Name: "package-reader", Permissions: `[]`}
	require.NoError(t, db.Create(&cluster).Error)
	require.NoError(t, db.Create(&role).Error)
	require.NoError(t, db.Create(&model.UserClusterRole{UserID: 7, ClusterID: cluster.ID, RoleID: role.ID, Namespaces: "team-a,team-b"}).Error)
	auth := NewRoleAuthorizer(roleRepo{role: &model.Role{Name: "package-reader", Permissions: `["package-releases:read"]`}}, db)
	require.True(t, auth.Allowed(context.Background(), Actor{UserID: 7, Role: "viewer"}, PermissionRead, "cluster-a", "team-a"))
	require.False(t, auth.Allowed(context.Background(), Actor{UserID: 7, Role: "viewer"}, PermissionRead, "cluster-a", ""))
	require.False(t, auth.Allowed(context.Background(), Actor{UserID: 7, Role: "viewer"}, PermissionRead, "cluster-a", "team-c"))
	require.False(t, auth.Allowed(context.Background(), Actor{UserID: 7, Role: "viewer"}, PermissionRead, "cluster-b", "team-a"))
}
