package api

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	db "github.com/mhdna/kashi/db/sqlc"
	"github.com/mhdna/kashi/token"
)

// Scheduled invoice reports: an email_schedules row says what to send (a
// period relative to the run, some registers, a format), to whom, and when
// -- daily, on some weekdays, or on one day of the month, at a time of day
// in businessZone. RunEmailScheduler sends each one when due through
// deliverEmailReport, the same path as the send dialog, so every run lands
// on the Emails page like a manual send.

// emailScheduleCheckInterval is how often due schedules are looked for; a
// run goes out within this of its time.
const emailScheduleCheckInterval = 30 * time.Second

var emailSchedulePeriods = map[string]string{
	"today":       "Today",
	"yesterday":   "Yesterday",
	"last_7_days": "Last 7 days",
	"this_week":   "This week",
	"this_month":  "This month",
	"last_month":  "Last month",
}

// emailSchedulePeriodDays is the [from, to] local days a period covers when
// it runs on runDay (a businessZone calendar date). Weeks start on Monday.
func emailSchedulePeriodDays(period string, runDay time.Time) (from, to time.Time) {
	switch period {
	case "yesterday":
		day := runDay.AddDate(0, 0, -1)
		return day, day
	case "last_7_days":
		return runDay.AddDate(0, 0, -6), runDay
	case "this_week":
		back := (int(runDay.Weekday()) + 6) % 7 // days since Monday
		return runDay.AddDate(0, 0, -back), runDay
	case "this_month":
		return time.Date(runDay.Year(), runDay.Month(), 1, 0, 0, 0, 0, time.UTC), runDay
	case "last_month":
		first := time.Date(runDay.Year(), runDay.Month(), 1, 0, 0, 0, 0, time.UTC)
		return first.AddDate(0, -1, 0), first.AddDate(0, 0, -1)
	default: // today
		return runDay, runDay
	}
}

// nextEmailScheduleRun is the first run strictly after `after`: the
// schedule's time of day in businessZone on the next day it applies to. A
// monthly day past a month's end runs on that month's last day.
func nextEmailScheduleRun(s db.EmailSchedule, after time.Time) time.Time {
	local := after.In(businessZone)
	day := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, businessZone)
	weekdays := map[int32]bool{}
	for _, w := range s.Weekdays {
		weekdays[w] = true
	}
	// A year of days always finds one: every schedule applies at least
	// monthly.
	for i := 0; i < 400; i++ {
		date := day.AddDate(0, 0, i)
		applies := true
		switch s.Frequency {
		case "weekly":
			applies = weekdays[int32(date.Weekday())]
		case "monthly":
			applies = date.Day() == clampDay(date.Year(), date.Month(), int(s.MonthDay))
		}
		if !applies {
			continue
		}
		run := time.Date(date.Year(), date.Month(), date.Day(), int(s.AtMinute)/60, int(s.AtMinute)%60, 0, 0, businessZone)
		if run.After(after) {
			return run
		}
	}
	return time.Time{}
}

// emailScheduleRequest turns a schedule into what deliverEmailReport needs
// for a run on runAt.
func emailScheduleRequest(s db.EmailSchedule, runAt time.Time) emailDelivery {
	y, m, d := runAt.In(businessZone).Date()
	from, to := emailSchedulePeriodDays(s.Period, time.Date(y, m, d, 0, 0, 0, 0, time.UTC))
	ids := make([]string, len(s.RegisterIds))
	for i, id := range s.RegisterIds {
		ids[i] = strconv.FormatInt(id, 10)
	}
	recipients, _ := parseEmailRecipients(s.Recipients)
	name := s.Name
	if name == "" {
		name = fmt.Sprintf("#%d", s.ID)
	}
	return emailDelivery{
		request: emailReportRequest{
			Scope:       "range",
			From:        from.Format("2006-01-02"),
			To:          to.Format("2006-01-02"),
			RegisterIDs: strings.Join(ids, ","),
			Format:      s.Format,
		},
		recipients: recipients,
		note:       s.Note,
		sentBy:     "schedule: " + name,
		scheduleID: sql.NullInt64{Int64: s.ID, Valid: true},
	}
}

// ---- running -------------------------------------------------------------------

// RunEmailScheduler sends due scheduled reports, on startup (catching up on
// one missed while kashi was down, once) and then every
// emailScheduleCheckInterval, for the life of the process.
func (server *Server) RunEmailScheduler(ctx context.Context) {
	server.runDueEmailSchedules(ctx)
	ticker := time.NewTicker(emailScheduleCheckInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			server.runDueEmailSchedules(ctx)
		}
	}
}

