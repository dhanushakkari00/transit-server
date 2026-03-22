package models

import "time"

// CachedDriverLocation is the struct stored in the in-memory cache
// for real-time driver location tracking. Used by both handlers and gtfsrt packages.
type CachedDriverLocation struct {
	DriverID  uint      `json:"driver_id"`
	Lat       float64   `json:"lat"`
	Lng       float64   `json:"lng"`
	Heading   float64   `json:"heading"`
	Speed     float64   `json:"speed"`
	UpdatedAt time.Time `json:"updated_at"`
}
