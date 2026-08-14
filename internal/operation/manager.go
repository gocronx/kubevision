package operation

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/gocronx/kubevision/internal/auth"
	"github.com/gocronx/kubevision/internal/model"
	bizerr "github.com/gocronx/kubevision/internal/pkg/errors"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

const queueSize = 256

const (
	heartbeatInterval = 5 * time.Second
	staleWorkerAfter  = 30 * time.Second
	historyRetention  = 30 * 24 * time.Hour
	cleanupInterval   = 6 * time.Hour
)

type Manager struct {
	db         *gorm.DB
	encryptKey string
	logger     *zap.Logger
	queue      chan string
	mu         sync.RWMutex
	executors  map[string]Executor
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
	started    bool
	workerID   string
}

func NewManager(db *gorm.DB, encryptKey string, logger *zap.Logger) *Manager {
	return &Manager{db: db, encryptKey: encryptKey, logger: logger, queue: make(chan string, queueSize), executors: make(map[string]Executor), workerID: uuid.NewString()}
}

func (m *Manager) Register(kind string, executor Executor) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.started {
		panic("operation executor registered after manager start")
	}
	m.executors[kind] = executor
}

func (m *Manager) Start(parent context.Context) error {
	m.mu.Lock()
	if m.started {
		m.mu.Unlock()
		return nil
	}
	m.ctx, m.cancel = context.WithCancel(parent)
	m.started = true
	m.mu.Unlock()

	if err := m.recoverStale(); err != nil {
		m.mu.Lock()
		m.cancel()
		m.ctx, m.cancel, m.started = nil, nil, false
		m.mu.Unlock()
		return err
	}

	m.wg.Add(4)
	go m.worker()
	go m.worker()
	go m.dispatchQueued()
	go m.cleanupCompleted()
	return nil
}

func (m *Manager) cleanupCompleted() {
	defer m.wg.Done()
	ticker := time.NewTicker(cleanupInterval)
	defer ticker.Stop()
	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			cutoff := time.Now().UTC().Add(-historyRetention)
			oldOperations := m.db.Model(&model.Operation{}).Select("id").Where("completed_at IS NOT NULL AND completed_at < ?", cutoff)
			if err := m.db.WithContext(m.ctx).Transaction(func(tx *gorm.DB) error {
				if err := tx.Where("operation_id IN (?)", oldOperations).Delete(&model.OperationEvent{}).Error; err != nil {
					return err
				}
				return tx.Where("completed_at IS NOT NULL AND completed_at < ?", cutoff).Delete(&model.Operation{}).Error
			}); err != nil && m.ctx.Err() == nil {
				m.logger.Warn("failed to clean completed operations", zap.Error(err))
			}
		}
	}
}

func (m *Manager) recoverStale() error {
	now := time.Now().UTC()
	if err := m.db.Model(&model.Operation{}).
		Where("status = ? AND (heartbeat_at IS NULL OR heartbeat_at < ?)", StatusRunning, now.Add(-staleWorkerAfter)).
		Updates(map[string]interface{}{
			"status": StatusFailed, "stage": "interrupted", "progress": 100,
			"error_code": "OPERATION_INTERRUPTED", "error_message": "The server stopped while this operation was running",
			"retryable": false, "completed_at": now, "worker_id": "", "heartbeat_at": nil,
		}).Error; err != nil {
		return fmt.Errorf("recover interrupted operations: %w", err)
	}
	return nil
}

func (m *Manager) Stop() {
	m.mu.Lock()
	cancel, started := m.cancel, m.started
	if !started {
		m.mu.Unlock()
		return
	}
	m.started = false
	m.mu.Unlock()
	cancel()
	m.wg.Wait()
}

func (m *Manager) Submit(ctx context.Context, input Input) (*View, error) {
	payload, err := json.Marshal(input.Payload)
	if err != nil {
		return nil, bizerr.New(bizerr.CodeParamInvalid, "operation payload is invalid")
	}
	encrypted, err := auth.Encrypt(string(payload), m.encryptKey)
	if err != nil {
		return nil, bizerr.New(bizerr.CodeInternal, "failed to protect operation payload")
	}
	op := model.Operation{
		ID: uuid.NewString(), UserID: input.UserID, Username: input.Username,
		Kind: input.Kind, Action: input.Action, Status: StatusQueued, Stage: "queued",
		Cluster: input.Cluster, Namespace: input.Namespace, ResourceName: input.ResourceName,
		RequestID: input.RequestID, PayloadEnc: encrypted,
	}
	if err := m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&op).Error; err != nil {
			return err
		}
		return tx.Create(&model.OperationEvent{OperationID: op.ID, Stage: op.Stage, Status: op.Status, Message: "Operation queued", Progress: 0}).Error
	}); err != nil {
		return nil, bizerr.New(bizerr.CodeInternal, "failed to persist operation")
	}
	m.enqueue(op.ID)
	return m.Get(ctx, op.ID, input.UserID, false)
}

