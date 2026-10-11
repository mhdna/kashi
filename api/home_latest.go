package api

import (
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	db "github.com/mhdna/kashi/db/sqlc"
)

// feedCursor is where the next page of a home feed starts: the sort key of
// the last row the feed already has (its timestamp, kind and id). The feeds
// page by this keyset rather than an offset, so new rows landing on top
// don't shift the pages and a deep page costs the same as the first. All
// empty on the first page.
type feedCursor struct {
	BeforeAt   string `form:"before_at"`
	BeforeKind string `form:"before_kind"`
	BeforeID   int64  `form:"before_id" binding:"min=0"`
}

// args turns the cursor into the queries' nullable keyset arguments, all
// NULL on the first page.
func (c feedCursor) args() (at sql.NullTime, kind sql.NullString, id sql.NullInt64, err error) {
	if c.BeforeAt == "" {
		return at, kind, id, nil
	}
	t, err := time.Parse(time.RFC3339Nano, c.BeforeAt)
	if err != nil {
		return at, kind, id, fmt.Errorf("before_at: %w", err)
	}
	return sql.NullTime{Time: t, Valid: true},
		sql.NullString{String: c.BeforeKind, Valid: true},
		sql.NullInt64{Int64: c.BeforeID, Valid: true}, nil
}

// listHomeActivities backs the home page's "Recent activity" feed: every
// kind of branch sync, newest received first, page_size rows after the
// cursor.
func (server *Server) listHomeActivities(ctx *gin.Context) {
	var req struct {
		feedCursor
		PageSize int32 `form:"page_size,default=30" binding:"min=1,max=100"`
	}
	if err := ctx.ShouldBindQuery(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	at, activity, id, err := req.args()
	if err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	rows, err := server.store.HomeActivityFeed(ctx, db.HomeActivityFeedParams{
		BeforeAt: at, BeforeActivity: activity, BeforeID: id, PageSize: req.PageSize,
	})
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	server.writeJSON(ctx, http.StatusOK, envelope{"activities": rows})
}

// listLatestAdditions backs the home page's "Latest added" feed: newest
// photos, transfers, expenses and purchases in one list. Open to every
// signed-in user, but each kind is included only when their role can view
// the section it belongs to, so nobody sees rows from a page they can't open.
func (server *Server) listLatestAdditions(ctx *gin.Context) {
	var req struct {
		feedCursor
		PageSize int32 `form:"page_size,default=20" binding:"min=1,max=500"`
	}
	if err := ctx.ShouldBindQuery(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	at, kind, id, err := req.args()
	if err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	access, _ := currentAccess(ctx)

	rows, err := server.store.ListLatestAdditions(ctx, db.ListLatestAdditionsParams{
		IncludeAccounting: access.canView("accounting"),
		IncludeHr:         access.canView("hr"),
		IncludeInventory:  access.canView("inventory"),
		BeforeAt:          at,
		BeforeKind:        kind,
		BeforeID:          id,
		PageSize:          req.PageSize,
	})
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	server.writeJSON(ctx, http.StatusOK, envelope{"items": rows})
}

// getHomeToday backs the "N in the last week" counts beside the home page's
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

	// Same rows the Recent activity feed lists (/home/activities).
	if access.canView("analytics") {
		n, err := server.store.CountHomeActivitiesSince(ctx, req.Since)
		if err != nil {
			server.writeError(ctx, http.StatusInternalServerError, err)
			return
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
