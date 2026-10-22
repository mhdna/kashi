package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	db "github.com/mhdna/kashi/db/sqlc"
)

type branchClientRequest struct {
	BranchClientID int64  `json:"branch_client_id" binding:"required"`
	Name           string `json:"name" binding:"required"`
	// Not "required": a legacy till may hold a client with a blank/garbage
	// phone. Rather than 400 (retried forever) or create an unmatchable
	// central row, that case is staged as an 'invalid_phone' conflict.
	Phone string `json:"phone"`
	// UID is the client's shared identity, minted by whichever side created
	// the client. Tills that predate uid sync omit it.
	UID string `json:"uid"`
}

// validBranchPhone reports whether a branch-reported phone is usable as the
// global clients.phone key: at least 6 digits after stripping the usual
// formatting characters.
// syntheticBranchPhone is the placeholder key used for a branch client
// with no usable phone, unique per (register, branch_client_id) -- the id
// is a register's local clients.id, so two tills both have a client 5.
// syncedPhoneValid and syncedPhoneCountry fill the clients page's phone_valid
// and phone_country for a client a till reported, which the admin form and
// the cleanup job otherwise set: without them every till-made client had no
// flag. The stored phone text stays exactly as reported (branch matching keys
// on it). A "nophone-..." placeholder isn't a typed number, so it isn't marked
// invalid and has no country.
func syncedPhoneValid(phone string) bool {
	if strings.HasPrefix(phone, "nophone-") {
		return true
	}
	_, _, valid := parseAnyPhone(phone)
	return valid
}

func syncedPhoneCountry(phone string) string {
	_, region, _ := parseAnyPhone(phone)
	return region
}

func syntheticBranchPhone(registerID, branchClientID int64) string {
	return fmt.Sprintf("nophone-r%d-%d", registerID, branchClientID)
}

// clientConflictRef is a client conflict's ref: the reporting register and
// its local client id, since the id alone collides across registers.
func clientConflictRef(registerID, branchClientID int64) string {
	return fmt.Sprintf("%d:%d", registerID, branchClientID)
}

