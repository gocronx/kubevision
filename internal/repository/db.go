package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/gocronx/kubevision/internal/auth"
	"github.com/gocronx/kubevision/internal/config"
	"github.com/gocronx/kubevision/internal/model"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// NewDB initialises a GORM database connection based on the application
// configuration, runs auto-migration for all known models, and seeds default
// data (admin user and system roles).
func NewDB(cfg *config.Config, logger *zap.Logger) (*gorm.DB, error) {
	var db *gorm.DB
	var err error

	switch cfg.Database.Driver {
	case "sqlite":
		db, err = gorm.Open(sqlite.Open(cfg.Database.DSN), &gorm.Config{})
	case "postgres", "postgresql":
		db, err = gorm.Open(postgres.Open(cfg.Database.DSN), &gorm.Config{})
	default:
		return nil, fmt.Errorf("unsupported database driver: %s", cfg.Database.Driver)
	}
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("get database connection: %w", err)
	}
	initialized := false
	defer func() {
		if !initialized {
			_ = sqlDB.Close()
		}
	}()
	configureConnectionPool(sqlDB, cfg.Database)
	pingTimeout := cfg.Database.PingTimeout
	if pingTimeout <= 0 {
		pingTimeout = 5 * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), pingTimeout)
	defer cancel()
	if err := sqlDB.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("ping database: %w", err)
	}

	// 启用 WAL 模式（SQLite）
	if cfg.Database.Driver == "sqlite" {
		db.Exec("PRAGMA journal_mode=WAL")
		db.Exec("PRAGMA foreign_keys=ON")
	}

	if err := runMigrations(db, cfg.Database.Driver); err != nil {
		return nil, fmt.Errorf("run database migrations: %w", err)
	}

	// Older versions soft-deleted cluster registrations while keeping a global
	// unique index on name. Those hidden rows prevented importing the same
	// cluster again and retained credentials after the user chose Remove.
	if err := purgeDeletedClusters(db); err != nil {
		return nil, fmt.Errorf("purge removed clusters: %w", err)
	}

	// 初始化默认 admin 用户（如果不存在）
	initDefaultAdmin(db, logger)
	// 初始化系统角色
	initSystemRoles(db, logger)

	initialized = true
	return db, nil
}

type schemaMigration struct {
	Version   int64     `gorm:"primaryKey"`
	AppliedAt time.Time `gorm:"not null"`
}

func (schemaMigration) TableName() string { return "schema_migrations" }

type databaseMigration struct {
	version int64
	up      func(*gorm.DB) error
}

var databaseMigrations = []databaseMigration{
	{
		version: 1,
		up: func(db *gorm.DB) error {
			return db.AutoMigrate(
				&model.User{},
				&model.PublicKeyCredential{},
				&model.PublicKeyCeremony{},
				&model.Cluster{},
				&model.Role{},
				&model.UserClusterRole{},
				&model.AuditLog{},
				&model.APIKey{},
				&model.Template{},
				&model.Setting{},
				&model.Webhook{},
				&model.Favorite{},
				&model.TerminalSession{},
				&model.PluginConfig{},
				&model.DirectoryConfig{},
				&model.DirectoryRoleMapping{},
			)
		},
	},
	{
		version: 2,
		up: func(db *gorm.DB) error {
			return db.AutoMigrate(&model.HelmRepository{}, &model.HelmUpgradePolicy{})
		},
	},
}

func runMigrations(db *gorm.DB, driver string) error {
	return db.Transaction(func(tx *gorm.DB) error {
		if driver == "postgres" || driver == "postgresql" {
			// Serialize schema changes when multiple replicas start together.
			if err := tx.Exec("SELECT pg_advisory_xact_lock(?)", int64(0x4b564953494f4e)).Error; err != nil {
				return fmt.Errorf("acquire migration lock: %w", err)
			}
		}
		if err := tx.AutoMigrate(&schemaMigration{}); err != nil {
			return fmt.Errorf("create migration history: %w", err)
		}
		for _, migration := range databaseMigrations {
			var count int64
			if err := tx.Model(&schemaMigration{}).Where("version = ?", migration.version).Count(&count).Error; err != nil {
				return err
			}
			if count != 0 {
				continue
			}
			if err := migration.up(tx); err != nil {
				return fmt.Errorf("apply migration %d: %w", migration.version, err)
			}
			if err := tx.Create(&schemaMigration{Version: migration.version, AppliedAt: time.Now().UTC()}).Error; err != nil {
				return fmt.Errorf("record migration %d: %w", migration.version, err)
			}
		}
		return nil
	})
}

func configureConnectionPool(db *sql.DB, cfg config.DatabaseConfig) {
	maxOpen, maxIdle := cfg.MaxOpenConns, cfg.MaxIdleConns
	if maxOpen <= 0 {
		if cfg.Driver == "postgres" || cfg.Driver == "postgresql" {
			maxOpen = 25
		} else {
			maxOpen = 1
		}
	}
	if maxIdle <= 0 {
		if cfg.Driver == "postgres" || cfg.Driver == "postgresql" {
			maxIdle = 5
		} else {
			maxIdle = 1
		}
	}
	if maxIdle > maxOpen {
		maxIdle = maxOpen
	}
	db.SetMaxOpenConns(maxOpen)
	db.SetMaxIdleConns(maxIdle)
	if cfg.ConnMaxLifetime > 0 {
		db.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	} else if cfg.Driver == "postgres" || cfg.Driver == "postgresql" {
		db.SetConnMaxLifetime(30 * time.Minute)
	}
	if cfg.ConnMaxIdleTime > 0 {
		db.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)
	} else if cfg.Driver == "postgres" || cfg.Driver == "postgresql" {
		db.SetConnMaxIdleTime(5 * time.Minute)
	}
}

