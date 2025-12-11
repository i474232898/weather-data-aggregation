# Weather Data Aggregation Service

A Go service that periodically fetches weather data from multiple public APIs, aggregates the information, and exposes it through a REST API using the Fiber framework.

## Overview

This service collects weather data from multiple weather API providers concurrently, aggregates the results to improve accuracy and reliability, and stores snapshots in memory with configurable retention policies. Built with Fiber v2, it provides a clean REST API for querying current weather, historical data, and forecasts.

## Features

### Core Functionality

✅ **Multi-Provider Aggregation**: Fetches data from 2+ different weather APIs simultaneously
   - OpenWeatherMap (fully implemented with current and forecast)
   - WeatherAPI.com (fully implemented with current and forecast)
   - Open-Meteo (not implemented yet)

✅ **Scheduled Data Collection**: Fetches weather data every 15 minutes (configurable) using `gocron`

✅ **Data Storage & Aggregation**: 
   - In-memory storage with configurable retention policies
   - Aggregates data from multiple sources (averages temperatures, merges forecasts)
   - Numeric values are averaged across providers
   - Weather conditions determined by majority vote

✅ **REST API**: Clean Fiber-based API endpoints for querying aggregated data

### Technical Implementation

#### Fiber Framework Features

✅ **Fiber v2**: Using latest Fiber framework with performance optimizations

✅ **Middleware**:
   - Logger middleware for request logging
   - Recover middleware for panic recovery
   - CORS middleware with configurable origins

✅ **Route Groups**: API versioning using `/api/v1` route group

✅ **Request Validation**: Using `go-playground/validator/v10` with Fiber context integration

✅ **Error Handling**: Custom error handler providing consistent JSON error responses

✅ **JSON Serialization**: Using Fiber's built-in JSON methods

✅ **Graceful Shutdown**: Implements `ShutdownWithContext()` with timeout handling

#### API Integration

✅ **Concurrent API Calls**: Fetches from all providers simultaneously using goroutines

✅ **Resilience Patterns**:
   - Circuit breaker pattern using `sony/gobreaker` to prevent cascading failures
   - Exponential backoff retries (500ms initial, max 5s, 3 retries)
   - Rate limit detection (HTTP 429 handling)
   - Context cancellation for timeout handling

✅ **Response Normalization**: Parses and normalizes different API response formats into unified data model

✅ **Graceful Failure Handling**: Continues with partial provider data if some providers fail

#### Data Processing

✅ **Data Aggregation**: 
   - Averages numeric fields (temperature, humidity, wind speed, pressure, precipitation)
   - Majority voting for weather conditions
   - Provider contribution tracking

✅ **Data Validation**: Request validation with proper error messages

✅ **Historical Data Storage**: In-memory store with retention policies (max snapshots, max age)

#### Scheduling

✅ **Robust Scheduler**: Uses `gocron` library for periodic task execution

✅ **Non-Overlapping Tasks**: Uses WaitGroup to ensure tasks complete before next execution starts

✅ **Execution Logging**: Logs execution times and errors for monitoring

## API Endpoints

All endpoints are under the `/api/v1` route group as per Fiber best practices.

### Health Check

```
GET /health
```

Returns service health status. *(Note: Currently returns basic status. Last successful fetch times can be added in future enhancement)*

**Example Request:**
```bash
curl http://localhost:8080/health
```

**Response:**
```json
{
  "status": "ok",
  "service": "weather-data-aggregation"
}
```

### Current Weather

```
GET /api/v1/weather/current?city={city_name}&country={country_code}
```

Returns the latest aggregated weather snapshot for the specified location.

**Query Parameters:**
- `city` (required): City name (e.g., "Prague", "London")
- `country` (required): Country code (e.g., "US", "UA", "TH", "CZ", "GB")

**Example Request:**
```bash
curl "http://localhost:8080/api/v1/weather/current?city=Prague&country=CZ"
```

