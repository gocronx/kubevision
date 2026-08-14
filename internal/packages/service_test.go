package packages

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/gocronx/kubevision/internal/model"
	bizerr "github.com/gocronx/kubevision/internal/pkg/errors"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/release"
)

type fakeAdapter struct {
	releases []Release
	rollback func()
	remove   func()
	preview  *Preview
	install  func(ChangeOptions)
	upgrade  func(ChangeOptions)
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
func (f *fakeAdapter) Preview(context.Context, string, string, ChangeOptions) (*Preview, error) {
	if f.preview != nil {
		copy := *f.preview
		return &copy, nil
	}
	return &Preview{Digest: "chart-digest"}, nil
}
func (f *fakeAdapter) Install(_ context.Context, _ string, opts ChangeOptions) error {
	if f.install != nil {
		f.install(opts)
	}
	return nil
}
func (f *fakeAdapter) Upgrade(_ context.Context, _ string, opts ChangeOptions) error {
	if f.upgrade != nil {
		f.upgrade(opts)
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

func TestServicePreviewBindsOneTimeConfirmationToRequest(t *testing.T) {
	var installed ChangeOptions
	adapter := &fakeAdapter{preview: &Preview{Digest: "abc"}, install: func(opts ChangeOptions) { installed = opts }}
	service := NewService(adapter, fakeAuth{permissions: map[string]bool{PermissionInstall: true}}, &fakeAudit{})
	actor := Actor{UserID: 7, Role: "editor"}
	opts := ChangeOptions{ReleaseName: "demo", Namespace: "team-a", Source: ChartSource{Chart: "oci://registry.example/demo"}, Values: map[string]interface{}{"replicas": 1}}
	preview, err := service.Preview(context.Background(), actor, "install", "cluster-a", opts)
	require.NoError(t, err)
	require.NotEmpty(t, preview.ConfirmationToken)
	opts.ConfirmationToken = preview.ConfirmationToken
	require.NoError(t, service.Install(context.Background(), actor, "cluster-a", opts))
	require.Equal(t, "abc", installed.ExpectedDigest)
	require.Error(t, service.Install(context.Background(), actor, "cluster-a", opts), "token must be one-time")
}

func TestServiceRejectsChangedRequestAfterPreview(t *testing.T) {
	service := NewService(&fakeAdapter{}, fakeAuth{permissions: map[string]bool{PermissionUpgrade: true}}, nil)
	actor := Actor{UserID: 9, Role: "editor"}
	opts := ChangeOptions{ReleaseName: "demo", Namespace: "team-a", Source: ChartSource{Chart: "oci://registry.example/demo"}}
	preview, err := service.Preview(context.Background(), actor, "upgrade", "cluster-a", opts)
	require.NoError(t, err)
	opts.Values = map[string]interface{}{"image": "changed"}
	opts.ConfirmationToken = preview.ConfirmationToken
	require.Error(t, service.Upgrade(context.Background(), actor, "cluster-a", opts))
}

func TestServiceCriticalRiskRequiresAdministrator(t *testing.T) {
	adapter := &fakeAdapter{preview: &Preview{Digest: "abc", Risks: []Risk{{Level: "critical", Code: "cluster-privilege"}}}}
	auth := fakeAuth{permissions: map[string]bool{PermissionInstall: true}}
	service := NewService(adapter, auth, nil)
	opts := ChangeOptions{ReleaseName: "demo", Namespace: "team-a", Source: ChartSource{Chart: "oci://registry.example/demo"}}
	preview, err := service.Preview(context.Background(), Actor{UserID: 1, Role: "editor"}, "install", "cluster", opts)
	require.NoError(t, err)
	require.False(t, preview.CanExecute)
	require.Empty(t, preview.ConfirmationToken)
	preview, err = service.Preview(context.Background(), Actor{UserID: 2, Role: "admin"}, "install", "cluster", opts)
	require.NoError(t, err)
	require.True(t, preview.CanExecute)
	require.NotEmpty(t, preview.ConfirmationToken)
}

func TestServiceCreateNamespaceRequiresAdministrator(t *testing.T) {
	service := NewService(&fakeAdapter{}, fakeAuth{permissions: map[string]bool{PermissionInstall: true}}, nil)
	opts := ChangeOptions{ReleaseName: "demo", Namespace: "new-team", Source: ChartSource{Chart: "oci://registry.example/demo"}, CreateNamespace: true}
	preview, err := service.Preview(context.Background(), Actor{UserID: 1, Role: "editor"}, "install", "cluster", opts)
	require.NoError(t, err)
	require.False(t, preview.CanExecute)
	require.Contains(t, preview.Risks[0].Resource, "new-team")
}

func TestServicePreviewOnlyDoesNotIssueConfirmation(t *testing.T) {
	service := NewService(&fakeAdapter{}, fakeAuth{permissions: map[string]bool{PermissionUpgrade: true}}, nil)
	opts := ChangeOptions{ReleaseName: "demo", Namespace: "default", Source: ChartSource{Chart: "oci://registry.example/demo"}}
	preview, err := service.Preview(context.Background(), Actor{UserID: 1, Role: "admin", PreviewOnly: true}, "upgrade", "cluster", opts)
	require.NoError(t, err)
	require.True(t, preview.CanExecute)
	require.Empty(t, preview.ConfirmationToken)
	require.Empty(t, service.previews)
}

func TestInspectManifestFindsCriticalWorkloadAccess(t *testing.T) {
	manifest := `apiVersion: apps/v1
kind: Deployment
metadata:
  name: unsafe
spec:
  template:
    spec:
      hostNetwork: true
      volumes:
        - name: host
          hostPath:
            path: /
      containers:
        - name: app
          image: example
          securityContext:
            privileged: true
`
	resources, risks := inspectManifest(manifest)
	require.Len(t, resources, 1)
	require.Len(t, risks, 3)
}

func TestReleaseManifestIncludesHooks(t *testing.T) {
	item := &release.Release{Manifest: "kind: Service", Hooks: []*release.Hook{{Manifest: "kind: ClusterRole"}}}
	manifest := releaseManifest(item)
	require.Contains(t, manifest, "kind: Service")
	require.Contains(t, manifest, "kind: ClusterRole")
}

func TestRedactManifestRemovesSecretValues(t *testing.T) {
	manifest := "apiVersion: v1\nkind: Secret\nmetadata:\n  name: credentials\nstringData:\n  password: hidden\n"
	redacted := redactManifest(manifest)
	require.NotContains(t, redacted, "hidden")
	require.Contains(t, redacted, "[REDACTED]")
}

func TestValidateChartSourceRejectsUnsafeInputs(t *testing.T) {
	tests := []ChartSource{
		{Chart: "/etc/passwd"},
		{Chart: "./local-chart"},
		{Chart: "file:///tmp/chart.tgz"},
		{Chart: "http://charts.example/demo.tgz"},
		{Chart: "https://user:password@charts.example/demo.tgz"},
		{Chart: "https://127.0.0.1/demo.tgz"},
		{Chart: "demo", RepoURL: "http://charts.example"},
	}
	for _, source := range tests {
		require.Error(t, validateChartSource(source), source.Chart)
	}
}

func TestLoadRemoteChartReturnsSourceValidationDetails(t *testing.T) {
	_, err := loadRemoteChart(context.Background(), ChartSource{
		Chart:   "https://charts.example/demo",
		RepoURL: "https://charts.example",
	})
	require.Error(t, err)
	var packageErr *bizerr.BizError
	require.ErrorAs(t, err, &packageErr)
	require.Equal(t, bizerr.CodeValidation, packageErr.Code)
	require.Equal(t, "chart must be a repository chart name when repoUrl is set", packageErr.Message)
}

func TestMapAdapterErrorReturnsChartValidationDetails(t *testing.T) {
	err := errors.New("execution error at (gocron/templates/deployment.yaml:19:28): db.host is required")
	mapped := mapAdapterError(err)
	var packageErr *bizerr.BizError
	require.ErrorAs(t, mapped, &packageErr)
	require.Equal(t, bizerr.CodeValidation, packageErr.Code)
	require.Equal(t, "chart values are invalid: db.host is required", packageErr.Message)
}

func TestMapAdapterErrorReturnsReleaseConflict(t *testing.T) {
	mapped := mapAdapterError(errors.New("cannot re-use a name that is still in use"))
	var packageErr *bizerr.BizError
	require.ErrorAs(t, mapped, &packageErr)
	require.Equal(t, bizerr.CodeConflict, packageErr.Code)
	require.Equal(t, "release already exists; upgrade or remove it before reinstalling", packageErr.Message)
}

func TestRejectClusterLookupsIncludesDependencies(t *testing.T) {
	clean := &chart.Chart{Metadata: &chart.Metadata{Name: "clean"}, Templates: []*chart.File{{Name: "deployment.yaml", Data: []byte(`{{ .Values.image }}`)}}}
	require.NoError(t, rejectClusterLookups(clean))
	dependency := &chart.Chart{Metadata: &chart.Metadata{Name: "dependency"}, Templates: []*chart.File{{Name: "secret.yaml", Data: []byte(`{{ lookup "v1" "Secret" "default" "credentials" }}`)}}}
	clean.AddDependency(dependency)
	err := rejectClusterLookups(clean)
	var packageErr *bizerr.BizError
	require.ErrorAs(t, err, &packageErr)
	require.Equal(t, bizerr.CodeValidation, packageErr.Code)
	require.Equal(t, "chart templates using Helm lookup are not supported for security reasons", packageErr.Message)
}

func TestPublicRoundTripperRejectsPlainHTTP(t *testing.T) {
	request, err := http.NewRequest(http.MethodGet, "http://registry.example/token", nil)
	require.NoError(t, err)
	_, err = (&publicRoundTripper{transport: http.DefaultTransport.(*http.Transport).Clone()}).RoundTrip(request)
	require.Error(t, err)
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
