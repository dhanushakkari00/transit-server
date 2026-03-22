package routes

import (
	"transit-server/handlers"
	"transit-server/middleware"

	"github.com/gin-gonic/gin"
)

// RegisterRoutes sets up all API routes.
func RegisterRoutes(router *gin.Engine) {
	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// API v1 auth routes
	v1 := router.Group("/api/v1")
	{
		auth := v1.Group("/auth")
		{
			// Public routes
			auth.POST("/register", handlers.Register)
			auth.POST("/login", handlers.Login)
			auth.POST("/forgot-password", handlers.ForgotPassword)
			auth.POST("/reset-password", handlers.ResetPassword)
			auth.POST("/refresh", handlers.RefreshToken)

			// Protected routes (require valid JWT)
			protected := auth.Group("")
			protected.Use(middleware.AuthRequired())
			{
				protected.POST("/logout", handlers.Logout)
				protected.GET("/me", handlers.Me)
			}
		}
	}
}
