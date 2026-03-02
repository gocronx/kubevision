package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/kubevision/kubevision/internal/config"
)

// Server wraps the HTTP server and Gin engine.
type Server struct {
	cfg    *config.Config
	engine *gin.Engine
	http   *http.Server
	logger *zap.Logger
}

// New creates a new Server with the given config, logger, and route dependencies.
func New(cfg *config.Config, logger *zap.Logger, deps *RouterDeps) *Server {
	gin.SetMode(gin.ReleaseMode)

	engine := gin.New()

	// Use Gin's built-in recovery middleware.
	engine.Use(gin.Recovery())

	// Register all routes with injected dependencies.
	RegisterRoutes(engine, deps)

	// TODO: Serve embedded frontend static files.
	// engine.StaticFS("/", http.FS(webFS))

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
		return fmt.Errorf("http listen: %w", err)
	}
	return nil
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
