package main

import (
	"log"
	"time"

	"transit-server/cache"
	"transit-server/config"
	"transit-server/database"
	"transit-server/gtfsrt"
	"transit-server/handlers"
	"transit-server/routes"
	"transit-server/ws"

	"github.com/gin-gonic/gin"
)

func main() {
	// Load configuration from .env / environment
	config.Load()

	// Initialize database connection and run migrations
	database.Connect()

	// Initialize in-memory cache with 30-second eviction cycle
	appCache := cache.New(30 * time.Second)
	defer appCache.Close()

	// Wire cache into location handlers
	handlers.LocationCache = appCache

	// Initialize WebSocket hub for live location push
	hub := ws.NewHub()
	go hub.Run()
	defer hub.Stop()
	handlers.LiveHub = hub

	// Initialize GTFS-RT feed generator (OOP chain):
	// DataSource (interface) → DBDataSource → FeedGenerator → Handler
	dataSource := gtfsrt.NewDBDataSource(database.DB, appCache)
	feedGenerator := gtfsrt.NewFeedGenerator(dataSource)
	feedHandler := gtfsrt.NewHandler(feedGenerator)

	// Initialize WebSocket handler
	wsHandler := ws.NewHandler(hub)

	// Create Gin router
	router := gin.Default()

	// CORS middleware
	router.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization, X-API-Key")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	// Register all routes
	routes.RegisterRoutes(router, feedHandler, wsHandler)

	// Start the server
	port := config.AppConfig.Port
	log.Printf("🚀 Transit Server starting on port %s", port)
	log.Printf("📍 API Base: http://localhost:%s/api/v1", port)
	log.Printf("❤️  Health:   http://localhost:%s/health", port)

	if err := router.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
