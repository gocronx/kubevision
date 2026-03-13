package repository

import (
	"context"

	"github.com/gocronx/kubevision/internal/model"
	"gorm.io/gorm"
)

// pluginConfigRepo is the GORM-backed implementation of PluginConfigRepo.
type pluginConfigRepo struct {
	db *gorm.DB
}

// NewPluginConfigRepo returns a new PluginConfigRepo backed by the given GORM database.
func NewPluginConfigRepo(db *gorm.DB) PluginConfigRepo {
	return &pluginConfigRepo{db: db}
}

func (r *pluginConfigRepo) Create(ctx context.Context, pc *model.PluginConfig) error {
	return r.db.WithContext(ctx).Create(pc).Error
}

func (r *pluginConfigRepo) GetByName(ctx context.Context, name string) (*model.PluginConfig, error) {
	var pc model.PluginConfig
	if err := r.db.WithContext(ctx).Where("name = ?", name).First(&pc).Error; err != nil {
		return nil, err
	}
	return &pc, nil
}

func (r *pluginConfigRepo) Update(ctx context.Context, pc *model.PluginConfig) error {
	return r.db.WithContext(ctx).Save(pc).Error
}

func (r *pluginConfigRepo) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&model.PluginConfig{}, id).Error
}

func (r *pluginConfigRepo) List(ctx context.Context) ([]model.PluginConfig, error) {
	var plugins []model.PluginConfig
	if err := r.db.WithContext(ctx).Find(&plugins).Error; err != nil {
		return nil, err
	}
	return plugins, nil
}
