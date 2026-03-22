package gtfsrt

import (
	"fmt"
	"time"

	pb "github.com/MobilityData/gtfs-realtime-bindings/golang/gtfs"
	"google.golang.org/protobuf/proto"
)

// FeedGenerator orchestrates the generation of GTFS-RT Vehicle Positions feeds.
// All methods are scoped to a specific agency — no public/global feed.
type FeedGenerator struct {
	source DataSource
}

// NewFeedGenerator creates a new feed generator with the given data source.
func NewFeedGenerator(source DataSource) *FeedGenerator {
	return &FeedGenerator{source: source}
}

// GenerateFeedForAgency builds a GTFS-RT FeedMessage for a specific aggregator.
// Only includes vehicles mapped to this aggregator that have active trips.
func (g *FeedGenerator) GenerateFeedForAgency(agencyID uint) ([]byte, error) {
	vehicles, err := g.source.GetActiveVehiclesForAgency(agencyID)
	if err != nil {
		return nil, fmt.Errorf("failed to get vehicles for agency %d: %w", agencyID, err)
	}
	return g.buildProtobuf(vehicles)
}

// GenerateFeedForVehicle builds a GTFS-RT FeedMessage for a single vehicle,
// after verifying the vehicle belongs to the given agency.
// Returns nil bytes if the driver is not mapped or has no active trip.
func (g *FeedGenerator) GenerateFeedForVehicle(agencyID uint, driverID uint) ([]byte, bool, error) {
	vehicle, err := g.source.GetVehicleForAgency(agencyID, driverID)
	if err != nil {
		return nil, false, fmt.Errorf("failed to get vehicle: %w", err)
	}
	if vehicle == nil {
		return nil, false, nil // Not found / not authorized
	}
	data, err := g.buildProtobuf([]VehicleData{*vehicle})
	return data, true, err
}

// buildProtobuf constructs the FeedMessage from vehicle data.
func (g *FeedGenerator) buildProtobuf(vehicles []VehicleData) ([]byte, error) {
	now := uint64(time.Now().Unix())

	header := &pb.FeedHeader{
		GtfsRealtimeVersion: proto.String("2.0"),
		Incrementality:      pb.FeedHeader_FULL_DATASET.Enum(),
		Timestamp:           proto.Uint64(now),
	}

	entities := make([]*pb.FeedEntity, 0, len(vehicles))
	for i, v := range vehicles {
		entityID := fmt.Sprintf("vehicle_%d", i+1)

		entity := NewVehiclePositionBuilder(entityID).
			WithTrip(v.TripID, v.RouteID).
			WithPosition(
				float32(v.Lat), float32(v.Lng),
				float32(v.Bearing), float32(v.Speed),
			).
			WithVehicle(v.VehicleID, v.VehicleID).
			WithTimestamp(uint64(v.Timestamp.Unix())).
			Build()

		entities = append(entities, entity)
	}

	feed := &pb.FeedMessage{
		Header: header,
		Entity: entities,
	}

	data, err := proto.Marshal(feed)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal feed: %w", err)
	}
	return data, nil
}

// --- JSON debug methods (same access control) ---

// GenerateDebugFeedForAgency returns a JSON-friendly feed for debugging.
func (g *FeedGenerator) GenerateDebugFeedForAgency(agencyID uint) (*FeedDebugResponse, error) {
	vehicles, err := g.source.GetActiveVehiclesForAgency(agencyID)
	if err != nil {
		return nil, err
	}
	return buildDebugResponse(vehicles), nil
}

// GenerateDebugFeedForVehicle returns a JSON-friendly feed for a single vehicle.
func (g *FeedGenerator) GenerateDebugFeedForVehicle(agencyID uint, driverID uint) (*FeedDebugResponse, bool, error) {
	vehicle, err := g.source.GetVehicleForAgency(agencyID, driverID)
	if err != nil {
		return nil, false, err
	}
	if vehicle == nil {
		return nil, false, nil
	}
	return buildDebugResponse([]VehicleData{*vehicle}), true, nil
}

func buildDebugResponse(vehicles []VehicleData) *FeedDebugResponse {
	positions := make([]VehiclePositionDebug, 0, len(vehicles))
	for _, v := range vehicles {
		positions = append(positions, VehiclePositionDebug{
			VehicleID: v.VehicleID,
			TripID:    v.TripID,
			RouteID:   v.RouteID,
			Lat:       v.Lat,
			Lng:       v.Lng,
			Bearing:   v.Bearing,
			Speed:     v.Speed,
			Timestamp: v.Timestamp,
			IsOnline:  v.IsOnline,
		})
	}

	return &FeedDebugResponse{
		Header:   FeedHeaderDebug{Version: "2.0", Timestamp: time.Now()},
		Entities: positions,
		Count:    len(positions),
	}
}

// --- Debug response types ---

type FeedDebugResponse struct {
	Header   FeedHeaderDebug        `json:"header"`
	Entities []VehiclePositionDebug `json:"entities"`
	Count    int                    `json:"count"`
}

type FeedHeaderDebug struct {
	Version   string    `json:"gtfs_realtime_version"`
	Timestamp time.Time `json:"timestamp"`
}

type VehiclePositionDebug struct {
	VehicleID string    `json:"vehicle_id"`
	TripID    string    `json:"trip_id"`
	RouteID   string    `json:"route_id"`
	Lat       float64   `json:"lat"`
	Lng       float64   `json:"lng"`
	Bearing   float64   `json:"bearing"`
	Speed     float64   `json:"speed"`
	Timestamp time.Time `json:"timestamp"`
	IsOnline  bool      `json:"is_online"`
}
