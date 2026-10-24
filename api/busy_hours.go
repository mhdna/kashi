package api

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	db "github.com/mhdna/kashi/db/sqlc"
)

// The Busy Hours page: customers in (the tills' people counters) and sales,
// by weekday and hour of the day, for heatmaps and hour/weekday charts. The
// page averages them per day itself; kashi just counts.

type busyHoursRequest struct {
	From      string `form:"from" binding:"required"` // YYYY-MM-DD, inclusive
	To        string `form:"to" binding:"required"`
	BranchIDs string `form:"branch_ids"` // comma separated; empty = all
}

func (server *Server) getBusyHours(ctx *gin.Context) {
	var req busyHoursRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	from, errFrom := time.Parse("2006-01-02", req.From)
	to, errTo := time.Parse("2006-01-02", req.To)
	if errFrom != nil || errTo != nil || to.Before(from) {
		server.writeError(ctx, http.StatusBadRequest, errors.New("from and to must be YYYY-MM-DD, from first"))
		return
	}
	if to.Sub(from) > 400*24*time.Hour {
		server.writeError(ctx, http.StatusBadRequest, errors.New("pick at most a year"))
		return
	}
	branchIDs := []int64{}
	for _, part := range strings.Split(req.BranchIDs, ",") {
		if part = strings.TrimSpace(part); part == "" {
			continue
		}
		id, err := strconv.ParseInt(part, 10, 64)
		if err != nil || id <= 0 {
			server.writeError(ctx, http.StatusBadRequest, fmt.Errorf("bad branch id %q", part))
			return
		}
		branchIDs = append(branchIDs, id)
	}

	visitors, err := server.store.BusyHoursVisitors(ctx, db.BusyHoursVisitorsParams{FromDay: from, ToDay: to, BranchIds: branchIDs})
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	sales, err := server.store.BusyHoursSales(ctx, db.BusyHoursSalesParams{FromDay: from, ToDay: to, BranchIds: branchIDs})
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	server.writeJSON(ctx, http.StatusOK, envelope{"visitors": visitors, "sales": sales})
}
