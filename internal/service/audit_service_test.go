package service

import (
	"context"
	"sync"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/gocronx/kubevision/internal/config"
	"github.com/gocronx/kubevision/internal/model"
	"github.com/gocronx/kubevision/internal/repository"
)

// ---------------------------------------------------------------------------
// Mock AuditRepo
// ---------------------------------------------------------------------------

type mockAuditRepo struct {
	mu   sync.Mutex
	logs []model.AuditLog
}

func newMockAuditRepo() *mockAuditRepo {
	return &mockAuditRepo{}
}

func (m *mockAuditRepo) BatchCreate(_ context.Context, logs []model.AuditLog) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.logs = append(m.logs, logs...)
	return nil
}

func (m *mockAuditRepo) List(_ context.Context, _ repository.AuditFilter) ([]model.AuditLog, int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.logs, int64(len(m.logs)), nil
}

func (m *mockAuditRepo) Purge(_ context.Context, _ time.Time) (int64, error) {
	return 0, nil
}

func (m *mockAuditRepo) count() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.logs)
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestAuditService_RecordAndFlush(t *testing.T) {
	repo := newMockAuditRepo()
	logger, _ := zap.NewDevelopment()
	cfg := config.AuditConfig{
		Enabled:       true,
		FlushInterval: 50 * time.Millisecond,
	}
	svc := NewAuditService(repo, cfg, logger)
	svc.Start()

	svc.Record(model.AuditLog{Action: "create", Resource: "pods"})
	svc.Record(model.AuditLog{Action: "delete", Resource: "services"})

	// Wait for flush.
	time.Sleep(200 * time.Millisecond)

	if repo.count() != 2 {
		t.Errorf("expected 2 flushed logs, got %d", repo.count())
	}

	svc.Stop()
}

func TestAuditService_StopDrainsRemaining(t *testing.T) {
	repo := newMockAuditRepo()
	logger, _ := zap.NewDevelopment()
	cfg := config.AuditConfig{
		Enabled:       true,
		FlushInterval: 10 * time.Second, // Long interval so flush only on Stop.
	}
	svc := NewAuditService(repo, cfg, logger)
	svc.Start()

	for i := 0; i < 5; i++ {
		svc.Record(model.AuditLog{Action: "create", Resource: "pods"})
	}

	svc.Stop()

	if repo.count() != 5 {
		t.Errorf("expected 5 drained logs after Stop, got %d", repo.count())
	}
}

func TestAuditService_RecordNonBlocking(t *testing.T) {
	repo := newMockAuditRepo()
	logger, _ := zap.NewDevelopment()
	cfg := config.AuditConfig{
		Enabled:       true,
		FlushInterval: 10 * time.Second,
	}
	svc := NewAuditService(repo, cfg, logger)
	// Do NOT start — channel will fill up, Record should not block.

	done := make(chan struct{})
	go func() {
		for i := 0; i < 2000; i++ {
			svc.Record(model.AuditLog{Action: "create", Resource: "pods"})
		}
		close(done)
	}()

	select {
	case <-done:
		// OK — Record did not block.
	case <-time.After(2 * time.Second):
		t.Fatal("Record blocked — should silently drop when channel is full")
	}
}

func TestAuditService_RecordDisabledIsNoop(t *testing.T) {
	repo := newMockAuditRepo()
	svc := NewAuditService(repo, config.AuditConfig{}, zap.NewNop())
	svc.Record(model.AuditLog{Action: "delete"})
	if repo.count() != 0 || len(svc.logCh) != 0 {
		t.Fatal("disabled audit service must not retain entries")
	}
}

func TestAuditService_RecordSyncPersistsImmediately(t *testing.T) {
	repo := newMockAuditRepo()
	svc := NewAuditService(repo, config.AuditConfig{Enabled: true, Sync: true}, zap.NewNop())
	svc.Record(model.AuditLog{Action: "delete"})
	if repo.count() != 1 {
		t.Fatalf("sync audit count = %d, want 1", repo.count())
	}
}
