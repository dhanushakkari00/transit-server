package models

import (
	"time"
)

// User represents a registered user in the system.
type User struct {
	ID               uint       `gorm:"primaryKey" json:"id"`
	Email            string     `gorm:"uniqueIndex:idx_users_email;size:255;not null" json:"email"`
	PasswordHash     string     `gorm:"not null" json:"-"`
	FirstName        string     `gorm:"index:idx_users_first_name;size:100" json:"first_name"`
	LastName         string     `gorm:"size:100" json:"last_name"`
	Role             string     `gorm:"index:idx_users_role;size:20;not null;default:'driver'" json:"role"`
	IsActive         bool       `gorm:"default:true;index:idx_users_is_active" json:"is_active"`
	ResetToken       string     `gorm:"index:idx_users_reset_token;size:255" json:"-"`
	ResetTokenExpiry *time.Time `json:"-"`
	CreatedAt        time.Time  `gorm:"index:idx_users_created_at" json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

// TableName overrides the default table name.
func (User) TableName() string {
	return "users"
}

// Role constants
const (
	RoleDriver     = "driver"
	RoleAggregator = "aggregator"
	RoleAdmin      = "admin"
)

// --- Request DTOs ---

// LoginRequest is the payload for user login.
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// ForgotPasswordRequest is the payload for forgot password.
type ForgotPasswordRequest struct {
	Email string `json:"email" binding:"required,email"`
}

// ResetPasswordRequest is the payload for resetting password.
type ResetPasswordRequest struct {
	Token       string `json:"token" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=8,max=72"`
}

// RefreshRequest is the payload for refreshing an access token.
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// --- Response DTOs ---

// UserResponse is the public representation of a user.
type UserResponse struct {
	ID        uint      `json:"id"`
	Email     string    `json:"email"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	Role      string    `json:"role"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ToResponse converts a User model to a UserResponse DTO.
func (u *User) ToResponse() UserResponse {
	return UserResponse{
		ID:        u.ID,
		Email:     u.Email,
		FirstName: u.FirstName,
		LastName:  u.LastName,
		Role:      u.Role,
		IsActive:  u.IsActive,
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
	}
}

// AuthResponse is returned after successful login.
type AuthResponse struct {
	AccessToken  string       `json:"access_token"`
	RefreshToken string       `json:"refresh_token"`
	TokenType    string       `json:"token_type"`
	ExpiresIn    int64        `json:"expires_in"`
	User         UserResponse `json:"user"`
}

// MessageResponse is a generic message response.
type MessageResponse struct {
	Message string `json:"message"`
}

// ErrorResponse is a standard error response.
type ErrorResponse struct {
	Error   string            `json:"error"`
	Details map[string]string `json:"details,omitempty"`
}
