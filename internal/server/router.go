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
	AuthHandler     *handler.AuthHandler
	ClusterHandler  *handler.ClusterHandler
	ResourceHandler *handler.ResourceHandler
	WSHub           *ws.Hub
	AuthMiddleware  gin.HandlerFunc
	Logger          *zap.Logger
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

				// Kubernetes resource routes (nested under cluster).
				if deps.ResourceHandler != nil {
					res := clusters.Group("/:id/resources")
					res.GET("/:resource", deps.ResourceHandler.List)
					res.GET("/:resource/:name", deps.ResourceHandler.Get)
					res.POST("/:resource", deps.ResourceHandler.Create)
					res.PUT("/:resource/:name", deps.ResourceHandler.Update)
					res.DELETE("/:resource/:name", deps.ResourceHandler.Delete)
					res.PATCH("/:resource/:name", deps.ResourceHandler.Patch)
				}
			}

			// WebSocket route.
			if deps != nil && deps.WSHub != nil {
				protected.GET("/ws/watch", deps.WSHub.HandleWatch)
			}
		}
	}
}
