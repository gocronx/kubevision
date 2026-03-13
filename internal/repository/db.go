package repository

import (
	"fmt"

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

	// 启用 WAL 模式（SQLite）
	if cfg.Database.Driver == "sqlite" {
		db.Exec("PRAGMA journal_mode=WAL")
		db.Exec("PRAGMA foreign_keys=ON")
	}

	// 自动迁移
	if err := db.AutoMigrate(
		&model.User{},
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
	); err != nil {
		return nil, fmt.Errorf("auto migrate: %w", err)
	}

	// 初始化默认 admin 用户（如果不存在）
	initDefaultAdmin(db, logger)
	// 初始化系统角色
	initSystemRoles(db, logger)

	return db, nil
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
			Permissions: `["clusters:get","clusters:list","resources:get","resources:list","resources:create","resources:update","resources:delete","favorites:get","favorites:list","favorites:create","favorites:delete","search:list","topology:list","compare:create","quota-summary:list","scale:update","restart:create","history:list","rollback:create","batch-delete:create","batch-restart:create","pods:exec","pods:list","overview:list"]`,
		},
		{
			Name:        "viewer",
			DisplayName: "Viewer",
			IsSystem:    true,
			// Read-only access to K8s resources and cluster views.
			// pods:list  — required by the WebSocket log-streaming handler (checkWSPermission).
			// overview:list — required by the cluster overview endpoint.
			Permissions: `["clusters:get","clusters:list","resources:get","resources:list","favorites:get","favorites:list","favorites:create","favorites:delete","search:list","topology:list","compare:create","quota-summary:list","history:list","pods:list","overview:list"]`,
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
