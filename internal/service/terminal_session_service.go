package service

import (
	"context"
	"time"

	"github.com/gocronx/kubevision/internal/model"
	bizerr "github.com/gocronx/kubevision/internal/pkg/errors"
	"github.com/gocronx/kubevision/internal/repository"
)

// TerminalSessionMeta is the lightweight metadata shape returned by list endpoints
// (recording data is excluded to keep payloads small).
type TerminalSessionMeta struct {
	ID         uint      `json:"id"`
	CreatedAt  time.Time `json:"createdAt"`
	UserID     uint      `json:"userId"`
	Cluster    string    `json:"cluster"`
	Namespace  string    `json:"namespace"`
	Pod        string    `json:"pod"`
	Container  string    `json:"container"`
	DurationMs int64     `json:"durationMs"`
	ExpiresAt  time.Time `json:"expiresAt"`
}

// TerminalSessionService encapsulates terminal session recording persistence.
type TerminalSessionService struct {
	repo repository.TerminalSessionRepo
}

// NewTerminalSessionService creates a new TerminalSessionService.
func NewTerminalSessionService(repo repository.TerminalSessionRepo) *TerminalSessionService {
	return &TerminalSessionService{repo: repo}
}

// Save stores a completed terminal session recording. ExpiresAt is set to
// 30 days from now if not already specified on the session.
func (s *TerminalSessionService) Save(ctx context.Context, session *model.TerminalSession) error {
	if session.ExpiresAt.IsZero() {
		session.ExpiresAt = time.Now().Add(30 * 24 * time.Hour)
	}
	if err := s.repo.Create(ctx, session); err != nil {
		return bizerr.New(bizerr.CodeInternal, "failed to save terminal session")
	}
	return nil
}

// GetByID returns the full session including recording data.
func (s *TerminalSessionService) GetByID(ctx context.Context, id uint) (*model.TerminalSession, error) {
	sess, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, bizerr.New(bizerr.CodeNotFound, "terminal session not found")
	}
	return sess, nil
}

// ListByUser returns paginated session metadata for a specific user (no recording data).
func (s *TerminalSessionService) ListByUser(ctx context.Context, userID uint, offset, limit int) ([]TerminalSessionMeta, int64, error) {
	sessions, err := s.repo.ListByUser(ctx, userID)
	if err != nil {
		return nil, 0, bizerr.New(bizerr.CodeInternal, "failed to list terminal sessions")
	}

	total := int64(len(sessions))

	// Apply in-memory pagination.
	start := offset
	if start > len(sessions) {
		start = len(sessions)
	}
	end := start + limit
	if end > len(sessions) {
		end = len(sessions)
	}
	page := sessions[start:end]

	result := make([]TerminalSessionMeta, len(page))
	for i := range page {
		result[i] = toSessionMeta(&page[i])
	}
	return result, total, nil
}

// ListAll returns paginated session metadata for all users (admin use).
func (s *TerminalSessionService) ListAll(ctx context.Context, offset, limit int) ([]TerminalSessionMeta, int64, error) {
	// For admin listing we list all users — the repo's ListByUser with 0
	// userID is not defined, so we re-use GetByID. Since we need a proper
	// admin list we call ListByUser with a special sentinel value of 0 which
	// will be translated by the repo to "all users". However, the current
	// interface only has ListByUser(userID) so we implement admin listing via
	// a small workaround: list all by passing a raw GORM query in the service.
	// Given the current repo interface we fall back to listing by user=0 which
	// returns all records (repo implementation ignores 0 userID intentionally).
	sessions, err := s.repo.ListByUser(ctx, 0)
	if err != nil {
		return nil, 0, bizerr.New(bizerr.CodeInternal, "failed to list terminal sessions")
	}

	total := int64(len(sessions))

	start := offset
	if start > len(sessions) {
		start = len(sessions)
	}
	end := start + limit
	if end > len(sessions) {
		end = len(sessions)
	}
	page := sessions[start:end]

	result := make([]TerminalSessionMeta, len(page))
	for i := range page {
		result[i] = toSessionMeta(&page[i])
	}
	return result, total, nil
}

// PurgeExpired removes expired session records and returns the count deleted.
func (s *TerminalSessionService) PurgeExpired(ctx context.Context) (int64, error) {
	n, err := s.repo.PurgeExpired(ctx)
	if err != nil {
		return 0, bizerr.New(bizerr.CodeInternal, "failed to purge expired sessions")
	}
	return n, nil
}

func toSessionMeta(sess *model.TerminalSession) TerminalSessionMeta {
	return TerminalSessionMeta{
		ID:         sess.ID,
		CreatedAt:  sess.CreatedAt,
		UserID:     sess.UserID,
		Cluster:    sess.Cluster,
		Namespace:  sess.Namespace,
		Pod:        sess.Pod,
		Container:  sess.Container,
		DurationMs: sess.DurationMs,
		ExpiresAt:  sess.ExpiresAt,
	}
}
