package weather

import (
	"time"
)

// Condition represents a normalized high-level weather condition.
type Condition string

const (
	ConditionUnknown Condition = "unknown"
	ConditionClear   Condition = "clear"
	ConditionCloudy  Condition = "cloudy"
	ConditionRain    Condition = "rain"
	ConditionSnow    Condition = "snow"
	ConditionStorm   Condition = "storm"
	ConditionMist    Condition = "mist"
)

// Location represents a logical place for which we track weather.
// City/Country must be provided.
type Location struct {
	City    string `json:"city"`
	Country string `json:"country"`
}

// Key returns a canonical string key for indexing this location in stores.
func (l Location) Key() string {
	return l.City + ":" + l.Country
}

// WeatherSnapshot is the normalized, aggregated weather view at a point in time.
type WeatherSnapshot struct {
	Location    Location  `json:"location"`
	Timestamp   time.Time `json:"timestamp"` // always UTC
	Temperature float64   `json:"temperatureC"`
	Humidity    float64   `json:"humidityPercent"`
	WindSpeed   float64   `json:"windSpeed"`
	Pressure    float64   `json:"pressureHpa"`
	PrecipMM    float64   `json:"precipMm"`
	Condition   Condition `json:"condition"`

	// Providers contributing to this snapshot.
	Providers []ProviderContribution `json:"providers,omitempty"`
}

// Forecast represents a simple multi-day weather forecast
// as a slice of normalized weather snapshots, one per day.
// Forecast entries are expected to be ordered by Timestamp ascending.
type Forecast []WeatherSnapshot

// ProviderContribution describes data coming from a single provider used in aggregation.
type ProviderContribution struct {
	ProviderName string    `json:"provider"`
	Timestamp    time.Time `json:"timestamp"`
}
