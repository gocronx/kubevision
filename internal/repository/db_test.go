package repository

import (
	"fmt"
	"os"
	"testing"

	"github.com/gocronx/kubevision/internal/auth"
	"github.com/gocronx/kubevision/internal/config"
	"github.com/gocronx/kubevision/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupTestDB creates an in-memory SQLite database for testing, with all
// migrations and seed data applied. Each call creates an isolated database
// using a unique DSN to prevent test interference.
func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	// Use a unique file name to ensure each test gets its own isolated DB.
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	cfg := &config.Config{
		Database: config.DatabaseConfig{
			Driver: "sqlite",
			DSN:    dsn,
		},
	}
	logger, _ := zap.NewDevelopment()
	db, err := NewDB(cfg, logger)
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, sqlDB.Close()) })
	return db
}

func TestNewDB_CreatesDatabaseAndRunsMigrations(t *testing.T) {
	db := setupTestDB(t)
	var migrationCount int64
	require.NoError(t, db.Model(&schemaMigration{}).Count(&migrationCount).Error)
	assert.Equal(t, int64(len(databaseMigrations)), migrationCount)

	// Verify that the users table exists by querying it.
	var count int64
	err := db.Model(&model.User{}).Count(&count).Error
	require.NoError(t, err, "users table should exist after migration")

	// Verify that the clusters table exists.
	err = db.Model(&model.Cluster{}).Count(&count).Error
	require.NoError(t, err, "clusters table should exist after migration")

	// Verify that the roles table exists.
	err = db.Model(&model.Role{}).Count(&count).Error
	require.NoError(t, err, "roles table should exist after migration")

	// Verify other migrated tables.
	err = db.Model(&model.AuditLog{}).Count(&count).Error
	require.NoError(t, err, "audit_logs table should exist after migration")

	err = db.Model(&model.APIKey{}).Count(&count).Error
	require.NoError(t, err, "api_keys table should exist after migration")

	err = db.Model(&model.Template{}).Count(&count).Error
	require.NoError(t, err, "templates table should exist after migration")

	err = db.Model(&model.Setting{}).Count(&count).Error
	require.NoError(t, err, "settings table should exist after migration")

	err = db.Model(&model.Webhook{}).Count(&count).Error
	require.NoError(t, err, "webhooks table should exist after migration")
}

func TestPurgeDeletedClustersReleasesNames(t *testing.T) {
	db := setupTestDB(t)
	cluster := &model.Cluster{Name: "local", AuthType: "kubeconfig", Status: "unhealthy"}
	require.NoError(t, db.Create(cluster).Error)
	require.NoError(t, db.Delete(cluster).Error)

	require.NoError(t, purgeDeletedClusters(db))
	require.NoError(t, db.Create(&model.Cluster{
		Name:     "local",
		AuthType: "kubeconfig",
		Status:   "healthy",
	}).Error, "purging a removed registration should release its unique name")
}

func TestNewDB_DefaultAdminUserCreated(t *testing.T) {
	db := setupTestDB(t)

	var admin model.User
	err := db.Where("username = ?", "admin").First(&admin).Error
	require.NoError(t, err, "default admin user should exist")

	assert.Equal(t, "admin", admin.Username)
	assert.Equal(t, "super-admin", admin.Role)
	assert.True(t, admin.IsActive, "admin account should be active")
	assert.NotEmpty(t, admin.PasswordHash, "admin password hash should not be empty")
}

func TestNewDB_DefaultAdminPasswordIsValid(t *testing.T) {
	db := setupTestDB(t)

	var admin model.User
	err := db.Where("username = ?", "admin").First(&admin).Error
	require.NoError(t, err)

	// The default password is "admin123" (set in db.go initDefaultAdmin).
	assert.True(t, auth.CheckPassword("admin123", admin.PasswordHash),
		"default admin password 'admin123' should be valid")

	// Wrong password should not match.
	assert.False(t, auth.CheckPassword("wrongpassword", admin.PasswordHash),
		"wrong password should not match admin hash")
}

