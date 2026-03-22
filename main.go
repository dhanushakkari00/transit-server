package main

import (
	"log"
	"time"

	"transit-server/cache"
	"transit-server/config"
	"transit-server/database"
	"transit-server/routes"

	"github.com/gin-gonic/gin"
)

// AppCache is the global in-memory cache available for general-purpose use.
var AppCache *cache.Store

func main() {
	// Load configuration from .env / environment
	config.Load()

	// Initialize database connection and run migrations
	database.Connect()

	// Initialize in-memory cache with 5-minute eviction cycle
	AppCache = cache.New(5 * time.Minute)
	defer AppCache.Close()

	// Create Gin router with default middleware (logger + recovery)
	router := gin.Default()

	// CORS middleware — allow all origins in development
	router.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	// Register all routes
	routes.RegisterRoutes(router)

	// Start the server
	port := config.AppConfig.Port
	log.Printf("🚀 Transit Server starting on port %s", port)
	log.Printf("📍 API Base: http://localhost:%s/api/v1", port)
	log.Printf("❤️  Health:   http://localhost:%s/health", port)

	if err := router.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
