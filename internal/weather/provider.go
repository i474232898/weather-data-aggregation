package weather

import (
	"context"
	"time"
)

// ProviderReading represents a single provider's normalized reading
// that can be aggregated into a WeatherSnapshot.
type ProviderReading struct {
	ProviderName string
	Timestamp    time.Time

	TemperatureC float64
	HumidityPct  float64
	WindSpeedMS  float64
	PressureHpa  float64
	PrecipMm     float64
	Condition    Condition
}

// Provider abstracts a weather data source (e.g. OpenWeatherMap, WeatherAPI, Open-Meteo).
type Provider interface {
	Name() string
	Fetch(ctx context.Context, loc Location) (ProviderReading, error)
}

// Store is the contract the in-memory store (and any future persistent store) must satisfy.
type Store interface {
	SaveSnapshot(loc Location, snapshot WeatherSnapshot)
	GetLatest(loc Location) (WeatherSnapshot, error)
	GetRange(loc Location, from, to time.Time) ([]WeatherSnapshot, error)
}


