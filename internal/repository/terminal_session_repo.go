package repository

import (
	"context"
	"time"

	"github.com/kubevision/kubevision/internal/model"
	"gorm.io/gorm"
)

// terminalSessionRepo is the GORM-backed implementation of TerminalSessionRepo.
type terminalSessionRepo struct {
	db *gorm.DB
}

// NewTerminalSessionRepo returns a new TerminalSessionRepo backed by the given GORM database.
func NewTerminalSessionRepo(db *gorm.DB) TerminalSessionRepo {
	return &terminalSessionRepo{db: db}
}

func (r *terminalSessionRepo) Create(ctx context.Context, session *model.TerminalSession) error {
	return r.db.WithContext(ctx).Create(session).Error
}

func (r *terminalSessionRepo) GetByID(ctx context.Context, id uint) (*model.TerminalSession, error) {
	var session model.TerminalSession
	if err := r.db.WithContext(ctx).First(&session, id).Error; err != nil {
		return nil, err
	}
	return &session, nil
}

func (r *terminalSessionRepo) ListByUser(ctx context.Context, userID uint) ([]model.TerminalSession, error) {
	var sessions []model.TerminalSession
	// Omit Recording field to keep list responses lightweight.
	q := r.db.WithContext(ctx).
		Select("id", "created_at", "user_id", "cluster", "namespace", "pod", "container", "duration_ms", "expires_at").
		Order("created_at DESC")

	// userID == 0 is a sentinel meaning "all users" (admin listing).
	if userID != 0 {
		q = q.Where("user_id = ?", userID)
	}

	if err := q.Find(&sessions).Error; err != nil {
		return nil, err
	}
	return sessions, nil
}

func (r *terminalSessionRepo) PurgeExpired(ctx context.Context) (int64, error) {
	result := r.db.WithContext(ctx).
		Where("expires_at < ?", time.Now()).
		Delete(&model.TerminalSession{})
	return result.RowsAffected, result.Error
}
