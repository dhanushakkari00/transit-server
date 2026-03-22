package models

import (
	"time"
)

// DriverAggregatorMapping links a driver to an aggregator.
type DriverAggregatorMapping struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	DriverID     uint      `gorm:"uniqueIndex:idx_mapping_driver_aggregator;not null" json:"driver_id"`
	Driver       Driver    `gorm:"foreignKey:DriverID;constraint:OnDelete:CASCADE" json:"-"`
	AggregatorID uint      `gorm:"uniqueIndex:idx_mapping_driver_aggregator;not null" json:"aggregator_id"`
	Aggregator   Aggregator `gorm:"foreignKey:AggregatorID;constraint:OnDelete:CASCADE" json:"-"`
	Status       string    `gorm:"index:idx_mapping_status;size:20;default:'active'" json:"status"`
	MappedAt     time.Time `json:"mapped_at"`
}

// TableName overrides the default table name.
func (DriverAggregatorMapping) TableName() string {
	return "driver_aggregator_mappings"
}

// Mapping status constants.
const (
	MappingStatusActive   = "active"
	MappingStatusInactive = "inactive"
)

// MappingResponse is the public representation of a driver-aggregator mapping.
type MappingResponse struct {
	ID           uint      `json:"id"`
	DriverID     uint      `json:"driver_id"`
	AggregatorID uint      `json:"aggregator_id"`
	Status       string    `json:"status"`
	MappedAt     time.Time `json:"mapped_at"`
}

// ToResponse converts a mapping to its response DTO.
func (m *DriverAggregatorMapping) ToResponse() MappingResponse {
	return MappingResponse{
		ID:           m.ID,
		DriverID:     m.DriverID,
		AggregatorID: m.AggregatorID,
		Status:       m.Status,
		MappedAt:     m.MappedAt,
	}
}
