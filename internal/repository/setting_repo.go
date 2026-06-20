package repository

import (
	"context"

	"github.com/gocronx/kubevision/internal/model"
	"gorm.io/gorm"
)

// settingRepo is the GORM-backed implementation of SettingRepo.
type settingRepo struct {
	db *gorm.DB
}

// NewSettingRepo returns a new SettingRepo backed by the given GORM database.
func NewSettingRepo(db *gorm.DB) SettingRepo {
	return &settingRepo{db: db}
}

func (r *settingRepo) Get(ctx context.Context, key string) (*model.Setting, error) {
	var s model.Setting
	if err := r.db.WithContext(ctx).First(&s, "key = ?", key).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &s, nil
}

// Set upserts a setting by its primary key.
func (r *settingRepo) Set(ctx context.Context, setting *model.Setting) error {
	return r.db.WithContext(ctx).Save(setting).Error
}

func (r *settingRepo) List(ctx context.Context, category string) ([]model.Setting, error) {
	var settings []model.Setting
	q := r.db.WithContext(ctx).Order("key ASC")
	if category != "" {
		q = q.Where("category = ?", category)
	}
	if err := q.Find(&settings).Error; err != nil {
		return nil, err
	}
	return settings, nil
}

func (r *settingRepo) Delete(ctx context.Context, key string) error {
	return r.db.WithContext(ctx).Delete(&model.Setting{}, "key = ?", key).Error
}
