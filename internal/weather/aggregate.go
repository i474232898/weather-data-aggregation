package weather

import "time"

// AggregateReadings combines multiple provider readings into a single WeatherSnapshot.
// Numeric fields are averaged; conditions are selected by majority (or first if tied).
func AggregateReadings(loc Location, readings []ProviderReading) WeatherSnapshot {
	if len(readings) == 0 {
		return WeatherSnapshot{
			Location:  loc,
			Timestamp: time.Now().UTC(),
			Condition: ConditionUnknown,
		}
	}

	var (
		sumTemp     float64
		sumHumidity float64
		sumWind     float64
		sumPressure float64
		sumPrecip   float64
	)

	conditionCounts := make(map[Condition]int)
	providers := make([]ProviderContribution, 0, len(readings))
	var newestTS time.Time

	for _, r := range readings {
		sumTemp += r.TemperatureC
		sumHumidity += r.HumidityPct
		sumWind += r.WindSpeedMS
		sumPressure += r.PressureHpa
		sumPrecip += r.PrecipMm

		conditionCounts[r.Condition]++

		if r.Timestamp.After(newestTS) {
			newestTS = r.Timestamp
		}

		providers = append(providers, ProviderContribution{
			ProviderName: r.ProviderName,
			Timestamp:    r.Timestamp,
		})
	}

	n := float64(len(readings))

	// Pick majority condition.
	bestCond := ConditionUnknown
	bestCount := 0
	for cond, count := range conditionCounts {
		if count > bestCount {
			bestCount = count
			bestCond = cond
		}
	}

	if newestTS.IsZero() {
		newestTS = time.Now().UTC()
	}

	return WeatherSnapshot{
		Location:    loc,
		Timestamp:   newestTS,
		Temperature: sumTemp / n,
		Humidity:    sumHumidity / n,
		WindSpeed:   sumWind / n,
		Pressure:    sumPressure / n,
		PrecipMM:    sumPrecip / n,
		Condition:   bestCond,
		Providers:   providers,
	}
}