func (m *Manager) List(ctx context.Context, userID uint, includeAll bool, limit int) ([]View, error) {
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	query := m.db.WithContext(ctx).Preload("Events", func(db *gorm.DB) *gorm.DB { return db.Order("id ASC") }).Order("created_at DESC").Limit(limit)
	if !includeAll {
		query = query.Where("user_id = ?", userID)
	}
	var items []model.Operation
	if err := query.Find(&items).Error; err != nil {
		return nil, err
	}
	views := make([]View, 0, len(items))
	for i := range items {
		views = append(views, toView(&items[i]))
	}
	return views, nil
}

func (m *Manager) Get(ctx context.Context, id string, userID uint, includeAll bool) (*View, error) {
	query := m.db.WithContext(ctx).Preload("Events", func(db *gorm.DB) *gorm.DB { return db.Order("id ASC") }).Where("id = ?", id)
	if !includeAll {
		query = query.Where("user_id = ?", userID)
	}
	var op model.Operation
	if err := query.First(&op).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, bizerr.ErrNotFound
		}
		return nil, err
	}
	view := toView(&op)
	return &view, nil
}

func (m *Manager) Retry(ctx context.Context, id string, principal Principal, includeAll bool) (*View, error) {
	query := m.db.WithContext(ctx).Where("id = ?", id)
	if !includeAll {
		query = query.Where("user_id = ?", principal.UserID)
	}
	var previous model.Operation
	if err := query.First(&previous).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, bizerr.ErrNotFound
		}
		return nil, err
	}
	if previous.Status != StatusFailed || !previous.Retryable {
		return nil, bizerr.New(bizerr.CodeConflict, "operation cannot be retried")
	}
	retry := previous
	retry.ID = uuid.NewString()
	retry.ParentID = previous.ID
	retry.CreatedAt, retry.UpdatedAt = time.Time{}, time.Time{}
	retry.StartedAt, retry.CompletedAt = nil, nil
	retry.Status, retry.Stage, retry.Progress = StatusQueued, "queued", 0
	retry.ErrorCode, retry.ErrorMessage, retry.SuggestionsJSON, retry.ResultJSON = "", "", "", ""
	retry.Retryable, retry.RollbackAvailable = false, false
	retry.RequestID = ""
	retry.WorkerID, retry.HeartbeatAt = "", nil
	retry.Events = nil
	if err := m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&retry).Error; err != nil {
			return err
		}
		return tx.Create(&model.OperationEvent{OperationID: retry.ID, Stage: retry.Stage, Status: retry.Status, Message: "Retry queued", Progress: 0}).Error
	}); err != nil {
		return nil, err
	}
	m.enqueue(retry.ID)
	return m.Get(ctx, retry.ID, principal.UserID, includeAll)
}

func (m *Manager) worker() {
	defer m.wg.Done()
	for {
		select {
		case <-m.ctx.Done():
			return
		case id := <-m.queue:
			m.execute(id)
		}
	}
}

