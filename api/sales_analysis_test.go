package api

import (
	"testing"
	"time"

	db "github.com/mhdna/kashi/db/sqlc"
	"github.com/stretchr/testify/require"
)

func day(s string) time.Time {
	d, _ := time.Parse("2006-01-02", s)
	return d
}

func TestTargetShare(t *testing.T) {
	// A $3,000 target for the 30 days of September (date_to inclusive) and
	// another branch's $310 for October.
	targets := []db.ReportTargetsInRangeRow{
		{BranchID: 1, DateFrom: day("2026-09-01"), DateTo: day("2026-09-30"), TargetAmount: 300_000},
		{BranchID: 2, DateFrom: day("2026-10-01"), DateTo: day("2026-10-31"), TargetAmount: 31_000},
	}

	got, ok := targetShare(targets, 0, day("2026-09-01"), day("2026-10-01"))
	require.True(t, ok)
	require.Equal(t, int64(300_000), got, "the whole of September")

	got, _ = targetShare(targets, 1, day("2026-09-01"), day("2026-09-11"))
	require.Equal(t, int64(100_000), got, "a third of the days, a third of the target")

	got, _ = targetShare(targets, 0, day("2026-09-21"), day("2026-10-11"))
	require.Equal(t, int64(100_000+10_000), got, "spans both")

	_, ok = targetShare(targets, 1, day("2026-10-01"), day("2026-11-01"))
	require.False(t, ok, "branch 1 has no target in October")
}
