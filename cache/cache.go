package cache

import (
	"fmt"
	"log"
	"sync"
	"time"
)

// Item represents a single cached item with an optional expiration.
type Item struct {
	Value     interface{}
	ExpiresAt time.Time // Zero value means no expiration
}

// IsExpired checks if the item has passed its expiration time.
func (item *Item) IsExpired() bool {
	if item.ExpiresAt.IsZero() {
		return false
	}
	return time.Now().After(item.ExpiresAt)
}

// Store is a thread-safe, in-memory key-value cache with TTL support.
// It supports automatic background eviction of expired entries.
type Store struct {
	mu       sync.RWMutex
	items    map[string]*Item
	stopCh   chan struct{}
	isClosed bool
}

// New creates a new cache Store and starts a background eviction goroutine.
// cleanupInterval controls how often expired items are evicted.
// Pass 0 to disable automatic eviction (manual cleanup only).
func New(cleanupInterval time.Duration) *Store {
	s := &Store{
		items:  make(map[string]*Item),
		stopCh: make(chan struct{}),
	}

	if cleanupInterval > 0 {
		go s.evictionLoop(cleanupInterval)
		log.Printf("🗄️  Cache initialized (cleanup every %s)", cleanupInterval)
	} else {
		log.Println("🗄️  Cache initialized (no auto-cleanup)")
	}

	return s
}

// Set stores a key-value pair with an optional TTL.
// Pass ttl=0 for no expiration.
func (s *Store) Set(key string, value interface{}, ttl time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()

	item := &Item{Value: value}
	if ttl > 0 {
		item.ExpiresAt = time.Now().Add(ttl)
	}
	s.items[key] = item
}

// Get retrieves a value by key. Returns the value and true if found and not expired.
// Expired items are lazily deleted on access.
func (s *Store) Get(key string) (interface{}, bool) {
	s.mu.RLock()
	item, found := s.items[key]
	s.mu.RUnlock()

	if !found {
		return nil, false
	}

	if item.IsExpired() {
		// Lazy eviction: delete expired item on access
		s.Delete(key)
		return nil, false
	}

	return item.Value, true
}

// GetString retrieves a string value by key. Returns empty string and false if not found.
func (s *Store) GetString(key string) (string, bool) {
	val, found := s.Get(key)
	if !found {
		return "", false
	}
	str, ok := val.(string)
	if !ok {
		return "", false
	}
	return str, true
}

// Has checks if a key exists and is not expired.
func (s *Store) Has(key string) bool {
	_, found := s.Get(key)
	return found
}

// Delete removes a key from the cache.
func (s *Store) Delete(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.items, key)
}

// Flush removes all items from the cache.
func (s *Store) Flush() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items = make(map[string]*Item)
}

// Count returns the number of items in the cache (including expired but not yet evicted).
func (s *Store) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.items)
}

// Keys returns all non-expired keys in the cache.
func (s *Store) Keys() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	keys := make([]string, 0, len(s.items))
	for k, item := range s.items {
		if !item.IsExpired() {
			keys = append(keys, k)
		}
	}
	return keys
}

// SetNX sets a value only if the key does not already exist (Set if Not eXists).
// Returns true if the value was set, false if the key already existed.
func (s *Store) SetNX(key string, value interface{}, ttl time.Duration) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if existing, found := s.items[key]; found && !existing.IsExpired() {
		return false
	}

	item := &Item{Value: value}
	if ttl > 0 {
		item.ExpiresAt = time.Now().Add(ttl)
	}
	s.items[key] = item
	return true
}

// Increment atomically increments an integer value by delta.
// If the key doesn't exist, it's initialized to delta.
// Returns the new value and any error.
func (s *Store) Increment(key string, delta int64) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	item, found := s.items[key]
	if !found || item.IsExpired() {
		s.items[key] = &Item{Value: delta}
		return delta, nil
	}

	switch v := item.Value.(type) {
	case int64:
		newVal := v + delta
		item.Value = newVal
		return newVal, nil
	case int:
		newVal := int64(v) + delta
		item.Value = newVal
		return newVal, nil
	default:
		return 0, fmt.Errorf("value for key '%s' is not an integer", key)
	}
}

// TTL returns the remaining time-to-live for a key.
// Returns -1 if the key has no expiration, -2 if the key doesn't exist.
func (s *Store) TTL(key string) time.Duration {
	s.mu.RLock()
	item, found := s.items[key]
	s.mu.RUnlock()

	if !found || item.IsExpired() {
		return -2
	}

	if item.ExpiresAt.IsZero() {
		return -1
	}

	return time.Until(item.ExpiresAt)
}

// Expire sets a new TTL on an existing key. Returns false if the key doesn't exist.
func (s *Store) Expire(key string, ttl time.Duration) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	item, found := s.items[key]
	if !found || item.IsExpired() {
		return false
	}

	if ttl > 0 {
		item.ExpiresAt = time.Now().Add(ttl)
	} else {
		item.ExpiresAt = time.Time{} // Remove expiration
	}
	return true
}

// Close stops the background eviction goroutine and clears the cache.
func (s *Store) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.isClosed {
		close(s.stopCh)
		s.isClosed = true
		s.items = make(map[string]*Item)
		log.Println("🗄️  Cache closed")
	}
}

// evictionLoop periodically removes expired items from the cache.
func (s *Store) evictionLoop(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.evictExpired()
		case <-s.stopCh:
			return
		}
	}
}

// evictExpired removes all expired items (called by eviction loop).
func (s *Store) evictExpired() {
	s.mu.Lock()
	defer s.mu.Unlock()

	evicted := 0
	for key, item := range s.items {
		if item.IsExpired() {
			delete(s.items, key)
			evicted++
		}
	}

	if evicted > 0 {
		log.Printf("🗄️  Cache eviction: removed %d expired item(s)", evicted)
	}
}
