package repository

import (
	"fmt"

	"github.com/kubevision/kubevision/internal/auth"
	"github.com/kubevision/kubevision/internal/config"
	"github.com/kubevision/kubevision/internal/model"
	"go.uber.org/zap"
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
	case "postgresql":
		// TODO: gorm.io/driver/postgres
		return nil, fmt.Errorf("postgresql driver not yet implemented")
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
			Role:         "admin",
			IsActive:     true,
		}
		if err := db.Create(&admin).Error; err != nil {
			logger.Warn("failed to create default admin", zap.Error(err))
		} else {
			logger.Info("default admin user created (password: admin)")
		}
	}
}

// initSystemRoles seeds the built-in system roles if they do not already exist.
func initSystemRoles(db *gorm.DB, logger *zap.Logger) {
	roles := []model.Role{
		{Name: "admin", DisplayName: "Administrator", IsSystem: true, Permissions: `["*:*"]`},
		{Name: "ops", DisplayName: "Operations", IsSystem: true, Permissions: `["*:get","*:list","*:create","*:update","*:delete","*:exec","*:logs"]`},
		{Name: "dev", DisplayName: "Developer", IsSystem: true, Permissions: `["*:get","*:list","pods:exec","pods:logs"]`},
		{Name: "readonly", DisplayName: "Read Only", IsSystem: true, Permissions: `["*:get","*:list"]`},
	}
	silent := db.Session(&gorm.Session{Logger: gormlogger.Discard})
	for _, role := range roles {
		var existing model.Role
		if silent.Where("name = ?", role.Name).First(&existing).Error != nil {
			if err := db.Create(&role).Error; err != nil {
				logger.Warn("failed to create system role", zap.String("role", role.Name), zap.Error(err))
			}
		}
	}
}
