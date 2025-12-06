<!-- 861f5a6d-882c-46c8-b40b-f1ee49414334 67a345f3-7f4c-46a3-8644-3f7a959cb400 -->
# Weather Aggregation Service Plan

## 1. Overall Architecture

- **Language & framework**: Go with Fiber v2 for the HTTP server.
- **Process responsibilities**:
- Periodically fetch weather data from OpenWeatherMap, WeatherAPI, and Open-Meteo.
- Normalize and aggregate provider-specific responses into a unified internal model.
- Store aggregated snapshots in an in-memory, concurrency-safe store (data is lost on restart).
- Expose REST endpoints for current and historical weather per location.
- **Key packages (proposed)**:
- `cmd/weather-data-aggregation/main.go`: application entrypoint, wiring, and startup.
- `internal/config`: configuration (API keys, rate limits, schedule interval, locations, etc.).
- `internal/weather`: domain models, provider interfaces, and aggregation logic.
- `internal/weather/providers`: concrete clients for OpenWeatherMap, WeatherAPI, and Open-Meteo.
- `internal/store`: in-memory storage for aggregated snapshots.
- `internal/scheduler`: job scheduling to trigger periodic fetches.
- `internal/api/http`: Fiber handlers, routing, and request/response DTOs.
- `internal/middleware`: logging, recovery, and request validation setup.

## 2. Domain Model & Data Normalization

- **Define core domain types** in `internal/weather`:
- `Location` (e.g. city, country, optional lat/lon) and a canonical key for indexing.
- `WeatherSnapshot` with normalized fields such as: temperature in °C, humidity %, wind speed (m/s or km/h), pressure, precipitation, conditions/description, observation time, and provider breakdown if needed.
- **Normalization rules**:
- Convert all temperature units to °C, wind speed to a single unit, and timestamps to UTC.
- Map provider-specific condition codes/strings to a small, common enum or string set (e.g. `clear`, `cloudy`, `rain`, `snow`).
- **Historical data**:
- Store a time-series of `WeatherSnapshot` per location in-memory (e.g. slice or ring buffer) ordered by timestamp.
- Optionally implement basic retention policy (e.g. keep last N hours/days) to avoid unbounded growth.

## 3. Provider Integrations (OpenWeatherMap, WeatherAPI, Open-Meteo)

- **Define provider interface** in `internal/weather`:
- `type Provider interface { Name() string; Fetch(ctx context.Context, loc Location) (ProviderReading, error) }`.
- `ProviderReading` is a raw-but-structured result that can be normalized into `WeatherSnapshot`.
- **Implement clients in `internal/weather/providers`**:
- `openweathermap`: implement API client using HTTP, include API key, units, language, and error handling.
- `weatherapi`: implement API client for current conditions with API key auth and response parsing.
- `openmeteo`: implement client for current weather variables matched to the common model.
- **HTTP client concerns**:
- Use a shared `http.Client` with sane timeouts and connection reuse.
- Make URLs, timeouts, and API keys configurable via `internal/config`.

## 4. Concurrency, Rate Limiting, and Resilience

- **Concurrent fetching**:
- For each scheduled run and each location, call all configured providers concurrently using `errgroup.Group` or goroutines + channels.
- Gather successful responses and ignore or log individual provider failures.
- **Rate limiting & retries with exponential backoff**:
- For each provider client, wrap the low-level HTTP call with a helper that:
- Detects rate-limit or transient errors (e.g. HTTP 429 and 5xx) and network timeouts.
- Retries with exponential backoff (e.g. base delay, multiplier, max attempts, optional jitter) respecting provider-specific rate limits from config.
- Ensure maximum total wait per request is bounded to avoid blocking the scheduler.
- **Circuit breaker (optional but preferred)**:
- Introduce a simple circuit breaker per provider (e.g. using a library like `sony/gobreaker` or a custom counter-based implementation).
- Open the circuit after a configurable number of consecutive failures and fail-fast for a cool-down period before allowing a limited number of trial requests.
- **Graceful degradation**:
- If some providers fail but at least one succeeds, still produce an aggregated snapshot with partial data and log missing providers.
- If all providers fail for a location, record the failure and avoid overwriting the last good snapshot (so queries can still see the last known data plus a flag).

## 5. Aggregation Logic

- **Aggregation strategy** (in `internal/weather/aggregation` or similar):
- Combine numeric fields (temperature, humidity, wind speed) using a simple policy (e.g. average of all successful providers or weighted average if needed later).
- For categorical fields (conditions/description), pick the most frequent or prefer a primary provider when there is disagreement.
- Include metadata about which providers contributed and when.
- **Output**:
- Produce a single `WeatherSnapshot` per location per run, which is then stored in the in-memory store.

## 6. In-Memory Storage Layer

- **Store design** in `internal/store`:
- Implement a thread-safe store using `sync.RWMutex`.
- Maintain a map from `LocationKey` to a struct containing:
- A slice/time-series of `WeatherSnapshot` (sorted by timestamp).
- Optional configuration for max history length or duration.
- **APIs**:
- `SaveSnapshot(loc Location, snapshot WeatherSnapshot)` appends a new snapshot, enforces retention, and updates `LastUpdated` metadata.
- `GetLatest(loc Location) (WeatherSnapshot, error)` returns the newest snapshot.
- `GetRange(loc Location, from, to time.Time) ([]WeatherSnapshot, error)` returns a slice for historical queries.

## 7. Scheduler and Periodic Fetching

