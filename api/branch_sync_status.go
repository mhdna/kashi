package api

import (
	"database/sql"
	"net/http"
	"reflect"
	"time"

	"github.com/gin-gonic/gin"
	db "github.com/mhdna/kashi/db/sqlc"
)

// What each till has of each kind of data, for the Commands page. Kashi works
// out what it can from the pulls themselves (recordSyncPull). The till adds what
// only it knows, whether applying worked and why not (branchSyncReport).

const (
	syncStateSynced   = "synced"   // nothing newer than the till's last pull
	syncStateSent     = "sent"     // sent to the till, which hasn't confirmed yet
	syncStateBehind   = "behind"   // kashi changed since the till last asked
	syncStateRetrying = "retrying" // the till asked for the same data again
	syncStateStuck    = "stuck"    // it keeps failing to apply it
)

// syncStuckRetries is how many repeat pulls count as stuck: one can be a lost
// connection, two in a row is a till that can't apply what it gets.
const syncStuckRetries = 2

// recordSyncPull notes one catch-up request. Best effort: this is only a report,
// and a failure here must never fail the till's sync.
func (server *Server) recordSyncPull(ctx *gin.Context, entity string, since time.Time, items any) {
	count := 0
	if v := reflect.ValueOf(items); v.Kind() == reflect.Slice {
		count = v.Len()
	}
	_ = server.store.RecordBranchSyncPull(ctx, db.RecordBranchSyncPullParams{
		RegisterID:    ctx.MustGet(registerIDKey).(int64),
		Entity:        entity,
		SyncedThrough: sql.NullTime{Time: since, Valid: !since.IsZero()},
		ItemsSent:     int32(count),
	})
}

type syncReportEntity struct {
	Entity      string `json:"entity" binding:"required"`
	Error       string `json:"error"`
	RowsApplied int64  `json:"rows_applied"`
}

type syncReportRequest struct {
	Entities      []syncReportEntity `json:"entities" binding:"dive"`
	OutboxPending int64              `json:"outbox_pending"`
	OutboxFailed  int64              `json:"outbox_failed"`
}

// branchSyncReport is the till's own word on its last pull. It sends it only
// when something changed, and a kashi without this endpoint just answers 404.
func (server *Server) branchSyncReport(ctx *gin.Context) {
	registerID := ctx.MustGet(registerIDKey).(int64)
	var req syncReportRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	for _, e := range req.Entities {
		if err := server.store.ReportBranchSyncEntity(ctx, db.ReportBranchSyncEntityParams{
			RegisterID: registerID, Entity: e.Entity, Error: e.Error, RowsApplied: e.RowsApplied,
		}); err != nil {
			server.writeError(ctx, http.StatusInternalServerError, err)
			return
		}
	}
	if err := server.store.ReportBranchSyncOutbox(ctx, db.ReportBranchSyncOutboxParams{
		RegisterID: registerID, Pending: req.OutboxPending, Failed: req.OutboxFailed,
	}); err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	server.writeJSON(ctx, http.StatusOK, envelope{"status": "ok"})
}

type syncEntityView struct {
	Entity        string     `json:"entity"`
	State         string     `json:"state"`
	LastPullAt    *time.Time `json:"last_pull_at"`
	SyncedThrough *time.Time `json:"synced_through"`
	// When kashi last changed this for the branch; only the main entities have it.
	LatestChangeAt *time.Time `json:"latest_change_at"`
	ItemsSent      int32      `json:"items_sent"`
	Retries        int32      `json:"retries"`
	StuckSince     *time.Time `json:"stuck_since"`
	Error          string     `json:"error"`
	RowsApplied    int64      `json:"rows_applied"`
	ReportedAt     *time.Time `json:"reported_at"`
}

type syncRegisterView struct {
	RegisterID    int64            `json:"register_id"`
	RegisterCode  string           `json:"register_code"`
	RegisterName  string           `json:"register_name"`
	BranchID      int64            `json:"branch_id"`
	BranchCode    string           `json:"branch_code"`
	BranchName    string           `json:"branch_name"`
	LastSeenAt    *time.Time       `json:"last_seen_at"`
	AppVersion    string           `json:"app_version"`
	OutboxPending int64            `json:"outbox_pending"`
	OutboxFailed  int64            `json:"outbox_failed"`
	OutboxAt      *time.Time       `json:"outbox_reported_at"`
	Entities      []syncEntityView `json:"entities"`
}

