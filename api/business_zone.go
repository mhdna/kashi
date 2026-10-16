package api

import (
	"time"
	_ "time/tzdata" // so the zone loads even on a host without zoneinfo
)

// businessZone is the shops' wall-clock zone: where kashi's report days
// start and end. Timestamps are stored as UTC instants (the tills stamp
// them that way too); this is only for turning them into calendar days.
// The report queries spell the same zone as 'Asia/Beirut', and each
// branch's own branches.timezone carries it for branch-scoped views.
var businessZone = func() *time.Location {
	loc, err := time.LoadLocation("Asia/Beirut")
	if err != nil {
		panic(err)
	}
	return loc
}()

// businessToday is today's calendar date in businessZone, held as a
// UTC-midnight value -- the same shape a scanned `date` column has.
func businessToday() time.Time {
	y, m, d := time.Now().In(businessZone).Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

// dayStart is the instant a calendar date (a UTC-midnight value, as above)
// begins in businessZone: what a timestamptz range bound needs.
func dayStart(day time.Time) time.Time {
	y, m, d := day.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, businessZone)
}
