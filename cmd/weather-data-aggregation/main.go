package main

import (
	"context"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"

	httpapi "github.com/i474232898/weather-data-aggregation/internal/api/http"
	"github.com/i474232898/weather-data-aggregation/internal/config"
	"github.com/i474232898/weather-data-aggregation/internal/scheduler"
	"github.com/i474232898/weather-data-aggregation/internal/store"
	"github.com/i474232898/weather-data-aggregation/internal/weather"
	"github.com/i474232898/weather-data-aggregation/internal/weather/providers"
)

func main() {
	// Load configuration.
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	// Shared HTTP client for outbound provider calls.
	httpClient := &http.Client{
		Timeout: time.Duration(10 * time.Second),
	}

	// In-memory store with configured retention.
	memStore := store.NewMemoryStore(cfg.StoreMaxHistory, cfg.StoreMaxAge)

	// Providers with resilience (backoff + circuit breaker).
	var provs []weather.Provider

	provs = append(provs, providers.NewOpenWeatherProvider(httpClient, cfg.OpenWeatherAPIKey))
	provs = append(provs, providers.NewWeatherAPIProvider(httpClient, cfg.WeatherAPIKey))

	// Open-Meteo does not require an API key, but geocoding requires a Google API key.
	// provs = append(provs, providers.NewOpenMeteoProvider(httpClient, cfg.GeocoderAPIKey))

	// Core service orchestrating providers and store.
	service := weather.NewService(memStore, provs)

	// Scheduler that periodically fetches and stores data.
	sched := scheduler.New(cfg.Locations, cfg.FetchInterval, service)
	if err := sched.Start(); err != nil {
		log.Fatalf("failed to start scheduler: %v", err)
	}
	defer sched.Stop()

	app := fiber.New(fiber.Config{
		AppName:               "weather-data-aggregation",
		DisableStartupMessage: true,
		ReadTimeout:           10 * time.Second,
		WriteTimeout:          10 * time.Second,
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			// Centralized error response
			code := fiber.StatusInternalServerError
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}
			return c.Status(code).JSON(fiber.Map{
				"error":   true,
				"message": err.Error(),
			})
		},
	})

	// Global middleware
	app.Use(logger.New())
	app.Use(recover.New())

	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":  "ok",
			"service": "weather-data-aggregation",
		})
	})

	// API routes.
	httpapi.RegisterRoutes(app, service)

	go func() {
		if err := app.Listen(":" + cfg.Port); err != nil {
			log.Printf("fiber server stopped: %v", err)
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	<-ctx.Done()
	log.Println("Received termination signal, shutting down.")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := app.ShutdownWithContext(shutdownCtx); err != nil {
		log.Printf("error during shutdown: %v", err)
	}
}
