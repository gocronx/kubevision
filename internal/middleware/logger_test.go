package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func TestLogger_LogsRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)

	core, obs := observer.New(zap.InfoLevel)
	logger := zap.New(core)

	w := httptest.NewRecorder()
	router := gin.New()
	router.Use(Logger(logger))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": true})
	})

	req := httptest.NewRequest("GET", "/test?q=hello", nil)
	router.ServeHTTP(w, req)

	if obs.Len() == 0 {
		t.Fatal("expected at least one log entry")
	}

	entry := obs.All()[0]
	if entry.Message != "request completed" {
		t.Errorf("expected message 'request completed', got %q", entry.Message)
	}

	fields := make(map[string]interface{})
	for _, f := range entry.Context {
		fields[f.Key] = f.Interface
	}

	if _, ok := fields["status"]; !ok {
		t.Error("expected 'status' field in log")
	}
	if _, ok := fields["method"]; !ok {
		t.Error("expected 'method' field in log")
	}
	if _, ok := fields["path"]; !ok {
		t.Error("expected 'path' field in log")
	}
	if _, ok := fields["latency"]; !ok {
		t.Error("expected 'latency' field in log")
	}
}

func TestLogger_LogsRequestWithRequestID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	core, obs := observer.New(zap.InfoLevel)
	logger := zap.New(core)

	w := httptest.NewRecorder()
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("requestID", "test-req-123")
		c.Next()
	})
	router.Use(Logger(logger))
	router.GET("/test", func(c *gin.Context) {
		c.Status(200)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	if obs.Len() == 0 {
		t.Fatal("expected at least one log entry")
	}

	fields := make(map[string]interface{})
	for _, f := range obs.All()[0].Context {
		fields[f.Key] = f.Interface
	}

	if _, ok := fields["requestID"]; !ok {
		t.Error("expected 'requestID' field when set in context")
	}
}

func TestLogger_LogsErrorRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)

	core, obs := observer.New(zap.InfoLevel)
	logger := zap.New(core)

	w := httptest.NewRecorder()
	router := gin.New()
	router.Use(Logger(logger))
	router.GET("/error", func(c *gin.Context) {
		_ = c.Error(http.ErrAbortHandler)
		c.Status(500)
	})

	req := httptest.NewRequest("GET", "/error", nil)
	router.ServeHTTP(w, req)

	if obs.Len() == 0 {
		t.Fatal("expected at least one log entry")
	}

	entry := obs.All()[0]
	if entry.Message != "request completed with errors" {
		t.Errorf("expected error message, got %q", entry.Message)
	}
}
