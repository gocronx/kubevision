package operation

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/gocronx/kubevision/internal/model"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func testManager(t *testing.T) (*Manager, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+strings.ReplaceAll(t.Name(), "/", "-")+"-"+uuid.NewString()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	// Production SQLite uses one connection to avoid table-level write locks.
	sqlDB.SetMaxOpenConns(1)
	if err := db.AutoMigrate(&model.User{}, &model.Operation{}, &model.OperationEvent{}); err != nil {
		t.Fatal(err)
	}
	user := model.User{Username: "operator", Role: "admin", IsActive: true}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	return NewManager(db, "test-encryption-key", zap.NewNop()), db
}

func TestManagerPersistsEncryptedTaskAndCompletes(t *testing.T) {
	manager, db := testManager(t)
	manager.Register("test", func(_ context.Context, principal Principal, payload json.RawMessage, report Reporter) (any, *Failure) {
		if principal.Username != "operator" || !strings.Contains(string(payload), "private-value") {
			t.Fatalf("executor received unexpected principal or payload")
		}
		report("applying", "Applying change", 50)
		return map[string]string{"state": "ready"}, nil
	})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := manager.Start(ctx); err != nil {
		t.Fatal(err)
	}
	defer manager.Stop()

	view, err := manager.Submit(context.Background(), Input{UserID: 1, Username: "operator", Kind: "test", Action: "apply", Payload: map[string]string{"secret": "private-value"}})
	if err != nil {
		t.Fatal(err)
	}
	var stored model.Operation
	if err := db.First(&stored, "id = ?", view.ID).Error; err != nil {
		t.Fatal(err)
	}
	if strings.Contains(stored.PayloadEnc, "private-value") {
		t.Fatal("operation payload was stored in plaintext")
	}

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		current, getErr := manager.Get(context.Background(), view.ID, 1, false)
		if getErr == nil && current.Status == StatusSucceeded {
			if current.Result["state"] != "ready" || len(current.Events) < 3 {
				t.Fatalf("unexpected completed operation: %#v", current)
			}
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("operation did not complete")
}

func TestManagerOnlyRecoversStaleWorkers(t *testing.T) {
	manager, db := testManager(t)
	now := time.Now().UTC()
	operations := []model.Operation{
		{ID: "stale", UserID: 1, Username: "operator", Kind: "test", Action: "apply", Status: StatusRunning, Stage: "applying", PayloadEnc: "x", HeartbeatAt: pointerTime(now.Add(-staleWorkerAfter - time.Second))},
		{ID: "active", UserID: 1, Username: "operator", Kind: "test", Action: "apply", Status: StatusRunning, Stage: "applying", PayloadEnc: "x", HeartbeatAt: &now},
	}
	if err := db.Create(&operations).Error; err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := manager.Start(ctx); err != nil {
		t.Fatal(err)
	}
	manager.Stop()
	var stale, active model.Operation
	_ = db.First(&stale, "id = ?", "stale").Error
	_ = db.First(&active, "id = ?", "active").Error
	if stale.Status != StatusFailed || stale.ErrorCode != "OPERATION_INTERRUPTED" || stale.Retryable {
		t.Fatalf("stale operation was not recovered: %#v", stale)
	}
	if active.Status != StatusRunning {
		t.Fatalf("active operation was incorrectly interrupted: %#v", active)
	}
}

func pointerTime(value time.Time) *time.Time { return &value }
