package weather

import (
	"fmt"
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
)

// Location represents a logical place for which we track weather.
// Either City/Country or Lat/Lon must be provided.
type Location struct {
	City    string   `json:"city,omitempty"`
	Country string   `json:"country,omitempty"`
	Lat     *float64 `json:"lat,omitempty"`
	Lon     *float64 `json:"lon,omitempty"`
}

// Key returns a canonical string key for indexing this location in stores.
func (l Location) Key() string {
	if l.Lat != nil && l.Lon != nil {
		return formatCoord(*l.Lat, *l.Lon)
	}
	// Fallback to city-country key
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

// ProviderContribution describes data coming from a single provider used in aggregation.
type ProviderContribution struct {
	ProviderName string    `json:"provider"`
	Timestamp    time.Time `json:"timestamp"`
}

// formatCoord formats latitude and longitude as a stable key string.
func formatCoord(lat, lon float64) string {
	// Fixed precision for key stability.
	return fmt.Sprintf("lat=%.4f,lon=%.4f", lat, lon)
}


