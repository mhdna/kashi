package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	db "github.com/mhdna/kashi/db/sqlc"
)

type branchClientRequest struct {
	BranchClientID int64  `json:"branch_client_id" binding:"required"`
	Name           string `json:"name" binding:"required"`
	// Not "required": a legacy till may hold a client with a blank/garbage
	// phone. Rather than 400 (retried forever) or create an unmatchable
	// central row, that case is staged as an 'invalid_phone' conflict.
	Phone string `json:"phone"`
}

// validBranchPhone reports whether a branch-reported phone is usable as the
// global clients.phone key: at least 6 digits after stripping the usual
// formatting characters.
// syntheticBranchPhone is the placeholder key used for a branch client
// with no usable phone, unique per (branch, branch_client_id).
func syntheticBranchPhone(branchID, branchClientID int64) string {
	return fmt.Sprintf("nophone-%d-%d", branchID, branchClientID)
}

func validBranchPhone(phone string) bool {
	digits := 0
	for _, r := range phone {
		if r >= '0' && r <= '9' {
			digits++
		}
	}
	return digits >= 6
}

// putBranchClient is the up-sync half of client sync: a branch reports a
// client it just created or edited locally. Resolution:
//
//  1. Already linked (branch_id, branch_client_id) -> an edit. Update the
//     linked kashi client to match; client_type is never touched.
//  2. Not linked, phone matches an existing kashi client -> link to it so
//     this branch's invoices attribute immediately. If the names differ,
//     stage a 'phone_name_mismatch' conflict for an admin rather than
//     silently keeping either spelling.
//  3. Not linked, phone unusable -> stage an 'invalid_phone' conflict;
//     create and link nothing. Still a 200 so the outbox row is consumed.
//  4. Not linked, no phone match -> genuinely new. Create and link.
func (server *Server) putBranchClient(ctx *gin.Context) {
	branchID := ctx.MustGet(branchIDKey).(int64)

	var req branchClientRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}

	outcome, err := server.resolveBranchClient(ctx, branchID, req)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	server.branchHub.broadcastAll(branchWSMessage{Type: "client_updated"})
	if outcome.conflictKind != "" {
		server.writeJSON(ctx, http.StatusOK, envelope{
			"client": outcome.client, "conflict": true, "conflict_kind": outcome.conflictKind,
		})
		return
	}
	server.writeJSON(ctx, http.StatusOK, envelope{"client": outcome.client})
}

// branchClientOutcome is what resolveBranchClient decided for one client:
// the central client the branch client now maps to, and -- only for the
// "linked to an existing phone under a different name" case -- the conflict
// kind that was staged for an admin.
type branchClientOutcome struct {
	client       db.Client
	conflictKind string
}

// resolveBranchClient applies the resolution rules documented on
// putBranchClient to one branch-reported client. It has no HTTP or websocket
// side effects of its own, so the single-client route and the batch route
// (branch_client_batch.go) share it and each decides how to respond and
// when to notify the other branches. A returned error is a server-side
// failure (retryable); the request itself is never rejected here.
func (server *Server) resolveBranchClient(ctx *gin.Context, branchID int64, req branchClientRequest) (branchClientOutcome, error) {
	req.Phone = strings.TrimSpace(req.Phone)
	payloadJSON, _ := json.Marshal(req)
	refStr := strconv.FormatInt(req.BranchClientID, 10)
	// A blank/garbage phone can't be the global clients.phone key and
	// can't collide-match anything, so give it a stable synthetic value
	// keyed to this branch client. The client still syncs and its invoices
	// still attribute; an admin can set the real number later.
	if !validBranchPhone(req.Phone) {
		req.Phone = syntheticBranchPhone(branchID, req.BranchClientID)
	}

	linkedClientID, err := server.store.GetClientLink(ctx, db.GetClientLinkParams{
		BranchID:       branchID,
		BranchClientID: req.BranchClientID,
	})
	switch {
	case err == nil:
		// An edit of an already-linked client. client_type is never touched
		// -- an admin may have reclassified this client to wholesale, and a
		// branch till has no business reverting that.
		linked, getErr := server.store.GetClient(ctx, linkedClientID)
		if getErr != nil {
			return branchClientOutcome{}, getErr
		}
		client, updErr := server.store.UpdateClient(ctx, db.UpdateClientParams{
			ID:         linkedClientID,
			Name:       req.Name,
			Phone:      req.Phone,
			ClientType: linked.ClientType,
		})
		if updErr != nil {
			return branchClientOutcome{}, updErr
		}
		return branchClientOutcome{client: client}, nil

	case err == sql.ErrNoRows:
		// Not linked yet.
		existing, phoneErr := server.store.GetClientByPhone(ctx, req.Phone)
		switch {
		case phoneErr == nil:
			if linkErr := server.store.UpsertClientLink(ctx, db.UpsertClientLinkParams{
				BranchID:       branchID,
				BranchClientID: req.BranchClientID,
				ClientID:       existing.ID,
			}); linkErr != nil {
				return branchClientOutcome{}, linkErr
			}
			if strings.TrimSpace(existing.Name) != strings.TrimSpace(req.Name) {
				server.stageBranchSyncConflict(ctx, branchID, "client", refStr, "phone_name_mismatch", payloadJSON,
					sql.NullInt64{Int64: existing.ID, Valid: true})
				return branchClientOutcome{client: existing, conflictKind: "phone_name_mismatch"}, nil
			}
			return branchClientOutcome{client: existing}, nil

		case phoneErr == sql.ErrNoRows:
			created, createErr := server.store.CreateClient(ctx, db.CreateClientParams{
				Name:       req.Name,
				Phone:      req.Phone,
				ClientType: "retail",
			})
			var client db.Client
			switch {
			case createErr == nil:
				client = created
			case isUniqueViolation(createErr):
				// A concurrent push won the race on the global phone index.
				// Resolve to the winner rather than 500ing the loser forever.
				won, lookupErr := server.store.GetClientByPhone(ctx, req.Phone)
				if lookupErr != nil {
					return branchClientOutcome{}, lookupErr
				}
				client = won
			default:
				return branchClientOutcome{}, createErr
			}
			if linkErr := server.store.UpsertClientLink(ctx, db.UpsertClientLinkParams{
				BranchID:       branchID,
				BranchClientID: req.BranchClientID,
				ClientID:       client.ID,
			}); linkErr != nil {
				return branchClientOutcome{}, linkErr
			}
			return branchClientOutcome{client: client}, nil

		default:
			return branchClientOutcome{}, phoneErr
		}

	default:
		return branchClientOutcome{}, err
	}
}
