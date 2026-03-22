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

// Register creates a new user account.
// POST /api/v1/auth/register
func Register(c *gin.Context) {
	var req models.RegisterRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusUnprocessableEntity, models.ErrorResponse{
			Error: "Validation failed",
			Details: map[string]string{
				"message": err.Error(),
			},
		})
		return
	}

	// Validate password strength
	if err := utils.ValidatePasswordStrength(req.Password); err != nil {
		c.JSON(http.StatusUnprocessableEntity, models.ErrorResponse{
			Error: err.Error(),
		})
		return
	}

	// Check if user already exists
	var existingUser models.User
	result := database.DB.Where("email = ?", req.Email).First(&existingUser)
	if result.Error == nil {
		c.JSON(http.StatusConflict, models.ErrorResponse{
			Error: "An account with this email already exists",
		})
		return
	}

	// Hash password
	hashedPassword, err := utils.HashPassword(req.Password)
	if err != nil {
		log.Printf("Error hashing password: %v", err)
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "Internal server error",
		})
		return
	}

	// Create user
	user := models.User{
		Email:        req.Email,
		PasswordHash: hashedPassword,
		FirstName:    req.FirstName,
		LastName:     req.LastName,
		IsActive:     true,
	}

	if err := database.DB.Create(&user).Error; err != nil {
		log.Printf("Error creating user: %v", err)
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "Failed to create account",
		})
		return
	}

	c.JSON(http.StatusCreated, models.MessageResponse{
		Message: "Account created successfully",
	})
}

// Login authenticates a user and returns JWT tokens.
// POST /api/v1/auth/login
func Login(c *gin.Context) {
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
		// Use generic message to prevent email enumeration
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

	// Verify password
	if !utils.CheckPassword(req.Password, user.PasswordHash) {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error: "Invalid email or password",
		})
		return
	}

	// Generate tokens
	accessToken, err := utils.GenerateAccessToken(user.ID, user.Email)
	if err != nil {
		log.Printf("Error generating access token: %v", err)
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "Internal server error",
		})
		return
	}

	refreshToken, err := utils.GenerateRefreshToken(user.ID, user.Email)
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
		ExpiresIn:    900, // 15 minutes in seconds
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

	// Validate the refresh token
	claims, err := utils.ValidateToken(req.RefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error: "Invalid or expired refresh token",
		})
		return
	}

	// Ensure it's actually a refresh token
	if claims.Type != "refresh" {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error: "Invalid token type, expected refresh token",
		})
		return
	}

	// Verify user still exists and is active
	var user models.User
	result := database.DB.First(&user, claims.UserID)
	if result.Error != nil || !user.IsActive {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error: "Account not found or deactivated",
		})
		return
	}

	// Blacklist the old refresh token (rotation for security)
	utils.BlacklistToken(req.RefreshToken)

	// Generate new token pair
	accessToken, err := utils.GenerateAccessToken(user.ID, user.Email)
	if err != nil {
		log.Printf("Error generating access token: %v", err)
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "Internal server error",
		})
		return
	}

	refreshToken, err := utils.GenerateRefreshToken(user.ID, user.Email)
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

	// Always return success to prevent email enumeration
	successResponse := models.MessageResponse{
		Message: "If an account with that email exists, a password reset link has been sent",
	}

	// Find user
	var user models.User
	result := database.DB.Where("email = ?", req.Email).First(&user)
	if result.Error != nil {
		// User not found — still return success to prevent enumeration
		c.JSON(http.StatusOK, successResponse)
		return
	}

	// Generate a cryptographically secure reset token
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		log.Printf("Error generating reset token: %v", err)
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "Internal server error",
		})
		return
	}
	resetToken := hex.EncodeToString(tokenBytes)

	// Set token expiry to 1 hour from now
	expiry := time.Now().Add(1 * time.Hour)

	// Save reset token to database
	database.DB.Model(&user).Updates(models.User{
		ResetToken:       resetToken,
		ResetTokenExpiry: &expiry,
	})

	// In production, send this via email. For now, log it to console.
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

	// Validate password strength
	if err := utils.ValidatePasswordStrength(req.NewPassword); err != nil {
		c.JSON(http.StatusUnprocessableEntity, models.ErrorResponse{
			Error: err.Error(),
		})
		return
	}

	// Find user by reset token
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

	// Check token expiry
	if user.ResetTokenExpiry == nil || time.Now().After(*user.ResetTokenExpiry) {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error: "Reset token has expired",
		})
		return
	}

	// Hash the new password
	hashedPassword, err := utils.HashPassword(req.NewPassword)
	if err != nil {
		log.Printf("Error hashing password: %v", err)
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "Internal server error",
		})
		return
	}

	// Update password and clear reset token
	database.DB.Model(&user).Updates(map[string]interface{}{
		"password_hash":      hashedPassword,
		"reset_token":        "",
		"reset_token_expiry": nil,
	})

	c.JSON(http.StatusOK, models.MessageResponse{
		Message: "Password has been reset successfully",
	})
}

// Me returns the currently authenticated user's profile.
// GET /api/v1/auth/me
func Me(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "Internal server error",
		})
		return
	}

	var user models.User
	result := database.DB.First(&user, userID)
	if result.Error != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{
			Error: "User not found",
		})
		return
	}

	c.JSON(http.StatusOK, user.ToResponse())
}
