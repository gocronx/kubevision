package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"

	"github.com/kubevision/kubevision/internal/auth"
	"github.com/kubevision/kubevision/internal/config"
	"github.com/kubevision/kubevision/internal/handler"
	"github.com/kubevision/kubevision/internal/handler/ws"
	"github.com/kubevision/kubevision/internal/kubernetes/cluster"
	"github.com/kubevision/kubevision/internal/kubernetes/informer"
	"github.com/kubevision/kubevision/internal/kubernetes/resource"
	"github.com/kubevision/kubevision/internal/middleware"
	"github.com/kubevision/kubevision/internal/repository"
	"github.com/kubevision/kubevision/internal/server"
	"github.com/kubevision/kubevision/internal/service"
)

func main() {
	// Parse command-line flags.
	configPath := flag.String("config", "", "path to config YAML file")
	flag.Parse()

	// ----- Logger -----
	logger, err := zap.NewProduction()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to init logger: %v\n", err)
		os.Exit(1)
	}
	defer func() { _ = logger.Sync() }()

	// ----- Configuration -----
	cfg, err := config.Load(*configPath)
	if err != nil {
		logger.Fatal("failed to load config", zap.Error(err))
	}
	logger.Info("configuration loaded",
		zap.Int("port", cfg.Server.Port),
		zap.String("db_driver", cfg.Database.Driver),
	)

	// ----- Database -----
	db, err := repository.NewDB(cfg, logger)
	if err != nil {
		logger.Fatal("failed to init database", zap.Error(err))
	}
	logger.Info("database connected", zap.String("driver", cfg.Database.Driver))

	// ----- Dependency Injection -----

	// Repositories
	userRepo := repository.NewUserRepo(db)
	clusterRepo := repository.NewClusterRepo(db)

	// Kubernetes components
	clusterManager := cluster.NewManager()
	resourceRegistry := resource.NewRegistry()
	informerMgr := informer.NewManager(logger)

	// WebSocket Hub
	wsHub := ws.NewHub(logger)
	go wsHub.Run()
	informerMgr.AddListener(wsHub)

	// JWT Manager
	jwtManager := auth.NewJWTManager(
		cfg.Auth.JWTSecret,
		cfg.Auth.AccessTokenTTL,
		cfg.Auth.RefreshTokenTTL,
	)

	// K8s Resource Repository
	k8sRepo := repository.NewK8sResourceRepo(informerMgr, clusterManager, resourceRegistry)

	// Services
	authService := service.NewAuthService(userRepo, jwtManager, logger)
	clusterService := service.NewClusterService(clusterRepo, clusterManager, informerMgr, resourceRegistry, logger, cfg.EncryptKey)
	resourceService := service.NewResourceService(k8sRepo, resourceRegistry, clusterRepo)

	// Handlers
	authHandler := handler.NewAuthHandler(authService)
	clusterHandler := handler.NewClusterHandler(clusterService)
	resourceHandler := handler.NewResourceHandler(resourceService)

	// Middleware
	authMiddleware := middleware.AuthMiddleware(jwtManager, userRepo)

	// Reconnect persisted clusters on startup.
	clusterService.InitClusters(context.Background())

	// Route dependencies
	routerDeps := &server.RouterDeps{
		AuthHandler:     authHandler,
		ClusterHandler:  clusterHandler,
		ResourceHandler: resourceHandler,
		WSHub:           wsHub,
		AuthMiddleware:  authMiddleware,
		Logger:          logger,
	}

	// ----- HTTP Server -----
	srv := server.New(cfg, logger, routerDeps)

	// Start HTTP server in a goroutine.
	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Start()
	}()

	// ----- Graceful Shutdown -----
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-quit:
		logger.Info("received shutdown signal", zap.String("signal", sig.String()))
	case err := <-errCh:
		if err != nil {
			logger.Error("server error", zap.Error(err))
		}
	}

	// Stop all informers.
	informerMgr.StopAll()

	if err := srv.Shutdown(); err != nil {
		logger.Error("shutdown error", zap.Error(err))
	}

	logger.Info("KubeVision exited")
}
