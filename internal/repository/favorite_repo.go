package repository

import (
	"context"

	"github.com/gocronx/kubevision/internal/model"
	"gorm.io/gorm"
)

// favoriteRepo is the GORM-backed implementation of FavoriteRepo.
type favoriteRepo struct {
	db *gorm.DB
}

// NewFavoriteRepo returns a new FavoriteRepo backed by the given GORM database.
func NewFavoriteRepo(db *gorm.DB) FavoriteRepo {
	return &favoriteRepo{db: db}
}

// Create inserts a new Favorite record.
func (r *favoriteRepo) Create(ctx context.Context, fav *model.Favorite) error {
	return r.db.WithContext(ctx).Create(fav).Error
}

// Delete removes a Favorite record by primary key.
func (r *favoriteRepo) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&model.Favorite{}, id).Error
}

// ListByUser returns all favorites owned by a specific user, ordered by SortOrder ascending.
func (r *favoriteRepo) ListByUser(ctx context.Context, userID uint) ([]model.Favorite, error) {
	var favs []model.Favorite
	if err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("sort_order ASC, created_at ASC").
		Find(&favs).Error; err != nil {
		return nil, err
	}
	return favs, nil
}

// UpdateSortOrder sets the SortOrder field on a specific favorite record.
func (r *favoriteRepo) UpdateSortOrder(ctx context.Context, id uint, sortOrder int) error {
	return r.db.WithContext(ctx).
		Model(&model.Favorite{}).
		Where("id = ?", id).
		Update("sort_order", sortOrder).Error
}

// GetByUserAndResource finds a favorite by owner, cluster, resource type, name, and namespace.
// Returns (nil, nil) when not found — callers must distinguish by nil check.
func (r *favoriteRepo) GetByUserAndResource(
	ctx context.Context,
	userID uint,
	clusterID, resourceType, resourceName, namespace string,
) (*model.Favorite, error) {
	var fav model.Favorite
	err := r.db.WithContext(ctx).
		Where(
			"user_id = ? AND cluster_id = ? AND resource_type = ? AND resource_name = ? AND namespace = ?",
			userID, clusterID, resourceType, resourceName, namespace,
		).
		First(&fav).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &fav, nil
}
