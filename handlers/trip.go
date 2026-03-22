package handlers

import (
	"log"
	"net/http"
	"time"

	"transit-server/database"
	"transit-server/models"

	"github.com/gin-gonic/gin"
)

// --- Route Management (Aggregator/Admin) ---

// CreateRoute creates a new transit route for the aggregator's agency.
// POST /api/v1/aggregator/routes
func CreateRoute(c *gin.Context) {
	var req models.CreateRouteRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusUnprocessableEntity, models.ErrorResponse{
			Error:   "Validation failed",
			Details: map[string]string{"message": err.Error()},
		})
		return
	}

	userID, _ := c.Get("userID")

	var aggregator models.Aggregator
	if database.DB.Where("user_id = ?", userID).First(&aggregator).Error != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{Error: "Aggregator not found"})
		return
	}

	// Check uniqueness of gtfs_route_id
	var count int64
	database.DB.Model(&models.Route{}).Where("gtfs_route_id = ?", req.RouteID).Count(&count)
	if count > 0 {
		c.JSON(http.StatusConflict, models.ErrorResponse{Error: "Route ID already exists"})
		return
	}

	route := models.Route{
		AgencyID:    aggregator.ID,
		GtfsRouteID: req.RouteID,
		ShortName:   req.ShortName,
		LongName:    req.LongName,
		Description: req.Description,
		RouteType:   req.RouteType,
		Color:       req.Color,
		TextColor:   req.TextColor,
		IsActive:    true,
	}

	if err := database.DB.Create(&route).Error; err != nil {
		log.Printf("Error creating route: %v", err)
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Failed to create route"})
		return
	}

	c.JSON(http.StatusCreated, models.RouteResponse{
		ID:          route.ID,
		RouteID:     route.GtfsRouteID,
		ShortName:   route.ShortName,
		LongName:    route.LongName,
		Description: route.Description,
		RouteType:   route.RouteType,
		IsActive:    route.IsActive,
	})
}

// ListRoutes returns all routes for the aggregator's agency.
// GET /api/v1/aggregator/routes
func ListRoutes(c *gin.Context) {
	userID, _ := c.Get("userID")

	var aggregator models.Aggregator
	if database.DB.Where("user_id = ?", userID).First(&aggregator).Error != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{Error: "Aggregator not found"})
		return
	}

	var routes []models.Route
	database.DB.Where("agency_id = ? AND is_active = ?", aggregator.ID, true).Find(&routes)

	responses := make([]models.RouteResponse, len(routes))
	for i, r := range routes {
		responses[i] = models.RouteResponse{
			ID: r.ID, RouteID: r.GtfsRouteID, ShortName: r.ShortName,
			LongName: r.LongName, Description: r.Description,
			RouteType: r.RouteType, IsActive: r.IsActive,
		}
	}

	c.JSON(http.StatusOK, gin.H{"routes": responses, "count": len(responses)})
}

// --- Trip Management (Aggregator/Admin) ---

// CreateTrip creates a new trip on a route.
// POST /api/v1/aggregator/trips
func CreateTrip(c *gin.Context) {
	var req models.CreateTripRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusUnprocessableEntity, models.ErrorResponse{
			Error:   "Validation failed",
			Details: map[string]string{"message": err.Error()},
		})
		return
	}

	userID, _ := c.Get("userID")
	var aggregator models.Aggregator
	if database.DB.Where("user_id = ?", userID).First(&aggregator).Error != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{Error: "Aggregator not found"})
		return
	}

	var route models.Route
	if database.DB.Where("id = ? AND agency_id = ?", req.RouteID, aggregator.ID).First(&route).Error != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{Error: "Route not found"})
		return
	}

	// Check trip_id uniqueness
	var count int64
	database.DB.Model(&models.Trip{}).Where("gtfs_trip_id = ?", req.TripID).Count(&count)
	if count > 0 {
		c.JSON(http.StatusConflict, models.ErrorResponse{Error: "Trip ID already exists"})
		return
	}

	trip := models.Trip{
		RouteRefID:  route.ID,
		GtfsTripID:  req.TripID,
		Headsign:    req.Headsign,
		DirectionID: req.DirectionID,
		IsActive:    true,
	}

	if err := database.DB.Create(&trip).Error; err != nil {
		log.Printf("Error creating trip: %v", err)
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Failed to create trip"})
		return
	}

	c.JSON(http.StatusCreated, models.TripResponse{
		ID: trip.ID, RouteID: trip.RouteRefID, TripID: trip.GtfsTripID,
		Headsign: trip.Headsign, DirectionID: trip.DirectionID, IsActive: trip.IsActive,
	})
}

