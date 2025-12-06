package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/i474232898/weather-data-aggregation/internal/weather"
)

// AppConfig holds application configuration loaded from environment variables.
type AppConfig struct {
	OpenWeatherAPIKey string
	WeatherAPIKey     string

	// SchedulerInterval controls how often we fetch data for each location.
	SchedulerInterval time.Duration

	// Locations to track.
	Locations []weather.Location

	// In-memory store retention.
	StoreMaxHistory int           // max number of snapshots per location (0 = unlimited)
	StoreMaxAge     time.Duration // max age of snapshots (0 = unlimited)

	// HTTP timeout for outbound provider calls.
	HTTPTimeout time.Duration
}

// Load reads configuration from environment with sensible defaults.
func Load() (*AppConfig, error) {
	cfg := &AppConfig{}

	cfg.OpenWeatherAPIKey = os.Getenv("OPENWEATHER_API_KEY")
	cfg.WeatherAPIKey = os.Getenv("WEATHERAPI_API_KEY")

	// Scheduler interval: default 15 minutes.
	intervalStr := getenvDefault("SCHEDULER_INTERVAL", "15m")
	interval, err := time.ParseDuration(intervalStr)
	if err != nil {
		return nil, fmt.Errorf("invalid SCHEDULER_INTERVAL: %w", err)
	}
	cfg.SchedulerInterval = interval

	// Store retention.
	cfg.StoreMaxHistory = getenvInt("STORE_MAX_HISTORY", 96) // roughly 24h at 15-minute intervals

	maxAgeStr := getenvDefault("STORE_MAX_AGE", "24h")
	maxAge, err := time.ParseDuration(maxAgeStr)
	if err != nil {
		return nil, fmt.Errorf("invalid STORE_MAX_AGE: %w", err)
	}
	cfg.StoreMaxAge = maxAge

	// HTTP timeout for outbound requests.
	httpTimeoutStr := getenvDefault("HTTP_TIMEOUT", "10s")
	httpTimeout, err := time.ParseDuration(httpTimeoutStr)
	if err != nil {
		return nil, fmt.Errorf("invalid HTTP_TIMEOUT: %w", err)
	}
	cfg.HTTPTimeout = httpTimeout

	// Locations: for simplicity, support a single primary location via env.
	loc, err := loadPrimaryLocation()
	if err != nil {
		return nil, err
	}
	cfg.Locations = []weather.Location{loc}

	return cfg, nil
}

func loadPrimaryLocation() (weather.Location, error) {
	city := os.Getenv("WEATHER_LOCATION_CITY")
	country := os.Getenv("WEATHER_LOCATION_COUNTRY")

	var loc weather.Location

	if city == "" {
		// Provide a sensible default for local development.
		loc.City = "Kyiv"
		loc.Country = "UA"
	} else {
		loc.City = city
		loc.Country = country
	}

	latStr := os.Getenv("WEATHER_LOCATION_LAT")
	lonStr := os.Getenv("WEATHER_LOCATION_LON")

	if latStr != "" && lonStr != "" {
		lat, err := strconv.ParseFloat(latStr, 64)
		if err != nil {
			return weather.Location{}, fmt.Errorf("invalid WEATHER_LOCATION_LAT: %w", err)
		}
		lon, err := strconv.ParseFloat(lonStr, 64)
		if err != nil {
			return weather.Location{}, fmt.Errorf("invalid WEATHER_LOCATION_LON: %w", err)
		}
		loc.Lat = &lat
		loc.Lon = &lon
	}

	return loc, nil
}

func getenvDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getenvInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		n, err := strconv.Atoi(v)
		if err == nil {
			return n
		}
	}
	return def
}


