package api

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	db "github.com/mhdna/kashi/db/sqlc"
)

// The completeness check: the updated_at cursor behind /branch/sync/changes
// only ever sees rows that changed after it, so a till that lost or never
// received a row would never learn about it. These routes let a till compare
// its full set of client/variant uids against kashi's and fetch whatever is
// missing or stale -- the "every till holds the same copy" guarantee.

const (
	syncIDsDefaultLimit = 5000
	syncIDsMaxLimit     = 10000
	syncFetchMaxUIDs    = 1000
)

var errTooManySyncFetchUIDs = errors.New("too many uids in one fetch")

type branchSyncIDsRequest struct {
	Entity string `json:"entity" binding:"required,oneof=clients products"`
	// After is the last uid of the previous page ("" for the first page).
	After string `json:"after"`
	Limit int    `json:"limit"`
}

type branchSyncID struct {
	UID       uuid.UUID `json:"uid"`
	UpdatedAt time.Time `json:"updated_at"`
}

// branchSyncIDs pages through every retail client's (or every variant's)
// uid and updated_at, ordered by uid. Inactive rows are included: a till
// must learn a row went inactive, not merely stop hearing about it.
func (server *Server) branchSyncIDs(ctx *gin.Context) {
	var req branchSyncIDsRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	after := uuid.Nil
	if req.After != "" {
		parsed, err := uuid.Parse(req.After)
		if err != nil {
			server.writeError(ctx, http.StatusBadRequest, err)
			return
		}
		after = parsed
	}
	limit := req.Limit
	if limit <= 0 {
		limit = syncIDsDefaultLimit
	}
	if limit > syncIDsMaxLimit {
		limit = syncIDsMaxLimit
	}

	ids := make([]branchSyncID, 0, limit)
	switch req.Entity {
	case "clients":
		rows, err := server.store.ListClientSyncIDs(ctx, db.ListClientSyncIDsParams{AfterUid: after, RowLimit: int32(limit)})
		if err != nil {
			server.writeError(ctx, http.StatusInternalServerError, err)
			return
		}
		for _, r := range rows {
			ids = append(ids, branchSyncID{UID: r.Uid, UpdatedAt: r.UpdatedAt})
		}
	case "products":
		rows, err := server.store.ListVariantSyncIDs(ctx, db.ListVariantSyncIDsParams{AfterUid: after, RowLimit: int32(limit)})
		if err != nil {
			server.writeError(ctx, http.StatusInternalServerError, err)
			return
		}
		for _, r := range rows {
			ids = append(ids, branchSyncID{UID: r.Uid, UpdatedAt: r.UpdatedAt})
		}
	}

	next := ""
	if len(ids) == limit {
		next = ids[len(ids)-1].UID.String()
	}
	server.writeJSON(ctx, http.StatusOK, envelope{"ids": ids, "next": next})
}

type branchSyncFetchRequest struct {
	Entity string   `json:"entity" binding:"required,oneof=clients products"`
	UIDs   []string `json:"uids" binding:"required,min=1"`
}

// branchSyncFetch returns full catch-up rows (the same shape
// /branch/sync/changes returns for the entity) for specific uids. Unknown
// uids are simply absent from the result.
func (server *Server) branchSyncFetch(ctx *gin.Context) {
	var req branchSyncFetchRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	if len(req.UIDs) > syncFetchMaxUIDs {
		server.writeError(ctx, http.StatusBadRequest, errTooManySyncFetchUIDs)
		return
	}
	uids := make([]uuid.UUID, 0, len(req.UIDs))
	for _, s := range req.UIDs {
		u, err := uuid.Parse(s)
		if err != nil {
			server.writeError(ctx, http.StatusBadRequest, err)
			return
		}
		uids = append(uids, u)
	}

	var items any
	var err error
	switch req.Entity {
	case "clients":
		items, err = server.store.ListClientsByUIDs(ctx, uids)
	case "products":
		items, err = server.store.ListProductVariantsForSyncByUIDs(ctx, uids)
	}
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	server.writeJSON(ctx, http.StatusOK, envelope{"items": items})
}

type branchClientUIDPair struct {
	BranchClientID int64     `json:"branch_client_id"`
	UID            uuid.UUID `json:"uid"`
}

// branchClientUIDs lists this branch's client_links as (local client id ->
// central uid) pairs, so a till adopting shared identities can stamp the
// exact client it was linked to instead of guessing by phone.
func (server *Server) branchClientUIDs(ctx *gin.Context) {
	branchID := ctx.MustGet(branchIDKey).(int64)
	rows, err := server.store.ListBranchClientUIDs(ctx, branchID)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	pairs := make([]branchClientUIDPair, 0, len(rows))
	for _, r := range rows {
		pairs = append(pairs, branchClientUIDPair{BranchClientID: r.BranchClientID, UID: r.Uid})
	}
	server.writeJSON(ctx, http.StatusOK, envelope{"pairs": pairs})
}
