package api

import (
	"math"
	"time"
)

// A deliberately simple sales forecast for the Financials page, built only
// from past daily sales so anyone can follow what it does:
//
//	forecast(day) = level × weekday factor × season
//
//   - level: the last 28 days' average daily sales, with the weekday pattern
//     taken out (so a window that happens to hold five Saturdays doesn't
//     read as growth).
//   - weekday factor: how each weekday sold against the average over the
//     last 12 weeks (Saturdays at 1.4 means Saturdays sell 40% above a
//     typical day).
//   - season: once a year of history exists, how last year's sales around
//     the same date compared with last year's sales over the level's window
//     -- last year's run-up to a holiday becomes this year's. Kept between
//     ⅓× and 3× so one odd day last year can't run away with it.
//
// The range around the forecast comes from how far off the same model was
// over the last 8 weeks (an 80% band, ±1.28 standard deviations of the
// relative error), widening the further ahead the day is.

const (
	forecastLevelDays   = 28
	forecastWeekdayDays = 84
	forecastErrorDays   = 56
	forecastBandZ       = 1.28
)

type forecastPoint struct {
	Forecast int64
	Low      int64
	High     int64
}

// salesForecaster holds daily sales (cents) indexed by day from `start`; a
// missing day is a day with no sales.
type salesForecaster struct {
	start time.Time
	sales []float64
}

func newSalesForecaster(start time.Time, byDay map[string]int64, until time.Time) *salesForecaster {
	f := &salesForecaster{start: start}
	for d := start; d.Before(until); d = d.AddDate(0, 0, 1) {
		f.sales = append(f.sales, float64(byDay[d.Format("2006-01-02")]))
	}
	return f
}

func (f *salesForecaster) index(day time.Time) int {
	return int(day.Sub(f.start).Hours() / 24)
}

// window is the sales of the n days before index i that are on record.
func (f *salesForecaster) window(i, n int) []float64 {
	lo := max(0, i-n)
	hi := min(i, len(f.sales))
	if hi <= lo {
		return nil
	}
	return f.sales[lo:hi]
}

func mean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	var sum float64
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

// predict forecasts `day` from what was known before `cutoff` (an index).
func (f *salesForecaster) predict(day time.Time, cutoff int) float64 {
	// Weekday factors over the last 12 weeks.
	weekStart := max(0, cutoff-forecastWeekdayDays)
	var sums, counts [7]float64
	for i := weekStart; i < cutoff && i < len(f.sales); i++ {
		wd := f.start.AddDate(0, 0, i).Weekday()
		sums[wd] += f.sales[i]
		counts[wd]++
	}
	overall := mean(f.window(cutoff, forecastWeekdayDays))
	factor := func(wd time.Weekday) float64 {
		if overall <= 0 || counts[wd] == 0 {
			return 1
		}
		return sums[wd] / counts[wd] / overall
	}

	// Level: the last 28 days with the weekday pattern divided out.
	levelStart := max(0, cutoff-forecastLevelDays)
	var level float64
	var n int
	for i := levelStart; i < cutoff && i < len(f.sales); i++ {
		if w := factor(f.start.AddDate(0, 0, i).Weekday()); w > 0 {
			level += f.sales[i] / w
			n++
		}
	}
	if n == 0 {
		return 0
	}
	level /= float64(n)

	// Season from last year, when there is one.
	season := 1.0
	lastYearDay := f.index(day.AddDate(-1, 0, 0))
	lastYearCutoff := cutoff - (f.index(day) - lastYearDay)
	if lastYearCutoff-forecastLevelDays >= 0 {
		around := mean(f.window(lastYearDay+forecastLevelDays/2, forecastLevelDays))
		before := mean(f.window(lastYearCutoff, forecastLevelDays))
		if before > 0 && around > 0 {
			season = math.Min(3, math.Max(1.0/3, around/before))
		}
	}

	return level * factor(day.Weekday()) * season
}

// forecastRange forecasts every day in [from, to). Days before `today`
// are each forecast from the days before them (what the model would have
// said that morning), days from `today` on from everything before today.
func (f *salesForecaster) forecastRange(from, to, today time.Time) map[string]forecastPoint {
	todayIdx := f.index(today)

	// How far off one-day-ahead forecasts were over the last 8 weeks.
	var errs []float64
	for i := max(0, todayIdx-forecastErrorDays); i < todayIdx && i < len(f.sales); i++ {
		if p := f.predict(f.start.AddDate(0, 0, i), i); p > 0 {
			errs = append(errs, f.sales[i]/p-1)
		}
	}
	m := mean(errs)
	var variance float64
	for _, e := range errs {
		variance += (e - m) * (e - m)
	}
	sigma := 0.25 // until there's a track record
	if len(errs) > 1 {
		sigma = math.Sqrt(variance / float64(len(errs)-1))
	}

	out := map[string]forecastPoint{}
	for d := from; d.Before(to); d = d.AddDate(0, 0, 1) {
		i := f.index(d)
		cutoff, ahead := i, 0
		if i >= todayIdx {
			cutoff, ahead = todayIdx, i-todayIdx
		}
		p := f.predict(d, cutoff)
		spread := p * forecastBandZ * sigma * math.Sqrt(1+float64(ahead)/forecastLevelDays)
		out[d.Format("2006-01-02")] = forecastPoint{
			Forecast: int64(math.Round(p)),
			Low:      int64(math.Round(math.Max(0, p-spread))),
			High:     int64(math.Round(p + spread)),
		}
	}
	return out
}
