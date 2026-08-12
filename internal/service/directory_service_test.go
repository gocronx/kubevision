package service

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/gocronx/kubevision/internal/auth"
	"github.com/gocronx/kubevision/internal/config"
	"github.com/gocronx/kubevision/internal/directory"
	"github.com/gocronx/kubevision/internal/model"
	"github.com/gocronx/kubevision/internal/repository"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type fakeDirectoryClient struct {
	identity           *directory.Identity
	authErr, lookupErr error
}

func (f *fakeDirectoryClient) Authenticate(context.Context, directory.Config, string, string) (*directory.Identity, error) {
	return f.identity, f.authErr
}
func (f *fakeDirectoryClient) Lookup(context.Context, directory.Config, string) (*directory.Identity, error) {
	return f.identity, f.lookupErr
}
func (f *fakeDirectoryClient) Ping(context.Context, directory.Config) error { return nil }

func directoryTestService(t *testing.T, fake *fakeDirectoryClient) (*DirectoryService, repository.UserRepo, repository.DirectoryRepo, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.User{}, &model.Role{}, &model.DirectoryConfig{}, &model.DirectoryRoleMapping{}))
	for _, role := range []string{"viewer", "editor", "admin"} {
		require.NoError(t, db.Create(&model.Role{Name: role}).Error)
	}
	users, directoryRepo, roles := repository.NewUserRepo(db), repository.NewDirectoryRepo(db), repository.NewRoleRepo(db)
	return NewDirectoryService(directoryRepo, users.(repository.DirectoryUserRepo), roles, fake, "test-encryption-key"), users, directoryRepo, db
}

func validDirectorySettings() DirectorySettings {
	return DirectorySettings{Enabled: true, ServerURL: "ldaps://directory.example.test", ConnectTimeoutSecs: 2, SearchTimeoutSecs: 3, UserBaseDN: "ou=people,dc=example,dc=test", UserFilter: "(uid={{username}})", StableIDAttribute: "entryUUID", UsernameAttribute: "uid", DisplayAttribute: "displayName", EmailAttribute: "mail", GroupAttribute: "memberOf", FallbackRole: "viewer", RefreshMapping: true, BindPassword: "bind-secret"}
}

func TestSelectDirectoryRolePriorityAndFallback(t *testing.T) {
	mappings := []model.DirectoryRoleMapping{{GroupID: "ops", Role: "editor", Priority: 20}, {GroupID: "security", Role: "admin", Priority: 10}}
	role, rule := selectRole([]string{"ops", "security"}, mappings, "viewer")
	require.Equal(t, "admin", role)
	require.NotNil(t, rule)
	require.Equal(t, "security", rule.GroupID)
	role, rule = selectRole([]string{"unknown"}, mappings, "viewer")
	require.Equal(t, "viewer", role)
	require.Nil(t, rule)
}

func TestDirectoryIdentityCollisionNeverMergesByEmail(t *testing.T) {
	fake := &fakeDirectoryClient{identity: &directory.Identity{StableID: "external-1", Username: "alice-directory", Email: "alice@example.test"}}
	svc, users, _, _ := directoryTestService(t, fake)
	require.NoError(t, users.Create(context.Background(), &model.User{Username: "alice-local", Email: "alice@example.test", PasswordHash: "hash", AuthProvider: "local", Role: "viewer", IsActive: true}))
	require.NoError(t, svc.SaveSettings(context.Background(), validDirectorySettings()))
	_, err := svc.Authenticate(context.Background(), "alice", "secret")
	require.Error(t, err)
	all, listErr := users.List(context.Background())
	require.NoError(t, listErr)
	require.Len(t, all, 1)
	require.Equal(t, "local", all[0].AuthProvider)
}

func TestDirectoryMatchFailuresAreGeneric(t *testing.T) {
	for _, upstream := range []error{directory.ErrNoMatch, directory.ErrAmbiguous, directory.ErrCredentials} {
		t.Run(upstream.Error(), func(t *testing.T) {
			fake := &fakeDirectoryClient{authErr: upstream}
			svc, _, _, _ := directoryTestService(t, fake)
			require.NoError(t, svc.SaveSettings(context.Background(), validDirectorySettings()))
			_, err := svc.Authenticate(context.Background(), "alice", "bad")
			require.EqualError(t, err, "[40100] invalid username or password")
		})
	}
}

