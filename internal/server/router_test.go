package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestRegisterRoutes_NilDeps(t *testing.T) {
	r := gin.New()
	// Should not panic with nil deps.
	RegisterRoutes(r, nil)

	// Health check should still work.
	req := httptest.NewRequest("GET", "/healthz", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]string
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "ok", resp["status"])
}

func TestRegisterRoutes_EmptyDeps(t *testing.T) {
	r := gin.New()
	RegisterRoutes(r, &RouterDeps{})

	req := httptest.NewRequest("GET", "/healthz", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRegisterRoutes_HealthCheck(t *testing.T) {
	r := gin.New()
	RegisterRoutes(r, &RouterDeps{})

	req := httptest.NewRequest("GET", "/healthz", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]string
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "ok", resp["status"])
}

func TestRegisterRoutes_UnknownRoute_Returns404(t *testing.T) {
	r := gin.New()
	RegisterRoutes(r, &RouterDeps{})

	req := httptest.NewRequest("GET", "/api/v1/nonexistent", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}
