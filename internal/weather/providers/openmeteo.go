package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/i474232898/weather-data-aggregation/internal/weather"
	"github.com/sony/gobreaker"
)

// OpenMeteoProvider implements the weather.Provider interface for Open-Meteo.
type OpenMeteoProvider struct {
	name    string
	baseURL string
	httpCfg HTTPClientConfig
	circuit *gobreaker.CircuitBreaker
}

func NewOpenMeteoProvider(client *http.Client) *OpenMeteoProvider {
	cb := gobreaker.NewCircuitBreaker(gobreaker.Settings{
		Name:        "openmeteo",
		MaxRequests: 5,
		Interval:    1 * time.Minute,
		Timeout:     2 * time.Minute,
	})

	return &OpenMeteoProvider{
		name:    "openmeteo",
		baseURL: "https://api.open-meteo.com/v1/forecast",
		httpCfg: HTTPClientConfig{
			Client: client,
			Backoff: BackoffConfig{
				MaxRetries:      3,
				InitialInterval: 500 * time.Millisecond,
				MaxInterval:     5 * time.Second,
			},
		},
		circuit: cb,
	}
}

func (p *OpenMeteoProvider) Name() string {
	return p.name
}

func (p *OpenMeteoProvider) Fetch(ctx context.Context, loc weather.Location) (weather.ProviderReading, error) {
	if loc.Lat == nil || loc.Lon == nil {
		return weather.ProviderReading{}, fmt.Errorf("openmeteo requires latitude and longitude")
	}

	buildRequest := func() (*http.Request, error) {
		values := url.Values{}
		values.Set("latitude", fmt.Sprintf("%f", *loc.Lat))
		values.Set("longitude", fmt.Sprintf("%f", *loc.Lon))
		values.Set("current_weather", "true")

		u := fmt.Sprintf("%s?%s", p.baseURL, values.Encode())
		req, err := http.NewRequest(http.MethodGet, u, nil)
		if err != nil {
			return nil, err
		}
		return req, nil
	}

	resp, err := doRequestWithResilience(ctx, p.httpCfg, p.circuit, buildRequest)
	if err != nil {
		return weather.ProviderReading{}, err
	}
	defer resp.Body.Close()

	var payload struct {
		CurrentWeather struct {
			Temperature float64 `json:"temperature"`
			WindSpeed   float64 `json:"windspeed"`
			Time        string  `json:"time"`
			WeatherCode int     `json:"weathercode"`
		} `json:"current_weather"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return weather.ProviderReading{}, err
	}

	ts, err := time.Parse(time.RFC3339, payload.CurrentWeather.Time)
	if err != nil {
		ts = time.Now().UTC()
	} else {
		ts = ts.UTC()
	}

	cond := mapOpenMeteoCondition(payload.CurrentWeather.WeatherCode)

	return weather.ProviderReading{
		ProviderName: p.name,
		Timestamp:    ts,
		TemperatureC: payload.CurrentWeather.Temperature,
		// Open-Meteo current_weather has limited fields; we fill what we can.
		WindSpeedMS: payload.CurrentWeather.WindSpeed,
		Condition:   cond,
	}, nil
}

func mapOpenMeteoCondition(code int) weather.Condition {
	// Mapping based on Open-Meteo weather codes (simplified).
	switch {
	case code == 0:
		return weather.ConditionClear
	case code >= 1 && code <= 3:
		return weather.ConditionCloudy
	case (code >= 51 && code <= 67) || (code >= 80 && code <= 82):
		return weather.ConditionRain
	case code >= 71 && code <= 77:
		return weather.ConditionSnow
	case code >= 95:
		return weather.ConditionStorm
	default:
		return weather.ConditionUnknown
	}
}


