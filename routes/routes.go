package routes

import (
	"transit-server/gtfsrt"
	"transit-server/handlers"
	"transit-server/middleware"
	"transit-server/models"
	"transit-server/ws"

	"github.com/gin-gonic/gin"
)

// RegisterRoutes sets up all API routes.
func RegisterRoutes(router *gin.Engine, feedHandler *gtfsrt.Handler, wsHandler *ws.Handler) {
	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	v1 := router.Group("/api/v1")
	{
		// ─── Shared auth routes ───
		auth := v1.Group("/auth")
		{
			auth.POST("/forgot-password", handlers.ForgotPassword)
			auth.POST("/reset-password", handlers.ResetPassword)
			auth.POST("/refresh", handlers.RefreshToken)

			authProtected := auth.Group("")
			authProtected.Use(middleware.AuthRequired())
			{
				authProtected.POST("/logout", handlers.Logout)
			}
		}

		// ─── Driver routes ───
		driver := v1.Group("/driver")
		{
			driver.POST("/register", handlers.DriverRegister)
			driver.POST("/login", handlers.DriverLogin)

			driverProtected := driver.Group("")
			driverProtected.Use(middleware.AuthRequired(), middleware.RequireRole(models.RoleDriver))
			{
				driverProtected.GET("/me", handlers.DriverMe)
				driverProtected.POST("/join", handlers.JoinAggregator)
				driverProtected.PUT("/location", handlers.UpdateLocation)
				driverProtected.POST("/locations/batch", handlers.BatchUpdateLocation)
				driverProtected.POST("/trip/start", handlers.StartTrip)
				driverProtected.POST("/trip/end", handlers.EndTrip)
			}
		}

		// ─── Aggregator routes ───
		aggregator := v1.Group("/aggregator")
		{
			aggregator.POST("/register", handlers.AggregatorRegister)
			aggregator.POST("/login", handlers.AggregatorLogin)

			// Profile and API key management require only JWT + role.
			aggAuthOnly := aggregator.Group("")
			aggAuthOnly.Use(
				middleware.AuthRequired(),
				middleware.RequireRole(models.RoleAggregator, models.RoleAdmin),
			)
			{
				aggAuthOnly.GET("/me", handlers.AggregatorMe)
				aggAuthOnly.GET("/api-key", handlers.AggregatorAPIKey)
				aggAuthOnly.PUT("/api-key", handlers.RotateAggregatorAPIKey)
			}

			// All other aggregator routes require JWT + matching API key + role.
			aggProtected := aggregator.Group("")
			aggProtected.Use(
				middleware.AuthRequired(),
				middleware.APIKeyRequired(),
				middleware.RequireRole(models.RoleAggregator, models.RoleAdmin),
			)
			{
				// Driver management
				aggProtected.GET("/drivers", handlers.ListDrivers)
				aggProtected.GET("/drivers/locations", handlers.GetAllDriverLocations)
				aggProtected.GET("/drivers/:id", handlers.GetDriver)
				aggProtected.GET("/drivers/:id/location", handlers.GetDriverLocation)

				// Route & trip management
				aggProtected.POST("/routes", handlers.CreateRoute)
				aggProtected.GET("/routes", handlers.ListRoutes)
				aggProtected.POST("/trips", handlers.CreateTrip)
				aggProtected.GET("/trips", handlers.ListTrips)

				// GTFS-RT feeds (all vehicles or single vehicle, protobuf + debug JSON)
				aggProtected.GET("/feed/vehicle-positions", feedHandler.ServeFeedForAgency)
				aggProtected.GET("/feed/vehicle-positions/debug", feedHandler.ServeDebugFeedForAgency)
				aggProtected.GET("/feed/vehicle-positions/:driverId", feedHandler.ServeFeedForVehicle)
				aggProtected.GET("/feed/vehicle-positions/:driverId/debug", feedHandler.ServeDebugFeedForVehicle)
			}

			// WebSocket subscribe — auth done inside handler via query params
			aggregator.GET("/subscribe", wsHandler.HandleSubscribe)
		}
	}
}
