package service

import (
	"context"
	"sync/atomic"
	"time"

	"go.uber.org/zap"

	"github.com/gocronx/kubevision/internal/config"
	"github.com/gocronx/kubevision/internal/model"
	"github.com/gocronx/kubevision/internal/repository"
)

const (
	auditChannelSize = 1000
	auditBatchSize   = 100
	auditFlushEvery  = 5 * time.Second
)

// AuditService buffers audit log entries and flushes them to persistent
// storage in batches to avoid per-request database round trips.
type AuditService struct {
	repo    repository.AuditRepo
	cfg     config.AuditConfig
	logger  *zap.Logger
	logCh   chan model.AuditLog
	stopCh  chan struct{}
	doneCh  chan struct{}
	started atomic.Bool
}

// NewAuditService creates an AuditService. Call Start() to begin background
// processing.
func NewAuditService(repo repository.AuditRepo, cfg config.AuditConfig, logger *zap.Logger) *AuditService {
	return &AuditService{
		repo:   repo,
		cfg:    cfg,
		logger: logger,
		logCh:  make(chan model.AuditLog, auditChannelSize),
		stopCh: make(chan struct{}),
		doneCh: make(chan struct{}),
	}
}

// Record enqueues an audit log entry for asynchronous persistence. If the
// internal buffer is full the entry is silently dropped rather than blocking
// the request path.
func (s *AuditService) Record(log model.AuditLog) {
	select {
	case s.logCh <- log:
	default:
		s.logger.Warn("audit log channel full, dropping entry",
			zap.String("action", log.Action),
			zap.String("resource", log.Resource),
		)
	}
}

// Start launches the background flush goroutine and the daily purge goroutine.
// It is safe to call Start only once.
func (s *AuditService) Start() {
	s.started.Store(true)
	go s.flushLoop()
	go s.purgeLoop()
}

// Stop signals the background goroutine to finish processing any pending
// entries and waits for it to exit cleanly. If Start was never called, Stop
// returns immediately without blocking.
func (s *AuditService) Stop() {
	close(s.stopCh)
	if s.started.Load() {
		<-s.doneCh
	}
}

// flushLoop reads entries from the channel and writes them to the database
// in batches, flushing either when the batch reaches auditBatchSize entries
// or when the flush interval elapses, whichever comes first.
func (s *AuditService) flushLoop() {
	defer close(s.doneCh)

	flushInterval := s.cfg.FlushInterval
	if flushInterval <= 0 {
		flushInterval = auditFlushEvery
	}
	ticker := time.NewTicker(flushInterval)
	defer ticker.Stop()

	batch := make([]model.AuditLog, 0, auditBatchSize)

	flush := func() {
		if len(batch) == 0 {
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := s.repo.BatchCreate(ctx, batch); err != nil {
			s.logger.Error("failed to flush audit logs", zap.Error(err), zap.Int("count", len(batch)))
		}
		batch = batch[:0]
	}

	for {
		select {
		case entry, ok := <-s.logCh:
			if !ok {
				flush()
				return
			}
			batch = append(batch, entry)
			if len(batch) >= auditBatchSize {
				flush()
			}

		case <-ticker.C:
			flush()

		case <-s.stopCh:
			// Drain remaining entries before stopping.
		drain:
			for {
				select {
				case entry := <-s.logCh:
					batch = append(batch, entry)
				default:
					break drain
				}
			}
			flush()
			return
		}
	}
}

// purgeLoop runs a daily job that deletes audit log entries older than the
// configured retention period.
func (s *AuditService) purgeLoop() {
	if s.cfg.RetentionDays <= 0 {
		return
	}

	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			cutoff := time.Now().AddDate(0, 0, -s.cfg.RetentionDays)
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			n, err := s.repo.Purge(ctx, cutoff)
			cancel()
			if err != nil {
				s.logger.Error("failed to purge audit logs", zap.Error(err))
			} else if n > 0 {
				s.logger.Info("purged old audit logs", zap.Int64("deleted", n))
			}
		case <-s.stopCh:
			return
		}
	}
}