**Response:**
```json
{
  "location": {
    "city": "Prague",
    "country": "CZ"
  },
  "timestamp": "2024-01-15T12:00:00Z",
  "temperatureC": 15.5,
  "humidityPercent": 65.0,
  "windSpeed": 3.2,
  "pressureHpa": 1013.25,
  "precipMm": 0.0,
  "condition": "cloudy",
  "providers": [
    {
      "provider": "openweathermap",
      "timestamp": "2024-01-15T12:00:00Z"
    },
    {
      "provider": "weatherapi",
      "timestamp": "2024-01-15T12:00:00Z"
    }
  ]
}
```

### Weather Forecast

```
GET /api/v1/weather/forecast?city={city_name}&country={country_code}&days={1-7}
```

Returns aggregated multi-day forecast data from providers that support forecasts. The `days` parameter is validated to be between 1-7.

**Query Parameters:**
- `city` (required): City name
- `country` (required): Country code
- `days` (required): Number of days (1-7, validated)

**Example Request:**
```bash
curl "http://localhost:8080/api/v1/weather/forecast?city=London&country=GB&days=5"
```

**Response:**
```json
{
  "location": {
    "city": "London",
    "country": "GB"
  },
  "days": 5,
  "forecast": [
    {
      "location": { "city": "London", "country": "GB" },
      "timestamp": "2024-01-16T00:00:00Z",
      "temperatureC": 14.2,
      "humidityPercent": 68.0,
      "windSpeed": 4.1,
      "pressureHpa": 1015.0,
      "precipMm": 0.5,
      "condition": "rain",
      "providers": [...]
    },
    ...
  ]
}
```

### Weather History

```
GET /api/v1/weather/history?city={city_name}&country={country_code}&from={timestamp}&to={timestamp}
```

Returns historical weather snapshots for a location within a specified time range.

**Query Parameters:**
- `city` (required): City name
- `country` (required): Country code
- `from` (required): Start timestamp (RFC3339 format or Unix seconds)
- `to` (required): End timestamp (RFC3339 format or Unix seconds)

**Example Request:**
```bash
# Using RFC3339 format
curl "http://localhost:8080/api/v1/weather/history?city=NewYork&country=US&from=2024-01-15T00:00:00Z&to=2024-01-15T23:59:59Z"

# Using Unix timestamp
curl "http://localhost:8080/api/v1/weather/history?city=NewYork&country=US&from=1705276800&to=1705363199"
```

**Response:**
```json
{
  "location": {
    "city": "NewYork",
    "country": "US"
  },
  "from": "2024-01-15T00:00:00Z",
  "to": "2024-01-15T23:59:59Z",
  "snapshots": [
    {
      "location": { "city": "NewYork", "country": "US" },
      "timestamp": "2024-01-15T00:00:00Z",
      "temperatureC": 12.3,
      "humidityPercent": 70.0,
      "windSpeed": 2.5,
      "pressureHpa": 1012.0,
      "precipMm": 0.0,
      "condition": "clear",
      "providers": [...]
    },
    ...
  ]
}
```

## Configuration

Configuration is managed through environment variables. Create a `.env` file in the project root or set environment variables directly.

### Environment Variables

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `OPENWEATHER_API_KEY` | API key for OpenWeatherMap | - | Yes* |
| `WEATHERAPI_API_KEY` | API key for WeatherAPI.com | - | Yes* |
| `FETCH_INTERVAL` | Interval between scheduled fetches (e.g., "15m", "1h") | `15m` | No |
| `STORE_MAX_HISTORY` | Maximum number of snapshots per location | `96` | No |
| `STORE_MAX_AGE` | Maximum age of stored snapshots (e.g., "24h", "7d") | `24h` | No |
| `WEATHER_LOCATION_CITY` | Comma-separated list of cities | - | Yes |
| `WEATHER_LOCATION_COUNTRY` | Comma-separated list of country codes (must match cities count) | - | Yes |
| `PORT` | HTTP server port | `8080` | No |

\* At least one API key is required for the service to function.

### Example `.env` File

```env
# API Keys (at least one required)
OPENWEATHER_API_KEY=your_openweather_api_key_here
WEATHERAPI_API_KEY=your_weatherapi_key_here

# Scheduler Configuration
FETCH_INTERVAL=15m

# Storage Configuration
STORE_MAX_HISTORY=96
STORE_MAX_AGE=24h

# Locations to Track (cities and countries must match count)
WEATHER_LOCATION_CITY=Prague,London,NewYork
WEATHER_LOCATION_COUNTRY=CZ,GB,US

# Server Configuration
PORT=8080
```

