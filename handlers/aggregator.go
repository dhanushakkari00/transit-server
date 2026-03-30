package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"log"
	"math/big"
	"net/http"
	"strconv"
	"strings"

	"transit-server/database"
	"transit-server/models"
	"transit-server/utils"

	"github.com/gin-gonic/gin"
)

const inviteCodeChars = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
const inviteCodeLength = 5
const apiKeyByteLength = 32

// generateInviteCode creates a unique random 5-character alphanumeric code.
func generateInviteCode() (string, error) {
	for attempts := 0; attempts < 10; attempts++ {
		var sb strings.Builder
		for i := 0; i < inviteCodeLength; i++ {
			idx, err := rand.Int(rand.Reader, big.NewInt(int64(len(inviteCodeChars))))
			if err != nil {
				return "", err
			}
			sb.WriteByte(inviteCodeChars[idx.Int64()])
		}
		code := sb.String()

		// Verify uniqueness
		var count int64
		database.DB.Model(&models.Aggregator{}).Where("invite_code = ?", code).Count(&count)
		if count == 0 {
			return code, nil
		}
	}
	return "", nil
}

// generateAPIKey creates a unique random 64-character hex API key.
func generateAPIKey() (string, error) {
	for attempts := 0; attempts < 10; attempts++ {
		apiKeyBytes := make([]byte, apiKeyByteLength)
		if _, err := rand.Read(apiKeyBytes); err != nil {
			return "", err
		}

		apiKey := hex.EncodeToString(apiKeyBytes)

		var count int64
		database.DB.Model(&models.Aggregator{}).Where("api_key = ?", apiKey).Count(&count)
		if count == 0 {
			return apiKey, nil
		}
	}

	return "", nil
}

func currentAggregator(c *gin.Context) (uint, models.Aggregator, bool) {
	userIDValue, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error: "Authentication required",
		})
		return 0, models.Aggregator{}, false
	}

	userID, ok := userIDValue.(uint)
	if !ok {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "Invalid authentication context",
		})
		return 0, models.Aggregator{}, false
	}

	var aggregator models.Aggregator
	if database.DB.Where("user_id = ?", userID).First(&aggregator).Error != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{
			Error: "Aggregator profile not found",
		})
		return 0, models.Aggregator{}, false
	}

	return userID, aggregator, true
}

// AggregatorRegister creates a new aggregator account (user + aggregator profile + invite code).
// POST /api/v1/aggregator/register
func AggregatorRegister(c *gin.Context) {
	var req models.AggregatorRegisterRequest

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

	// Generate unique invite code
	inviteCode, err := generateInviteCode()
	if err != nil || inviteCode == "" {
		log.Printf("Error generating invite code: %v", err)
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "Failed to generate invite code",
		})
		return
	}

	apiKey, err := generateAPIKey()
	if err != nil || apiKey == "" {
		log.Printf("Error generating API key: %v", err)
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "Failed to generate API key",
		})
		return
	}

	// Create user with aggregator role
	user := models.User{
		Email:        req.Email,
		PasswordHash: hashedPassword,
		FirstName:    req.FirstName,
		LastName:     req.LastName,
		Role:         models.RoleAggregator,
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

	// Create aggregator profile
	aggregator := models.Aggregator{
		UserID:      user.ID,
		CompanyName: req.CompanyName,
		Phone:       req.Phone,
		InviteCode:  inviteCode,
		APIKey:      apiKey,
	}

	if err := tx.Create(&aggregator).Error; err != nil {
		tx.Rollback()
		log.Printf("Error creating aggregator profile: %v", err)
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "Failed to create aggregator profile",
		})
		return
	}

	tx.Commit()

	c.JSON(http.StatusCreated, gin.H{
		"message":     "Aggregator account created successfully",
		"invite_code": inviteCode,
		"api_key":     apiKey,
	})
}

// AggregatorLogin authenticates an aggregator or admin.
// POST /api/v1/aggregator/login
func AggregatorLogin(c *gin.Context) {
	loginUser(c, models.RoleAggregator, models.RoleAdmin)
}

// AggregatorMe returns the current aggregator's profile.
// GET /api/v1/aggregator/me
func AggregatorMe(c *gin.Context) {
	userID, aggregator, ok := currentAggregator(c)
	if !ok {
		return
	}

	var user models.User
	if database.DB.First(&user, userID).Error != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{
			Error: "User not found",
		})
		return
	}

	c.JSON(http.StatusOK, aggregator.ToResponse(user))
}

