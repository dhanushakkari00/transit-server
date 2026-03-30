package ws

import (
	"log"
	"net/http"
	"strings"

	"transit-server/database"
	"transit-server/models"
	"transit-server/utils"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// Allow all origins in dev — tighten in production
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// Handler holds the hub reference and handles WebSocket upgrades.
type Handler struct {
	hub *Hub
}

// NewHandler creates a new WebSocket handler.
func NewHandler(hub *Hub) *Handler {
	return &Handler{hub: hub}
}

// HandleSubscribe upgrades an HTTP request to a WebSocket connection.
// Auth is done via query params since WebSocket can't send custom headers.
//
// GET /api/v1/aggregator/subscribe?token=<jwt>&api_key=<key>
//
// Flow:
//  1. Extract token + api_key from query params
//  2. Validate JWT → userID, role
//  3. Validate API key → aggregatorID
//  4. Verify role is aggregator or admin
//  5. Upgrade to WebSocket
//  6. Register client in Hub
//  7. Start read/write pumps
func (h *Handler) HandleSubscribe(c *gin.Context) {
	// 1. Extract auth from query params
	token := c.Query("token")
	apiKey := c.Query("api_key")

	if token == "" || apiKey == "" {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error: "Both 'token' and 'api_key' query parameters are required",
		})
		return
	}

	// Remove "Bearer " prefix if present
	token = strings.TrimPrefix(token, "Bearer ")

	// 2. Validate JWT
	claims, err := utils.ValidateToken(token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error: "Invalid or expired token",
		})
		return
	}

	// 3. Check role — only aggregators and admins can subscribe
	if claims.Role != string(models.RoleAggregator) && claims.Role != string(models.RoleAdmin) {
		c.JSON(http.StatusForbidden, models.ErrorResponse{
			Error: "Only aggregators can subscribe to live updates",
		})
		return
	}

	// 4. Validate API key → get aggregator
	var aggregator models.Aggregator
	if database.DB.Where("api_key = ?", apiKey).First(&aggregator).Error != nil {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error: "Invalid API key",
		})
		return
	}

	// 5. Verify the JWT user matches the API key's aggregator
	if aggregator.UserID != claims.UserID {
		c.JSON(http.StatusForbidden, models.ErrorResponse{
			Error: "Token does not match API key owner",
		})
		return
	}

	// 6. Upgrade to WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("[WS Handler] Upgrade failed: %v", err)
		return
	}

	log.Printf("[WS Handler] Aggregator %d (%s) subscribed to live updates",
		aggregator.ID, aggregator.CompanyName)

	// 7. Create client and register
	client := NewClient(h.hub, conn, aggregator.ID)
	h.hub.Register(aggregator.ID, client)

	// Start pumps in goroutines
	go client.WritePump()
	go client.ReadPump()
}
