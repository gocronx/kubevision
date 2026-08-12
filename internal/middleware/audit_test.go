package middleware

import (
	"bytes"
	"context"
	"encoding/json"
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

func TestRedactSensitiveFieldsRecursivelyAndCaseInsensitively(t *testing.T) {
	body := `{"bindPassword":"bind-secret","apiKey":"sk-secret","nested":{"refreshToken":"refresh-secret","TOTPCode":"123456"},"items":[{"privateKey":"key"}],"serverUrl":"ldaps://directory.example.test"}`
	redacted := redactSensitiveFields(body)

	var got map[string]any
	if err := json.Unmarshal([]byte(redacted), &got); err != nil {
		t.Fatalf("redacted body is invalid JSON: %v", err)
	}
	if got["bindPassword"] != "[REDACTED]" {
		t.Fatalf("bind password was not redacted: %s", redacted)
	}
	if got["apiKey"] != "[REDACTED]" {
		t.Fatalf("API key was not redacted: %s", redacted)
	}
	nested := got["nested"].(map[string]any)
	if nested["refreshToken"] != "[REDACTED]" || nested["TOTPCode"] != "[REDACTED]" {
		t.Fatalf("nested authentication material was not redacted: %s", redacted)
	}
	item := got["items"].([]any)[0].(map[string]any)
	if item["privateKey"] != "[REDACTED]" {
		t.Fatalf("authentication material in an array was not redacted: %s", redacted)
	}
	if got["serverUrl"] != "ldaps://directory.example.test" {
		t.Fatalf("non-sensitive audit context was removed: %s", redacted)
	}
}

func TestRedactSensitiveFieldsDropsUnparseableInput(t *testing.T) {
	if got := redactSensitiveFields(`{"bindPassword":"truncated`); got != "[UNAVAILABLE]" {
		t.Fatalf("unparseable input must not be retained, got %q", got)
	}
}

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

func TestAuditMiddleware_OmitsAIAndPublicKeyBodies(t *testing.T) {
	for _, path := range []string{"/api/v1/ai/chat", "/api/v1/ai/continue-action", "/api/v1/auth/public-key/register/finish"} {
		t.Run(path, func(t *testing.T) {
			repo := &mockAuditRepo{}
			svc := newTestAuditService(repo)
			router := gin.New()
			router.Use(AuditMiddleware(svc))
			router.POST(path, func(c *gin.Context) { c.Status(200) })

			req := httptest.NewRequest("POST", path, strings.NewReader(`{"messages":[{"content":"secret"}],"sessionId":"token","credential":"key"}`))
			router.ServeHTTP(httptest.NewRecorder(), req)
			svc.Stop()

			if len(repo.logs) != 1 || repo.logs[0].RequestBody != "" {
				t.Fatalf("sensitive body retained: %+v", repo.logs)
			}
		})
	}
}

func TestAuditMiddleware_OmitsSecretResourceBodies(t *testing.T) {
	repo := &mockAuditRepo{}
	svc := newTestAuditService(repo)
	router := gin.New()
	router.Use(AuditMiddleware(svc))
	router.PUT("/api/v1/clusters/:id/resources/:resource/:name/dry-run", func(c *gin.Context) { c.Status(200) })

	body := `{"stringData":{"REDIS_PASSWORD":"root1"}}`
	req := httptest.NewRequest("PUT", "/api/v1/clusters/1/resources/secrets/redis-secret/dry-run", strings.NewReader(body))
	router.ServeHTTP(httptest.NewRecorder(), req)
	svc.Stop()

	if len(repo.logs) != 1 {
		t.Fatalf("logs = %d, want 1", len(repo.logs))
	}
	if repo.logs[0].RequestBody != "" {
		t.Fatalf("Secret body retained in audit log: %q", repo.logs[0].RequestBody)
	}
	if repo.logs[0].Resource == "" || repo.logs[0].Name != "redis-secret" {
		t.Fatalf("safe audit metadata missing: %+v", repo.logs[0])
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
