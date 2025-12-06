package store

import (
	"errors"
	"sync"
	"time"

	"github.com/i474232898/weather-data-aggregation/internal/weather"
)

var (
	// ErrNotFound is returned when no data is available for a given location.
	ErrNotFound = errors.New("no weather data for location")
)

// SnapshotHistory holds a time-ordered list of weather snapshots for a location.
type SnapshotHistory struct {
	Snapshots []weather.WeatherSnapshot
}

// MemoryStore is a concurrency-safe in-memory implementation of a weather store.
type MemoryStore struct {
	mu sync.RWMutex

	// key: location key, value: history
	data map[string]*SnapshotHistory

	// retention configuration
	maxHistory int           // max number of snapshots per location
	maxAge     time.Duration // optional max age for snapshots
}

// NewMemoryStore creates a new MemoryStore with optional limits.
// If maxHistory is <= 0, it is treated as unlimited.
func NewMemoryStore(maxHistory int, maxAge time.Duration) *MemoryStore {
	return &MemoryStore{
		data:       make(map[string]*SnapshotHistory),
		maxHistory: maxHistory,
		maxAge:     maxAge,
	}
}

// SaveSnapshot appends a new snapshot for a location and enforces retention.
func (s *MemoryStore) SaveSnapshot(loc weather.Location, snapshot weather.WeatherSnapshot) {
	key := loc.Key()

	s.mu.Lock()
	defer s.mu.Unlock()

	history, ok := s.data[key]
	if !ok {
		history = &SnapshotHistory{}
		s.data[key] = history
	}

	history.Snapshots = append(history.Snapshots, snapshot)

	// Enforce retention by count.
	if s.maxHistory > 0 && len(history.Snapshots) > s.maxHistory {
		over := len(history.Snapshots) - s.maxHistory
		history.Snapshots = history.Snapshots[over:]
	}

	// Enforce retention by age.
	if s.maxAge > 0 {
		cutoff := time.Now().Add(-s.maxAge)
		i := 0
		for ; i < len(history.Snapshots); i++ {
			if history.Snapshots[i].Timestamp.After(cutoff) || history.Snapshots[i].Timestamp.Equal(cutoff) {
				break
			}
		}
		if i > 0 && i < len(history.Snapshots) {
			history.Snapshots = history.Snapshots[i:]
		}
	}
}

// GetLatest returns the most recent snapshot for a location.
func (s *MemoryStore) GetLatest(loc weather.Location) (weather.WeatherSnapshot, error) {
	key := loc.Key()

	s.mu.RLock()
	defer s.mu.RUnlock()

	history, ok := s.data[key]
	if !ok || len(history.Snapshots) == 0 {
		return weather.WeatherSnapshot{}, ErrNotFound
	}
	return history.Snapshots[len(history.Snapshots)-1], nil
}

// GetRange returns all snapshots for a location between from and to (inclusive).
func (s *MemoryStore) GetRange(loc weather.Location, from, to time.Time) ([]weather.WeatherSnapshot, error) {
	key := loc.Key()

	s.mu.RLock()
	defer s.mu.RUnlock()

	history, ok := s.data[key]
	if !ok || len(history.Snapshots) == 0 {
		return nil, ErrNotFound
	}

	var result []weather.WeatherSnapshot
	for _, snap := range history.Snapshots {
		if (snap.Timestamp.Equal(from) || snap.Timestamp.After(from)) &&
			(snap.Timestamp.Equal(to) || snap.Timestamp.Before(to)) {
			result = append(result, snap)
		}
	}

	if len(result) == 0 {
		return nil, ErrNotFound
	}

	return result, nil
}


