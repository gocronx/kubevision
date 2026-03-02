package repository

import (
	"context"

	"github.com/kubevision/kubevision/internal/model"
	"gorm.io/gorm"
)

// apiKeyRepo is the GORM-backed implementation of APIKeyRepo.
type apiKeyRepo struct {
	db *gorm.DB
}

// NewAPIKeyRepo returns a new APIKeyRepo backed by the given GORM database.
func NewAPIKeyRepo(db *gorm.DB) APIKeyRepo {
	return &apiKeyRepo{db: db}
}

// Create inserts a new API key record.
func (r *apiKeyRepo) Create(ctx context.Context, key *model.APIKey) error {
	return r.db.WithContext(ctx).Create(key).Error
}

// GetByKeyHash retrieves an API key by its SHA256 hash.
func (r *apiKeyRepo) GetByKeyHash(ctx context.Context, keyHash string) (*model.APIKey, error) {
	var key model.APIKey
	if err := r.db.WithContext(ctx).Where("key_hash = ?", keyHash).First(&key).Error; err != nil {
		return nil, err
	}
	return &key, nil
}

// ListByUser returns all API keys belonging to the given user (hash is never returned).
func (r *apiKeyRepo) ListByUser(ctx context.Context, userID uint) ([]model.APIKey, error) {
	var keys []model.APIKey
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).Order("created_at DESC").Find(&keys).Error; err != nil {
		return nil, err
	}
	return keys, nil
}

// Delete removes an API key by its primary key.
func (r *apiKeyRepo) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&model.APIKey{}, id).Error
}
