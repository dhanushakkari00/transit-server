package gtfsrt

import (
	"log"
	"net/http"
	"strconv"

	"transit-server/models"

	"github.com/gin-gonic/gin"
)

// Handler holds the feed generator and exposes HTTP handlers.
// All endpoints require JWT + API Key authentication — no public feeds.
type Handler struct {
	generator *FeedGenerator
}

// NewHandler creates a new GTFS-RT HTTP handler.
func NewHandler(generator *FeedGenerator) *Handler {
	return &Handler{generator: generator}
}

// getAggregatorID extracts the aggregator ID from the API key middleware context.
func getAggregatorID(c *gin.Context) (uint, bool) {
	agencyID, exists := c.Get("apiKeyAggregatorID")
	if !exists {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error: "API key required",
		})
		return 0, false
	}
	return agencyID.(uint), true
}

// ServeFeedForAgency serves the GTFS-RT protobuf feed for the authenticated aggregator.
// Only includes vehicles mapped to THIS aggregator with active trips.
// GET /api/v1/aggregator/feed/vehicle-positions
func (h *Handler) ServeFeedForAgency(c *gin.Context) {
	agencyID, ok := getAggregatorID(c)
	if !ok {
		return
	}

	data, err := h.generator.GenerateFeedForAgency(agencyID)
	if err != nil {
		log.Printf("Error generating feed for agency %d: %v", agencyID, err)
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "Failed to generate feed",
		})
		return
	}

	c.Data(http.StatusOK, "application/x-protobuf", data)
}

// ServeFeedForVehicle serves the GTFS-RT protobuf feed for a single vehicle.
// Verifies the driver is mapped to the requesting aggregator before returning data.
// GET /api/v1/aggregator/feed/vehicle-positions/:driverId
func (h *Handler) ServeFeedForVehicle(c *gin.Context) {
	agencyID, ok := getAggregatorID(c)
	if !ok {
		return
	}

	driverIDParam := c.Param("driverId")
	driverID, err := strconv.ParseUint(driverIDParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "Invalid driver ID"})
		return
	}

	data, found, err := h.generator.GenerateFeedForVehicle(agencyID, uint(driverID))
	if err != nil {
		log.Printf("Error generating feed for vehicle %d: %v", driverID, err)
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "Failed to generate feed",
		})
		return
	}
	if !found {
		c.JSON(http.StatusNotFound, models.ErrorResponse{
			Error: "Driver not found, not mapped to your account, or has no active trip",
		})
		return
	}

	c.Data(http.StatusOK, "application/x-protobuf", data)
}

// ServeDebugFeedForAgency serves the JSON debug feed for the aggregator.
// GET /api/v1/aggregator/feed/vehicle-positions/debug
func (h *Handler) ServeDebugFeedForAgency(c *gin.Context) {
	agencyID, ok := getAggregatorID(c)
	if !ok {
		return
	}

	response, err := h.generator.GenerateDebugFeedForAgency(agencyID)
	if err != nil {
		log.Printf("Error generating debug feed: %v", err)
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "Failed to generate feed",
		})
		return
	}

	c.JSON(http.StatusOK, response)
}

// ServeDebugFeedForVehicle serves the JSON debug feed for a single vehicle.
// GET /api/v1/aggregator/feed/vehicle-positions/:driverId/debug
func (h *Handler) ServeDebugFeedForVehicle(c *gin.Context) {
	agencyID, ok := getAggregatorID(c)
	if !ok {
		return
	}

	driverIDParam := c.Param("driverId")
	driverID, err := strconv.ParseUint(driverIDParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "Invalid driver ID"})
		return
	}

	response, found, err := h.generator.GenerateDebugFeedForVehicle(agencyID, uint(driverID))
	if err != nil {
		log.Printf("Error generating debug feed for vehicle: %v", err)
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "Failed to generate feed",
		})
		return
	}
	if !found {
		c.JSON(http.StatusNotFound, models.ErrorResponse{
			Error: "Driver not found, not mapped to your account, or has no active trip",
		})
		return
	}

	c.JSON(http.StatusOK, response)
}
