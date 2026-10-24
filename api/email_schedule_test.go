package api

import (
	"database/sql"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	mockdb "github.com/mhdna/kashi/db/mock"
	db "github.com/mhdna/kashi/db/sqlc"
	"github.com/stretchr/testify/require"
)

func beirut(y int, m time.Month, d, h, min int) time.Time {
	return time.Date(y, m, d, h, min, 0, 0, businessZone)
}

func TestNextEmailScheduleRun(t *testing.T) {
	daily := db.EmailSchedule{Frequency: "daily", AtMinute: 22 * 60}
	// Before today's time: today. At or after it: tomorrow.
	require.Equal(t, beirut(2026, 10, 13, 22, 0), nextEmailScheduleRun(daily, beirut(2026, 10, 13, 21, 59)))
	require.Equal(t, beirut(2026, 10, 14, 22, 0), nextEmailScheduleRun(daily, beirut(2026, 10, 13, 22, 0)))

	// Mondays and Fridays at 08:30; 2026-10-13 is a Tuesday.
	weekly := db.EmailSchedule{Frequency: "weekly", AtMinute: 8*60 + 30, Weekdays: []int32{1, 5}}
	require.Equal(t, beirut(2026, 10, 16, 8, 30), nextEmailScheduleRun(weekly, beirut(2026, 10, 13, 9, 0)))
	require.Equal(t, beirut(2026, 10, 19, 8, 30), nextEmailScheduleRun(weekly, beirut(2026, 10, 16, 8, 30)))

	// The 31st: November has 30 days, so its run is on the 30th.
	monthly := db.EmailSchedule{Frequency: "monthly", AtMinute: 9 * 60, MonthDay: 31}
	require.Equal(t, beirut(2026, 10, 31, 9, 0), nextEmailScheduleRun(monthly, beirut(2026, 10, 13, 9, 0)))
	require.Equal(t, beirut(2026, 11, 30, 9, 0), nextEmailScheduleRun(monthly, beirut(2026, 10, 31, 9, 0)))
}

func TestEmailSchedulePeriodDays(t *testing.T) {
	day := func(y int, m time.Month, d int) time.Time { return time.Date(y, m, d, 0, 0, 0, 0, time.UTC) }
	run := day(2026, 10, 15) // a Thursday
	cases := map[string][2]time.Time{
		"today":       {run, run},
		"yesterday":   {day(2026, 10, 14), day(2026, 10, 14)},
		"last_7_days": {day(2026, 10, 9), run},
		"this_week":   {day(2026, 10, 12), run},
		"this_month":  {day(2026, 10, 1), run},
		"last_month":  {day(2026, 9, 1), day(2026, 9, 30)},
	}
	for period, want := range cases {
		from, to := emailSchedulePeriodDays(period, run)
		require.Equal(t, want, [2]time.Time{from, to}, period)
	}
}

func TestCleanEmailSchedule(t *testing.T) {
	s, err := cleanEmailSchedule(emailScheduleRequestBody{
		Period: "today", Recipients: "a@x.com", Frequency: "weekly", Time: "22:15", Weekdays: []int32{5, 1, 5},
	})
	require.NoError(t, err)
	require.Equal(t, int32(22*60+15), s.AtMinute)
	require.Equal(t, []int32{1, 5}, s.Weekdays)
	require.True(t, s.NextRunAt.Valid)

	_, err = cleanEmailSchedule(emailScheduleRequestBody{Period: "today", Recipients: "a@x.com", Frequency: "weekly", Time: "22:15"})
	require.Error(t, err)
	_, err = cleanEmailSchedule(emailScheduleRequestBody{Period: "today", Recipients: "a@x.com", Frequency: "daily", Time: "25:00"})
	require.Error(t, err)
}

// A due run another check already claimed isn't sent again.
func TestRunDueEmailSchedulesClaimsOnce(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	due := db.EmailSchedule{ID: 7, Enabled: true, Frequency: "daily", AtMinute: 60, Period: "today",
		Recipients: "a@x.com", NextRunAt: sql.NullTime{Time: time.Now().Add(-time.Minute), Valid: true}}
	store.EXPECT().ListDueEmailSchedules(gomock.Any()).Return([]db.EmailSchedule{due}, nil)
	store.EXPECT().ClaimEmailScheduleRun(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ any, arg db.ClaimEmailScheduleRunParams) (int64, error) {
			require.Equal(t, due.NextRunAt, arg.DueAt)
			require.True(t, arg.NextRunAt.Time.After(time.Now()))
			return 0, nil
		})
	// Nothing else: no send, no record.
	server := newTestServer(t, store)
	server.runDueEmailSchedules(t.Context())
}

func TestRunEmailScheduleSkipsEmpty(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	store.EXPECT().GetEmailResendAPIKey(gomock.Any()).Return("re_test", nil)
	store.EXPECT().GetEmailSettings(gomock.Any()).Return(db.GetEmailSettingsRow{EmailBusinessName: "Grand Bazaar"}, nil)
	store.EXPECT().ListEmailReportInvoices(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ any, arg db.ListEmailReportInvoicesParams) ([]db.ListEmailReportInvoicesRow, error) {
			// "yesterday" as of a run on the 15th.
			require.Equal(t, "2026-10-14", arg.FromDay.Format("2006-01-02"))
			require.Equal(t, "2026-10-14", arg.ToDay.Format("2006-01-02"))
			return nil, nil
		})
	store.EXPECT().RecordEmailScheduleRun(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ any, arg db.RecordEmailScheduleRunParams) error {
			require.Equal(t, "skipped", arg.Status)
			return nil
		})
	server := newTestServer(t, store)
	_, err := server.runEmailSchedule(t.Context(), db.EmailSchedule{
		ID: 3, Period: "yesterday", Recipients: "a@x.com", SkipEmpty: true,
	}, beirut(2026, 10, 15, 8, 0))
	require.ErrorIs(t, err, errEmailReportEmpty)
}
