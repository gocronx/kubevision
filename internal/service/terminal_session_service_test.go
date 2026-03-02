package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/kubevision/kubevision/internal/model"
)

// ---------------------------------------------------------------------------
// Mock TerminalSessionRepo
// ---------------------------------------------------------------------------

type mockTerminalSessionRepo struct {
	sessions map[uint]*model.TerminalSession
	nextID   uint
}

func newMockTerminalSessionRepo() *mockTerminalSessionRepo {
	return &mockTerminalSessionRepo{
		sessions: make(map[uint]*model.TerminalSession),
		nextID:   1,
	}
}

func (m *mockTerminalSessionRepo) Create(_ context.Context, s *model.TerminalSession) error {
	s.ID = m.nextID
	s.CreatedAt = time.Now()
	m.nextID++
	cp := *s
	m.sessions[s.ID] = &cp
	return nil
}

func (m *mockTerminalSessionRepo) GetByID(_ context.Context, id uint) (*model.TerminalSession, error) {
	s, ok := m.sessions[id]
	if !ok {
		return nil, errors.New("not found")
	}
	cp := *s
	return &cp, nil
}

func (m *mockTerminalSessionRepo) ListByUser(_ context.Context, userID uint) ([]model.TerminalSession, error) {
	var result []model.TerminalSession
	for _, s := range m.sessions {
		if userID == 0 || s.UserID == userID {
			result = append(result, *s)
		}
	}
	return result, nil
}

func (m *mockTerminalSessionRepo) PurgeExpired(_ context.Context) (int64, error) {
	var deleted int64
	for id, s := range m.sessions {
		if time.Now().After(s.ExpiresAt) {
			delete(m.sessions, id)
			deleted++
		}
	}
	return deleted, nil
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestTerminalSessionService_Save(t *testing.T) {
	repo := newMockTerminalSessionRepo()
	svc := NewTerminalSessionService(repo)
	ctx := context.Background()

	sess := &model.TerminalSession{
		UserID:    1,
		Cluster:   "prod",
		Pod:       "nginx",
		Container: "nginx",
		Recording: `{"version":2}`,
	}
	err := svc.Save(ctx, sess)
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}
	if sess.ExpiresAt.IsZero() {
		t.Error("ExpiresAt should be set automatically")
	}
}

func TestTerminalSessionService_GetByID(t *testing.T) {
	repo := newMockTerminalSessionRepo()
	svc := NewTerminalSessionService(repo)
	ctx := context.Background()

	sess := &model.TerminalSession{UserID: 1, Pod: "test-pod"}
	_ = svc.Save(ctx, sess)

	got, err := svc.GetByID(ctx, sess.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if got.Pod != "test-pod" {
		t.Errorf("expected pod 'test-pod', got %q", got.Pod)
	}
}

func TestTerminalSessionService_GetByID_NotFound(t *testing.T) {
	repo := newMockTerminalSessionRepo()
	svc := NewTerminalSessionService(repo)
	ctx := context.Background()

	_, err := svc.GetByID(ctx, 999)
	if err == nil {
		t.Fatal("expected error for non-existent session")
	}
}

func TestTerminalSessionService_ListByUser(t *testing.T) {
	repo := newMockTerminalSessionRepo()
	svc := NewTerminalSessionService(repo)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		_ = svc.Save(ctx, &model.TerminalSession{UserID: 1, Pod: "pod"})
	}

	metas, total, err := svc.ListByUser(ctx, 1, 0, 3)
	if err != nil {
		t.Fatalf("ListByUser failed: %v", err)
	}
	if total != 5 {
		t.Errorf("expected total 5, got %d", total)
	}
	if len(metas) != 3 {
		t.Errorf("expected 3 results (limit), got %d", len(metas))
	}
}

func TestTerminalSessionService_ListAll(t *testing.T) {
	repo := newMockTerminalSessionRepo()
	svc := NewTerminalSessionService(repo)
	ctx := context.Background()

	_ = svc.Save(ctx, &model.TerminalSession{UserID: 1, Pod: "pod-1"})
	_ = svc.Save(ctx, &model.TerminalSession{UserID: 2, Pod: "pod-2"})

	metas, total, err := svc.ListAll(ctx, 0, 10)
	if err != nil {
		t.Fatalf("ListAll failed: %v", err)
	}
	if total != 2 {
		t.Errorf("expected total 2, got %d", total)
	}
	if len(metas) != 2 {
		t.Errorf("expected 2 results, got %d", len(metas))
	}
}

func TestTerminalSessionService_PurgeExpired(t *testing.T) {
	repo := newMockTerminalSessionRepo()
	svc := NewTerminalSessionService(repo)
	ctx := context.Background()

	// Add an expired session directly to mock.
	repo.sessions[100] = &model.TerminalSession{
		ID:        100,
		UserID:    1,
		Pod:       "expired",
		ExpiresAt: time.Now().Add(-1 * time.Hour),
	}

	_ = svc.Save(ctx, &model.TerminalSession{UserID: 1, Pod: "active"})

	deleted, err := svc.PurgeExpired(ctx)
	if err != nil {
		t.Fatalf("PurgeExpired failed: %v", err)
	}
	if deleted != 1 {
		t.Errorf("expected 1 deleted, got %d", deleted)
	}
}
