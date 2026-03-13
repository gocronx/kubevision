package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"

	"github.com/gocronx/kubevision/internal/auth"
	"github.com/gocronx/kubevision/internal/config"
	"github.com/gocronx/kubevision/internal/handler"
	"github.com/gocronx/kubevision/internal/handler/ws"
	"github.com/gocronx/kubevision/internal/kubernetes/cluster"
	"github.com/gocronx/kubevision/internal/kubernetes/informer"
	"github.com/gocronx/kubevision/internal/kubernetes/resource"
	"github.com/gocronx/kubevision/internal/middleware"
	"github.com/gocronx/kubevision/internal/repository"
	"github.com/gocronx/kubevision/internal/server"
	"github.com/gocronx/kubevision/internal/service"
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
	favoriteRepo := repository.NewFavoriteRepo(db)
	roleRepo := repository.NewRoleRepo(db)
	auditRepo := repository.NewAuditRepo(db)
	apiKeyRepo := repository.NewAPIKeyRepo(db)
	webhookRepo := repository.NewWebhookRepo(db)
	terminalSessionRepo := repository.NewTerminalSessionRepo(db)

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
	userService := service.NewUserService(userRepo)
	authService := service.NewAuthService(userRepo, jwtManager, cfg, logger)
	clusterService := service.NewClusterService(clusterRepo, clusterManager, informerMgr, resourceRegistry, logger, cfg.EncryptKey)
	resourceService := service.NewResourceService(k8sRepo, resourceRegistry, clusterRepo)
	resourceActionService := service.NewResourceActionService(clusterRepo, clusterManager)
	quotaService := service.NewQuotaService(k8sRepo, clusterRepo)
	overviewService := service.NewOverviewService(k8sRepo, clusterRepo)
	favoriteService := service.NewFavoriteService(favoriteRepo)
	searchService := service.NewSearchService(informerMgr, clusterManager, resourceRegistry, clusterRepo)
	apiKeyService := service.NewAPIKeyService(apiKeyRepo, userRepo)

	// Audit service — start background flush / purge goroutines.
	auditService := service.NewAuditService(auditRepo, cfg.Audit, logger)
	if cfg.Audit.Enabled {
		auditService.Start()
	}

	// P3: Webhook, terminal session recording, and compare services.
	webhookService := service.NewWebhookService(webhookRepo, logger)
	terminalSessionService := service.NewTerminalSessionService(terminalSessionRepo)
	compareService := service.NewCompareService(k8sRepo)
	topologyService := service.NewTopologyService(k8sRepo, clusterRepo)

	// P4: CRD discovery, OAuth, and Plugin services.
	crdService := service.NewCRDService(clusterManager, clusterRepo, logger)
	oauthService := service.NewOAuthService(userRepo, jwtManager, cfg, logger)
	templateRepo := repository.NewTemplateRepo(db)
	pluginConfigRepo := repository.NewPluginConfigRepo(db)
	pluginService := service.NewPluginService(pluginConfigRepo, logger)
	pluginService.InitFromDB(context.Background())
	templateService := service.NewTemplateService(templateRepo)
	if err := templateService.SeedBuiltinTemplates(context.Background()); err != nil {
		logger.Warn("failed to seed built-in templates", zap.Error(err))
	}

	// Start periodic CRD discovery in the background.
	crdCtx, crdCancel := context.WithCancel(context.Background())
	go crdService.StartPeriodicDiscovery(crdCtx, clusterManager.ListIDs, cfg.Kube.CRDDiscoveryInterval)

	// Periodically cleanup expired OAuth states.
	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			oauthService.CleanupExpiredStates()
		}
	}()

	// Register webhook service as an event listener so it dispatches on K8s events.
	informerMgr.AddListener(webhookService)

	// Handlers
	authHandler := handler.NewAuthHandler(authService)
	userHandler := handler.NewUserHandler(userService)
	clusterHandler := handler.NewClusterHandler(clusterService)
	resourceHandler := handler.NewResourceHandler(resourceService, resourceActionService)
	resourceActionHandler := handler.NewResourceActionHandler(resourceActionService)
	quotaHandler := handler.NewQuotaHandler(quotaService)
	overviewHandler := handler.NewOverviewHandler(overviewService)
	favoriteHandler := handler.NewFavoriteHandler(favoriteService)
	searchHandler := handler.NewSearchHandler(searchService)
	auditHandler := handler.NewAuditHandler(auditRepo)
	apiKeyHandler := handler.NewAPIKeyHandler(apiKeyService)

	// P3: Webhook, terminal session, and compare handlers.
	webhookHandler := handler.NewWebhookHandler(webhookService)
	terminalSessionHandler := handler.NewTerminalSessionHandler(terminalSessionService)
	compareHandler := handler.NewCompareHandler(compareService)
	topologyHandler := handler.NewTopologyHandler(topologyService)

	// P4: CRD, OAuth, and Plugin handlers.
	crdHandler := handler.NewCRDHandler(crdService)
	oauthHandler := handler.NewOAuthHandler(oauthService)
	pluginHandler := handler.NewPluginHandler(pluginService)
	templateHandler := handler.NewTemplateHandler(templateService)

	// Pod terminal and log streaming handlers.
	terminalHandler := ws.NewTerminalHandler(clusterManager, clusterRepo, jwtManager, userRepo, roleRepo, logger).
		WithSessionService(terminalSessionService)
	logsHandler := ws.NewLogsHandler(clusterManager, clusterRepo, jwtManager, userRepo, roleRepo, logger)

	// Middleware
	authMiddleware := middleware.AuthMiddleware(jwtManager, userRepo, apiKeyService)
	rbacMiddleware := middleware.RBACMiddleware(roleRepo)
	auditMiddleware := middleware.AuditMiddleware(auditService)

	// Reconnect persisted clusters on startup.
	clusterService.InitClusters(context.Background())

	// Route dependencies
	routerDeps := &server.RouterDeps{
		AuthHandler:            authHandler,
		UserHandler:            userHandler,
		ClusterHandler:         clusterHandler,
		ResourceHandler:        resourceHandler,
		SearchHandler:          searchHandler,
		ResourceActionHandler:  resourceActionHandler,
		FavoriteHandler:        favoriteHandler,
		QuotaHandler:           quotaHandler,
		OverviewHandler:        overviewHandler,
		AuditHandler:           auditHandler,
		APIKeyHandler:          apiKeyHandler,
		WebhookHandler:         webhookHandler,
		TerminalSessionHandler: terminalSessionHandler,
		CompareHandler:         compareHandler,
		TopologyHandler:        topologyHandler,
		CRDHandler:             crdHandler,
		OAuthHandler:           oauthHandler,
		PluginHandler:          pluginHandler,
		TemplateHandler:        templateHandler,
		WSHub:                  wsHub,
		TerminalHandler:        terminalHandler,
		LogsHandler:            logsHandler,
		AuthMiddleware:         authMiddleware,
		RBACMiddleware:         rbacMiddleware,
		AuditMiddleware:        auditMiddleware,
		Logger:                 logger,
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

	// Stop CRD discovery.
	crdCancel()

	// Stop all informers.
	informerMgr.StopAll()

	// Stop the WebSocket hub event loop.
	wsHub.Stop()

	// Stop audit service (flushes remaining entries).
	if cfg.Audit.Enabled {
		auditService.Stop()
	}

	if err := srv.Shutdown(); err != nil {
		logger.Error("shutdown error", zap.Error(err))
	}

	logger.Info("KubeVision exited")
}
