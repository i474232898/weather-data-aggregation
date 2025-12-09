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

// OpenWeatherProvider implements the weather.Provider interface for OpenWeatherMap.
type OpenWeatherProvider struct {
	name    string
	apiKey  string
	baseURL string
	httpCfg HTTPClientConfig
	circuit *gobreaker.CircuitBreaker
}

func NewOpenWeatherProvider(client *http.Client, apiKey string) *OpenWeatherProvider {
	cb := gobreaker.NewCircuitBreaker(gobreaker.Settings{
		Name:        "openweather",
		MaxRequests: 5,
		Interval:    1 * time.Minute,
		Timeout:     2 * time.Minute,
	})

	return &OpenWeatherProvider{
		name:    "openweathermap",
		apiKey:  apiKey,
		baseURL: "https://api.openweathermap.org/data/2.5/weather",
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

func (p *OpenWeatherProvider) Name() string {
	return p.name
}

func (p *OpenWeatherProvider) Fetch(ctx context.Context, loc weather.Location) (weather.ProviderReading, error) {
	if p.apiKey == "" {
		return weather.ProviderReading{}, fmt.Errorf("openweather api key is not configured")
	}

	buildRequest := func() (*http.Request, error) {
		values := url.Values{}
		values.Set("appid", p.apiKey)
		values.Set("units", "metric")

		// if loc.Lat != nil && loc.Lon != nil {
		// 	values.Set("lat", fmt.Sprintf("%f", *loc.Lat))
		// 	values.Set("lon", fmt.Sprintf("%f", *loc.Lon))
		// } else {
		// city,country
		q := loc.City
		if loc.Country != "" {
			q = fmt.Sprintf("%s,%s", loc.City, loc.Country)
		}
		values.Set("q", q)
		// }

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
		Dt   int64 `json:"dt"`
		Main struct {
			Temp     float64 `json:"temp"`
			Humidity float64 `json:"humidity"`
			Pressure float64 `json:"pressure"`
		} `json:"main"`
		Wind struct {
			Speed float64 `json:"speed"`
		} `json:"wind"`
		Rain struct {
			OneH   float64 `json:"1h"`
			ThreeH float64 `json:"3h"`
		} `json:"rain"`
		Weather []struct {
			Main string `json:"main"`
		} `json:"weather"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return weather.ProviderReading{}, err
	}

	ts := time.Unix(payload.Dt, 0).UTC()
	if ts.IsZero() {
		ts = time.Now().UTC()
	}

	precip := payload.Rain.OneH
	if precip == 0 {
		precip = payload.Rain.ThreeH
	}

	cond := mapOpenWeatherCondition(payload.Weather)

	return weather.ProviderReading{
		ProviderName: p.name,
		Timestamp:    ts,
		TemperatureC: payload.Main.Temp,
		HumidityPct:  payload.Main.Humidity,
		WindSpeedMS:  payload.Wind.Speed,
		PressureHpa:  payload.Main.Pressure,
		PrecipMm:     precip,
		Condition:    cond,
	}, nil
}

func mapOpenWeatherCondition(items []struct {
	Main string `json:"main"`
}) weather.Condition {
	if len(items) == 0 {
		return weather.ConditionUnknown
	}
	switch items[0].Main {
	case "Clear":
		return weather.ConditionClear
	case "Clouds":
		return weather.ConditionCloudy
	case "Rain", "Drizzle":
		return weather.ConditionRain
	case "Snow":
		return weather.ConditionSnow
	case "Thunderstorm":
		return weather.ConditionStorm
	default:
		return weather.ConditionUnknown
	}
}
