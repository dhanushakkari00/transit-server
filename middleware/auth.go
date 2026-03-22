package middleware

import (
	"net/http"
	"strings"

	"transit-server/models"
	"transit-server/utils"

	"github.com/gin-gonic/gin"
)

// AuthRequired is a Gin middleware that validates JWT access tokens.
// It extracts the Bearer token from the Authorization header,
// validates it, and sets the user claims in the Gin context.
func AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, models.ErrorResponse{
				Error: "Authorization header is required",
			})
			c.Abort()
			return
		}

		// Expect "Bearer <token>"
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			c.JSON(http.StatusUnauthorized, models.ErrorResponse{
				Error: "Authorization header must be in the format: Bearer <token>",
			})
			c.Abort()
			return
		}

		tokenString := parts[1]

		claims, err := utils.ValidateToken(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, models.ErrorResponse{
				Error: "Invalid or expired token",
			})
			c.Abort()
			return
		}

		// Only allow access tokens (not refresh tokens) for API access
		if claims.Type != "access" {
			c.JSON(http.StatusUnauthorized, models.ErrorResponse{
				Error: "Invalid token type",
			})
			c.Abort()
			return
		}

		// Store claims and raw token in context for downstream handlers
		c.Set("userID", claims.UserID)
		c.Set("userEmail", claims.Email)
		c.Set("token", tokenString)

		c.Next()
	}
}
