package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/i474232898/weather-data-aggregation/internal/weather"
	"github.com/sony/gobreaker"
)

// WeatherAPIProvider implements the weather.Provider interface for WeatherAPI.com.
type WeatherAPIProvider struct {
	name    string
	apiKey  string
	baseURL string
	httpCfg HTTPClientConfig
	circuit *gobreaker.CircuitBreaker
}

func NewWeatherAPIProvider(client *http.Client, apiKey string) *WeatherAPIProvider {
	cb := gobreaker.NewCircuitBreaker(gobreaker.Settings{
		Name:        "weatherapi",
		MaxRequests: 5,
		Interval:    1 * time.Minute,
		Timeout:     2 * time.Minute,
	})

	return &WeatherAPIProvider{
		name:    "weatherapi",
		apiKey:  apiKey,
		baseURL: "https://api.weatherapi.com/v1/current.json",
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

func (p *WeatherAPIProvider) Name() string {
	return p.name
}

func (p *WeatherAPIProvider) Fetch(ctx context.Context, loc weather.Location) (weather.ProviderReading, error) {
	if p.apiKey == "" {
		return weather.ProviderReading{}, fmt.Errorf("weatherapi api key is not configured")
	}

	buildRequest := func() (*http.Request, error) {
		values := url.Values{}
		values.Set("key", p.apiKey)
		// WeatherAPI uses "q" for location; it accepts "city,country" or "lat,lon".
		if loc.Lat != nil && loc.Lon != nil {
			values.Set("q", fmt.Sprintf("%f,%f", *loc.Lat, *loc.Lon))
		} else {
			q := loc.City
			if loc.Country != "" {
				q = fmt.Sprintf("%s,%s", loc.City, loc.Country)
			}
			values.Set("q", q)
		}

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
		Location struct {
			LocaltimeEpoch int64 `json:"localtime_epoch"`
		} `json:"location"`
		Current struct {
			TempC      float64 `json:"temp_c"`
			Humidity   float64 `json:"humidity"`
			WindKph    float64 `json:"wind_kph"`
			PressureMb float64 `json:"pressure_mb"`
			PrecipMm   float64 `json:"precip_mm"`
			Condition  struct {
				Text string `json:"text"`
			} `json:"condition"`
		} `json:"current"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return weather.ProviderReading{}, err
	}

	ts := time.Unix(payload.Location.LocaltimeEpoch, 0).UTC()
	if ts.IsZero() {
		ts = time.Now().UTC()
	}

	// Convert wind from kph to m/s (approx).
	windMS := payload.Current.WindKph / 3.6

	cond := mapWeatherAPICondition(payload.Current.Condition.Text)

	return weather.ProviderReading{
		ProviderName: p.name,
		Timestamp:    ts,
		TemperatureC: payload.Current.TempC,
		HumidityPct:  payload.Current.Humidity,
		WindSpeedMS:  windMS,
		PressureHpa:  payload.Current.PressureMb,
		PrecipMm:     payload.Current.PrecipMm,
		Condition:    cond,
	}, nil
}

func mapWeatherAPICondition(text string) weather.Condition {
	switch {
	case text == "":
		return weather.ConditionUnknown
	case contains(text, "rain") || contains(text, "shower") || contains(text, "drizzle"):
		return weather.ConditionRain
	case contains(text, "snow") || contains(text, "sleet") || contains(text, "blizzard"):
		return weather.ConditionSnow
	case contains(text, "thunder") || contains(text, "storm"):
		return weather.ConditionStorm
	case contains(text, "cloud"):
		return weather.ConditionCloudy
	case contains(text, "sunny") || contains(text, "clear"):
		return weather.ConditionClear
	default:
		return weather.ConditionUnknown
	}
}

func contains(s, sub string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(sub))
}


