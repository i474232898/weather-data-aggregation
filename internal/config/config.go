package config

import (
	"fmt"
	"github.com/joho/godotenv"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/i474232898/weather-data-aggregation/internal/weather"
)

type AppConfig struct {
	OpenWeatherAPIKey string
	WeatherAPIKey     string

	// FetchInterval controls how often we fetch data for each location.
	FetchInterval time.Duration

	// Locations to track.
	Locations []weather.Location

	// In-memory store retention.
	StoreMaxHistory int           // max number of snapshots per location (0 = unlimited)
	StoreMaxAge     time.Duration // max age of snapshots (0 = unlimited)

	Port string
}

// Load reads configuration from environment with sensible defaults.
func Load() (*AppConfig, error) {
	if err := godotenv.Load(); err != nil {
		log.Printf("INFO: No .env file found or error loading it: %v", err)
	}
	cfg := &AppConfig{}

	cfg.OpenWeatherAPIKey = os.Getenv("OPENWEATHER_API_KEY")
	cfg.WeatherAPIKey = os.Getenv("WEATHERAPI_API_KEY")

	// Scheduler interval: default 15 minutes.
	intervalStr := getenvDefault("FETCH_INTERVAL", "15m")
	interval, err := time.ParseDuration(intervalStr)
	if err != nil {
		return nil, fmt.Errorf("invalid FETCH_INTERVAL: %w", err)
	}
	cfg.FetchInterval = interval

	// Store retention.
	cfg.StoreMaxHistory = getenvInt("STORE_MAX_HISTORY", 96) // roughly 24h at 15-minute intervals

	maxAgeStr := getenvDefault("STORE_MAX_AGE", "24h")
	maxAge, err := time.ParseDuration(maxAgeStr)
	if err != nil {
		return nil, fmt.Errorf("invalid STORE_MAX_AGE: %w", err)
	}
	cfg.StoreMaxAge = maxAge
	cfg.Port = getenvDefault("PORT", "8080")

	locs, err := loadPrimaryLocation()
	if err != nil {
		return nil, err
	}
	cfg.Locations = locs

	return cfg, nil
}

func loadPrimaryLocation() ([]weather.Location, error) {
	city := os.Getenv("WEATHER_LOCATION_CITY")
	country := os.Getenv("WEATHER_LOCATION_COUNTRY")
	cities := strings.Split(city, ",")
	countries := strings.Split(country, ",")
	if len(cities) != len(countries) {
		return nil, fmt.Errorf("number of cities and countries must be the same")
	}
	var locs []weather.Location
	for i := range cities {
		locs = append(locs, weather.Location{
			City:    cities[i],
			Country: countries[i],
		})
	}

	return locs, nil
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
