package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kubevision/kubevision/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func TestNew_ReturnsValidServer(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{Port: 0},
	}
	logger := zaptest.NewLogger(t)

	srv := New(cfg, logger, &RouterDeps{})
	require.NotNil(t, srv)
	require.NotNil(t, srv.Engine())
}

func TestNew_HealthCheck(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{Port: 0},
	}
	logger := zaptest.NewLogger(t)

	srv := New(cfg, logger, &RouterDeps{})

	req := httptest.NewRequest("GET", "/healthz", nil)
	w := httptest.NewRecorder()
	srv.Engine().ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]string
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "ok", resp["status"])
}

func TestNew_NilDeps(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{Port: 0},
	}
	logger := zaptest.NewLogger(t)

	srv := New(cfg, logger, nil)
	require.NotNil(t, srv)

	req := httptest.NewRequest("GET", "/healthz", nil)
	w := httptest.NewRecorder()
	srv.Engine().ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}
