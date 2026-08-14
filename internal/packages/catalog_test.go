package packages

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"testing"
	"time"

	"github.com/gocronx/kubevision/internal/model"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"helm.sh/helm/v3/pkg/action"
)

func newCatalogTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.HelmRepository{}, &model.HelmUpgradePolicy{}))
	return db
}

func TestCatalogEncryptsRepositoryCredentials(t *testing.T) {
	db := newCatalogTestDB(t)
	catalog := NewCatalog(db, "test-encryption-key")
	item, err := catalog.SaveRepository(context.Background(), Actor{Role: "admin"}, 0, RepositoryInput{Name: "charts", Type: "oci", URL: "oci://registry.example/charts", Username: "alice", Password: "plain-secret", AllowPrivateNetwork: true, Enabled: true})
	require.NoError(t, err)
	require.Empty(t, item.PasswordEnc, "API model must not expose encrypted credentials")
	var stored model.HelmRepository
	require.NoError(t, db.First(&stored, item.ID).Error)
	require.NotEmpty(t, stored.PasswordEnc)
	require.NotContains(t, stored.PasswordEnc, "plain-secret")
	resolved, _, err := catalog.repositorySource(context.Background(), item.ID)
	require.NoError(t, err)
	require.Equal(t, "plain-secret", resolved.Password)
}

func TestCatalogRepositoryManagementRequiresAdmin(t *testing.T) {
	catalog := NewCatalog(newCatalogTestDB(t), "key")
	_, err := catalog.SaveRepository(context.Background(), Actor{Role: "editor"}, 0, RepositoryInput{Name: "charts", Type: "oci", URL: "oci://registry.example/charts", Enabled: true})
	require.Error(t, err)
}

func TestCatalogDoesNotExposeStoredRepositoryCredentials(t *testing.T) {
	db := newCatalogTestDB(t)
	catalog := NewCatalog(db, "key")
	created, err := catalog.SaveRepository(context.Background(), Actor{Role: "admin"}, 0, RepositoryInput{Name: "charts", Type: "oci", URL: "oci://registry.example/charts", Username: "alice", Password: "secret", AllowPrivateNetwork: true, Enabled: true})
	require.NoError(t, err)
	require.Empty(t, created.PasswordEnc)

	items, err := catalog.ListRepositories(context.Background(), Actor{Role: "admin"})
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Empty(t, items[0].PasswordEnc)
}

func TestCatalogManagedRepositoryUseRequiresAdmin(t *testing.T) {
	db := newCatalogTestDB(t)
	catalog := NewCatalog(db, "key")
	item, err := catalog.SaveRepository(context.Background(), Actor{Role: "admin"}, 0, RepositoryInput{Name: "charts", Type: "oci", URL: "oci://registry.example/charts", AllowPrivateNetwork: true, Enabled: true})
	require.NoError(t, err)

	_, err = catalog.Inspect(context.Background(), Actor{Role: "editor"}, ChartSource{RepositoryID: item.ID, Chart: "demo"})
	require.Error(t, err)
}

func TestCatalogUploadInspectsAndExpiresChart(t *testing.T) {
	catalog := NewCatalog(newCatalogTestDB(t), "key")
	archive := testChartArchive(t)
	inspection, err := catalog.Upload(context.Background(), Actor{Role: "editor"}, bytes.NewReader(archive), int64(len(archive)))
	require.NoError(t, err)
	require.Equal(t, "demo", inspection.Name)
	require.Equal(t, "hello", inspection.Readme)
	require.NotEmpty(t, inspection.UploadID)
	require.Contains(t, inspection.Templates, "templates/deployment.yaml")
	catalog.mu.Lock()
	entry := catalog.uploads[inspection.UploadID]
	entry.createdAt = time.Now().Add(-31 * time.Minute)
	catalog.uploads[inspection.UploadID] = entry
	catalog.pruneUploadsLocked(time.Now())
	catalog.mu.Unlock()
	_, err = catalog.loadChart(context.Background(), ChartSource{UploadID: inspection.UploadID})
	require.Error(t, err)
	require.Zero(t, catalog.uploadSize)
}