// AggregatorAPIKey returns the current aggregator API key and invite code.
// GET /api/v1/aggregator/api-key
func AggregatorAPIKey(c *gin.Context) {
	_, aggregator, ok := currentAggregator(c)
	if !ok {
		return
	}

	c.JSON(http.StatusOK, models.AggregatorAPIKeyResponse{
		APIKey:     aggregator.APIKey,
		InviteCode: aggregator.InviteCode,
		UpdatedAt:  aggregator.UpdatedAt,
	})
}

// RotateAggregatorAPIKey regenerates the current aggregator API key.
// PUT /api/v1/aggregator/api-key
func RotateAggregatorAPIKey(c *gin.Context) {
	_, aggregator, ok := currentAggregator(c)
	if !ok {
		return
	}

	apiKey, err := generateAPIKey()
	if err != nil || apiKey == "" {
		log.Printf("Error rotating API key: %v", err)
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "Failed to rotate API key",
		})
		return
	}

	if err := database.DB.Model(&aggregator).Update("api_key", apiKey).Error; err != nil {
		log.Printf("Error updating API key: %v", err)
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "Failed to update API key",
		})
		return
	}

	if err := database.DB.First(&aggregator, aggregator.ID).Error; err != nil {
		log.Printf("Error loading rotated API key: %v", err)
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "API key updated but failed to reload aggregator profile",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":     "API key rotated successfully",
		"api_key":     aggregator.APIKey,
		"invite_code": aggregator.InviteCode,
		"updated_at":  aggregator.UpdatedAt,
	})
}

// ListDrivers returns all drivers mapped to the current aggregator.
// GET /api/v1/aggregator/drivers
func ListDrivers(c *gin.Context) {
	userID, _ := c.Get("userID")

	// Get aggregator profile
	var aggregator models.Aggregator
	if database.DB.Where("user_id = ?", userID).First(&aggregator).Error != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{
			Error: "Aggregator profile not found",
		})
		return
	}

	// Get active mappings
	var mappings []models.DriverAggregatorMapping
	database.DB.Where("aggregator_id = ? AND status = ?", aggregator.ID, models.MappingStatusActive).Find(&mappings)

	// Collect driver IDs
	driverIDs := make([]uint, len(mappings))
	for i, m := range mappings {
		driverIDs[i] = m.DriverID
	}

	if len(driverIDs) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"drivers": []interface{}{},
			"count":   0,
		})
		return
	}

	// Fetch drivers with their users
	var drivers []models.Driver
	database.DB.Where("id IN ?", driverIDs).Find(&drivers)

	// Fetch associated users
	userIDs := make([]uint, len(drivers))
	for i, d := range drivers {
		userIDs[i] = d.UserID
	}

	var users []models.User
	database.DB.Where("id IN ?", userIDs).Find(&users)

	// Build user lookup map
	userMap := make(map[uint]models.User)
	for _, u := range users {
		userMap[u.ID] = u
	}

	// Build response
	driverResponses := make([]models.DriverResponse, 0, len(drivers))
	for _, d := range drivers {
		if user, ok := userMap[d.UserID]; ok {
			driverResponses = append(driverResponses, d.ToResponse(user))
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"drivers": driverResponses,
		"count":   len(driverResponses),
	})
}

// GetDriver returns a specific driver mapped to the current aggregator.
// GET /api/v1/aggregator/drivers/:id
func GetDriver(c *gin.Context) {
	userID, _ := c.Get("userID")

	// Parse driver ID from URL
	driverIDParam := c.Param("id")
	driverID, err := strconv.ParseUint(driverIDParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error: "Invalid driver ID",
		})
		return
	}

	// Get aggregator profile
	var aggregator models.Aggregator
	if database.DB.Where("user_id = ?", userID).First(&aggregator).Error != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{
			Error: "Aggregator profile not found",
		})
		return
	}

	// Verify this driver is actively mapped to this aggregator
	var mapping models.DriverAggregatorMapping
	result := database.DB.Where("driver_id = ? AND aggregator_id = ? AND status = ?",
		uint(driverID), aggregator.ID, models.MappingStatusActive).First(&mapping)
	if result.Error != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{
			Error: "Driver not found or not mapped to your account",
		})
		return
	}

	// Fetch driver and user
	var driver models.Driver
	if database.DB.First(&driver, uint(driverID)).Error != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{
			Error: "Driver not found",
		})
		return
	}

	var user models.User
	if database.DB.First(&user, driver.UserID).Error != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{
			Error: "User not found",
		})
		return
	}

	c.JSON(http.StatusOK, driver.ToResponse(user))
}
