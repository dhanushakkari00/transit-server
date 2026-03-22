package gtfsrt

import (
	pb "github.com/MobilityData/gtfs-realtime-bindings/golang/gtfs"
	"google.golang.org/protobuf/proto"
)

// VehiclePositionBuilder constructs a single GTFS-RT VehiclePosition entity
// using the builder pattern for clean, readable construction.
type VehiclePositionBuilder struct {
	entity *pb.FeedEntity
}

// NewVehiclePositionBuilder creates a new builder for a VehiclePosition.
func NewVehiclePositionBuilder(entityID string) *VehiclePositionBuilder {
	return &VehiclePositionBuilder{
		entity: &pb.FeedEntity{
			Id: proto.String(entityID),
			Vehicle: &pb.VehiclePosition{},
		},
	}
}

// WithTrip sets the trip descriptor (trip_id and route_id).
func (b *VehiclePositionBuilder) WithTrip(tripID, routeID string) *VehiclePositionBuilder {
	b.entity.Vehicle.Trip = &pb.TripDescriptor{
		TripId:  proto.String(tripID),
		RouteId: proto.String(routeID),
	}
	return b
}

// WithPosition sets the geographic position (lat, lng, bearing, speed).
func (b *VehiclePositionBuilder) WithPosition(lat, lng float32, bearing, speed float32) *VehiclePositionBuilder {
	b.entity.Vehicle.Position = &pb.Position{
		Latitude:  proto.Float32(lat),
		Longitude: proto.Float32(lng),
		Bearing:   proto.Float32(bearing),
		Speed:     proto.Float32(speed),
	}
	return b
}

// WithVehicle sets the vehicle descriptor (vehicle ID and label).
func (b *VehiclePositionBuilder) WithVehicle(vehicleID, label string) *VehiclePositionBuilder {
	b.entity.Vehicle.Vehicle = &pb.VehicleDescriptor{
		Id:    proto.String(vehicleID),
		Label: proto.String(label),
	}
	return b
}

// WithTimestamp sets the timestamp of the position observation.
func (b *VehiclePositionBuilder) WithTimestamp(timestamp uint64) *VehiclePositionBuilder {
	b.entity.Vehicle.Timestamp = proto.Uint64(timestamp)
	return b
}

// Build returns the constructed FeedEntity.
func (b *VehiclePositionBuilder) Build() *pb.FeedEntity {
	return b.entity
}
