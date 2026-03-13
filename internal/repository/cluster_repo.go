package repository

import (
	"context"

	"github.com/gocronx/kubevision/internal/model"
	"gorm.io/gorm"
)

// clusterRepo is the GORM-backed implementation of ClusterRepo.
type clusterRepo struct {
	db *gorm.DB
}

// NewClusterRepo returns a new ClusterRepo backed by the given GORM database.
func NewClusterRepo(db *gorm.DB) ClusterRepo {
	return &clusterRepo{db: db}
}

func (r *clusterRepo) Create(ctx context.Context, cluster *model.Cluster) error {
	return r.db.WithContext(ctx).Create(cluster).Error
}

func (r *clusterRepo) GetByID(ctx context.Context, id uint) (*model.Cluster, error) {
	var cluster model.Cluster
	if err := r.db.WithContext(ctx).First(&cluster, id).Error; err != nil {
		return nil, err
	}
	return &cluster, nil
}

func (r *clusterRepo) GetByName(ctx context.Context, name string) (*model.Cluster, error) {
	var cluster model.Cluster
	if err := r.db.WithContext(ctx).Where("name = ?", name).First(&cluster).Error; err != nil {
		return nil, err
	}
	return &cluster, nil
}

func (r *clusterRepo) Update(ctx context.Context, cluster *model.Cluster) error {
	return r.db.WithContext(ctx).Save(cluster).Error
}

func (r *clusterRepo) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&model.Cluster{}, id).Error
}

func (r *clusterRepo) List(ctx context.Context) ([]model.Cluster, error) {
	var clusters []model.Cluster
	if err := r.db.WithContext(ctx).Find(&clusters).Error; err != nil {
		return nil, err
	}
	return clusters, nil
}