func (server *Server) runDueEmailSchedules(ctx context.Context) {
	due, err := server.store.ListDueEmailSchedules(ctx)
	if err != nil {
		log.Printf("email schedules: list due: %v", err)
		return
	}
	now := time.Now()
	for _, s := range due {
		// Claim the run by moving the schedule on first, so it's never sent
		// twice; a run missed while kashi was down goes out once, now.
		next := nextEmailScheduleRun(s, now)
		claimed, err := server.store.ClaimEmailScheduleRun(ctx, db.ClaimEmailScheduleRunParams{
			ID:        s.ID,
			NextRunAt: sql.NullTime{Time: next, Valid: !next.IsZero()},
			DueAt:     s.NextRunAt,
		})
		if err != nil {
			log.Printf("email schedule %d: claim: %v", s.ID, err)
			continue
		}
		if claimed == 0 {
			continue
		}
		server.runEmailSchedule(ctx, s, now)
	}
}

// runEmailSchedule sends one run and records how it went on the schedule.
func (server *Server) runEmailSchedule(ctx context.Context, s db.EmailSchedule, runAt time.Time) (db.EmailReport, error) {
	report, err := server.deliverEmailReport(ctx, emailScheduleRequest(s, runAt))
	status, message := "sent", ""
	switch {
	case errors.Is(err, errEmailReportEmpty) && s.SkipEmpty:
		status, message = "skipped", err.Error()
	case err != nil:
		status, message = "failed", err.Error()
		log.Printf("email schedule %d: %v", s.ID, err)
	}
	if recordErr := server.store.RecordEmailScheduleRun(ctx, db.RecordEmailScheduleRunParams{
		ID: s.ID, Status: status, Error: message,
	}); recordErr != nil {
		log.Printf("email schedule %d: record run: %v", s.ID, recordErr)
	}
	server.adminHub.broadcastAll(adminWSMessage{Type: "email_schedule_ran"})
	return report, err
}

// ---- API -----------------------------------------------------------------------

// emailScheduleResponse flattens the nullable times for the UI.
type emailScheduleResponse struct {
	db.EmailSchedule
	NextRunAt   *time.Time `json:"next_run_at"`
	LastRunAt   *time.Time `json:"last_run_at"`
	PeriodLabel string     `json:"period_label"`
}

func newEmailScheduleResponse(s db.EmailSchedule) emailScheduleResponse {
	return emailScheduleResponse{
		EmailSchedule: s,
		NextRunAt:     nullTimeValue(s.NextRunAt),
		LastRunAt:     nullTimeValue(s.LastRunAt),
		PeriodLabel:   emailSchedulePeriods[s.Period],
	}
}

func (server *Server) listEmailSchedules(ctx *gin.Context) {
	schedules, err := server.store.ListEmailSchedules(ctx)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	res := make([]emailScheduleResponse, len(schedules))
	for i, s := range schedules {
		res[i] = newEmailScheduleResponse(s)
	}
	server.writeJSON(ctx, http.StatusOK, envelope{"schedules": res})
}

type emailScheduleRequestBody struct {
	Name        string  `json:"name"`
	Enabled     bool    `json:"enabled"`
	Period      string  `json:"period" binding:"required"`
	RegisterIDs []int64 `json:"register_ids"`
	Recipients  string  `json:"recipients" binding:"required"`
	Format      string  `json:"format" binding:"omitempty,oneof=csv xlsx"`
	Note        string  `json:"note" binding:"max=2000"`
	SkipEmpty   bool    `json:"skip_empty"`
	Frequency   string  `json:"frequency" binding:"required,oneof=daily weekly monthly"`
	// "HH:MM", in the shops' zone.
	Time     string  `json:"time" binding:"required"`
	Weekdays []int32 `json:"weekdays"`
	MonthDay int32   `json:"month_day"`
}

