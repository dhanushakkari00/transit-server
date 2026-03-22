// Package gtfsrt provides a clean, OOP-based builder for generating
// GTFS-Realtime Vehicle Positions protobuf feeds.
//
// Architecture:
//   - FeedGenerator: Orchestrator that builds the complete GTFS-RT FeedMessage
//   - VehiclePositionBuilder: Builds individual VehiclePosition entities
//   - DataSource: Interface for fetching vehicle data (decoupled from DB)
//
// Usage:
//
//	source := gtfsrt.NewDBDataSource(db, cache)
//	generator := gtfsrt.NewFeedGenerator(source)
//	feedBytes, err := generator.GenerateFeed()
package gtfsrt