func purgeDeletedClusters(db *gorm.DB) error {
	return db.Unscoped().Where("deleted_at IS NOT NULL").Delete(&model.Cluster{}).Error
}

// initDefaultAdmin creates the default admin user if no users exist yet.
// The default password is "admin" and should be changed immediately in production.
func initDefaultAdmin(db *gorm.DB, logger *zap.Logger) {
	var count int64
	db.Model(&model.User{}).Count(&count)
	if count == 0 {
		hash, err := auth.HashPassword("admin123")
		if err != nil {
			logger.Warn("failed to hash default admin password", zap.Error(err))
			return
		}
		admin := model.User{
			Username:     "admin",
			PasswordHash: hash,
			Role:         "super-admin",
			IsActive:     true,
		}
		if err := db.Create(&admin).Error; err != nil {
			logger.Warn("failed to create default admin", zap.Error(err))
		} else {
			logger.Info("default admin user created", zap.String("username", "admin"))
		}
	}
}

// initSystemRoles seeds the built-in system roles if they do not already exist.
//
// Permission format: "<resource>:<action>" where "*" is a wildcard.
// Resources are derived from the last non-parameter URL segment by the RBAC
// middleware (e.g. /clusters/:id/resources/:resource → "resources").
//
// Role hierarchy:
//   - super-admin: full access, RBAC bypass in middleware, user management visible in UI
//   - admin:       full access, RBAC bypass in middleware, user management hidden in UI
//   - editor:      CRUD on K8s resources + cluster views; cannot touch users/webhooks/audit/api-keys
//   - viewer:      read-only on K8s resources + cluster views
//   - custom:      minimal read-only baseline; permissions can be edited by admins
func initSystemRoles(db *gorm.DB, logger *zap.Logger) {
	roles := []model.Role{
		{Name: "super-admin", DisplayName: "Super Administrator", IsSystem: true, Permissions: `["*:*"]`},
		{Name: "admin", DisplayName: "Administrator", IsSystem: true, Permissions: `["*:*"]`},
		{
			Name:        "editor",
			DisplayName: "Editor",
			IsSystem:    true,
			// Can CRUD K8s resources, use cluster tools, and manage favorites.
			// Cannot access users, webhooks, audit-logs, api-keys, or terminal-sessions.
			// pods:exec  — required by the WebSocket terminal handler (checkWSPermission).
			// pods:list  — required by the WebSocket log-streaming handler (checkWSPermission).
			// overview:list — required by the cluster overview endpoint.
			Permissions: `["clusters:get","clusters:list","resources:get","resources:list","resources:create","resources:update","resources:delete","favorites:get","favorites:list","favorites:create","favorites:delete","search:list","topology:list","compare:create","quota-summary:list","scale:update","restart:create","history:list","rollback:create","batch-delete:create","batch-restart:create","pods:exec","pods:list","overview:list","registry-tags:list","package-releases:read","package-releases:install","package-releases:upgrade","package-releases:rollback","package-releases:remove"]`,
		},
		{
			Name:        "viewer",
			DisplayName: "Viewer",
			IsSystem:    true,
			// Read-only access to K8s resources and cluster views.
			// pods:list  — required by the WebSocket log-streaming handler (checkWSPermission).
			// overview:list — required by the cluster overview endpoint.
			Permissions: `["clusters:get","clusters:list","resources:get","resources:list","favorites:get","favorites:list","favorites:create","favorites:delete","search:list","topology:list","compare:create","quota-summary:list","history:list","pods:list","overview:list","registry-tags:list","package-releases:read"]`,
		},
		{
			Name:        "custom",
			DisplayName: "Custom",
			IsSystem:    false,
			// Minimal read-only baseline; admins can update this role's Permissions JSON.
			Permissions: `["clusters:get","clusters:list","resources:get","resources:list"]`,
		},
	}
	silent := db.Session(&gorm.Session{Logger: gormlogger.Discard})
	for _, role := range roles {
		var existing model.Role
		if silent.Where("name = ?", role.Name).First(&existing).Error != nil {
			// Role does not exist yet — create it.
			if err := db.Create(&role).Error; err != nil {
				logger.Warn("failed to create system role", zap.String("role", role.Name), zap.Error(err))
			}
		} else {
			// Role exists — update its permissions and display name so that
			// permission additions introduced by security patches are applied
			// to already-running databases without requiring a manual migration.
			existing.Permissions = role.Permissions
			existing.DisplayName = role.DisplayName
			if err := db.Save(&existing).Error; err != nil {
				logger.Warn("failed to update system role permissions", zap.String("role", role.Name), zap.Error(err))
			}
		}
	}
}