func parseClientConflictRef(ref string) (registerID, branchClientID int64, err error) {
	registerPart, clientPart, ok := strings.Cut(ref, ":")
	if !ok {
		return 0, 0, fmt.Errorf("client conflict ref %q has no register", ref)
	}
	if registerID, err = strconv.ParseInt(registerPart, 10, 64); err != nil {
		return 0, 0, err
	}
	if branchClientID, err = strconv.ParseInt(clientPart, 10, 64); err != nil {
		return 0, 0, err
	}
	return registerID, branchClientID, nil
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
	registerID := ctx.MustGet(registerIDKey).(int64)

	var req branchClientRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}

	outcome, err := server.resolveBranchClient(ctx, branchID, registerID, req)
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
func (server *Server) resolveBranchClient(ctx *gin.Context, branchID, registerID int64, req branchClientRequest) (branchClientOutcome, error) {
	req.Phone = strings.TrimSpace(req.Phone)
	payloadJSON, _ := json.Marshal(req)
	refStr := clientConflictRef(registerID, req.BranchClientID)
	// A blank/garbage phone can't be the global clients.phone key and
	// can't collide-match anything, so give it a stable synthetic value
	// keyed to this register's client. The client still syncs and its invoices
	// still attribute; an admin can set the real number later.
	if !validBranchPhone(req.Phone) {
		req.Phone = syntheticBranchPhone(registerID, req.BranchClientID)
	}

	// A till that sent a uid already knows which central client it means:
	// either kashi has it (edit) or it's a new client the till created.
	if uid := parseNullUUID(req.UID); uid.Valid {
		return server.resolveBranchClientByUID(ctx, branchID, registerID, req, uid.UUID, refStr, payloadJSON)
	}

	linkedClientID, err := server.store.GetClientLink(ctx, db.GetClientLinkParams{
		RegisterID:     registerID,
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
			ID:           linkedClientID,
			Name:         req.Name,
			Phone:        req.Phone,
			ClientType:   linked.ClientType,
			PhoneValid:   syncedPhoneValid(req.Phone),
			PhoneCountry: syncedPhoneCountry(req.Phone),
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
				RegisterID:     registerID,
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
				Name:         req.Name,
				Phone:        req.Phone,
				ClientType:   "retail",
				PhoneValid:   syncedPhoneValid(req.Phone),
				PhoneCountry: syncedPhoneCountry(req.Phone),
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
				RegisterID:     registerID,
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

// resolveBranchClientByUID is resolveBranchClient for a till that sends the
// client's shared uid. Resolution order:
//  1. uid known centrally: update its name/phone (kashi keeps client_type).
//  2. phone already belongs to another client: that client is canonical --
//     the till adopts its uid (the response carries it). Nothing is created.
//  3. otherwise: create the client under the till's uid.
//
// client_links is still upserted every time so older reports and
// invoice attribution keyed on branch_client_id keep working.
func (server *Server) resolveBranchClientByUID(ctx *gin.Context, branchID, registerID int64, req branchClientRequest, uid uuid.UUID, refStr string, payloadJSON []byte) (branchClientOutcome, error) {
	var client db.Client
	conflictKind := ""

	existing, err := server.store.GetClientByUID(ctx, uid)
	switch {
	case err == nil:
		// If the new phone is taken by a different client, keep this
		// client's phone as is rather than violating the unique index; an
		// admin resolves the clash from the staged conflict.
		phone := req.Phone
		if other, phoneErr := server.store.GetClientByPhone(ctx, req.Phone); phoneErr == nil && other.ID != existing.ID {
			phone = existing.Phone
			conflictKind = "phone_taken"
			server.stageBranchSyncConflict(ctx, branchID, "client", refStr, conflictKind, payloadJSON,
				sql.NullInt64{Int64: other.ID, Valid: true})
		}
		client, err = server.store.UpdateClient(ctx, db.UpdateClientParams{
			ID:           existing.ID,
			Name:         req.Name,
			Phone:        phone,
			ClientType:   existing.ClientType,
			PhoneValid:   syncedPhoneValid(phone),
			PhoneCountry: syncedPhoneCountry(phone),
		})
		if err != nil {
			return branchClientOutcome{}, err
		}

	case errors.Is(err, sql.ErrNoRows):
		byPhone, phoneErr := server.store.GetClientByPhone(ctx, req.Phone)
		switch {
		case phoneErr == nil:
			client = byPhone
			if err := server.store.RepointBranchInvoicesClientUID(ctx, db.RepointBranchInvoicesClientUIDParams{
				CanonicalUid: byPhone.Uid,
				BranchID:     branchID,
				TillUid:      uid,
			}); err != nil {
				return branchClientOutcome{}, err
			}
			if strings.TrimSpace(byPhone.Name) != strings.TrimSpace(req.Name) {
				conflictKind = "phone_name_mismatch"
				server.stageBranchSyncConflict(ctx, branchID, "client", refStr, conflictKind, payloadJSON,
					sql.NullInt64{Int64: byPhone.ID, Valid: true})
			}
		case errors.Is(phoneErr, sql.ErrNoRows):
			created, createErr := server.store.CreateClientWithUID(ctx, db.CreateClientWithUIDParams{
				Uid:   uid,
				Name:  req.Name,
				Phone: req.Phone,
			})
			switch {
			case createErr == nil:
				client = created
			case isUniqueViolation(createErr):
				// A concurrent push won the race on phone or uid; resolve
				// to whichever row now holds this phone.
				won, lookupErr := server.store.GetClientByPhone(ctx, req.Phone)
				if lookupErr != nil {
					return branchClientOutcome{}, lookupErr
				}
				client = won
			default:
				return branchClientOutcome{}, createErr
			}
		default:
			return branchClientOutcome{}, phoneErr
		}

	default:
		return branchClientOutcome{}, err
	}

	if err := server.store.UpsertClientLink(ctx, db.UpsertClientLinkParams{
		BranchID:       branchID,
		RegisterID:     registerID,
		BranchClientID: req.BranchClientID,
		ClientID:       client.ID,
	}); err != nil {
		return branchClientOutcome{}, err
	}
	return branchClientOutcome{client: client, conflictKind: conflictKind}, nil
}
