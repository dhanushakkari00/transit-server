package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"transit-server/cache"
	"transit-server/database"
	"transit-server/models"
	"transit-server/ws"

	"github.com/gin-gonic/gin"
)

// LocationCache is the global cache reference for location data.
var LocationCache *cache.Store

// LiveHub is the WebSocket hub for broadcasting live location updates.
var LiveHub *ws.Hub

const locationTTL = 60 * time.Second

func locationKey(driverID uint) string {
	return fmt.Sprintf("location:driver:%d", driverID)
}

// UpdateLocation receives a GPS update from a driver.
// PUT /api/v1/driver/location
func UpdateLocation(c *gin.Context) {
	var req models.UpdateLocationRequest

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

	now := time.Now()

	// 1. Write to cache (live/real-time, 60s TTL)
	loc := models.CachedDriverLocation{
		DriverID:  driver.ID,
		Lat:       req.Lat,
		Lng:       req.Lng,
		Heading:   req.Heading,
		Speed:     req.Speed,
		UpdatedAt: now,
	}
	LocationCache.Set(locationKey(driver.ID), loc, locationTTL)

	// 2. Write to DB (persistent last-known location)
	database.DB.Model(&driver).Updates(map[string]interface{}{
		"last_lat":     req.Lat,
		"last_lng":     req.Lng,
		"last_heading": req.Heading,
		"last_speed":   req.Speed,
		"last_seen_at": now,
	})

	// 3. Broadcast to WebSocket subscribers
	if LiveHub != nil {
		LiveHub.Broadcast(driver.ID, loc)
	}

	c.JSON(http.StatusOK, models.MessageResponse{Message: "Location updated"})
}

// GetDriverLocation returns a specific driver's location (live or last-known).
// GET /api/v1/aggregator/drivers/:id/location
func GetDriverLocation(c *gin.Context) {
	userID, _ := c.Get("userID")

	driverIDParam := c.Param("id")
	driverID, err := strconv.ParseUint(driverIDParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "Invalid driver ID"})
		return
	}

	var aggregator models.Aggregator
	if database.DB.Where("user_id = ?", userID).First(&aggregator).Error != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{Error: "Aggregator profile not found"})
		return
	}

	// Verify driver is mapped
	var mapping models.DriverAggregatorMapping
	if database.DB.Where("driver_id = ? AND aggregator_id = ? AND status = ?",
		uint(driverID), aggregator.ID, models.MappingStatusActive).First(&mapping).Error != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{Error: "Driver not found or not mapped to your account"})
		return
	}

	// Try cache first (online)
	if val, found := LocationCache.Get(locationKey(uint(driverID))); found {
		loc := val.(models.CachedDriverLocation)
		updatedAt := loc.UpdatedAt
		c.JSON(http.StatusOK, models.DriverLocationResponse{
			DriverID: loc.DriverID, Status: "online",
			Lat: loc.Lat, Lng: loc.Lng, Heading: loc.Heading, Speed: loc.Speed,
			UpdatedAt: &updatedAt,
		})
		return
	}

	// Cache miss → fall back to DB (offline)
	var driver models.Driver
	if database.DB.First(&driver, uint(driverID)).Error != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{Error: "Driver not found"})
		return
	}

	if driver.LastSeenAt == nil {
		c.JSON(http.StatusOK, models.DriverLocationResponse{DriverID: driver.ID, Status: "offline"})
		return
	}

	c.JSON(http.StatusOK, models.DriverLocationResponse{
		DriverID: driver.ID, Status: "offline",
		Lat: driver.LastLat, Lng: driver.LastLng,
		Heading: driver.LastHeading, Speed: driver.LastSpeed,
		UpdatedAt: driver.LastSeenAt,
	})
}

// GetAllDriverLocations returns locations for all drivers mapped to the aggregator.
// GET /api/v1/aggregator/drivers/locations
func GetAllDriverLocations(c *gin.Context) {
	userID, _ := c.Get("userID")

	var aggregator models.Aggregator
	if database.DB.Where("user_id = ?", userID).First(&aggregator).Error != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{Error: "Aggregator profile not found"})
		return
	}

	var mappings []models.DriverAggregatorMapping
	database.DB.Where("aggregator_id = ? AND status = ?",
		aggregator.ID, models.MappingStatusActive).Find(&mappings)

	if len(mappings) == 0 {
		c.JSON(http.StatusOK, gin.H{"locations": []interface{}{}, "count": 0})
		return
	}

	driverIDs := make([]uint, len(mappings))
	for i, m := range mappings {
		driverIDs[i] = m.DriverID
	}

	var drivers []models.Driver
	database.DB.Where("id IN ?", driverIDs).Find(&drivers)

	driverMap := make(map[uint]models.Driver)
	for _, d := range drivers {
		driverMap[d.ID] = d
	}

	locations := make([]models.DriverLocationResponse, 0, len(driverIDs))
	for _, driverID := range driverIDs {
		if val, found := LocationCache.Get(locationKey(driverID)); found {
			loc := val.(models.CachedDriverLocation)
			updatedAt := loc.UpdatedAt
			locations = append(locations, models.DriverLocationResponse{
				DriverID: loc.DriverID, Status: "online",
				Lat: loc.Lat, Lng: loc.Lng, Heading: loc.Heading, Speed: loc.Speed,
				UpdatedAt: &updatedAt,
			})
		} else if driver, ok := driverMap[driverID]; ok {
			resp := models.DriverLocationResponse{DriverID: driver.ID, Status: "offline"}
			if driver.LastSeenAt != nil {
				resp.Lat = driver.LastLat
				resp.Lng = driver.LastLng
				resp.Heading = driver.LastHeading
				resp.Speed = driver.LastSpeed
				resp.UpdatedAt = driver.LastSeenAt
			}
			locations = append(locations, resp)
		}
	}

	c.JSON(http.StatusOK, gin.H{"locations": locations, "count": len(locations)})
}