func TestNewDB_SystemRolesSeeded(t *testing.T) {
	db := setupTestDB(t)

	expectedRoles := []string{"super-admin", "admin", "editor", "viewer", "custom"}

	for _, roleName := range expectedRoles {
		var role model.Role
		err := db.Where("name = ?", roleName).First(&role).Error
		require.NoError(t, err, "system role %q should exist", roleName)
		assert.NotEmpty(t, role.DisplayName, "role %q should have a display name", roleName)
		assert.NotEmpty(t, role.Permissions, "role %q should have permissions", roleName)
	}

	// Verify the total number of roles is exactly 5.
	var count int64
	err := db.Model(&model.Role{}).Count(&count).Error
	require.NoError(t, err)
	assert.Equal(t, int64(5), count, "there should be exactly 5 system roles")
}

func TestNewDB_CalledTwiceDoesNotDuplicateSeedData(t *testing.T) {
	// Use a shared DSN so the second NewDB call sees the same database.
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	cfg := &config.Config{
		Database: config.DatabaseConfig{
			Driver: "sqlite",
			DSN:    dsn,
		},
	}
	logger, _ := zap.NewDevelopment()

	// First call initializes and seeds.
	db1, err := NewDB(cfg, logger)
	require.NoError(t, err)

	// Count users and roles after first call.
	var userCount1, roleCount1 int64
	db1.Model(&model.User{}).Count(&userCount1)
	db1.Model(&model.Role{}).Count(&roleCount1)

	// Second call should be idempotent.
	db2, err := NewDB(cfg, logger)
	require.NoError(t, err)

	var userCount2, roleCount2 int64
	db2.Model(&model.User{}).Count(&userCount2)
	db2.Model(&model.Role{}).Count(&roleCount2)

	assert.Equal(t, userCount1, userCount2, "user count should not change after second NewDB call")
	assert.Equal(t, roleCount1, roleCount2, "role count should not change after second NewDB call")

	// Verify exact counts.
	assert.Equal(t, int64(1), userCount2, "should have exactly 1 admin user")
	assert.Equal(t, int64(5), roleCount2, "should have exactly 5 system roles")
	var migrationCount int64
	require.NoError(t, db2.Model(&schemaMigration{}).Count(&migrationCount).Error)
	assert.Equal(t, int64(len(databaseMigrations)), migrationCount)
}

func TestNewDB_ConfiguresSQLiteForSingleWriter(t *testing.T) {
	db := setupTestDB(t)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	assert.Equal(t, 1, sqlDB.Stats().MaxOpenConnections)
}

func TestNewDB_AdoptsLegacySQLiteSchemaWithoutLosingData(t *testing.T) {
	dsn := t.TempDir() + "/legacy.db"
	legacy, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, legacy.AutoMigrate(&model.User{}))
	require.NoError(t, legacy.Create(&model.User{
		Username: "existing", PasswordHash: "hash", Role: "viewer", IsActive: true,
	}).Error)
	legacySQL, err := legacy.DB()
	require.NoError(t, err)
	require.NoError(t, legacySQL.Close())

	db, err := NewDB(&config.Config{Database: config.DatabaseConfig{Driver: "sqlite", DSN: dsn}}, zap.NewNop())
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, sqlDB.Close()) })

	var existing model.User
	require.NoError(t, db.Where("username = ?", "existing").First(&existing).Error)
	var migrationCount int64
	require.NoError(t, db.Model(&schemaMigration{}).Count(&migrationCount).Error)
	assert.Equal(t, int64(len(databaseMigrations)), migrationCount)
}

func TestNewDB_PostgresIntegration(t *testing.T) {
	dsn := os.Getenv("KUBEVISION_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("KUBEVISION_TEST_POSTGRES_DSN is not configured")
	}
	cfg := &config.Config{Database: config.DatabaseConfig{
		Driver: "postgres", DSN: dsn, MaxOpenConns: 7, MaxIdleConns: 3,
	}}

	db, err := NewDB(cfg, zap.NewNop())
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, sqlDB.Close()) })
	assert.Equal(t, 7, sqlDB.Stats().MaxOpenConnections)

	var migrationCount int64
	require.NoError(t, db.Model(&schemaMigration{}).Count(&migrationCount).Error)
	assert.Equal(t, int64(len(databaseMigrations)), migrationCount)

	// A second startup must observe the migration history and remain idempotent.
	db2, err := NewDB(cfg, zap.NewNop())
	require.NoError(t, err)
	sqlDB2, err := db2.DB()
	require.NoError(t, err)
	require.NoError(t, sqlDB2.Close())
}
