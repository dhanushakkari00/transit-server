package gtfsrt

import (
	"fmt"
	"time"

	"transit-server/cache"
	"transit-server/models"

	"gorm.io/gorm"
)

// VehicleData holds all the information needed to build a VehiclePosition entity.
type VehicleData struct {
	VehicleID string
	DriverID  uint
	TripID    string
	RouteID   string
	Lat       float64
	Lng       float64
	Bearing   float64
	Speed     float64
	Timestamp time.Time
	IsOnline  bool
}

// DataSource is the interface for fetching active vehicle data.
// All methods are scoped to an agency for security.
type DataSource interface {
	// GetActiveVehiclesForAgency returns all active vehicles for a specific aggregator.
	GetActiveVehiclesForAgency(agencyID uint) ([]VehicleData, error)

	// GetVehicleForAgency returns a single vehicle's data, only if it belongs to the agency.
	// Returns nil if the driver is not mapped to this agency or has no active trip.
	GetVehicleForAgency(agencyID uint, driverID uint) (*VehicleData, error)

	// IsDriverMappedToAgency checks if a driver is actively mapped to the given aggregator.
	IsDriverMappedToAgency(agencyID uint, driverID uint) (bool, error)
}

// DBDataSource implements DataSource using GORM + cache.
type DBDataSource struct {
	db    *gorm.DB
	cache *cache.Store
}

// NewDBDataSource creates a new database-backed data source.
func NewDBDataSource(db *gorm.DB, cache *cache.Store) *DBDataSource {
	return &DBDataSource{db: db, cache: cache}
}

// activeTripRow is the shape returned by the JOIN query.
type activeTripRow struct {
	DriverID      uint
	VehicleID     string
	TripGTFSID    string
	RouteGTFSID   string
	AgencyID      uint
	DriverTableID uint
}

// IsDriverMappedToAgency checks if a driver is actively mapped to the given aggregator.
func (ds *DBDataSource) IsDriverMappedToAgency(agencyID uint, driverID uint) (bool, error) {
	var count int64
	err := ds.db.Table("driver_aggregator_mappings").
		Where("driver_id = ? AND aggregator_id = ? AND status = ?", driverID, agencyID, "active").
		Count(&count).Error
	return count > 0, err
}

// GetActiveVehiclesForAgency returns all active vehicles for the given aggregator.
// Only returns vehicles that are:
//  1. Mapped to this aggregator (via driver_aggregator_mappings)
//  2. Currently on an active trip
func (ds *DBDataSource) GetActiveVehiclesForAgency(agencyID uint) ([]VehicleData, error) {
	return ds.queryVehicles("routes.agency_id = ?", agencyID)
}

// GetVehicleForAgency returns a single vehicle's data with ownership verification.
func (ds *DBDataSource) GetVehicleForAgency(agencyID uint, driverID uint) (*VehicleData, error) {
	// First verify ownership
	mapped, err := ds.IsDriverMappedToAgency(agencyID, driverID)
	if err != nil {
		return nil, fmt.Errorf("failed to check mapping: %w", err)
	}
	if !mapped {
		return nil, nil // Not mapped = no access
	}

	// Query this specific driver's active trip
	vehicles, err := ds.queryVehicles(
		"routes.agency_id = ? AND active_trips.driver_id = ?", agencyID, driverID,
	)
	if err != nil {
		return nil, err
	}
	if len(vehicles) == 0 {
		return nil, nil // Driver mapped but not on an active trip
	}

	return &vehicles[0], nil
}

func (ds *DBDataSource) queryVehicles(condition string, args ...interface{}) ([]VehicleData, error) {
	var rows []activeTripRow

	query := ds.db.Table("active_trips").
		Select(`
			active_trips.driver_id,
			active_trips.vehicle_id,
			trips.gtfs_trip_id AS trip_gtfs_id,
			routes.gtfs_route_id AS route_gtfs_id,
			routes.agency_id,
			drivers.id AS driver_table_id
		`).
		Joins("JOIN trips ON trips.id = active_trips.trip_ref_id").
		Joins("JOIN routes ON routes.id = trips.route_ref_id").
		Joins("JOIN drivers ON drivers.id = active_trips.driver_id").
		Where("active_trips.is_active = ?", true)

	if condition != "" {
		query = query.Where(condition, args...)
	}

	if err := query.Scan(&rows).Error; err != nil {
		return nil, err
	}

	vehicles := make([]VehicleData, 0, len(rows))
	for _, row := range rows {
		vd := VehicleData{
			VehicleID: row.VehicleID,
			DriverID:  row.DriverID,
			TripID:    row.TripGTFSID,
			RouteID:   row.RouteGTFSID,
		}

		// Try cache first for live location
		key := locationCacheKey(row.DriverTableID)
		if val, found := ds.cache.Get(key); found {
			loc := val.(models.CachedDriverLocation)
			vd.Lat = loc.Lat
			vd.Lng = loc.Lng
			vd.Bearing = loc.Heading
			vd.Speed = loc.Speed
			vd.Timestamp = loc.UpdatedAt
			vd.IsOnline = true
		} else {
			// Fall back to DB last-known location
			var driver struct {
				LastLat     float64
				LastLng     float64
				LastHeading float64
				LastSpeed   float64
				LastSeenAt  *time.Time
			}
			ds.db.Table("drivers").
				Select("last_lat, last_lng, last_heading, last_speed, last_seen_at").
				Where("id = ?", row.DriverTableID).
				Scan(&driver)

			vd.Lat = driver.LastLat
			vd.Lng = driver.LastLng
			vd.Bearing = driver.LastHeading
			vd.Speed = driver.LastSpeed
			if driver.LastSeenAt != nil {
				vd.Timestamp = *driver.LastSeenAt
			}
			vd.IsOnline = false
		}

		if vd.Lat != 0 || vd.Lng != 0 {
			vehicles = append(vehicles, vd)
		}
	}

	return vehicles, nil
}

func locationCacheKey(driverID uint) string {
	return "location:driver:" + uintToString(driverID)
}

func uintToString(n uint) string {
	buf := make([]byte, 0, 10)
	if n == 0 {
		return "0"
	}
	for n > 0 {
		buf = append([]byte{byte('0' + n%10)}, buf...)
		n /= 10
	}
	return string(buf)
}
