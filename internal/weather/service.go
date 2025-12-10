package weather

import (
	"context"
	"fmt"
	"log"
	"sort"
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

// GetForecast generates a simple multi-day forecast based on the latest snapshot.
// For now this is a placeholder implementation that extrapolates from the most
// recent reading by repeating its values for the requested number of days and
// advancing the timestamp by whole days.
func (s *Service) getForecastPlaceholder(loc Location, days int) (Forecast, error) {
	// latest, err := s.store.GetLatest(loc)
	// if err != nil {
	// 	return nil, err
	// }

	// // Normalize the base date to midnight UTC for clearer daily buckets.
	// baseDate := time.Date(
	// 	latest.Timestamp.Year(),
	// 	latest.Timestamp.Month(),
	// 	latest.Timestamp.Day(),
	// 	0, 0, 0, 0,
	// 	time.UTC,
	// )

	// forecast := make(Forecast, 0, days)
	// for i := 0; i < days; i++ {
	// 	snap := latest
	// 	snap.Timestamp = baseDate.AddDate(0, 0, i)
	// 	forecast = append(forecast, snap)
	// }

	// return forecast, nil

	return nil, fmt.Errorf("getForecastPlaceholder is deprecated and should not be used")
}

// GetForecast fetches multi-day forecasts from providers that support it,
// aggregates them per day, and returns a normalized Forecast.
func (s *Service) GetForecast(loc Location, days int) (Forecast, error) {
	if days <= 0 {
		return nil, fmt.Errorf("days must be greater than zero")
	}

	log.Printf("DEBUG: GetForecast called for %s for %d days", loc.Key(), days)

	// Use a bounded context for outbound provider calls.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	type dayKey string

	var (
		wg            sync.WaitGroup
		mu            sync.Mutex
		dayReadings   = make(map[dayKey][]ProviderReading)
		dayTimestamps = make(map[dayKey]time.Time)
	)

	for _, p := range s.providers {
		fp, ok := p.(ForecastProvider)
		if !ok {
			continue
		}

		providerName := p.Name()

		wg.Add(1)
		go func(fp ForecastProvider, providerName string) {
			defer wg.Done()

			readings, err := fp.FetchForecast(ctx, loc, days)
			if err != nil {
				log.Printf("provider %s forecast failed for %s: %v", providerName, loc.Key(), err)
				return
			}

			if len(readings) == 0 {
				return
			}

			mu.Lock()
			defer mu.Unlock()

			for _, r := range readings {
				ts := r.Timestamp.UTC()
				k := dayKey(ts.Format("2006-01-02"))

				dayReadings[k] = append(dayReadings[k], r)

				if _, exists := dayTimestamps[k]; !exists {
					dayTimestamps[k] = time.Date(ts.Year(), ts.Month(), ts.Day(), 0, 0, 0, 0, time.UTC)
				}
			}
		}(fp, providerName)
	}

	wg.Wait()

	if len(dayReadings) == 0 {
		log.Printf("no successful forecast readings for %s", loc.Key())
		return nil, fmt.Errorf("no forecast data available")
	}

	// Collect and sort all date keys.
	keys := make([]string, 0, len(dayReadings))
	for k := range dayReadings {
		keys = append(keys, string(k))
	}
	sort.Strings(keys)

	forecast := make(Forecast, 0, days)

	for _, k := range keys {
		if len(forecast) >= days {
			break
		}

		dk := dayKey(k)
		readings := dayReadings[dk]
		if len(readings) == 0 {
			continue
		}

		snapshot := AggregateReadings(loc, readings)
		if ts, ok := dayTimestamps[dk]; ok {
			snapshot.Timestamp = ts
		}

		forecast = append(forecast, snapshot)
	}

	if len(forecast) == 0 {
		log.Printf("forecast aggregation produced no entries for %s", loc.Key())
		return nil, fmt.Errorf("no forecast data available")
	}

	return forecast, nil
}

// GetLatest delegates to the underlying store.
func (s *Service) GetLatest(loc Location) (WeatherSnapshot, error) {
	return s.store.GetLatest(loc)
}

// GetRange delegates to the underlying store.
func (s *Service) GetRange(loc Location, from, to time.Time) ([]WeatherSnapshot, error) {
	return s.store.GetRange(loc, from, to)
}
