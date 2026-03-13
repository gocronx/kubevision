package ws

// ws_test_helpers_test.go — shared test utilities for the ws package tests.
// These helpers are only compiled during `go test` (the _test.go suffix).

import (
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gocronx/kubevision/internal/auth"
	"go.uber.org/zap"
)

// newDiscardLogger returns a zap logger that discards all output, suitable for
// unit tests where log output is not relevant.
func newDiscardLogger() *zap.Logger {
	return zap.NewNop()
}

// newTestJWTManager returns a JWTManager backed by a fixed test secret, used
// across ws package tests that need to generate or validate tokens.
func newTestJWTManager() *auth.JWTManager {
	return auth.NewJWTManager(
		"ws-test-secret-key-for-unit-tests",
		5*time.Minute,
		24*time.Hour,
	)
}

// newTestGinContext creates a minimal *gin.Context wrapping the given
// ResponseWriter and Request. It sets Gin to test mode so that it does not
// print route warnings during tests.
func newTestGinContext(w http.ResponseWriter, r *http.Request) *gin.Context {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(w.(*httptest.ResponseRecorder))
	c.Request = r
	return c
}
