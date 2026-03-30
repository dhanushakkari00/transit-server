package middleware

import (
	"net/http"

	"transit-server/database"
	"transit-server/models"

	"github.com/gin-gonic/gin"
)

// APIKeyRequired validates the X-API-Key header against aggregator API keys.
// Sets the aggregator ID in context on success.
func APIKeyRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		apiKey := c.GetHeader("X-API-Key")
		if apiKey == "" {
			c.JSON(http.StatusUnauthorized, models.ErrorResponse{
				Error: "X-API-Key header is required",
			})
			c.Abort()
			return
		}

		// Look up aggregator by API key
		var aggregator models.Aggregator
		result := database.DB.Where("api_key = ?", apiKey).First(&aggregator)
		if result.Error != nil {
			c.JSON(http.StatusUnauthorized, models.ErrorResponse{
				Error: "Invalid API key",
			})
			c.Abort()
			return
		}

		if userIDValue, exists := c.Get("userID"); exists {
			userID, ok := userIDValue.(uint)
			if !ok {
				c.JSON(http.StatusInternalServerError, models.ErrorResponse{
					Error: "Invalid authentication context",
				})
				c.Abort()
				return
			}

			if aggregator.UserID != userID {
				c.JSON(http.StatusForbidden, models.ErrorResponse{
					Error: "API key does not belong to authenticated user",
				})
				c.Abort()
				return
			}
		}

		// Store aggregator info in context
		c.Set("apiKeyAggregatorID", aggregator.ID)
		c.Set("apiKeyUserID", aggregator.UserID)

		c.Next()
	}
}
