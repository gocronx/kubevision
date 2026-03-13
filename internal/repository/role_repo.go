package repository

import (
	"context"

	"github.com/gocronx/kubevision/internal/model"
	"gorm.io/gorm"
)

// roleRepo is the GORM-backed implementation of RoleRepo.
type roleRepo struct {
	db *gorm.DB
}

// NewRoleRepo returns a new RoleRepo backed by the given GORM database.
func NewRoleRepo(db *gorm.DB) RoleRepo {
	return &roleRepo{db: db}
}

// Create inserts a new role record.
func (r *roleRepo) Create(ctx context.Context, role *model.Role) error {
	return r.db.WithContext(ctx).Create(role).Error
}

// GetByID retrieves a role by its primary key.
func (r *roleRepo) GetByID(ctx context.Context, id uint) (*model.Role, error) {
	var role model.Role
	if err := r.db.WithContext(ctx).First(&role, id).Error; err != nil {
		return nil, err
	}
	return &role, nil
}

// GetByName retrieves a role by its unique name.
func (r *roleRepo) GetByName(ctx context.Context, name string) (*model.Role, error) {
	var role model.Role
	if err := r.db.WithContext(ctx).Where("name = ?", name).First(&role).Error; err != nil {
		return nil, err
	}
	return &role, nil
}

// Update saves changes to an existing role.
func (r *roleRepo) Update(ctx context.Context, role *model.Role) error {
	return r.db.WithContext(ctx).Save(role).Error
}

// Delete removes a role by its primary key.
func (r *roleRepo) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&model.Role{}, id).Error
}

// List returns all roles.
func (r *roleRepo) List(ctx context.Context) ([]model.Role, error) {
	var roles []model.Role
	if err := r.db.WithContext(ctx).Find(&roles).Error; err != nil {
		return nil, err
	}
	return roles, nil
}
