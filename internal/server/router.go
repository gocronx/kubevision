package server

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/kubevision/kubevision/internal/handler"
	"github.com/kubevision/kubevision/internal/handler/ws"
	"github.com/kubevision/kubevision/internal/middleware"
)

// RouterDeps holds all handler and middleware dependencies required to register routes.
type RouterDeps struct {
	AuthHandler           *handler.AuthHandler
	ClusterHandler        *handler.ClusterHandler
	ResourceHandler       *handler.ResourceHandler
	SearchHandler         *handler.SearchHandler
	ResourceActionHandler *handler.ResourceActionHandler
	FavoriteHandler       *handler.FavoriteHandler
	QuotaHandler          *handler.QuotaHandler
	WSHub                 *ws.Hub
	TerminalHandler       *ws.TerminalHandler
	LogsHandler           *ws.LogsHandler
	AuthMiddleware        gin.HandlerFunc
	Logger                *zap.Logger
}

// RegisterRoutes sets up all API route groups on the given engine.
func RegisterRoutes(r *gin.Engine, deps *RouterDeps) {
	// Global middleware.
	r.Use(middleware.RequestID())
	if deps != nil && deps.Logger != nil {
		r.Use(middleware.Logger(deps.Logger))
	}

	// Health check - always available.
	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// API v1 group.
	v1 := r.Group("/api/v1")
	{
		// ---- Public auth routes (no authentication required) ----
		authGroup := v1.Group("/auth")
		{
			if deps != nil && deps.AuthHandler != nil {
				authGroup.POST("/login", deps.AuthHandler.Login)
				authGroup.POST("/refresh", deps.AuthHandler.Refresh)
			}
		}

		// ---- Protected routes (authentication required) ----
		protected := v1.Group("")
		if deps != nil && deps.AuthMiddleware != nil {
			protected.Use(deps.AuthMiddleware)
		} else {
			protected.Use(middleware.Auth())
		}
		{
			// User profile.
			if deps != nil && deps.AuthHandler != nil {
				protected.GET("/users/me", deps.AuthHandler.Me)
			}

			// Cluster management routes.
			if deps != nil && deps.ClusterHandler != nil {
				clusters := protected.Group("/clusters")
				clusters.GET("", deps.ClusterHandler.List)
				clusters.GET("/:id", deps.ClusterHandler.Get)
				clusters.POST("", deps.ClusterHandler.Create)
				clusters.DELETE("/:id", deps.ClusterHandler.Delete)

				// Generic Kubernetes resource CRUD (nested under cluster).
				if deps.ResourceHandler != nil {
					res := clusters.Group("/:id/resources")

					// Standard CRUD operations.
					res.GET("/:resource", deps.ResourceHandler.List)
					res.GET("/:resource/:name", deps.ResourceHandler.Get)
					res.POST("/:resource", deps.ResourceHandler.Create)
					res.PUT("/:resource/:name", deps.ResourceHandler.Update)
					res.DELETE("/:resource/:name", deps.ResourceHandler.Delete)
					res.PATCH("/:resource/:name", deps.ResourceHandler.Patch)

					// Dry-run preview operations.
					// POST .../resources/:resource/dry-run  -> preview a create
					// PUT  .../resources/:resource/:name/dry-run -> preview an update
					res.POST("/:resource/dry-run", deps.ResourceHandler.DryRunCreate)
					res.PUT("/:resource/:name/dry-run", deps.ResourceHandler.DryRunUpdate)
				}

				// Quota summary route (nested under cluster).
				if deps.QuotaHandler != nil {
					clusters.GET("/:id/quota-summary", deps.QuotaHandler.GetQuotaSummary)
				}

				// Global search route (nested under cluster).
				if deps.SearchHandler != nil {
					clusters.GET("/:id/search", deps.SearchHandler.Search)
				}

				// Workload lifecycle action routes.
				// Pattern: /api/v1/clusters/:id/namespaces/:namespace/:kind/:name/<action>
				if deps.ResourceActionHandler != nil {
					actions := clusters.Group("/:id/namespaces/:namespace")

					// Scale: deployments, statefulsets, replicasets.
					actions.PUT("/:kind/:name/scale", deps.ResourceActionHandler.Scale)

					// Restart: deployments, statefulsets, daemonsets.
					actions.POST("/:kind/:name/restart", deps.ResourceActionHandler.Restart)

					// Rollout history and rollback: deployments only.
					actions.GET("/deployments/:name/history", deps.ResourceActionHandler.RolloutHistory)
					actions.POST("/deployments/:name/rollback", deps.ResourceActionHandler.Rollback)
				}
			}

			// Favorites routes.
			if deps != nil && deps.FavoriteHandler != nil {
				favs := protected.Group("/favorites")
				favs.GET("", deps.FavoriteHandler.List)
				favs.POST("", deps.FavoriteHandler.Create)
				favs.DELETE("/:id", deps.FavoriteHandler.Delete)
				favs.POST("/toggle", deps.FavoriteHandler.Toggle)
				favs.PUT("/reorder", deps.FavoriteHandler.Reorder)
				favs.GET("/check", deps.FavoriteHandler.Check)
			}

			// Pod terminal and log streaming routes.
			// Authentication is handled inside the handlers via the ?token= query param
			// because browsers cannot send custom headers during a WebSocket upgrade.
			// These routes are registered outside the protected group on purpose.
			if deps != nil && deps.TerminalHandler != nil {
				v1.GET("/clusters/:id/namespaces/:namespace/pods/:name/exec", deps.TerminalHandler.HandleExec)
			}
			if deps != nil && deps.LogsHandler != nil {
				v1.GET("/clusters/:id/namespaces/:namespace/pods/:name/logs", deps.LogsHandler.HandleLogs)
			}

			// WebSocket route.
			if deps != nil && deps.WSHub != nil {
				protected.GET("/ws/watch", deps.WSHub.HandleWatch)
			}
		}
	}
}
