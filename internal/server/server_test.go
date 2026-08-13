package server

import (
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gocronx/kubevision/internal/config"
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

func TestPortInUseMessage_IsFriendlyAndActionable(t *testing.T) {
	msg := portInUseMessage(18082)

	assert.Contains(t, msg, "port 18082 is already in use")
	assert.Contains(t, msg, "DEV_PORT=18083 make dev")
	assert.NotContains(t, strings.ToLower(msg), "kill ")
}

func TestCheckPortAvailable_ReturnsPortInUse(t *testing.T) {
	ln, err := net.Listen("tcp", ":0")
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, ln.Close())
	})

	port := ln.Addr().(*net.TCPAddr).Port
	err = CheckPortAvailable(port)

	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrPortInUse), "error = %v", err)
	assert.NotContains(t, err.Error(), ErrPortInUse.Error()+":")
}
