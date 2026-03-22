package models

import (
	"time"
)

// Driver holds driver-specific profile details, linked 1:1 to a User.
type Driver struct {
	ID            uint       `gorm:"primaryKey" json:"id"`
	UserID        uint       `gorm:"uniqueIndex:idx_drivers_user_id;not null" json:"user_id"`
	User          User       `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"-"`
	LicenseNumber string     `gorm:"index:idx_drivers_license;size:50" json:"license_number"`
	Phone         string     `gorm:"index:idx_drivers_phone;size:20" json:"phone"`
	VehicleNumber string     `gorm:"size:20" json:"vehicle_number"`
	VehicleType   string     `gorm:"size:50" json:"vehicle_type"`
	IsAvailable   bool       `gorm:"default:true" json:"is_available"`
	LastLat       float64    `gorm:"type:real;default:0" json:"last_lat"`
	LastLng       float64    `gorm:"type:real;default:0" json:"last_lng"`
	LastHeading   float64    `gorm:"type:real;default:0" json:"last_heading"`
	LastSpeed     float64    `gorm:"type:real;default:0" json:"last_speed"`
	LastSeenAt    *time.Time `gorm:"index:idx_drivers_last_seen" json:"last_seen_at"`
	CreatedAt     time.Time  `gorm:"index:idx_drivers_created_at" json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// TableName overrides the default table name.
func (Driver) TableName() string {
	return "drivers"
}

// --- Request DTOs ---

// DriverRegisterRequest is the payload for driver registration.
type DriverRegisterRequest struct {
	Email         string `json:"email" binding:"required,email,max=255"`
	Password      string `json:"password" binding:"required,min=8,max=72"`
	FirstName     string `json:"first_name" binding:"required,max=100"`
	LastName      string `json:"last_name" binding:"max=100"`
	LicenseNumber string `json:"license_number" binding:"required,max=50"`
	Phone         string `json:"phone" binding:"required,max=20"`
	VehicleNumber string `json:"vehicle_number" binding:"max=20"`
	VehicleType   string `json:"vehicle_type" binding:"max=50"`
}

// JoinAggregatorRequest is the payload for a driver to join an aggregator.
type JoinAggregatorRequest struct {
	InviteCode string `json:"invite_code" binding:"required,len=5"`
}

// UpdateLocationRequest is the payload for a driver to push GPS location.
type UpdateLocationRequest struct {
	Lat     float64 `json:"lat" binding:"required"`
	Lng     float64 `json:"lng" binding:"required"`
	Heading float64 `json:"heading"`
	Speed   float64 `json:"speed"`
}

// --- Response DTOs ---

// DriverResponse is the public representation of a driver profile.
type DriverResponse struct {
	ID            uint         `json:"id"`
	User          UserResponse `json:"user"`
	LicenseNumber string       `json:"license_number"`
	Phone         string       `json:"phone"`
	VehicleNumber string       `json:"vehicle_number"`
	VehicleType   string       `json:"vehicle_type"`
	IsAvailable   bool         `json:"is_available"`
	CreatedAt     time.Time    `json:"created_at"`
	UpdatedAt     time.Time    `json:"updated_at"`
}

// ToResponse converts a Driver model to a DriverResponse DTO.
func (d *Driver) ToResponse(user User) DriverResponse {
	return DriverResponse{
		ID:            d.ID,
		User:          user.ToResponse(),
		LicenseNumber: d.LicenseNumber,
		Phone:         d.Phone,
		VehicleNumber: d.VehicleNumber,
		VehicleType:   d.VehicleType,
		IsAvailable:   d.IsAvailable,
		CreatedAt:     d.CreatedAt,
		UpdatedAt:     d.UpdatedAt,
	}
}

// DriverLocationResponse is the location response for a driver.
type DriverLocationResponse struct {
	DriverID  uint       `json:"driver_id"`
	Status    string     `json:"status"` // "online" or "offline"
	Lat       float64    `json:"lat"`
	Lng       float64    `json:"lng"`
	Heading   float64    `json:"heading"`
	Speed     float64    `json:"speed"`
	UpdatedAt *time.Time `json:"updated_at"`
}
