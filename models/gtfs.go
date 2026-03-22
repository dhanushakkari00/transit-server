package models

import (
	"time"
)

// Route represents a transit route (e.g., "Bus 101 - Downtown Express").
type Route struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	AgencyID    uint      `gorm:"index:idx_routes_agency;not null" json:"agency_id"`
	GtfsRouteID string    `gorm:"column:gtfs_route_id;uniqueIndex:idx_routes_gtfs_id;size:50;not null" json:"route_id"`
	ShortName   string    `gorm:"size:50" json:"short_name"`
	LongName    string    `gorm:"size:255" json:"long_name"`
	Description string    `gorm:"size:500" json:"description"`
	RouteType   int       `gorm:"default:3" json:"route_type"` // GTFS route_type: 3=Bus
	Color       string    `gorm:"size:6" json:"color"`
	TextColor   string    `gorm:"size:6" json:"text_color"`
	IsActive    bool      `gorm:"default:true;index:idx_routes_active" json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (Route) TableName() string {
	return "routes"
}

// Trip represents a specific journey on a route.
type Trip struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	RouteRefID  uint      `gorm:"column:route_ref_id;index:idx_trips_route;not null" json:"route_id"`
	GtfsTripID  string    `gorm:"column:gtfs_trip_id;uniqueIndex:idx_trips_gtfs_id;size:50;not null" json:"trip_id"`
	Headsign    string    `gorm:"size:255" json:"headsign"`
	DirectionID int       `gorm:"default:0" json:"direction_id"`
	IsActive    bool      `gorm:"default:true" json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (Trip) TableName() string {
	return "trips"
}

// ActiveTrip represents a driver currently operating a trip.
// This is what links a driver to a route/trip for GTFS-RT feed generation.
type ActiveTrip struct {
	ID        uint       `gorm:"primaryKey" json:"id"`
	DriverID  uint       `gorm:"index:idx_active_trips_driver;not null" json:"driver_id"`
	TripRefID uint       `gorm:"column:trip_ref_id;index:idx_active_trips_trip;not null" json:"trip_id"`
	VehicleID string     `gorm:"index:idx_active_trips_vehicle;size:50" json:"vehicle_id"`
	StartedAt time.Time  `json:"started_at"`
	EndedAt   *time.Time `json:"ended_at"`
	IsActive  bool       `gorm:"default:true;index:idx_active_trips_active" json:"is_active"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

func (ActiveTrip) TableName() string {
	return "active_trips"
}

// --- Request DTOs ---

// CreateRouteRequest is the payload for creating a route.
type CreateRouteRequest struct {
	RouteID     string `json:"route_id" binding:"required,max=50"`
	ShortName   string `json:"short_name" binding:"required,max=50"`
	LongName    string `json:"long_name" binding:"max=255"`
	Description string `json:"description" binding:"max=500"`
	RouteType   int    `json:"route_type"`
	Color       string `json:"color" binding:"max=6"`
	TextColor   string `json:"text_color" binding:"max=6"`
}

// CreateTripRequest is the payload for creating a trip.
type CreateTripRequest struct {
	RouteID     uint   `json:"route_id" binding:"required"`
	TripID      string `json:"trip_id" binding:"required,max=50"`
	Headsign    string `json:"headsign" binding:"max=255"`
	DirectionID int    `json:"direction_id"`
}

// StartTripRequest is the payload for a driver to start a trip.
type StartTripRequest struct {
	TripID    uint   `json:"trip_id" binding:"required"`
	VehicleID string `json:"vehicle_id" binding:"max=50"`
}

// BatchLocationRequest is the payload for offline sync of multiple locations.
type BatchLocationRequest struct {
	Locations []BatchLocationEntry `json:"locations" binding:"required,min=1"`
}

// BatchLocationEntry is a single location entry in a batch.
type BatchLocationEntry struct {
	Lat       float64   `json:"lat" binding:"required"`
	Lng       float64   `json:"lng" binding:"required"`
	Heading   float64   `json:"heading"`
	Speed     float64   `json:"speed"`
	Timestamp time.Time `json:"timestamp" binding:"required"`
}

// --- Response DTOs ---

// RouteResponse is the public representation of a route.
type RouteResponse struct {
	ID          uint   `json:"id"`
	RouteID     string `json:"route_id"`
	ShortName   string `json:"short_name"`
	LongName    string `json:"long_name"`
	Description string `json:"description"`
	RouteType   int    `json:"route_type"`
	IsActive    bool   `json:"is_active"`
}

// TripResponse is the public representation of a trip.
type TripResponse struct {
	ID          uint   `json:"id"`
	RouteID     uint   `json:"route_id"`
	TripID      string `json:"trip_id"`
	Headsign    string `json:"headsign"`
	DirectionID int    `json:"direction_id"`
	IsActive    bool   `json:"is_active"`
}

// ActiveTripResponse is the public representation of an active trip.
type ActiveTripResponse struct {
	ID        uint       `json:"id"`
	DriverID  uint       `json:"driver_id"`
	TripID    uint       `json:"trip_id"`
	VehicleID string     `json:"vehicle_id"`
	StartedAt time.Time  `json:"started_at"`
	EndedAt   *time.Time `json:"ended_at"`
	IsActive  bool       `json:"is_active"`
}
