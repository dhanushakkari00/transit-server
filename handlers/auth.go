package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"log"
	"net/http"
	"time"

	"transit-server/database"
	"transit-server/models"
	"transit-server/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Register handles generic registration (kept for backward compat but not routed).
// Role-specific registration is in driver.go and aggregator.go.

// Login authenticates a user by role and returns JWT tokens.
// Used internally by role-specific login handlers.
func loginUser(c *gin.Context, allowedRoles ...string) {
	var req models.LoginRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusUnprocessableEntity, models.ErrorResponse{
			Error: "Validation failed",
			Details: map[string]string{
				"message": err.Error(),
			},
		})
		return
	}

	// Find user by email
	var user models.User
	result := database.DB.Where("email = ?", req.Email).First(&user)
	if result.Error != nil {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error: "Invalid email or password",
		})
		return
	}

	// Check if account is active
	if !user.IsActive {
		c.JSON(http.StatusForbidden, models.ErrorResponse{
			Error: "Account is deactivated",
		})
		return
	}

	// Check role matches
	roleAllowed := false
	for _, r := range allowedRoles {
		if user.Role == r {
			roleAllowed = true
			break
		}
	}
	if !roleAllowed {
		c.JSON(http.StatusForbidden, models.ErrorResponse{
			Error: "This login endpoint is not for your account type",
		})
		return
	}

	// Verify password
	if !utils.CheckPassword(req.Password, user.PasswordHash) {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error: "Invalid email or password",
		})
		return
	}

	// Generate tokens
	accessToken, err := utils.GenerateAccessToken(user.ID, user.Email, user.Role)
	if err != nil {
		log.Printf("Error generating access token: %v", err)
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "Internal server error",
		})
		return
	}

	refreshToken, err := utils.GenerateRefreshToken(user.ID, user.Email, user.Role)
	if err != nil {
		log.Printf("Error generating refresh token: %v", err)
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "Internal server error",
		})
		return
	}

	c.JSON(http.StatusOK, models.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    900,
		User:         user.ToResponse(),
	})
}

// Logout revokes the current token by blacklisting it.
// POST /api/v1/auth/logout
func Logout(c *gin.Context) {
	token, exists := c.Get("token")
	if !exists {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "Internal server error",
		})
		return
	}

	utils.BlacklistToken(token.(string))

	c.JSON(http.StatusOK, models.MessageResponse{
		Message: "Successfully logged out",
	})
}

// RefreshToken issues a new access token using a valid refresh token.
// POST /api/v1/auth/refresh
func RefreshToken(c *gin.Context) {
	var req models.RefreshRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusUnprocessableEntity, models.ErrorResponse{
			Error: "Validation failed",
			Details: map[string]string{
				"message": err.Error(),
			},
		})
		return
	}

	claims, err := utils.ValidateToken(req.RefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error: "Invalid or expired refresh token",
		})
		return
	}

	if claims.Type != "refresh" {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error: "Invalid token type, expected refresh token",
		})
		return
	}

	var user models.User
	result := database.DB.First(&user, claims.UserID)
	if result.Error != nil || !user.IsActive {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error: "Account not found or deactivated",
		})
		return
	}

	utils.BlacklistToken(req.RefreshToken)

	accessToken, err := utils.GenerateAccessToken(user.ID, user.Email, user.Role)
	if err != nil {
		log.Printf("Error generating access token: %v", err)
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "Internal server error",
		})
		return
	}

	refreshToken, err := utils.GenerateRefreshToken(user.ID, user.Email, user.Role)
	if err != nil {
		log.Printf("Error generating refresh token: %v", err)
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "Internal server error",
		})
		return
	}

	c.JSON(http.StatusOK, models.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    900,
		User:         user.ToResponse(),
	})
}

// ForgotPassword generates a password reset token.
// POST /api/v1/auth/forgot-password
func ForgotPassword(c *gin.Context) {
	var req models.ForgotPasswordRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusUnprocessableEntity, models.ErrorResponse{
			Error: "Validation failed",
			Details: map[string]string{
				"message": err.Error(),
			},
		})
		return
	}

	successResponse := models.MessageResponse{
		Message: "If an account with that email exists, a password reset link has been sent",
	}

	var user models.User
	result := database.DB.Where("email = ?", req.Email).First(&user)
	if result.Error != nil {
		c.JSON(http.StatusOK, successResponse)
		return
	}

	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		log.Printf("Error generating reset token: %v", err)
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "Internal server error",
		})
		return
	}
	resetToken := hex.EncodeToString(tokenBytes)

	expiry := time.Now().Add(1 * time.Hour)

	database.DB.Model(&user).Updates(models.User{
		ResetToken:       resetToken,
		ResetTokenExpiry: &expiry,
	})

	log.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	log.Printf("PASSWORD RESET TOKEN for %s", user.Email)
	log.Printf("Token: %s", resetToken)
	log.Printf("Expires: %s", expiry.Format(time.RFC3339))
	log.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	c.JSON(http.StatusOK, successResponse)
}

// ResetPassword resets a user's password using a valid reset token.
// POST /api/v1/auth/reset-password
func ResetPassword(c *gin.Context) {
	var req models.ResetPasswordRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusUnprocessableEntity, models.ErrorResponse{
			Error: "Validation failed",
			Details: map[string]string{
				"message": err.Error(),
			},
		})
		return
	}

	if err := utils.ValidatePasswordStrength(req.NewPassword); err != nil {
		c.JSON(http.StatusUnprocessableEntity, models.ErrorResponse{
			Error: err.Error(),
		})
		return
	}

	var user models.User
	result := database.DB.Where("reset_token = ?", req.Token).First(&user)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			c.JSON(http.StatusBadRequest, models.ErrorResponse{
				Error: "Invalid or expired reset token",
			})
			return
		}
		log.Printf("Error finding user by reset token: %v", result.Error)
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "Internal server error",
		})
		return
	}

	if user.ResetTokenExpiry == nil || time.Now().After(*user.ResetTokenExpiry) {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error: "Reset token has expired",
		})
		return
	}

	hashedPassword, err := utils.HashPassword(req.NewPassword)
	if err != nil {
		log.Printf("Error hashing password: %v", err)
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "Internal server error",
		})
		return
	}

	database.DB.Model(&user).Updates(map[string]interface{}{
		"password_hash":      hashedPassword,
		"reset_token":        "",
		"reset_token_expiry": nil,
	})

	c.JSON(http.StatusOK, models.MessageResponse{
		Message: "Password has been reset successfully",
	})
}