func TestDirectoryRefreshPrivilegeRemovalRevokesOldSession(t *testing.T) {
	fake := &fakeDirectoryClient{identity: &directory.Identity{StableID: "external-2", Username: "bob", Groups: []string{"admins"}}}
	directorySvc, users, _, _ := directoryTestService(t, fake)
	settings := validDirectorySettings()
	settings.Mappings = []model.DirectoryRoleMapping{{GroupID: "admins", Role: "admin", Priority: 1}}
	require.NoError(t, directorySvc.SaveSettings(context.Background(), settings))
	user, err := directorySvc.Authenticate(context.Background(), "bob", "secret")
	require.NoError(t, err)
	require.Equal(t, "admin", user.Role)
	oldVersion := user.TokenVersion
	jwtManager := auth.NewJWTManager("01234567890123456789012345678901", time.Minute, time.Hour)
	authSvc := NewAuthService(users, jwtManager, config.Default(), zap.NewNop())
	authSvc.SetDirectoryService(directorySvc)
	oldRefresh, err := jwtManager.GenerateRefreshToken(user.ID, user.TokenVersion)
	require.NoError(t, err)
	fake.identity.Groups = nil
	_, err = authSvc.RefreshToken(context.Background(), oldRefresh)
	require.Error(t, err)
	updated, err := users.GetByID(context.Background(), user.ID)
	require.NoError(t, err)
	require.Equal(t, "viewer", updated.Role)
	require.Greater(t, updated.TokenVersion, oldVersion)
}

func TestDirectoryConfigEncryptsCredentialAndRevokesSessions(t *testing.T) {
	svc, users, repo, _ := directoryTestService(t, &fakeDirectoryClient{})
	require.NoError(t, users.Create(context.Background(), &model.User{Username: "existing", DirectoryID: "id", AuthProvider: "directory", Role: "editor", IsActive: true}))
	require.NoError(t, svc.SaveSettings(context.Background(), validDirectorySettings()))
	cfg, err := repo.GetConfig(context.Background())
	require.NoError(t, err)
	require.NotEqual(t, "bind-secret", cfg.BindPasswordEnc)
	require.NotEmpty(t, cfg.BindPasswordEnc)
	plain, err := auth.Decrypt(cfg.BindPasswordEnc, "test-encryption-key")
	require.NoError(t, err)
	require.Equal(t, "bind-secret", plain)
	user, err := users.GetByUsername(context.Background(), "existing")
	require.NoError(t, err)
	require.Equal(t, 1, user.TokenVersion)
}

func TestDirectoryOnlyUserCannotReceiveLocalPassword(t *testing.T) {
	_, users, _, _ := directoryTestService(t, &fakeDirectoryClient{})
	user := &model.User{Username: "directory-only", DirectoryID: "stable", AuthProvider: "directory", Role: "viewer", IsActive: true}
	require.NoError(t, users.Create(context.Background(), user))
	userService := NewUserService(users)
	require.EqualError(t, userService.ResetPassword(context.Background(), user.ID, "long-enough-password"), "[40300] directory-only accounts do not have a local password")
	require.EqualError(t, userService.ChangePassword(context.Background(), user.ID, "old-password", "long-enough-password"), "[40300] directory-only accounts do not have a local password")
}

func TestDirectorySettingsJSONRedactsCredential(t *testing.T) {
	payload, err := json.Marshal(DirectorySettings{Enabled: true, ServerURL: "ldaps://example.test", BindPassword: "secret", Mappings: []model.DirectoryRoleMapping{}, CredentialConfigured: true})
	require.NoError(t, err)
	require.JSONEq(t, `{"enabled":true,"startTls":false,"allowPlaintext":false,"refreshMapping":false,"serverUrl":"ldaps://example.test","caBundle":"","bindDn":"","userBaseDn":"","userFilter":"","stableIdAttribute":"","usernameAttribute":"","displayAttribute":"","emailAttribute":"","groupAttribute":"","fallbackRole":"","connectTimeoutSecs":0,"searchTimeoutSecs":0,"mappings":[],"credentialConfigured":true}`, string(payload))
	require.NotContains(t, string(payload), "secret")
}

func TestDirectorySettingsAlwaysReturnMappingsArray(t *testing.T) {
	settings := settingsFromModel(&model.DirectoryConfig{}, nil)
	payload, err := json.Marshal(settings)
	require.NoError(t, err)
	require.Contains(t, string(payload), `"mappings":[]`)
}

var _ = errors.Is
