package ws

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"transit-server/database"
	"transit-server/models"
)

// Hub manages all WebSocket connections grouped by aggregator ID.
// When a driver pushes a location, the Hub finds which aggregators
// own that driver and pushes the update to their connected clients.
type Hub struct {
	// subscribers maps aggregatorID → set of connected clients
	subscribers map[uint]map[*Client]bool

	// driverAggregatorCache caches driver→aggregator mappings to avoid
	// hitting DB on every location push. TTL-based invalidation.
	driverAggregatorCache map[uint]*driverMappingCache

	mu       sync.RWMutex
	cacheMu  sync.RWMutex
	register chan *registration
	remove   chan *registration
	done     chan struct{}
}

type registration struct {
	aggregatorID uint
	client       *Client
}

type driverMappingCache struct {
	aggregatorIDs []uint
	cachedAt      time.Time
}

const mappingCacheTTL = 30 * time.Second

// LocationEvent is the JSON payload pushed to WebSocket subscribers.
type LocationEvent struct {
	Event string              `json:"event"`
	Data  LocationEventData   `json:"data"`
}

type LocationEventData struct {
	DriverID  uint      `json:"driver_id"`
	VehicleID string    `json:"vehicle_id,omitempty"`
	Lat       float64   `json:"lat"`
	Lng       float64   `json:"lng"`
	Heading   float64   `json:"heading"`
	Speed     float64   `json:"speed"`
	Timestamp time.Time `json:"timestamp"`
	IsOnline  bool      `json:"is_online"`
}

// NewHub creates a new WebSocket hub.
func NewHub() *Hub {
	return &Hub{
		subscribers:           make(map[uint]map[*Client]bool),
		driverAggregatorCache: make(map[uint]*driverMappingCache),
		register:              make(chan *registration, 64),
		remove:                make(chan *registration, 64),
		done:                  make(chan struct{}),
	}
}

// Run starts the hub's event loop. Call this as a goroutine.
func (h *Hub) Run() {
	for {
		select {
		case reg := <-h.register:
			h.mu.Lock()
			if h.subscribers[reg.aggregatorID] == nil {
				h.subscribers[reg.aggregatorID] = make(map[*Client]bool)
			}
			h.subscribers[reg.aggregatorID][reg.client] = true
			count := len(h.subscribers[reg.aggregatorID])
			h.mu.Unlock()
			log.Printf("[WS Hub] Client registered for aggregator %d (total: %d)", reg.aggregatorID, count)

		case reg := <-h.remove:
			h.mu.Lock()
			if clients, ok := h.subscribers[reg.aggregatorID]; ok {
				delete(clients, reg.client)
				if len(clients) == 0 {
					delete(h.subscribers, reg.aggregatorID)
				}
			}
			h.mu.Unlock()
			reg.client.conn.Close()
			log.Printf("[WS Hub] Client unregistered from aggregator %d", reg.aggregatorID)

		case <-h.done:
			return
		}
	}
}

// Register adds a client to the hub for the given aggregator.
func (h *Hub) Register(aggregatorID uint, client *Client) {
	h.register <- &registration{aggregatorID: aggregatorID, client: client}
}

// Unregister removes a client from the hub.
func (h *Hub) Unregister(aggregatorID uint, client *Client) {
	h.remove <- &registration{aggregatorID: aggregatorID, client: client}
}

// Stop shuts down the hub event loop.
func (h *Hub) Stop() {
	close(h.done)
}

// Broadcast pushes a location update to all aggregator subscribers
// that own the given driver (via driver_aggregator_mappings).
func (h *Hub) Broadcast(driverID uint, loc models.CachedDriverLocation) {
	// Find which aggregators own this driver
	aggregatorIDs := h.getAggregatorIDsForDriver(driverID)
	if len(aggregatorIDs) == 0 {
		return
	}

	// Build the event payload
	event := LocationEvent{
		Event: "location_update",
		Data: LocationEventData{
			DriverID:  loc.DriverID,
			Lat:       loc.Lat,
			Lng:       loc.Lng,
			Heading:   loc.Heading,
			Speed:     loc.Speed,
			Timestamp: loc.UpdatedAt,
			IsOnline:  true,
		},
	}

	// Also try to get the vehicle ID from the active trip
	var activeTrip struct{ VehicleID string }
	database.DB.Table("active_trips").
		Select("vehicle_id").
		Where("driver_id = ? AND is_active = ?", driverID, true).
		Scan(&activeTrip)
	event.Data.VehicleID = activeTrip.VehicleID

	msg, err := json.Marshal(event)
	if err != nil {
		log.Printf("[WS Hub] Failed to marshal event: %v", err)
		return
	}

	// Push to all connected clients of each owning aggregator
	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, aggID := range aggregatorIDs {
		clients, ok := h.subscribers[aggID]
		if !ok || len(clients) == 0 {
			continue
		}
		for client := range clients {
			select {
			case client.send <- msg:
			default:
				// Client buffer full — drop (they'll get next update)
				log.Printf("[WS Hub] Dropping message for slow client (agg %d)", aggID)
			}
		}
	}
}

// getAggregatorIDsForDriver returns cached aggregator IDs mapped to a driver.
func (h *Hub) getAggregatorIDsForDriver(driverID uint) []uint {
	h.cacheMu.RLock()
	cached, ok := h.driverAggregatorCache[driverID]
	h.cacheMu.RUnlock()

	if ok && time.Since(cached.cachedAt) < mappingCacheTTL {
		return cached.aggregatorIDs
	}

	// Cache miss or expired — query DB
	var mappings []struct{ AggregatorID uint }
	database.DB.Table("driver_aggregator_mappings").
		Select("aggregator_id").
		Where("driver_id = ? AND status = ?", driverID, "active").
		Scan(&mappings)

	ids := make([]uint, len(mappings))
	for i, m := range mappings {
		ids[i] = m.AggregatorID
	}

	h.cacheMu.Lock()
	h.driverAggregatorCache[driverID] = &driverMappingCache{
		aggregatorIDs: ids,
		cachedAt:      time.Now(),
	}
	h.cacheMu.Unlock()

	return ids
}

// SubscriberCount returns the total number of active WebSocket connections.
func (h *Hub) SubscriberCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	count := 0
	for _, clients := range h.subscribers {
		count += len(clients)
	}
	return count
}
