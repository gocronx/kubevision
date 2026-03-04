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
	AuthHandler            *handler.AuthHandler
	UserHandler            *handler.UserHandler
	ClusterHandler         *handler.ClusterHandler
	ResourceHandler        *handler.ResourceHandler
	SearchHandler          *handler.SearchHandler
	ResourceActionHandler  *handler.ResourceActionHandler
	FavoriteHandler        *handler.FavoriteHandler
	QuotaHandler           *handler.QuotaHandler
	OverviewHandler        *handler.OverviewHandler
	AuditHandler           *handler.AuditHandler
	APIKeyHandler          *handler.APIKeyHandler
	WebhookHandler         *handler.WebhookHandler
	TerminalSessionHandler *handler.TerminalSessionHandler
	CompareHandler         *handler.CompareHandler
	TopologyHandler        *handler.TopologyHandler
	CRDHandler             *handler.CRDHandler
	OAuthHandler           *handler.OAuthHandler
	PluginHandler          *handler.PluginHandler
	TemplateHandler        *handler.TemplateHandler
	WSHub                  *ws.Hub
	TerminalHandler        *ws.TerminalHandler
	LogsHandler            *ws.LogsHandler
	AuthMiddleware         gin.HandlerFunc
	RBACMiddleware         gin.HandlerFunc
	AuditMiddleware        gin.HandlerFunc
	Logger                 *zap.Logger
}