// syncEntityState reads one entity's row, worst first. Reported or repeated
// failure wins; then the last pull, which a till only proves it applied by
// reporting so or by pulling again from a later cursor.
func syncEntityState(row db.ListBranchSyncStatusRow, latest *time.Time) string {
	switch {
	case row.Error != "" || row.Retries >= syncStuckRetries:
		return syncStateStuck
	case row.Retries > 0:
		return syncStateRetrying
	case latest != nil && row.LastPullAt.Valid && latest.After(row.LastPullAt.Time):
		return syncStateBehind
	// Only a pull with a cursor can wait on a later one: a whole-set entity sends
	// everything every time, so its items say nothing about what's new.
	case row.SyncedThrough.Valid && row.ItemsSent > 0 && (!row.ReportedAt.Valid || row.ReportedAt.Time.Before(row.LastPullAt.Time)):
		return syncStateSent
	}
	return syncStateSynced
}

// latestByEntity is BranchSyncLatestChanges by entity name, the epoch dropped.
func latestByEntity(row db.BranchSyncLatestChangesRow) map[string]*time.Time {
	out := map[string]*time.Time{}
	for entity, at := range map[string]time.Time{
		"discount_lists": row.DiscountLists, "price_lists": row.PriceLists,
		"products": row.Products, "clients": row.Clients,
	} {
		if at.Unix() > 0 {
			out[entity] = &at
		}
	}
	return out
}

type listBranchSyncStatusRequest struct {
	BranchID int64 `form:"branch_id" binding:"min=0"`
}

// listBranchSyncStatus answers every active register of one branch, or of all.
func (server *Server) listBranchSyncStatus(ctx *gin.Context) {
	var req listBranchSyncStatusRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	rows, err := server.store.ListBranchSyncStatus(ctx, sql.NullInt64{Int64: req.BranchID, Valid: req.BranchID > 0})
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	latest := map[int64]map[string]*time.Time{}
	registers := []syncRegisterView{}
	at := map[int64]int{}
	for _, row := range rows {
		if _, seen := latest[row.BranchID]; !seen {
			changes, err := server.store.BranchSyncLatestChanges(ctx, row.BranchID)
			if err != nil {
				server.writeError(ctx, http.StatusInternalServerError, err)
				return
			}
			latest[row.BranchID] = latestByEntity(changes)
		}
		i, seen := at[row.RegisterID]
		if !seen {
			i = len(registers)
			at[row.RegisterID] = i
			registers = append(registers, syncRegisterView{
				RegisterID: row.RegisterID, RegisterCode: row.RegisterCode, RegisterName: row.RegisterName,
				BranchID: row.BranchID, BranchCode: row.BranchCode, BranchName: row.BranchName,
				LastSeenAt: nullTimePtr(row.LastSeenAt), AppVersion: row.AppVersion,
				OutboxPending: row.OutboxPending, OutboxFailed: row.OutboxFailed,
				OutboxAt: nullTimePtr(row.OutboxReportedAt), Entities: []syncEntityView{},
			})
		}
		if row.Entity == "" {
			continue // this register hasn't pulled since kashi started recording
		}
		changed := latest[row.BranchID][row.Entity]
		registers[i].Entities = append(registers[i].Entities, syncEntityView{
			Entity: row.Entity, State: syncEntityState(row, changed),
			LastPullAt: nullTimePtr(row.LastPullAt), SyncedThrough: nullTimePtr(row.SyncedThrough),
			LatestChangeAt: changed, ItemsSent: row.ItemsSent, Retries: row.Retries,
			StuckSince: nullTimePtr(row.StuckSince), Error: row.Error, RowsApplied: row.RowsApplied,
			ReportedAt: nullTimePtr(row.ReportedAt),
		})
	}
	server.writeJSON(ctx, http.StatusOK, envelope{"registers": registers})
}
