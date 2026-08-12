package repository

import (
	"context"

	"github.com/gocronx/kubevision/internal/model"
	"gorm.io/gorm"
)

type directoryRepo struct{ db *gorm.DB }

func NewDirectoryRepo(db *gorm.DB) DirectoryRepo { return &directoryRepo{db: db} }

func (r *directoryRepo) GetConfig(ctx context.Context) (*model.DirectoryConfig, error) {
	var cfg model.DirectoryConfig
	err := r.db.WithContext(ctx).First(&cfg, 1).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &cfg, err
}

func (r *directoryRepo) ListMappings(ctx context.Context) ([]model.DirectoryRoleMapping, error) {
	var mappings []model.DirectoryRoleMapping
	err := r.db.WithContext(ctx).Order("priority ASC, id ASC").Find(&mappings).Error
	return mappings, err
}

func (r *directoryRepo) SaveConfig(ctx context.Context, cfg *model.DirectoryConfig, mappings []model.DirectoryRoleMapping) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		cfg.ID = 1
		if err := tx.Save(cfg).Error; err != nil {
			return err
		}
		if err := tx.Where("1 = 1").Delete(&model.DirectoryRoleMapping{}).Error; err != nil {
			return err
		}
		for i := range mappings {
			mappings[i].ID = 0
			if err := tx.Create(&mappings[i]).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