- **Scheduling mechanism** in `internal/scheduler`:
- Use a Go scheduling library like `github.com/go-co-op/gocron` or `github.com/robfig/cron/v3` to run jobs every 15 minutes.
- Configure the scheduler with the set of locations to track (from config).
- **Job execution flow**:
- On each tick, for each configured location:
- Spawn concurrent provider fetches with rate-limited, backoff-enabled clients.
- Aggregate the results into a `WeatherSnapshot`.
- Persist snapshot into the in-memory store.
- Ensure jobs respect context cancellation on shutdown so the service can exit cleanly.

## 8. REST API Design (Fiber v2)

- **App setup** (in `main.go` and `internal/api/http`):
- Initialize Fiber v2 with custom error handler for consistent JSON error responses.
- Register global middleware: logging, recovery, and request ID/correlation ID if desired.
- Wire handler dependencies: in-memory store, aggregation service, and any configuration.
- **Request validation using Fiber**:
- Define request DTO structs with `validate` tags (e.g. for query parameters and JSON bodies).
- Integrate Fiber’s validation hook (using `go-playground/validator` wired into Fiber’s validator support) so handlers can call a simple `ctx.Validate(data)` or shared helper and return standardized 400 responses on validation failures.
- **Endpoints (v1)**:
- `GET /health` – simple health check.
- `GET /api/v1/weather/current` – query current aggregated weather for a location.
- Query params: `city` (required), optional `country`, or alternatively `lat` and `lon`.
- Validate that either city or lat/lon is provided and well-formed.
- `GET /api/v1/weather/history` – query historical snapshots for a location.
- Query params: same location parameters plus `from` and `to` timestamps (e.g. ISO8601 or Unix seconds).
- Validate ranges (`from <= to`, not too far in the past relative to retention).
- **Responses**:
- Return JSON using a unified schema: `location`, `timestamp`, `temperatureC`, `humidity`, `windSpeed`, `condition`, and optional `providers` metadata.
- For history, return a list of snapshots plus paging info if needed later.

## 9. Middleware, Logging, and Error Handling

- **Middleware** (in `internal/middleware`):
- Configure Fiber’s logger middleware for structured request logs.
- Configure Fiber’s recover middleware to handle panics and return a safe error response.
- Optionally add a simple request ID middleware and attach ID to logs.
- **Error handling**:
- Implement a centralized error-handling strategy, mapping domain/storage errors to HTTP status codes (e.g. 404 for missing location data, 500 for internal errors).
- Ensure provider and scheduler errors are logged with enough context (provider name, location key), but not exposed directly to clients.

## 10. Configuration and Secrets

- **Config source**:
- Add a configuration struct in `internal/config` with fields for API keys, base URLs, retry/backoff parameters, schedule interval (default 15 minutes), and tracked locations.
- Load configuration from environment variables and/or a small config file, with defaults for non-sensitive values.
- **Secrets**:
- Read API keys for OpenWeatherMap, WeatherAPI, and Open-Meteo from environment variables.
- Document required env vars in a `README.md`.

## 11. Startup & Shutdown Flow

- **Startup** (in `main.go`):
- Load configuration and validate it.
- Initialize HTTP client(s), provider clients, store, aggregation service, and scheduler.
- Start scheduler, then start Fiber HTTP server (on a configurable port).
- **Shutdown**:
- Handle OS signals (e.g. SIGINT, SIGTERM) to initiate graceful shutdown.
- Stop accepting new HTTP requests, stop the scheduler, and wait for in-flight jobs to finish (with timeout).

## 12. Testing Strategy (High-Level)

- **Unit tests**:
- Test provider normalization logic with canned JSON responses for each API.
- Test aggregation strategies (averaging, condition selection) with controlled inputs.
- Test the in-memory store’s concurrency behavior and retention logic.
- **Integration tests**:
- Use mocked HTTP servers for the three providers to simulate normal, rate-limited, and error scenarios, verifying backoff and circuit breaker behavior.
- Test REST endpoints using Fiber’s testing utilities to validate request validation, routing, and responses.

## 13. Future Enhancements (Optional)

- Swap in-memory store for a persistent DB (e.g. PostgreSQL) transparently behind the store interface.
- Add authentication/authorization for APIs if exposed beyond trusted networks.
- Add more providers or support for forecast data in addition to current conditions.
- Expose metrics (Prometheus) for provider success rates, latency, scheduler runs, and HTTP handler performance.

### To-dos

- [ ] Set up Go module, folder structure, and basic `main.go` with Fiber v2, logging, and recovery middleware wired.
- [ ] Define domain models (locations, weather snapshots) and implement the in-memory, concurrency-safe storage layer.
- [ ] Implement provider interfaces and concrete clients for OpenWeatherMap, WeatherAPI, and Open-Meteo with normalization logic.
- [ ] Add concurrency, retry with exponential backoff, and a simple circuit breaker around provider calls, plus aggregation logic.
- [ ] Introduce a scheduler that runs every 15 minutes and orchestrates per-location data fetching and storage.
- [ ] Implement REST endpoints for current and historical weather, including request validation and error handling, using Fiber v2.
- [ ] Set up Go module, folder structure, and basic `main.go` with Fiber v2, logging, and recovery middleware wired.
- [ ] Define domain models (locations, weather snapshots) and implement the in-memory, concurrency-safe storage layer.
- [ ] Implement provider interfaces and concrete clients for OpenWeatherMap, WeatherAPI, and Open-Meteo with normalization logic.
- [ ] Add concurrency, retry with exponential backoff, and a simple circuit breaker around provider calls, plus aggregation logic.
- [ ] Introduce a scheduler that runs every 15 minutes and orchestrates per-location data fetching and storage.
- [ ] Implement REST endpoints for current and historical weather, including request validation and error handling, using Fiber v2.