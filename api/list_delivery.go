package api

import (
	"context"
	"database/sql"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	db "github.com/mhdna/kashi/db/sqlc"
)

// Whether a till holds a price or discount list's latest version, for the lists
// page. A till that reports fingerprints (list_fingerprint.go) is checked: what it
// holds is compared with what it should hold. An older one can only be judged by
// its sync cursor, which says what it should have received, so its "reached" comes
// back with Verified false.

const (
	deliveryReached   = "reached"   // the till has this version of the list
	deliverySending   = "sending"   // it pulled the change and hasn't confirmed yet
	deliveryWaiting   = "waiting"   // it's online and hasn't asked for it yet
	deliveryOffline   = "offline"   // it hasn't been seen lately
	deliveryStuck     = "stuck"     // it keeps failing to apply it
	deliveryDifferent = "different" // it's up to date by its cursor, yet holds something else
)

// deliveryOfflineAfter: a connected till touches kashi far more often than this.
const deliveryOfflineAfter = 10 * time.Minute

type listDeliveryRegister struct {
	RegisterID   int64      `json:"register_id"`
	RegisterCode string     `json:"register_code"`
	BranchID     int64      `json:"branch_id"`
	BranchName   string     `json:"branch_name"`
	State        string     `json:"state"`
	Verified     bool       `json:"verified"`
	TillRows     *int32     `json:"till_rows"`
	LastSeenAt   *time.Time `json:"last_seen_at"`
	LastPullAt   *time.Time `json:"last_pull_at"`
	ReachedAt    *time.Time `json:"reached_at"`
	Error        string     `json:"error"`
}

// tillListView is what a till holds of one list: known is whether it reports
// fingerprints at all, and matches whether this one is what it should be.
type tillListView struct {
	known, matches bool
	reportedAt     time.Time
}

// listDeliveryState reads one till's row for a list changed at changedAt. Holding
// the right rows wins over everything else; a till with the right list that has
// since gone offline still has it.
func listDeliveryState(row db.ListBranchSyncStatusRow, till tillListView, changedAt, now time.Time) (state string, reachedAt *time.Time, verified bool) {
	if till.known && till.matches {
		return deliveryReached, &till.reportedAt, true
	}
	cursorPassed := row.SyncedThrough.Valid && !row.SyncedThrough.Time.Before(changedAt)
	pulledSince := row.LastPullAt.Valid && !row.LastPullAt.Time.Before(changedAt)
	if !till.known {
		if cursorPassed {
			return deliveryReached, nullTimePtr(row.LastPullAt), false
		}
		// A pull that sent nothing after the change can't have carried it.
		if pulledSince && row.ItemsSent > 0 && row.Error == "" && row.ReportedAt.Valid && !row.ReportedAt.Time.Before(row.LastPullAt.Time) {
			return deliveryReached, nullTimePtr(row.ReportedAt), false
		}
	}
	switch {
	case row.Error != "" || row.Retries >= syncStuckRetries:
		return deliveryStuck, nil, false
	case till.known && cursorPassed:
		return deliveryDifferent, nil, false
	case !row.LastSeenAt.Valid || now.Sub(row.LastSeenAt.Time) > deliveryOfflineAfter:
		return deliveryOffline, nil, false
	case pulledSince && row.ItemsSent > 0:
		return deliverySending, nil, false
	}
	return deliveryWaiting, nil, false
}

// listDelivery answers one list's tills: every register of the branches given,
// or of all branches for a default list, which every till carries.
func (server *Server) listDelivery(ctx *gin.Context, kind, entity string, listUID string, changedAt time.Time, isActive, isDefault bool, branchIDs []int64, expected listFingerprint) {
	rows, err := server.store.ListBranchSyncStatus(ctx, sql.NullInt64{})
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	states, err := server.store.ListBranchRegisterListStates(ctx, kind)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	reports := map[int64]bool{}
	holds := map[int64]db.BranchRegisterListState{}
	for _, s := range states {
		reports[s.RegisterID] = true
		if s.ListUid == listUID {
			holds[s.RegisterID] = s
		}
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
			state := deliveryWaiting
			if !row.LastSeenAt.Valid || now.Sub(row.LastSeenAt.Time) > deliveryOfflineAfter {
				state = deliveryOffline
			}
			registers = append(registers, listDeliveryRegister{
				RegisterID: row.RegisterID, RegisterCode: row.RegisterCode,
				BranchID: row.BranchID, BranchName: row.BranchName,
				State: state, LastSeenAt: nullTimePtr(row.LastSeenAt),
			})
		}
		if row.Entity != entity {
			continue
		}
		till := tillListView{known: reports[row.RegisterID]}
		if till.known {
			// A list the till doesn't have is an empty one.
			hold, has := holds[row.RegisterID]
			if has {
				till.matches, till.reportedAt = hold.Hash == expected.Hash, hold.ReportedAt
				tillRows := hold.Rows
				registers[i].TillRows = &tillRows
			} else {
				till.matches, till.reportedAt = expected.Rows == 0, now
				tillRows := int32(0)
				registers[i].TillRows = &tillRows
			}
		}
		registers[i].State, registers[i].ReachedAt, registers[i].Verified = listDeliveryState(row, till, changedAt, now)
		registers[i].LastPullAt, registers[i].Error = nullTimePtr(row.LastPullAt), row.Error
	}
	server.writeJSON(ctx, http.StatusOK, envelope{
		"changed_at": changedAt, "is_active": isActive, "expected_rows": expected.Rows, "registers": registers,
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
	expected, err := server.expectedDiscountList(context.Context(ctx), list)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	server.listDelivery(ctx, "discount", "discount_lists", list.Uid.String(), list.UpdatedAt, list.IsActive, list.IsDefault, branchIDs, expected)
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
	expected, err := server.expectedPriceList(context.Context(ctx), list)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	server.listDelivery(ctx, "price", "price_lists", list.Uid.String(), list.UpdatedAt, list.IsActive, list.IsDefault, branchIDs, expected)
}