func TestCatalogUploadIsBoundToItsOwner(t *testing.T) {
	catalog := NewCatalog(newCatalogTestDB(t), "key")
	archive := testChartArchive(t)
	inspection, err := catalog.Upload(context.Background(), Actor{UserID: 7, Role: "editor"}, bytes.NewReader(archive), int64(len(archive)))
	require.NoError(t, err)

	require.Error(t, catalog.AuthorizeUpload(Actor{UserID: 8, Role: "editor"}, inspection.UploadID))
	require.NoError(t, catalog.AuthorizeUpload(Actor{UserID: 7, Role: "editor"}, inspection.UploadID))
	require.NoError(t, catalog.AuthorizeUpload(Actor{UserID: 1, Role: "admin"}, inspection.UploadID))
}

func TestUpgradePolicyRejectsOCIRepository(t *testing.T) {
	db := newCatalogTestDB(t)
	repository := model.HelmRepository{Name: "oci", Type: "oci", URL: "oci://registry.example/charts", Enabled: true}
	require.NoError(t, db.Create(&repository).Error)
	manager := NewUpgradeManager(db, NewCatalog(db, "key"), nil, nil)
	_, err := manager.Save(context.Background(), Actor{Role: "admin"}, 0, UpgradePolicyInput{Cluster: "local", Namespace: "default", ReleaseName: "demo", RepositoryID: repository.ID, Chart: "demo", VersionConstraint: "^1.0.0", IntervalMinutes: 60, Enabled: true})
	require.Error(t, err)
}

func TestUpgradeManagerStopIsIdempotent(t *testing.T) {
	manager := NewUpgradeManager(newCatalogTestDB(t), nil, nil, nil)
	manager.Start(context.Background())
	manager.Start(context.Background())
	manager.Stop()
	manager.Stop()
}

func TestUpgradeReusesExistingReleaseValues(t *testing.T) {
	client := action.NewUpgrade(new(action.Configuration))
	configureUpgrade(client, ChangeOptions{Namespace: "default", Timeout: time.Minute}, true)
	require.True(t, client.ResetThenReuseValues)
}

func TestConfigureInstallOnlyEnablesServerDryRunForPreview(t *testing.T) {
	preview := action.NewInstall(new(action.Configuration))
	configureInstall(preview, ChangeOptions{Namespace: "default", Timeout: time.Minute}, true)
	require.True(t, preview.DryRun)
	require.Equal(t, "server", preview.DryRunOption)

	execute := action.NewInstall(new(action.Configuration))
	configureInstall(execute, ChangeOptions{Namespace: "default", Timeout: time.Minute}, false)
	require.False(t, execute.DryRun)
	require.Empty(t, execute.DryRunOption)
}

func TestConfigureUpgradeOnlyEnablesServerDryRunForPreview(t *testing.T) {
	preview := action.NewUpgrade(new(action.Configuration))
	configureUpgrade(preview, ChangeOptions{Namespace: "default", Timeout: time.Minute}, true)
	require.True(t, preview.DryRun)
	require.Equal(t, "server", preview.DryRunOption)

	execute := action.NewUpgrade(new(action.Configuration))
	configureUpgrade(execute, ChangeOptions{Namespace: "default", Timeout: time.Minute}, false)
	require.False(t, execute.DryRun)
	require.Empty(t, execute.DryRunOption)
}

func testChartArchive(t *testing.T) []byte {
	t.Helper()
	var output bytes.Buffer
	gzipWriter := gzip.NewWriter(&output)
	tarWriter := tar.NewWriter(gzipWriter)
	files := map[string]string{"demo/Chart.yaml": "apiVersion: v2\nname: demo\nversion: 1.0.0\n", "demo/values.yaml": "replicas: 1\n", "demo/README.md": "hello", "demo/templates/deployment.yaml": "apiVersion: apps/v1\nkind: Deployment\nmetadata:\n  name: demo\n"}
	for name, content := range files {
		require.NoError(t, tarWriter.WriteHeader(&tar.Header{Name: name, Mode: 0600, Size: int64(len(content))}))
		_, err := tarWriter.Write([]byte(content))
		require.NoError(t, err)
	}
	require.NoError(t, tarWriter.Close())
	require.NoError(t, gzipWriter.Close())
	return output.Bytes()
}
