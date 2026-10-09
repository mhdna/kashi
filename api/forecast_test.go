package api

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// sales builds a day -> cents map from start until `until` with fn.
func salesSeries(start, until time.Time, fn func(time.Time) int64) map[string]int64 {
	out := map[string]int64{}
	for d := start; d.Before(until); d = d.AddDate(0, 0, 1) {
		out[d.Format("2006-01-02")] = fn(d)
	}
	return out
}

func TestForecastFollowsTheWeekdayPattern(t *testing.T) {
	start := day("2026-05-01")
	today := day("2026-09-01")
	// $100 a day, $200 on Saturdays, steady.
	byDay := salesSeries(start, today, func(d time.Time) int64 {
		if d.Weekday() == time.Saturday {
			return 20_000
		}
		return 10_000
	})
	f := newSalesForecaster(start, byDay, today)
	got := f.forecastRange(today, today.AddDate(0, 0, 14), today)

	require.InDelta(t, 20_000, got["2026-09-05"].Forecast, 50, "a Saturday")
	require.InDelta(t, 10_000, got["2026-09-07"].Forecast, 50, "a Monday")
	// A perfectly regular past leaves almost no error to spread.
	require.InDelta(t, got["2026-09-07"].Forecast, got["2026-09-07"].High, 50)
}

func TestForecastCarriesLastYearsSeason(t *testing.T) {
	start := day("2024-10-01")
	today := day("2025-11-01")
	// Flat $100 a day, except every December sells double.
	byDay := salesSeries(start, today, func(d time.Time) int64 {
		if d.Month() == time.December {
			return 20_000
		}
		return 10_000
	})
	f := newSalesForecaster(start, byDay, today)
	got := f.forecastRange(today, day("2026-01-01"), today)

	// Last year is averaged over the 4 weeks around the date, so the run-up
	// starts showing about two weeks before December.
	require.InDelta(t, 10_000, got["2025-11-05"].Forecast, 600, "early November, before the run-up")
	require.InDelta(t, 20_000, got["2025-12-15"].Forecast, 600, "mid-December doubles like last year")
}

func TestForecastBandWidensWithDistance(t *testing.T) {
	start := day("2026-05-01")
	today := day("2026-09-01")
	// Noisy: alternating $80 / $120 days.
	byDay := salesSeries(start, today, func(d time.Time) int64 {
		if d.YearDay()%2 == 0 {
			return 8_000
		}
		return 12_000
	})
	f := newSalesForecaster(start, byDay, today)
	got := f.forecastRange(today, today.AddDate(0, 0, 60), today)

	near := got["2026-09-01"]
	far := got["2026-10-30"]
	require.Greater(t, near.High, near.Forecast)
	require.Less(t, near.Low, near.Forecast)
	require.Greater(t, far.High-far.Low, near.High-near.Low)
	require.GreaterOrEqual(t, far.Low, int64(0))
}
