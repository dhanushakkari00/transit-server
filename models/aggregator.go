package models

import (
	"time"
)

// Aggregator holds aggregator-specific profile details, linked 1:1 to a User.
type Aggregator struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	UserID      uint      `gorm:"uniqueIndex:idx_aggregators_user_id;not null" json:"user_id"`
	User        User      `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"-"`
	CompanyName string    `gorm:"size:200" json:"company_name"`
	Phone       string    `gorm:"index:idx_aggregators_phone;size:20" json:"phone"`
	InviteCode  string    `gorm:"uniqueIndex:idx_aggregators_invite_code;size:5;not null" json:"invite_code"`
	APIKey      string    `gorm:"uniqueIndex:idx_aggregators_api_key;size:64;not null" json:"-"`
	CreatedAt   time.Time `gorm:"index:idx_aggregators_created_at" json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// TableName overrides the default table name.
func (Aggregator) TableName() string {
	return "aggregators"
}

// --- Request DTOs ---

// AggregatorRegisterRequest is the payload for aggregator registration.
type AggregatorRegisterRequest struct {
	Email       string `json:"email" binding:"required,email,max=255"`
	Password    string `json:"password" binding:"required,min=8,max=72"`
	FirstName   string `json:"first_name" binding:"required,max=100"`
	LastName    string `json:"last_name" binding:"max=100"`
	CompanyName string `json:"company_name" binding:"required,max=200"`
	Phone       string `json:"phone" binding:"required,max=20"`
}

// --- Response DTOs ---

// AggregatorResponse is the public representation of an aggregator profile.
type AggregatorResponse struct {
	ID          uint         `json:"id"`
	User        UserResponse `json:"user"`
	CompanyName string       `json:"company_name"`
	Phone       string       `json:"phone"`
	InviteCode  string       `json:"invite_code"`
	APIKey      string       `json:"api_key"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
}

// AggregatorAPIKeyResponse is the API key payload for self-service key retrieval.
type AggregatorAPIKeyResponse struct {
	APIKey     string    `json:"api_key"`
	InviteCode string    `json:"invite_code"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// ToResponse converts an Aggregator model to an AggregatorResponse DTO.
func (a *Aggregator) ToResponse(user User) AggregatorResponse {
	return AggregatorResponse{
		ID:          a.ID,
		User:        user.ToResponse(),
		CompanyName: a.CompanyName,
		Phone:       a.Phone,
		InviteCode:  a.InviteCode,
		APIKey:      a.APIKey,
		CreatedAt:   a.CreatedAt,
		UpdatedAt:   a.UpdatedAt,
	}
}
