package middleware

import (
	"bytes"
	"context"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/gocronx/kubevision/internal/config"
	"github.com/gocronx/kubevision/internal/model"
	"github.com/gocronx/kubevision/internal/repository"
	"github.com/gocronx/kubevision/internal/service"
	"go.uber.org/zap"
)

// mockAuditRepo implements repository.AuditRepo for tests.
type mockAuditRepo struct {
	logs []model.AuditLog
}

func (m *mockAuditRepo) BatchCreate(_ context.Context, logs []model.AuditLog) error {
	m.logs = append(m.logs, logs...)
	return nil
}

func (m *mockAuditRepo) List(_ context.Context, _ repository.AuditFilter) ([]model.AuditLog, int64, error) {
	return m.logs, int64(len(m.logs)), nil
}

func (m *mockAuditRepo) Purge(_ context.Context, _ time.Time) (int64, error) {
	return 0, nil
}

func newTestAuditService(repo *mockAuditRepo) *service.AuditService {
	logger, _ := zap.NewDevelopment()
	cfg := config.AuditConfig{
		Enabled:       true,
		FlushInterval: 50 * time.Millisecond,
		RetentionDays: 0,
	}
	svc := service.NewAuditService(repo, cfg, logger)
	svc.Start()
	return svc
}

func TestAuditMiddleware_SkipsGET(t *testing.T) {
	repo := &mockAuditRepo{}
	svc := newTestAuditService(repo)
	defer svc.Stop()

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()

	router := gin.New()
	router.Use(AuditMiddleware(svc))
	router.GET("/api/v1/clusters", func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": true})
	})

	req := httptest.NewRequest("GET", "/api/v1/clusters", nil)
	router.ServeHTTP(w, req)

	// Give time for async flush.
	time.Sleep(100 * time.Millisecond)

	if len(repo.logs) != 0 {
		t.Errorf("GET should not be audited, got %d entries", len(repo.logs))
	}
}

func TestAuditMiddleware_RecordsPOST(t *testing.T) {
	repo := &mockAuditRepo{}
	svc := newTestAuditService(repo)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("userID", uint(42))
		c.Set("username", "testuser")
		c.Next()
	})
	router.Use(AuditMiddleware(svc))
	router.POST("/api/v1/clusters", func(c *gin.Context) {
		c.JSON(201, gin.H{"ok": true})
	})

	body := `{"name":"test-cluster"}`
	req := httptest.NewRequest("POST", "/api/v1/clusters", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	// Stop to flush remaining entries.
	svc.Stop()

	if len(repo.logs) == 0 {
		t.Fatal("expected at least one audit log entry for POST")
	}

	entry := repo.logs[0]
	if entry.Action != "create" {
		t.Errorf("expected action 'create', got %q", entry.Action)
	}
	if entry.UserID != 42 {
		t.Errorf("expected userID 42, got %d", entry.UserID)
	}
	if entry.Username != "testuser" {
		t.Errorf("expected username 'testuser', got %q", entry.Username)
	}
	if entry.RequestBody != body {
		t.Errorf("expected request body %q, got %q", body, entry.RequestBody)
	}
}

func TestAuditMiddleware_RecordsDELETE(t *testing.T) {
	repo := &mockAuditRepo{}
	svc := newTestAuditService(repo)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()

	router := gin.New()
	router.Use(AuditMiddleware(svc))
	router.DELETE("/api/v1/clusters/:id", func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": true})
	})

	req := httptest.NewRequest("DELETE", "/api/v1/clusters/1", nil)
	router.ServeHTTP(w, req)

	svc.Stop()

	if len(repo.logs) == 0 {
		t.Fatal("expected audit log entry for DELETE")
	}
	if repo.logs[0].Action != "delete" {
		t.Errorf("expected action 'delete', got %q", repo.logs[0].Action)
	}
}

func TestAuditMiddleware_BodyCapped(t *testing.T) {
	repo := &mockAuditRepo{}
	svc := newTestAuditService(repo)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()

	router := gin.New()
	router.Use(AuditMiddleware(svc))
	router.PUT("/api/v1/clusters/:id", func(c *gin.Context) {
		// Read body in handler to verify it was restored.
		buf := new(bytes.Buffer)
		buf.ReadFrom(c.Request.Body)
		c.JSON(200, gin.H{"bodyLen": buf.Len()})
	})

	// Create body larger than maxRequestBodySize (4KB).
	largeBody := strings.Repeat("x", 5000)
	req := httptest.NewRequest("PUT", "/api/v1/clusters/1", strings.NewReader(largeBody))
	router.ServeHTTP(w, req)

	svc.Stop()

	if len(repo.logs) == 0 {
		t.Fatal("expected audit log entry")
	}
	if len(repo.logs[0].RequestBody) > maxRequestBodySize {
		t.Errorf("request body should be capped at %d bytes, got %d", maxRequestBodySize, len(repo.logs[0].RequestBody))
	}
}

func TestAudit_NoopMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	handlerCalled := false

	router := gin.New()
	router.Use(Audit())
	router.GET("/test", func(c *gin.Context) {
		handlerCalled = true
		c.Status(200)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	if !handlerCalled {
		t.Error("Audit noop middleware should pass through")
	}
}