// RegisterRoutes sets up all API route groups on the given engine.
func RegisterRoutes(r *gin.Engine, deps *RouterDeps) {
	// Global middleware.
	r.Use(middleware.SecurityHeaders())
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
		authGroup.Use(middleware.AuthRateLimit())
		{
			if deps != nil && deps.AuthHandler != nil {
				authGroup.POST("/login", deps.AuthHandler.Login)
				authGroup.POST("/refresh", deps.AuthHandler.Refresh)

				// 2FA verification endpoints — guarded by the short-lived
				// tempToken issued during login rather than a full JWT.
				authGroup.POST("/2fa/verify", deps.AuthHandler.Verify2FA)
				authGroup.POST("/2fa/recovery", deps.AuthHandler.Recovery2FA)
			}

			// OAuth/OIDC routes — public (users are redirected here from providers).
			if deps != nil && deps.OAuthHandler != nil {
				oauth := authGroup.Group("/oauth")
				oauth.GET("/providers", deps.OAuthHandler.ListProviders)
				oauth.GET("/:provider/authorize", deps.OAuthHandler.Authorize)
				oauth.GET("/:provider/callback", deps.OAuthHandler.Callback)
			}
		}

		// ---- Auth-only routes (authentication required, NO RBAC) ----
		// These routes are available to all authenticated users regardless of role.
		authOnly := v1.Group("")
		if deps != nil && deps.AuthMiddleware != nil {
			authOnly.Use(deps.AuthMiddleware)
		} else {
			authOnly.Use(middleware.Auth())
		}
		{
			// User profile.
			if deps != nil && deps.AuthHandler != nil {
				authOnly.GET("/users/me", deps.AuthHandler.Me)

				// 2FA management — require a valid session JWT.
				twoFA := authOnly.Group("/auth/2fa")
				twoFA.POST("/setup", deps.AuthHandler.Setup2FA)
				twoFA.POST("/enable", deps.AuthHandler.Enable2FA)
				twoFA.POST("/disable", deps.AuthHandler.Disable2FA)
			}

			// Change own password — any authenticated user.
			if deps != nil && deps.UserHandler != nil {
				authOnly.PUT("/users/me/password", deps.UserHandler.ChangePassword)
			}
		}

		// ---- Protected routes (authentication + RBAC + audit) ----
		protected := v1.Group("")
		if deps != nil && deps.AuthMiddleware != nil {
			protected.Use(deps.AuthMiddleware)
		} else {
			protected.Use(middleware.Auth())
		}
		// Apply RBAC middleware after auth so the user role is available.
		if deps != nil && deps.RBACMiddleware != nil {
			protected.Use(deps.RBACMiddleware)
		}
		// Apply audit middleware to capture mutating operations.
		if deps != nil && deps.AuditMiddleware != nil {
			protected.Use(deps.AuditMiddleware)
		}
		{

			// User management routes — admin+ only.
			// The RBAC middleware already enforces admin bypass; non-admin roles
			// must have explicit "users:*" permissions in their role record.
			if deps != nil && deps.UserHandler != nil {
				users := protected.Group("/users")
				users.GET("", deps.UserHandler.List)
				users.POST("", deps.UserHandler.Create)
				users.GET("/:id", deps.UserHandler.Get)
				users.PUT("/:id", deps.UserHandler.Update)
				users.DELETE("/:id", deps.UserHandler.Delete)
				users.PUT("/:id/reset-password", deps.UserHandler.ResetPassword)
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

				// Batch operations (nested under cluster).
				if deps.ResourceHandler != nil {
					clusters.POST("/:id/resources/batch-delete", deps.ResourceHandler.BatchDelete)
					clusters.POST("/:id/batch-restart", deps.ResourceHandler.BatchRestart)
				}

				// Quota summary route (nested under cluster).
				if deps.QuotaHandler != nil {
					clusters.GET("/:id/quota-summary", deps.QuotaHandler.GetQuotaSummary)
				}

				// Cluster overview route (nested under cluster).
				if deps.OverviewHandler != nil {
					clusters.GET("/:id/overview", deps.OverviewHandler.GetOverview)
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

			// CRD discovery routes (nested under cluster).
			if deps != nil && deps.CRDHandler != nil && deps.ClusterHandler != nil {
				protected.GET("/clusters/:id/crds", deps.CRDHandler.List)
				protected.POST("/clusters/:id/crds/refresh", deps.CRDHandler.Refresh)
			}

			// Plugin management routes.
			if deps != nil && deps.PluginHandler != nil {
				plugins := protected.Group("/plugins")
				plugins.GET("", deps.PluginHandler.List)
				plugins.GET("/:name", deps.PluginHandler.GetConfig)
				plugins.PUT("/:name", deps.PluginHandler.Configure)
				plugins.GET("/:name/health", deps.PluginHandler.HealthCheck)

				// Plugin-specific data endpoints.
				plugins.GET("/prometheus/query", deps.PluginHandler.PrometheusQuery)
				plugins.GET("/grafana/dashboards", deps.PluginHandler.GrafanaDashboards)
				plugins.GET("/argocd/applications", deps.PluginHandler.ArgoCDApplications)
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

			// Template routes.
			if deps != nil && deps.TemplateHandler != nil {
				templates := protected.Group("/templates")
				templates.GET("", deps.TemplateHandler.List)
				templates.GET("/:id", deps.TemplateHandler.Get)
				templates.POST("", deps.TemplateHandler.Create)
				templates.DELETE("/:id", deps.TemplateHandler.Delete)
			}

			// Audit log routes.
			if deps != nil && deps.AuditHandler != nil {
				protected.GET("/audit-logs", deps.AuditHandler.List)
			}

			// API key routes.
			if deps != nil && deps.APIKeyHandler != nil {
				apiKeys := protected.Group("/api-keys")
				apiKeys.GET("", deps.APIKeyHandler.List)
				apiKeys.POST("", deps.APIKeyHandler.Generate)
				apiKeys.DELETE("/:id", deps.APIKeyHandler.Revoke)
			}

			// Webhook routes.
			if deps != nil && deps.WebhookHandler != nil {
				webhooks := protected.Group("/webhooks")
				webhooks.GET("", deps.WebhookHandler.List)
				webhooks.POST("", deps.WebhookHandler.Create)
				webhooks.PUT("/:id", deps.WebhookHandler.Update)
				webhooks.DELETE("/:id", deps.WebhookHandler.Delete)
				webhooks.POST("/:id/test", deps.WebhookHandler.Test)
			}

			// Terminal session recording routes.
			if deps != nil && deps.TerminalSessionHandler != nil {
				ts := protected.Group("/terminal-sessions")
				ts.GET("", deps.TerminalSessionHandler.List)
				ts.GET("/:id", deps.TerminalSessionHandler.Get)
				ts.GET("/:id/play", deps.TerminalSessionHandler.Play)
			}

			// Cross-cluster resource comparison route.
			if deps != nil && deps.CompareHandler != nil {
				protected.POST("/compare", deps.CompareHandler.Compare)
			}

			// Topology route (nested under clusters).
			if deps != nil && deps.TopologyHandler != nil && deps.ClusterHandler != nil {
				protected.GET("/clusters/:id/namespaces/:namespace/topology", deps.TopologyHandler.GetTopology)
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
