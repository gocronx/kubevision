package server

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/gocronx/kubevision/internal/config"
	"github.com/gocronx/kubevision/web"
)

// Server wraps the HTTP server and Gin engine.
type Server struct {
	cfg    *config.Config
	engine *gin.Engine
	http   *http.Server
	logger *zap.Logger
}

var ErrPortInUse = errors.New("port already in use")

type PortInUseError struct {
	Port    int
	Message string
}

func (e *PortInUseError) Error() string {
	return e.Message
}

func (e *PortInUseError) Unwrap() error {
	return ErrPortInUse
}

// CheckPortAvailable verifies that the configured HTTP port can be bound before
// expensive startup work begins.
func CheckPortAvailable(port int) error {
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		var opErr *net.OpError
		if errors.As(err, &opErr) && strings.Contains(opErr.Error(), "address already in use") {
			return &PortInUseError{Port: port, Message: portInUseMessage(port)}
		}
		return fmt.Errorf("check port %d: %w", port, err)
	}
	if err := ln.Close(); err != nil {
		return fmt.Errorf("release port %d: %w", port, err)
	}
	return nil
}

// New creates a new Server with the given config, logger, and route dependencies.
func New(cfg *config.Config, logger *zap.Logger, deps *RouterDeps) *Server {
	// Respect GIN_MODE env var; default to release mode for security.
	if os.Getenv("GIN_MODE") != "" {
		gin.SetMode(os.Getenv("GIN_MODE"))
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	engine := gin.New()

	// Use Gin's built-in recovery middleware.
	engine.Use(gin.Recovery())

	// Register all routes with injected dependencies.
	RegisterRoutes(engine, deps)

	// Serve embedded frontend static files (SPA with history mode fallback).
	serveFrontend(engine, logger)

	return &Server{
		cfg:    cfg,
		engine: engine,
		logger: logger,
	}
}

// Engine returns the underlying Gin engine, useful for registering additional
// middleware or routes before starting the server.
func (s *Server) Engine() *gin.Engine {
	return s.engine
}

// Start begins listening and serving HTTP requests.
// It blocks until the server is shut down or an error occurs.
func (s *Server) Start() error {
	addr := fmt.Sprintf(":%d", s.cfg.Server.Port)
	s.http = &http.Server{
		Addr:              addr,
		Handler:           s.engine,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	s.logger.Info("HTTP server starting", zap.String("addr", addr))

	if err := s.http.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		var opErr *net.OpError
		if errors.As(err, &opErr) && strings.Contains(opErr.Error(), "address already in use") {
			return &PortInUseError{Port: s.cfg.Server.Port, Message: portInUseMessage(s.cfg.Server.Port)}
		}
		return fmt.Errorf("http listen: %w", err)
	}
	return nil
}

// portInUseMessage returns a short, actionable message for local startup
// failures without forcing users toward a specific kill command.
func portInUseMessage(port int) string {
	if runtime.GOOS == "windows" {
		return fmt.Sprintf("port %d is already in use; stop the process using it or choose another port", port)
	}
	out, err := exec.Command("lsof", "-i", fmt.Sprintf(":%d", port), "-sTCP:LISTEN", "-P", "-n").Output()
	if err != nil || len(out) == 0 {
		return fmt.Sprintf("port %d is already in use; stop the process using it or start with another port, for example: DEV_PORT=%d make dev", port, port+1)
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) < 2 {
		return fmt.Sprintf("port %d is already in use; stop the process using it or start with another port, for example: DEV_PORT=%d make dev", port, port+1)
	}
	fields := strings.Fields(lines[1])
	if len(fields) >= 2 {
		return fmt.Sprintf("port %d is already in use by %s (PID %s); stop that process or start with another port, for example: DEV_PORT=%d make dev", port, fields[0], fields[1], port+1)
	}
	return fmt.Sprintf("port %d is already in use; stop the process using it or start with another port, for example: DEV_PORT=%d make dev", port, port+1)
}

// serveFrontend mounts the embedded React build as a static file server.
// API routes and health endpoints take priority via Gin's router; any path not
// matching a known route falls back to index.html for SPA client-side routing.
func serveFrontend(engine *gin.Engine, logger *zap.Logger) {
	distFS, err := fs.Sub(web.DistFS, "dist")
	if err != nil {
		logger.Warn("Failed to locate embedded frontend assets", zap.Error(err))
		return
	}

	// Check whether a real frontend build is embedded (not just .gitkeep).
	if _, err := fs.Stat(distFS, "index.html"); err != nil {
		logger.Info("No frontend build embedded, skipping static file serving")
		return
	}

	fileServer := http.FileServer(http.FS(distFS))

	engine.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path

		// Skip API and health paths (should not reach here, but guard anyway).
		if strings.HasPrefix(path, "/api/") || path == "/healthz" || path == "/readyz" {
			c.Next()
			return
		}

		// Try to serve the exact file (JS, CSS, images, etc.).
		if f, err := distFS.Open(strings.TrimPrefix(path, "/")); err == nil {
			f.Close()
			fileServer.ServeHTTP(c.Writer, c.Request)
			return
		}

		// Fallback to index.html for SPA client-side routing.
		c.Request.URL.Path = "/"
		fileServer.ServeHTTP(c.Writer, c.Request)
	})
}

// Shutdown gracefully shuts down the HTTP server with a 5-second timeout.
func (s *Server) Shutdown() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	s.logger.Info("HTTP server shutting down")

	if err := s.http.Shutdown(ctx); err != nil {
		return fmt.Errorf("http shutdown: %w", err)
	}

	s.logger.Info("HTTP server stopped")
	return nil
}
