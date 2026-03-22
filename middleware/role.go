package middleware

import (
	"net/http"

	"transit-server/models"

	"github.com/gin-gonic/gin"
)

// RequireRole returns a middleware that restricts access to users with one of the specified roles.
// Must be used AFTER AuthRequired() in the middleware chain.
func RequireRole(allowedRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("userRole")
		if !exists {
			c.JSON(http.StatusUnauthorized, models.ErrorResponse{
				Error: "Authentication required",
			})
			c.Abort()
			return
		}

		userRole := role.(string)
		for _, allowed := range allowedRoles {
			if userRole == allowed {
				c.Next()
				return
			}
		}

		c.JSON(http.StatusForbidden, models.ErrorResponse{
			Error: "You do not have permission to access this resource",
		})
		c.Abort()
	}
}
