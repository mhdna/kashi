package api

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	db "github.com/mhdna/kashi/db/sqlc"
	"github.com/mhdna/kashi/token"
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
	activities, err := server.markResolvedHomeActivities(ctx, rows)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	server.writeJSON(ctx, http.StatusOK, envelope{"activities": activities})
}

// homeActivity is a feed row plus, for an urgent kind, whether someone
// resolved it -- so a resolved row can show its check (and be reopened).
type homeActivity struct {
	db.HomeActivityFeedRow
	Resolved   bool       `json:"resolved"`
	ResolvedBy string     `json:"resolved_by,omitempty"`
	ResolvedAt *time.Time `json:"resolved_at,omitempty"`
}

func (server *Server) markResolvedHomeActivities(ctx context.Context, rows []db.HomeActivityFeedRow) ([]homeActivity, error) {
	out := make([]homeActivity, len(rows))
	ids := map[string][]int64{}
	for i, row := range rows {
		out[i] = homeActivity{HomeActivityFeedRow: row}
		if urgentHomeActivities[row.Activity] {
			ids[row.Activity] = append(ids[row.Activity], row.ID)
		}
	}
	for activity, sourceIDs := range ids {
		resolutions, err := server.store.ListHomeActivityResolutions(ctx, db.ListHomeActivityResolutionsParams{
			Activity: activity, SourceIds: sourceIDs,
		})
		if err != nil {
			return nil, err
		}
		for _, r := range resolutions {
			for i := range out {
				if out[i].Activity == activity && out[i].ID == r.SourceID {
					at := r.ResolvedAt
					out[i].Resolved, out[i].ResolvedBy, out[i].ResolvedAt = true, r.ResolvedBy, &at
				}
			}
		}
	}
	return out, nil
}

// urgentHomeActivities are the Recent activity rows that stay pinned above
// the feed until someone resolves them (ListUnresolvedUrgentHomeActivities
// decides which rows of these kinds are urgent).
var urgentHomeActivities = map[string]bool{
	"Shift closed": true,
}

// listUrgentHomeActivities backs the pinned top of the Recent activity feed:
// every urgent row nobody has resolved, newest first.
func (server *Server) listUrgentHomeActivities(ctx *gin.Context) {
	rows, err := server.store.ListUnresolvedUrgentHomeActivities(ctx)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	server.writeJSON(ctx, http.StatusOK, envelope{"activities": rows})
}

type resolveHomeActivityRequest struct {
	Activity string `json:"activity" binding:"required"`
	ID       int64  `json:"id" binding:"required,min=1"`
}

// resolveHomeActivity unpins an urgent row. The route needs the
// resolve_urgent_activity permission (see routeRules).
func (server *Server) resolveHomeActivity(ctx *gin.Context) {
	var req resolveHomeActivityRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	if !urgentHomeActivities[req.Activity] {
		server.writeError(ctx, http.StatusBadRequest, fmt.Errorf("%q is not an urgent activity", req.Activity))
		return
	}
	by := ""
	if payload, ok := ctx.MustGet(authoizationPayloadKey).(*token.Payload); ok {
		by = payload.Username
	}
	if err := server.store.ResolveHomeActivity(ctx, db.ResolveHomeActivityParams{
		Activity: req.Activity, SourceID: req.ID, ResolvedBy: by,
	}); err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	server.writeJSON(ctx, http.StatusOK, envelope{"resolved": true})
}

// unresolveHomeActivity reopens a resolved urgent row, pinning it again --
// for a resolve clicked by mistake. Same route rule (and permission) as
// resolving: DELETE on the same path.
func (server *Server) unresolveHomeActivity(ctx *gin.Context) {
	var req resolveHomeActivityRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	if !urgentHomeActivities[req.Activity] {
		server.writeError(ctx, http.StatusBadRequest, fmt.Errorf("%q is not an urgent activity", req.Activity))
		return
	}
	if err := server.store.UnresolveHomeActivity(ctx, db.UnresolveHomeActivityParams{
		Activity: req.Activity, SourceID: req.ID,
	}); err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	server.writeJSON(ctx, http.StatusOK, envelope{"resolved": false})
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
