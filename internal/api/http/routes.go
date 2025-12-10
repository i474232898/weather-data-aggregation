package httpapi

import (
	"errors"
	"strconv"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"

	"github.com/i474232898/weather-data-aggregation/internal/store"
	"github.com/i474232898/weather-data-aggregation/internal/weather"
)

var validate = validator.New()

// RegisterRoutes wires the HTTP handlers into the Fiber app.
func RegisterRoutes(app *fiber.App, service *weather.Service) {
	v1 := app.Group("/api/v1")

	v1.Get("/weather/current", func(c *fiber.Ctx) error {
		locReq, err := parseLocationQuery(c)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}

		loc := locReq.toLocation()
		snapshot, err := service.GetLatest(loc)
		if err != nil {
			if errors.Is(err, store.ErrNotFound) {
				return fiber.NewError(fiber.StatusNotFound, "no weather data for requested location")
			}
			return fiber.NewError(fiber.StatusInternalServerError, "failed to fetch weather data")
		}

		return c.JSON(snapshot)
	})

	v1.Get("/weather/history", func(c *fiber.Ctx) error {
		var req historyQuery
		if err := req.bind(c); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}

		if err := validate.Struct(req); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}

		loc := req.Location.toLocation()
		snapshots, err := service.GetRange(loc, req.From, req.To)
		if err != nil {
			if errors.Is(err, store.ErrNotFound) {
				return fiber.NewError(fiber.StatusNotFound, "no weather history for requested range")
			}
			return fiber.NewError(fiber.StatusInternalServerError, "failed to fetch weather history")
		}

		return c.JSON(fiber.Map{
			"location":  loc,
			"from":      req.From,
			"to":        req.To,
			"snapshots": snapshots,
		})
	})
}

// locationQuery holds query parameters for identifying a location.
type locationQuery struct {
	City    string `validate:"required"`
	Country string `validate:"required"`
}

func (l locationQuery) toLocation() weather.Location {
	return weather.Location{
		City:    l.City,
		Country: l.Country,
	}
}

func parseLocationQuery(c *fiber.Ctx) (locationQuery, error) {
	var q locationQuery

	q.City = c.Query("city")
	q.Country = c.Query("country")

	if err := validate.Struct(q); err != nil {
		return q, err
	}

	return q, nil
}

// historyQuery holds query parameters for the history endpoint.
type historyQuery struct {
	Location locationQuery
	From     time.Time `validate:"required"`
	To       time.Time `validate:"required,gtefield=From"`
}

func (h *historyQuery) bind(c *fiber.Ctx) error {
	loc, err := parseLocationQuery(c)
	if err != nil {
		return err
	}
	h.Location = loc

	fromStr := c.Query("from")
	toStr := c.Query("to")
	if fromStr == "" || toStr == "" {
		return errors.New("from and to query parameters are required")
	}

	from, err := parseTime(fromStr)
	if err != nil {
		return err
	}
	to, err := parseTime(toStr)
	if err != nil {
		return err
	}

	h.From = from
	h.To = to
	return nil
}

// parseTime tries to parse either RFC3339 or Unix seconds.
func parseTime(s string) (time.Time, error) {
	if ts, err := time.Parse(time.RFC3339, s); err == nil {
		return ts, nil
	}
	if unix, err := strconv.ParseInt(s, 10, 64); err == nil {
		return time.Unix(unix, 0).UTC(), nil
	}
	return time.Time{}, errors.New("invalid time format; use RFC3339 or unix seconds")
}