func (m *Manager) dispatchQueued() {
	defer m.wg.Done()
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		if err := m.recoverStale(); err != nil && m.ctx.Err() == nil {
			m.logger.Warn("failed to recover stale operations", zap.Error(err))
		}
		m.loadQueued()
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (m *Manager) loadQueued() {
	var ids []string
	if err := m.db.WithContext(m.ctx).Model(&model.Operation{}).Where("status = ?", StatusQueued).Order("created_at ASC").Limit(queueSize).Pluck("id", &ids).Error; err != nil {
		if m.ctx.Err() == nil {
			m.logger.Warn("failed to load queued operations", zap.Error(err))
		}
		return
	}
	for _, id := range ids {
		m.enqueue(id)
	}
}

func (m *Manager) enqueue(id string) {
	m.mu.RLock()
	ctx, started := m.ctx, m.started
	m.mu.RUnlock()
	if !started {
		return
	}
	select {
	case m.queue <- id:
	case <-ctx.Done():
	default:
	}
}

func (m *Manager) execute(id string) {
	now := time.Now().UTC()
	claimed := m.db.WithContext(m.ctx).Model(&model.Operation{}).
		Where("id = ? AND status = ?", id, StatusQueued).
		Updates(map[string]interface{}{"status": StatusRunning, "stage": "preparing", "progress": 5, "started_at": now, "worker_id": m.workerID, "heartbeat_at": now})
	if claimed.Error != nil || claimed.RowsAffected == 0 {
		return
	}
	var op model.Operation
	if err := m.loadClaimedOperation(id, &op); err != nil {
		op.ID = id
		m.fail(&op, &Failure{Stage: "preparing", Code: "OPERATION_LOAD_FAILED", Message: "The queued operation could not be loaded", Retryable: true})
		return
	}
	m.event(id, "preparing", StatusRunning, "Preparing operation", 5)

	var user model.User
	if err := m.db.WithContext(m.ctx).First(&user, op.UserID).Error; err != nil || !user.IsActive {
		m.fail(&op, &Failure{Stage: "authorization", Code: "USER_NOT_ACTIVE", Message: "The operation owner is no longer active"})
		return
	}
	m.mu.RLock()
	executor := m.executors[op.Kind]
	m.mu.RUnlock()
	if executor == nil {
		m.fail(&op, &Failure{Stage: "preparing", Code: "EXECUTOR_UNAVAILABLE", Message: "No executor is registered for this operation", Retryable: true})
		return
	}
	plain, err := auth.Decrypt(op.PayloadEnc, m.encryptKey)
	if err != nil {
		m.fail(&op, &Failure{Stage: "preparing", Code: "PAYLOAD_DECRYPT_FAILED", Message: "The protected operation input could not be read"})
		return
	}
	reporter := func(stage, message string, progress int) {
		if progress < 5 {
			progress = 5
		}
		if progress > 95 {
			progress = 95
		}
		_ = m.db.WithContext(m.ctx).Model(&model.Operation{}).Where("id = ?", id).Updates(map[string]interface{}{"stage": stage, "progress": progress}).Error
		m.event(id, stage, StatusRunning, message, progress)
	}
	heartbeatCtx, stopHeartbeat := context.WithCancel(m.ctx)
	heartbeatDone := make(chan struct{})
	go m.heartbeat(heartbeatCtx, id, heartbeatDone)
	result, failure := executor(m.ctx, Principal{UserID: user.ID, Username: user.Username, Role: user.Role}, json.RawMessage(plain), reporter)
	stopHeartbeat()
	<-heartbeatDone
	if m.ctx.Err() != nil {
		return
	}
	if failure != nil {
		m.fail(&op, failure)
		return
	}
	resultJSON := ""
	if result != nil {
		if encoded, marshalErr := json.Marshal(result); marshalErr == nil {
			resultJSON = string(encoded)
		}
	}
	completed := time.Now().UTC()
	_ = m.db.WithContext(m.ctx).Model(&model.Operation{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status": StatusSucceeded, "stage": "completed", "progress": 100, "completed_at": completed, "result_json": resultJSON,
		"worker_id": "", "heartbeat_at": nil,
	}).Error
	m.event(id, "completed", StatusSucceeded, "Operation completed", 100)
}

func (m *Manager) loadClaimedOperation(id string, op *model.Operation) error {
	var err error
	for attempt := 0; attempt < 5; attempt++ {
		err = m.db.WithContext(m.ctx).First(op, "id = ?", id).Error
		if err == nil {
			return nil
		}
		timer := time.NewTimer(time.Duration(attempt+1) * 20 * time.Millisecond)
		select {
		case <-m.ctx.Done():
			timer.Stop()
			return m.ctx.Err()
		case <-timer.C:
		}
	}
	return err
}

func (m *Manager) heartbeat(ctx context.Context, id string, done chan<- struct{}) {
	defer close(done)
	ticker := time.NewTicker(heartbeatInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			_ = m.db.WithContext(ctx).Model(&model.Operation{}).
				Where("id = ? AND status = ? AND worker_id = ?", id, StatusRunning, m.workerID).
				Update("heartbeat_at", now.UTC()).Error
		}
	}
}

func (m *Manager) fail(op *model.Operation, failure *Failure) {
	if failure == nil {
		failure = &Failure{Stage: "executing", Code: "OPERATION_FAILED", Message: "Operation failed"}
	}
	suggestions, _ := json.Marshal(failure.Suggestions)
	completed := time.Now().UTC()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = m.db.WithContext(ctx).Model(&model.Operation{}).Where("id = ?", op.ID).Updates(map[string]interface{}{
		"status": StatusFailed, "stage": failure.Stage, "progress": 100,
		"error_code": failure.Code, "error_message": failure.Message, "suggestions_json": string(suggestions),
		"retryable": failure.Retryable, "rollback_available": failure.RollbackAvailable, "completed_at": completed,
		"worker_id": "", "heartbeat_at": nil,
	}).Error
	m.eventWithContext(ctx, op.ID, failure.Stage, StatusFailed, failure.Message, 100)
}

func (m *Manager) event(id, stage, status, message string, progress int) {
	m.eventWithContext(m.ctx, id, stage, status, message, progress)
}

func (m *Manager) eventWithContext(ctx context.Context, id, stage, status, message string, progress int) {
	if len(message) > 512 {
		message = message[:512]
	}
	_ = m.db.WithContext(ctx).Create(&model.OperationEvent{OperationID: id, Stage: stage, Status: status, Message: message, Progress: progress}).Error
}

func toView(op *model.Operation) View {
	view := View{
		ID: op.ID, CreatedAt: op.CreatedAt, UpdatedAt: op.UpdatedAt, StartedAt: op.StartedAt, CompletedAt: op.CompletedAt,
		ParentID: op.ParentID, UserID: op.UserID, Username: op.Username, Kind: op.Kind, Action: op.Action,
		Status: op.Status, Stage: op.Stage, Cluster: op.Cluster, Namespace: op.Namespace, ResourceName: op.ResourceName,
		Progress: op.Progress, ErrorCode: op.ErrorCode, ErrorMessage: op.ErrorMessage, RequestID: op.RequestID,
		Retryable: op.Retryable, RollbackAvailable: op.RollbackAvailable, Events: op.Events,
	}
	_ = json.Unmarshal([]byte(op.SuggestionsJSON), &view.Suggestions)
	_ = json.Unmarshal([]byte(op.ResultJSON), &view.Result)
	return view
}
