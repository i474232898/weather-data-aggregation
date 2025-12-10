package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/i474232898/weather-data-aggregation/internal/store"
	"github.com/i474232898/weather-data-aggregation/internal/weather"
)

// TestForecastDaysValidation verifies that the forecast endpoint enforces the
// expected 1-7 range for the `days` query parameter.
func TestForecastDaysValidation(t *testing.T) {
	app := fiber.New()

	memStore := store.NewMemoryStore(10, time.Hour)
	svc := weather.NewService(memStore, nil)
	RegisterRoutes(app, svc)

	// Missing days parameter should return 400.
	req := httptest.NewRequest(http.MethodGet, "/api/v1/weather/forecast?city=Paris&country=FR", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, resp.StatusCode)
	}

	// Out-of-range days value should also return 400.
	req = httptest.NewRequest(http.MethodGet, "/api/v1/weather/forecast?city=Paris&country=FR&days=8", nil)
	resp, err = app.Test(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, resp.StatusCode)
	}
}
