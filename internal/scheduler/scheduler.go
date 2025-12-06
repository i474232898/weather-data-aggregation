package scheduler

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/go-co-op/gocron"
	"github.com/i474232898/weather-data-aggregation/internal/weather"
)

// Scheduler periodically fetches weather data for configured locations.
type Scheduler struct {
	scheduler *gocron.Scheduler
	service   *weather.Service
	locations []weather.Location
	interval  time.Duration
}

// New creates a new Scheduler.
func New(locations []weather.Location, interval time.Duration, service *weather.Service) *Scheduler {
	s := gocron.NewScheduler(time.UTC)
	return &Scheduler{
		scheduler: s,
		service:   service,
		locations: locations,
		interval:  interval,
	}
}

// Start schedules the periodic job and starts the underlying scheduler.
func (s *Scheduler) Start() error {
	if len(s.locations) == 0 {
		log.Println("scheduler: no locations configured; nothing to schedule")
		return nil
	}

	minutes := int(s.interval.Minutes())
	if minutes <= 0 {
		minutes = 15
	}

	_, err := s.scheduler.Every(minutes).Minutes().Do(func() {
		log.Println("scheduler: running weather fetch job")

		var wg sync.WaitGroup
		for _, loc := range s.locations {
			loc := loc
			wg.Add(1)
			go func() {
				defer wg.Done()

				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()

				if err := s.service.FetchAndStore(ctx, loc); err != nil {
					log.Printf("scheduler: fetch failed for %s: %v", loc.Key(), err)
				}
			}()
		}
		wg.Wait()
		log.Println("scheduler: completed weather fetch job")
	})
	if err != nil {
		return err
	}

	s.scheduler.StartAsync()
	return nil
}

// Stop stops the scheduler and cancels any future jobs.
func (s *Scheduler) Stop() {
	if s.scheduler != nil {
		s.scheduler.Stop()
	}
}