// cleanEmailSchedule validates a create/update body into a schedule (with
// its next run worked out from now).
func cleanEmailSchedule(req emailScheduleRequestBody) (db.EmailSchedule, error) {
	if _, ok := emailSchedulePeriods[req.Period]; !ok {
		return db.EmailSchedule{}, fmt.Errorf("unknown period %q", req.Period)
	}
	recipients, err := parseEmailRecipients(req.Recipients)
	if err != nil {
		return db.EmailSchedule{}, err
	}
	if len(recipients) == 0 {
		return db.EmailSchedule{}, errors.New("add a recipient")
	}
	at, err := time.Parse("15:04", strings.TrimSpace(req.Time))
	if err != nil {
		return db.EmailSchedule{}, errors.New("time must be HH:MM")
	}
	s := db.EmailSchedule{
		Name:        strings.TrimSpace(req.Name),
		Enabled:     req.Enabled,
		Period:      req.Period,
		RegisterIds: req.RegisterIDs,
		Recipients:  strings.Join(recipients, ", "),
		Format:      req.Format,
		Note:        strings.TrimSpace(req.Note),
		SkipEmpty:   req.SkipEmpty,
		Frequency:   req.Frequency,
		AtMinute:    int32(at.Hour()*60 + at.Minute()),
		Weekdays:    []int32{},
		MonthDay:    1,
	}
	if s.RegisterIds == nil {
		s.RegisterIds = []int64{}
	}
	switch req.Frequency {
	case "weekly":
		seen := map[int32]bool{}
		for _, w := range req.Weekdays {
			if w < 0 || w > 6 {
				return db.EmailSchedule{}, fmt.Errorf("bad weekday %d", w)
			}
			if !seen[w] {
				seen[w] = true
				s.Weekdays = append(s.Weekdays, w)
			}
		}
		if len(s.Weekdays) == 0 {
			return db.EmailSchedule{}, errors.New("pick at least one weekday")
		}
		sort.Slice(s.Weekdays, func(i, j int) bool { return s.Weekdays[i] < s.Weekdays[j] })
	case "monthly":
		if req.MonthDay < 1 || req.MonthDay > 31 {
			return db.EmailSchedule{}, errors.New("day of month must be 1-31")
		}
		s.MonthDay = req.MonthDay
	}
	if next := nextEmailScheduleRun(s, time.Now()); !next.IsZero() {
		s.NextRunAt = sql.NullTime{Time: next, Valid: true}
	}
	return s, nil
}

func (server *Server) createEmailSchedule(ctx *gin.Context) {
	var req emailScheduleRequestBody
	if err := ctx.ShouldBindJSON(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	s, err := cleanEmailSchedule(req)
	if err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	createdBy := ""
	if payload, ok := ctx.MustGet(authoizationPayloadKey).(*token.Payload); ok {
		createdBy = payload.Username
	}
	created, err := server.store.CreateEmailSchedule(ctx, db.CreateEmailScheduleParams{
		Name: s.Name, Enabled: s.Enabled, Period: s.Period, RegisterIds: s.RegisterIds,
		Recipients: s.Recipients, Format: s.Format, Note: s.Note, SkipEmpty: s.SkipEmpty,
		Frequency: s.Frequency, AtMinute: s.AtMinute, Weekdays: s.Weekdays, MonthDay: s.MonthDay,
		NextRunAt: s.NextRunAt, CreatedBy: createdBy,
	})
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	server.writeJSON(ctx, http.StatusOK, envelope{"schedule": newEmailScheduleResponse(created)})
}

type emailScheduleURI struct {
	ID int64 `uri:"id" binding:"required,min=1"`
}

func (server *Server) updateEmailSchedule(ctx *gin.Context) {
	var uri emailScheduleURI
	if err := ctx.ShouldBindUri(&uri); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	var req emailScheduleRequestBody
	if err := ctx.ShouldBindJSON(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	s, err := cleanEmailSchedule(req)
	if err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	updated, err := server.store.UpdateEmailSchedule(ctx, db.UpdateEmailScheduleParams{
		ID: uri.ID, Name: s.Name, Enabled: s.Enabled, Period: s.Period, RegisterIds: s.RegisterIds,
		Recipients: s.Recipients, Format: s.Format, Note: s.Note, SkipEmpty: s.SkipEmpty,
		Frequency: s.Frequency, AtMinute: s.AtMinute, Weekdays: s.Weekdays, MonthDay: s.MonthDay,
		NextRunAt: s.NextRunAt,
	})
	if errors.Is(err, sql.ErrNoRows) {
		server.writeError(ctx, http.StatusNotFound, errors.New("schedule not found"))
		return
	} else if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	server.writeJSON(ctx, http.StatusOK, envelope{"schedule": newEmailScheduleResponse(updated)})
}

func (server *Server) deleteEmailSchedule(ctx *gin.Context) {
	var uri emailScheduleURI
	if err := ctx.ShouldBindUri(&uri); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	if err := server.store.DeleteEmailSchedule(ctx, uri.ID); err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	server.writeJSON(ctx, http.StatusOK, envelope{"deleted": true})
}

// runEmailScheduleNow is the Schedules tab's "Send now": one run as of now,
// leaving the schedule's next run where it was.
func (server *Server) runEmailScheduleNow(ctx *gin.Context) {
	var uri emailScheduleURI
	if err := ctx.ShouldBindUri(&uri); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	s, err := server.store.GetEmailSchedule(ctx, uri.ID)
	if errors.Is(err, sql.ErrNoRows) {
		server.writeError(ctx, http.StatusNotFound, errors.New("schedule not found"))
		return
	} else if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	report, err := server.runEmailSchedule(ctx, s, time.Now())
	if err != nil {
		server.writeEmailDeliveryError(ctx, err)
		return
	}
	server.writeJSON(ctx, http.StatusOK, envelope{"report": report})
}
