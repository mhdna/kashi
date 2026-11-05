package api

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	db "github.com/mhdna/kashi/db/sqlc"
)

// Whether a price or discount list's latest change has reached each till that
// uses it, for the lists page. Read from what branch_sync_status.go records:
// a list is "reached" once the till's cursor has passed the change, or the till
// has reported applying a pull made after it.

const (
	deliveryReached = "reached" // the till has this version of the list
	deliverySending = "sending" // it pulled the change and hasn't confirmed yet
	deliveryWaiting = "waiting" // it's online and hasn't asked for it yet
	deliveryOffline = "offline" // it hasn't been seen lately
	deliveryStuck   = "stuck"   // it keeps failing to apply it
)

// deliveryOfflineAfter: a connected till touches kashi far more often than this.
const deliveryOfflineAfter = 10 * time.Minute

type listDeliveryRegister struct {
	RegisterID   int64      `json:"register_id"`
	RegisterCode string     `json:"register_code"`
	BranchID     int64      `json:"branch_id"`
	BranchName   string     `json:"branch_name"`
	State        string     `json:"state"`
	LastSeenAt   *time.Time `json:"last_seen_at"`
	LastPullAt   *time.Time `json:"last_pull_at"`
	ReachedAt    *time.Time `json:"reached_at"`
	Error        string     `json:"error"`
}

// listDeliveryState reads one till's row for a list changed at changedAt. Having
// the change wins over everything else: a till that got it and then went offline
// still has it.
func listDeliveryState(row db.ListBranchSyncStatusRow, changedAt, now time.Time) (string, *time.Time) {
	pulledSince := row.LastPullAt.Valid && !row.LastPullAt.Time.Before(changedAt)
	if row.SyncedThrough.Valid && !row.SyncedThrough.Time.Before(changedAt) {
		return deliveryReached, nullTimePtr(row.LastPullAt)
	}
	// A pull that sent nothing after the change can't have carried it.
	if pulledSince && row.ItemsSent > 0 && row.Error == "" && row.ReportedAt.Valid && !row.ReportedAt.Time.Before(row.LastPullAt.Time) {
		return deliveryReached, nullTimePtr(row.ReportedAt)
	}
	switch {
	case row.Error != "" || row.Retries >= syncStuckRetries:
		return deliveryStuck, nil
	case !row.LastSeenAt.Valid || now.Sub(row.LastSeenAt.Time) > deliveryOfflineAfter:
		return deliveryOffline, nil
	case pulledSince && row.ItemsSent > 0:
		return deliverySending, nil
	}
	return deliveryWaiting, nil
}

// listDelivery answers one list's tills: every register of the branches given,
// or of all branches for a default list, which every till carries.
func (server *Server) listDelivery(ctx *gin.Context, entity string, changedAt time.Time, isActive, isDefault bool, branchIDs []int64) {
	rows, err := server.store.ListBranchSyncStatus(ctx, sql.NullInt64{})
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	inScope := map[int64]bool{}
	for _, id := range branchIDs {
		inScope[id] = true
	}
	now := time.Now()
	registers := []listDeliveryRegister{}
	at := map[int64]int{}
	for _, row := range rows {
		if !isDefault && !inScope[row.BranchID] {
			continue
		}
		i, seen := at[row.RegisterID]
		if !seen {
			i = len(registers)
			at[row.RegisterID] = i
			offline := deliveryWaiting
			if !row.LastSeenAt.Valid || now.Sub(row.LastSeenAt.Time) > deliveryOfflineAfter {
				offline = deliveryOffline
			}
			registers = append(registers, listDeliveryRegister{
				RegisterID: row.RegisterID, RegisterCode: row.RegisterCode,
				BranchID: row.BranchID, BranchName: row.BranchName,
				State: offline, LastSeenAt: nullTimePtr(row.LastSeenAt),
			})
		}
		if row.Entity != entity {
			continue
		}
		state, reachedAt := listDeliveryState(row, changedAt, now)
		registers[i].State, registers[i].ReachedAt = state, reachedAt
		registers[i].LastPullAt, registers[i].Error = nullTimePtr(row.LastPullAt), row.Error
	}
	server.writeJSON(ctx, http.StatusOK, envelope{
		"changed_at": changedAt, "is_active": isActive, "registers": registers,
	})
}

func (server *Server) discountListDelivery(ctx *gin.Context) {
	var uri discountListBranchesURI
	if err := ctx.ShouldBindUri(&uri); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	list, err := server.store.GetDiscountList(ctx, uri.ID)
	if err != nil {
		server.writeError(ctx, statusForLookup(err), err)
		return
	}
	branchIDs, err := server.store.ListDiscountListBranchIDs(ctx, uri.ID)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	server.listDelivery(ctx, "discount_lists", list.UpdatedAt, list.IsActive, list.IsDefault, branchIDs)
}

func (server *Server) priceListDelivery(ctx *gin.Context) {
	var uri discountListBranchesURI
	if err := ctx.ShouldBindUri(&uri); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	list, err := server.store.GetPriceList(ctx, uri.ID)
	if err != nil {
		server.writeError(ctx, statusForLookup(err), err)
		return
	}
	branchIDs, err := server.store.ListPriceListBranchIDs(ctx, uri.ID)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	server.listDelivery(ctx, "price_lists", list.UpdatedAt, list.IsActive, list.IsDefault, branchIDs)
}
