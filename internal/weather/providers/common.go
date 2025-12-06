package providers

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net/http"
	"time"

	"github.com/sony/gobreaker"
)

// BackoffConfig controls exponential backoff behaviour.
type BackoffConfig struct {
	MaxRetries      int
	InitialInterval time.Duration
	MaxInterval     time.Duration
}

// HTTPClientConfig bundles HTTP client and resilience settings.
type HTTPClientConfig struct {
	Client  *http.Client
	Backoff BackoffConfig
}

var (
	errRateLimited   = errors.New("rate limited")
	errServerError   = errors.New("server error")
	errUnexpected    = errors.New("unexpected status code")
	errCircuitOpen   = errors.New("circuit breaker open")
	errNoHTTPClient  = errors.New("http client not configured")
	errInvalidConfig = errors.New("invalid backoff configuration")
)

// doRequestWithResilience executes the HTTP request with retries, exponential backoff,
// and a circuit breaker.
func doRequestWithResilience(
	ctx context.Context,
	cfg HTTPClientConfig,
	cb *gobreaker.CircuitBreaker,
	buildRequest func() (*http.Request, error),
) (*http.Response, error) {
	if cfg.Client == nil {
		return nil, errNoHTTPClient
	}
	if cfg.Backoff.MaxRetries < 0 || cfg.Backoff.InitialInterval <= 0 {
		return nil, errInvalidConfig
	}

	var attempt int
	var lastErr error

	for {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		req, err := buildRequest()
		if err != nil {
			return nil, err
		}

		// Ensure the request obeys context cancellation.
		req = req.WithContext(ctx)

		result, err := cb.Execute(func() (interface{}, error) {
			resp, execErr := cfg.Client.Do(req)
			if execErr != nil {
				return nil, execErr
			}

			// Handle rate limiting and server errors explicitly.
			if resp.StatusCode == http.StatusTooManyRequests {
				return nil, errRateLimited
			}
			if resp.StatusCode >= 500 {
				return nil, errServerError
			}
			if resp.StatusCode < 200 || resp.StatusCode >= 300 {
				return nil, fmt.Errorf("%w: %d", errUnexpected, resp.StatusCode)
			}

			return resp, nil
		})

		if err == nil {
			// Success.
			resp, ok := result.(*http.Response)
			if !ok {
				return nil, fmt.Errorf("unexpected result type from circuit breaker")
			}
			return resp, nil
		}

		// If circuit is open, propagate immediately.
		if errors.Is(err, gobreaker.ErrOpenState) || errors.Is(err, gobreaker.ErrTooManyRequests) {
			return nil, fmt.Errorf("%w: %v", errCircuitOpen, err)
		}

		lastErr = err
		if attempt >= cfg.Backoff.MaxRetries {
			return nil, lastErr
		}

		// Backoff with exponential delay.
		delay := cfg.Backoff.InitialInterval * time.Duration(math.Pow(2, float64(attempt)))
		if delay > cfg.Backoff.MaxInterval && cfg.Backoff.MaxInterval > 0 {
			delay = cfg.Backoff.MaxInterval
		}

		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, ctx.Err()
		case <-timer.C:
			// continue to next attempt
		}

		attempt++
	}
}