**Note:** The number of cities and countries must match. For multiple locations, provide comma-separated lists where each position corresponds.

## Setup Instructions

### Prerequisites

- Go 1.21 or later
- API keys for weather providers:
  - [OpenWeatherMap](https://openweathermap.org/api) (free tier available)
  - [WeatherAPI.com](https://www.weatherapi.com/) (free tier available)

### Installation

1. Clone the repository:
```bash
git clone <repository-url>
cd weather-data-aggregation
```

2. Install dependencies:
```bash
go mod download
```

3. Create a `.env` file with your configuration (see example above)

4. Build the service:
```bash
go build -o weather-service ./cmd/weather-data-aggregation
```

### Running the Service

**Option 1: Run the compiled binary**
```bash
./weather-service
```

**Option 2: Run directly with Go**
```bash
go run ./cmd/weather-data-aggregation/main.go
```

The service will:
- Start the HTTP server on the configured port (default: 8080)
- Begin scheduled data collection for configured locations every 15 minutes (or configured interval)
- Accept API requests immediately (initial fetch happens on startup)

### Graceful Shutdown

The service handles SIGINT and SIGTERM signals for graceful shutdown, allowing up to 10 seconds for in-flight requests to complete using Fiber's `ShutdownWithContext()` method.

## Architecture

### Project Structure

```
.
├── cmd/
│   └── weather-data-aggregation/
│       └── main.go              # Application entry point, Fiber app setup
├── internal/
│   ├── api/
│   │   └── http/
│   │       ├── routes.go        # HTTP route handlers with Fiber route groups
│   │       └── routes_test.go   # Route validation tests
│   ├── common/
│   │   └── utils.go             # Common utility functions
│   ├── config/
│   │   └── config.go            # Configuration management from env vars
│   ├── scheduler/
│   │   └── scheduler.go         # Periodic data fetching using gocron
│   ├── store/
│   │   └── memory.go            # Thread-safe in-memory storage implementation
│   └── weather/
│       ├── aggregate.go         # Data aggregation logic (averaging, voting)
│       ├── models.go            # Domain models (Location, WeatherSnapshot, etc.)
│       ├── provider.go          # Provider and Store interfaces
│       ├── service.go           # Core business logic orchestration
│       └── providers/
│           ├── common.go        # Shared resilience utilities (backoff, circuit breaker)
│           ├── openmeteo.go     # Open-Meteo provider implementation
│           ├── openweather.go   # OpenWeatherMap provider with forecast support
│           └── weatherapi.go    # WeatherAPI.com provider with forecast support
├── .env.example                 # Example environment configuration
├── go.mod                       # Go module definition
└── README.md                    # This file
```

### Key Components

1. **Main Application** (`cmd/weather-data-aggregation/main.go`):
   - Initializes Fiber app with middleware (logger, recover, CORS)
   - Sets up custom error handler for consistent responses
   - Configures graceful shutdown
   - Wires together all components

2. **Service Layer** (`internal/weather/service.go`):
   - Orchestrates concurrent data fetching from providers
   - Aggregates provider readings
   - Manages storage operations

3. **Providers** (`internal/weather/providers/`):
   - Interface-based design for extensibility
   - Each provider implements `Provider` interface
   - Providers supporting forecasts implement `ForecastProvider` interface
   - Resilience patterns: circuit breaker + exponential backoff

4. **Store** (`internal/store/memory.go`):
   - Thread-safe in-memory implementation
   - Configurable retention policies
   - Fast lookups with location-based keys

5. **Scheduler** (`internal/scheduler/scheduler.go`):
   - Uses `gocron` for periodic execution
   - Ensures non-overlapping executions
   - Concurrent fetching for multiple locations

6. **HTTP API** (`internal/api/http/routes.go`):
   - Fiber route groups for versioning (`/api/v1`)
   - Request validation using `go-playground/validator`
   - Consistent error handling

### Architecture Decisions

#### Fiber Framework Choices

1. **Route Groups**: Used `/api/v1` route groups for clean API versioning, allowing future v2 without breaking changes

2. **Error Handler**: Custom error handler provides consistent JSON error responses across all endpoints, improving API usability

3. **Middleware Stack**:
   - Logger: Essential for debugging and monitoring in production
   - Recover: Prevents service crashes from unhandled panics
   - CORS: Enables web application integration

4. **Graceful Shutdown**: Using `ShutdownWithContext()` with timeout ensures clean shutdown without dropping requests

5. **Built-in JSON**: Leveraging Fiber's optimized JSON serialization for performance

#### Resilience Patterns

1. **Circuit Breaker**: Prevents cascading failures when providers are down, reducing unnecessary load and improving response times

2. **Exponential Backoff**: Handles transient failures and rate limiting gracefully without overwhelming providers

3. **Partial Success Strategy**: Service continues operating even if some providers fail, improving availability

#### Data Storage

1. **In-Memory Storage**: Chosen for simplicity and performance. Trade-off is data loss on restart, acceptable for this use case

2. **Retention Policies**: Both count-based and time-based retention prevent unbounded memory growth

#### Concurrency

1. **Concurrent Provider Fetching**: Using goroutines with WaitGroup ensures parallel API calls while maintaining synchronization

2. **Thread-Safe Store**: Mutex-based locking ensures safe concurrent access from scheduler and API handlers

## Weather Conditions

The service normalizes weather conditions across providers into the following standardized categories:

- `clear`: Clear skies
- `cloudy`: Cloudy conditions
- `rain`: Rainy conditions (includes drizzle, showers)
- `snow`: Snowy conditions (includes sleet, blizzard)
- `storm`: Stormy/thunderstorm conditions
- `mist`: Mist/fog/haze conditions
- `unknown`: Unrecognized or unavailable condition

## Data Flow

1. **Scheduled Collection**: Scheduler (gocron) triggers `FetchAndStore()` at configured intervals (default: 15 minutes)

2. **Concurrent Fetching**: Service launches goroutines to fetch from all providers simultaneously for each location

3. **Resilience Layer**: Each provider request goes through:
   - Circuit breaker check
   - HTTP request with timeout
   - Retry logic with exponential backoff on failure

4. **Aggregation**: Successful readings are aggregated:
   - Numeric fields (temperature, humidity, etc.) are averaged
   - Weather conditions are determined by majority vote
   - Provider metadata is tracked for traceability

5. **Storage**: Aggregated snapshot is stored in memory store with automatic retention policy enforcement

6. **API Access**: HTTP endpoints query the store for current/historical/forecast data using Fiber's JSON serialization

## Error Handling

- **Provider Failures**: Logged but don't block aggregation if other providers succeed (graceful degradation)
- **No Successful Reads**: Last good snapshot is retained, not overwritten with empty data
- **API Errors**: Standardized JSON error responses with appropriate HTTP status codes via Fiber error handler
- **Configuration Errors**: Service fails fast at startup with clear error messages
- **Request Validation**: Invalid requests return 400 Bad Request with descriptive messages

## Implementation Status vs Requirements

### ✅ Fully Implemented

- [x] Fetch weather data from at least 2 different APIs
- [x] Schedule data fetching every 15 minutes (configurable)
- [x] Store and aggregate data from multiple sources
- [x] REST API using Fiber v2
- [x] Proper middleware (logger, recover, CORS)
- [x] Request validation using Fiber's context methods
- [x] Custom error handler for consistent error responses
- [x] Route groups for versioning (`/api/v1`)
- [x] Concurrent API calls to multiple weather services
- [x] Rate limiting and retries with exponential backoff
- [x] Circuit breaker pattern
- [x] Parse and normalize responses from different API formats
- [x] Aggregate data (average temperatures, merge forecasts)
- [x] In-memory storage with retention policies
- [x] Robust task scheduler (gocron) with non-overlapping execution
- [x] Log execution times and errors
- [x] Graceful shutdown using Fiber's `ShutdownWithContext()`
- [x] Environment variable configuration

### ⚠️ Partially Implemented / Limitations

- [~] Health endpoint: Basic implementation exists, but doesn't show "last successful API fetch times" (can be enhanced)
- [~] Open-Meteo provider: Implemented but commented out (requires geocoding API key)
- [~] Persistent storage: Only in-memory storage (acceptable per requirements)
