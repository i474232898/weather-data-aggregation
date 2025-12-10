package weather

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// Service orchestrates fetching from multiple providers and persisting snapshots.
type Service struct {
	store     Store
	providers []Provider
}

// NewService creates a new Service.
func NewService(store Store, providers []Provider) *Service {
	return &Service{
		store:     store,
		providers: providers,
	}
}

// FetchAndStore fetches data from all providers concurrently for the given location,
// aggregates successful readings, and stores a snapshot.
func (s *Service) FetchAndStore(ctx context.Context, loc Location) error {
	var (
		wg       sync.WaitGroup
		mu       sync.Mutex
		readings []ProviderReading
	)

	log.Printf("DEBUG: FetchAndStore called for %s with %d providers", loc.Key(), len(s.providers))
	if len(s.providers) == 0 {
		log.Printf("ERROR: No providers available to fetch weather data for %s", loc.Key())
		return fmt.Errorf("no weather providers configured")
	}

	for _, p := range s.providers {
		p := p
		wg.Add(1)
		go func() {
			defer wg.Done()

			r, err := p.Fetch(ctx, loc)
			if err != nil {
				// Log and continue; we want partial success when possible.
				log.Printf("provider %s fetch failed for %s: %v", p.Name(), loc.Key(), err)
				return
			}

			mu.Lock()
			readings = append(readings, r)
			fmt.Println("readings>>", readings)
			mu.Unlock()
		}()
	}

	wg.Wait()

	if len(readings) == 0 {
		// No providers succeeded; do not overwrite last good snapshot.
		log.Printf("no successful provider readings for %s; keeping last good snapshot if any", loc.Key())
		return nil
	}

	snapshot := AggregateReadings(loc, readings)
	if snapshot.Timestamp.IsZero() {
		snapshot.Timestamp = time.Now().UTC()
	}
	s.store.SaveSnapshot(loc, snapshot)
	return nil
}

// GetLatest delegates to the underlying store.
func (s *Service) GetLatest(loc Location) (WeatherSnapshot, error) {
	return s.store.GetLatest(loc)
}

// GetRange delegates to the underlying store.
func (s *Service) GetRange(loc Location, from, to time.Time) ([]WeatherSnapshot, error) {
	return s.store.GetRange(loc, from, to)
}
