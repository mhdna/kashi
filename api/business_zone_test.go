package api

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestDayStartIsBeirutMidnight(t *testing.T) {
	// Summer (UTC+3) and winter (UTC+2) both land on local midnight.
	summer := dayStart(time.Date(2026, 10, 13, 0, 0, 0, 0, time.UTC))
	require.Equal(t, time.Date(2026, 10, 12, 21, 0, 0, 0, time.UTC), summer.UTC())

	winter := dayStart(time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC))
	require.Equal(t, time.Date(2026, 11, 30, 22, 0, 0, 0, time.UTC), winter.UTC())
}

func TestBusinessTodayIsADate(t *testing.T) {
	today := businessToday()
	require.Equal(t, time.UTC, today.Location())
	require.Equal(t, today, today.Truncate(24*time.Hour))
	require.Equal(t, time.Now().In(businessZone).Format(time.DateOnly), today.Format(time.DateOnly))
}
