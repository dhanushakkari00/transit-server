package handlers

import (
	"log"
	"net/http"
	"time"

	"transit-server/database"
	"transit-server/models"
	"transit-server/utils"

	"github.com/gin-gonic/gin"
)

// DriverRegister creates a new driver account (user + driver profile).
// POST /api/v1/driver/register
func DriverRegister(c *gin.Context) {
	var req models.DriverRegisterRequest

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

	// Check if email already exists
	var existingUser models.User
	if database.DB.Where("email = ?", req.Email).First(&existingUser).Error == nil {
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

	// Create user with driver role
	user := models.User{
		Email:        req.Email,
		PasswordHash: hashedPassword,
		FirstName:    req.FirstName,
		LastName:     req.LastName,
		Role:         models.RoleDriver,
		IsActive:     true,
	}

	tx := database.DB.Begin()

	if err := tx.Create(&user).Error; err != nil {
		tx.Rollback()
		log.Printf("Error creating user: %v", err)
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "Failed to create account",
		})
		return
	}

	// Create driver profile
	driver := models.Driver{
		UserID:        user.ID,
		LicenseNumber: req.LicenseNumber,
		Phone:         req.Phone,
		VehicleNumber: req.VehicleNumber,
		VehicleType:   req.VehicleType,
		IsAvailable:   true,
	}

	if err := tx.Create(&driver).Error; err != nil {
		tx.Rollback()
		log.Printf("Error creating driver profile: %v", err)
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "Failed to create driver profile",
		})
		return
	}

	tx.Commit()

	c.JSON(http.StatusCreated, models.MessageResponse{
		Message: "Driver account created successfully",
	})
}

// DriverLogin authenticates a driver.
// POST /api/v1/driver/login
func DriverLogin(c *gin.Context) {
	loginUser(c, models.RoleDriver)
}

// DriverMe returns the current driver's profile.
// GET /api/v1/driver/me
func DriverMe(c *gin.Context) {
	userID, _ := c.Get("userID")

	var user models.User
	if database.DB.First(&user, userID).Error != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{
			Error: "User not found",
		})
		return
	}

	var driver models.Driver
	if database.DB.Where("user_id = ?", userID).First(&driver).Error != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{
			Error: "Driver profile not found",
		})
		return
	}

	c.JSON(http.StatusOK, driver.ToResponse(user))
}

// JoinAggregator maps the current driver to an aggregator using an invite code.
// POST /api/v1/driver/join
func JoinAggregator(c *gin.Context) {
	var req models.JoinAggregatorRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusUnprocessableEntity, models.ErrorResponse{
			Error: "Validation failed",
			Details: map[string]string{
				"message": err.Error(),
			},
		})
		return
	}

	userID, _ := c.Get("userID")

	// Get driver profile for the current user
	var driver models.Driver
	if database.DB.Where("user_id = ?", userID).First(&driver).Error != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{
			Error: "Driver profile not found",
		})
		return
	}

	// Find aggregator by invite code
	var aggregator models.Aggregator
	if database.DB.Where("invite_code = ?", req.InviteCode).First(&aggregator).Error != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{
			Error: "Invalid invite code",
		})
		return
	}

	// Check if already mapped to this aggregator
	var existingMapping models.DriverAggregatorMapping
	result := database.DB.Where("driver_id = ? AND aggregator_id = ?", driver.ID, aggregator.ID).First(&existingMapping)
	if result.Error == nil {
		if existingMapping.Status == models.MappingStatusActive {
			c.JSON(http.StatusConflict, models.ErrorResponse{
				Error: "You are already mapped to this aggregator",
			})
			return
		}
		// Reactivate inactive mapping
		database.DB.Model(&existingMapping).Updates(map[string]interface{}{
			"status":    models.MappingStatusActive,
			"mapped_at": time.Now(),
		})
		c.JSON(http.StatusOK, models.MessageResponse{
			Message: "Successfully re-joined aggregator",
		})
		return
	}

	// Create new mapping
	mapping := models.DriverAggregatorMapping{
		DriverID:     driver.ID,
		AggregatorID: aggregator.ID,
		Status:       models.MappingStatusActive,
		MappedAt:     time.Now(),
	}

	if err := database.DB.Create(&mapping).Error; err != nil {
		log.Printf("Error creating mapping: %v", err)
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "Failed to join aggregator",
		})
		return
	}

	c.JSON(http.StatusCreated, models.MessageResponse{
		Message: "Successfully joined aggregator",
	})
}