// ListTrips returns all trips for the aggregator's agency.
// GET /api/v1/aggregator/trips
func ListTrips(c *gin.Context) {
	userID, _ := c.Get("userID")

	var aggregator models.Aggregator
	if database.DB.Where("user_id = ?", userID).First(&aggregator).Error != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{Error: "Aggregator not found"})
		return
	}

	var trips []models.Trip
	database.DB.Joins("JOIN routes ON routes.id = trips.route_ref_id").
		Where("routes.agency_id = ? AND trips.is_active = ?", aggregator.ID, true).
		Find(&trips)

	responses := make([]models.TripResponse, len(trips))
	for i, t := range trips {
		responses[i] = models.TripResponse{
			ID: t.ID, RouteID: t.RouteRefID, TripID: t.GtfsTripID,
			Headsign: t.Headsign, DirectionID: t.DirectionID, IsActive: t.IsActive,
		}
	}

	c.JSON(http.StatusOK, gin.H{"trips": responses, "count": len(responses)})
}

// --- Active Trip Management (Driver) ---

// StartTrip assigns the current driver to a trip.
// POST /api/v1/driver/trip/start
func StartTrip(c *gin.Context) {
	var req models.StartTripRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusUnprocessableEntity, models.ErrorResponse{
			Error:   "Validation failed",
			Details: map[string]string{"message": err.Error()},
		})
		return
	}

	userID, _ := c.Get("userID")

	var driver models.Driver
	if database.DB.Where("user_id = ?", userID).First(&driver).Error != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{Error: "Driver profile not found"})
		return
	}

	var trip models.Trip
	if database.DB.First(&trip, req.TripID).Error != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{Error: "Trip not found"})
		return
	}

	// Check if driver already has an active trip
	var existing models.ActiveTrip
	if database.DB.Where("driver_id = ? AND is_active = ?", driver.ID, true).First(&existing).Error == nil {
		c.JSON(http.StatusConflict, models.ErrorResponse{
			Error: "You already have an active trip. End it first.",
		})
		return
	}

	vehicleID := req.VehicleID
	if vehicleID == "" {
		vehicleID = driver.VehicleNumber
	}

	activeTrip := models.ActiveTrip{
		DriverID:  driver.ID,
		TripRefID: trip.ID,
		VehicleID: vehicleID,
		StartedAt: time.Now(),
		IsActive:  true,
	}

	if err := database.DB.Create(&activeTrip).Error; err != nil {
		log.Printf("Error starting trip: %v", err)
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Failed to start trip"})
		return
	}

	c.JSON(http.StatusCreated, models.ActiveTripResponse{
		ID: activeTrip.ID, DriverID: activeTrip.DriverID,
		TripID: activeTrip.TripRefID, VehicleID: activeTrip.VehicleID,
		StartedAt: activeTrip.StartedAt, IsActive: true,
	})
}

// EndTrip ends the current driver's active trip.
// POST /api/v1/driver/trip/end
func EndTrip(c *gin.Context) {
	userID, _ := c.Get("userID")

	var driver models.Driver
	if database.DB.Where("user_id = ?", userID).First(&driver).Error != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{Error: "Driver profile not found"})
		return
	}

	var activeTrip models.ActiveTrip
	if database.DB.Where("driver_id = ? AND is_active = ?", driver.ID, true).First(&activeTrip).Error != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{Error: "No active trip found"})
		return
	}

	now := time.Now()
	database.DB.Model(&activeTrip).Updates(map[string]interface{}{
		"is_active": false,
		"ended_at":  now,
	})

	c.JSON(http.StatusOK, models.MessageResponse{Message: "Trip ended successfully"})
}

// --- Batch Location Sync (Offline Mode) ---

// BatchUpdateLocation receives multiple location entries from offline sync.
// POST /api/v1/driver/locations/batch
func BatchUpdateLocation(c *gin.Context) {
	var req models.BatchLocationRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusUnprocessableEntity, models.ErrorResponse{
			Error:   "Validation failed",
			Details: map[string]string{"message": err.Error()},
		})
		return
	}

	userID, _ := c.Get("userID")

	var driver models.Driver
	if database.DB.Where("user_id = ?", userID).First(&driver).Error != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{Error: "Driver profile not found"})
		return
	}

	// Find the latest entry to update DB with the most recent location
	var latest models.BatchLocationEntry
	for _, loc := range req.Locations {
		if loc.Timestamp.After(latest.Timestamp) {
			latest = loc
		}
	}

	// Update the cache with the latest entry
	if LocationCache != nil {
		loc := models.CachedDriverLocation{
			DriverID:  driver.ID,
			Lat:       latest.Lat,
			Lng:       latest.Lng,
			Heading:   latest.Heading,
			Speed:     latest.Speed,
			UpdatedAt: latest.Timestamp,
		}
		LocationCache.Set(locationKey(driver.ID), loc, locationTTL)
	}

	// Update DB with latest location
	database.DB.Model(&driver).Updates(map[string]interface{}{
		"last_lat":     latest.Lat,
		"last_lng":     latest.Lng,
		"last_heading": latest.Heading,
		"last_speed":   latest.Speed,
		"last_seen_at": latest.Timestamp,
	})

	c.JSON(http.StatusOK, gin.H{
		"message":  "Batch location sync complete",
		"received": len(req.Locations),
	})
}
