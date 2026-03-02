package repository

import (
	"context"
	"time"

	"github.com/kubevision/kubevision/internal/model"
	"gorm.io/gorm"
)

// auditRepo is the GORM-backed implementation of AuditRepo.
type auditRepo struct {
	db *gorm.DB
}

// NewAuditRepo returns a new AuditRepo backed by the given GORM database.
func NewAuditRepo(db *gorm.DB) AuditRepo {
	return &auditRepo{db: db}
}

// BatchCreate inserts multiple audit log entries in batches of 100.
func (r *auditRepo) BatchCreate(ctx context.Context, logs []model.AuditLog) error {
	return r.db.WithContext(ctx).CreateInBatches(logs, 100).Error
}

// List returns paginated audit logs matching the given filter. It returns the
// matching entries, the total count (before pagination), and any error.
func (r *auditRepo) List(ctx context.Context, filter AuditFilter) ([]model.AuditLog, int64, error) {
	query := r.db.WithContext(ctx).Model(&model.AuditLog{})

	if filter.UserID > 0 {
		query = query.Where("user_id = ?", filter.UserID)
	}
	if filter.Action != "" {
		query = query.Where("action = ?", filter.Action)
	}
	if filter.Cluster != "" {
		query = query.Where("cluster = ?", filter.Cluster)
	}
	if !filter.Since.IsZero() {
		query = query.Where("created_at >= ?", filter.Since)
	}

	var total int64
	query.Count(&total)

	var logs []model.AuditLog
	err := query.Order("created_at DESC").Offset(filter.Offset).Limit(filter.Limit).Find(&logs).Error
	return logs, total, err
}

// Purge deletes audit log entries created before the given time and returns
// the number of rows deleted.
func (r *auditRepo) Purge(ctx context.Context, before time.Time) (int64, error) {
	result := r.db.WithContext(ctx).Where("created_at < ?", before).Delete(&model.AuditLog{})
	return result.RowsAffected, result.Error
}
