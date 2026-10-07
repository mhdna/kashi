package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	db "github.com/mhdna/kashi/db/sqlc"
)

// listLatestAdditions backs the home page's "Latest added" feed: newest
// photos, transfers, expenses and purchases in one list. Open to every
// signed-in user, but each kind is included only when their role can view
// the section it belongs to, so nobody sees rows from a page they can't open.
func (server *Server) listLatestAdditions(ctx *gin.Context) {
	var req struct {
		PageSize int32 `form:"page_size,default=20" binding:"min=1,max=100"`
	}
	if err := ctx.ShouldBindQuery(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	access, _ := currentAccess(ctx)

	rows, err := server.store.ListLatestAdditions(ctx, db.ListLatestAdditionsParams{
		IncludeAccounting: access.canView("accounting"),
		IncludeHr:         access.canView("hr"),
		IncludeInventory:  access.canView("inventory"),
		PageSize:          req.PageSize,
	})
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	server.writeJSON(ctx, http.StatusOK, envelope{"items": rows})
}

// getHomeToday backs the "N in the last 48h" counts beside the home page's
// three feed titles: everything from `since` (an RFC3339 instant the client
// picks) up to now. Each count is null when the user can't view that feed, mirroring
// the access checks of the feed's own route.
func (server *Server) getHomeToday(ctx *gin.Context) {
	var req struct {
		Since time.Time `form:"since" binding:"required" time_format:"2006-01-02T15:04:05Z07:00"`
	}
	if err := ctx.ShouldBindQuery(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	access, _ := currentAccess(ctx)
	// nil marshals to null: a feed this user can't view.
	var adminActivities, activities *int64

	if access.canView("system") {
		n, err := server.store.CountAdminActivitiesSince(ctx, req.Since)
		if err != nil {
			server.writeError(ctx, http.StatusInternalServerError, err)
			return
		}
		adminActivities = &n
	}

	// Same feed the home page lists from /dashboard/activities; its window
	// count rides on every row, so one row is enough.
	if access.canView("analytics") {
		rows, err := server.store.DashboardActivitiesList(ctx, db.DashboardActivitiesListParams{
			Scope: "all", DateFrom: req.Since, DateTo: time.Now(),
			PageSize: 1, PageID: 0,
		})
		if err != nil {
			server.writeError(ctx, http.StatusInternalServerError, err)
			return
		}
		var n int64
		if len(rows) > 0 {
			n = rows[0].TotalCount
		}
		activities = &n
	}

	n, err := server.store.CountLatestAdditionsSince(ctx, db.CountLatestAdditionsSinceParams{
		IncludeAccounting: access.canView("accounting"),
		IncludeHr:         access.canView("hr"),
		IncludeInventory:  access.canView("inventory"),
		Since:             req.Since,
	})
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	server.writeJSON(ctx, http.StatusOK, envelope{
		"admin_activities": adminActivities,
		"activities":       activities,
		"latest":           n,
	})
}
