package api

import (
	"context"
	"crypto/rand"
	"database/sql"
	_ "embed"
	"encoding/hex"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	db "github.com/mhdna/kashi/db/sqlc"
)

// A phone paired with a till. The till's QR code carries its pairing link
// instead of one sale's: the phone scans it once, keeps the page open, and
// from then on shows whatever unknown items the till's cart holds, sale after
// sale. The till keeps its cart's session up to date as units are added
// (createUnknownSession with background), so each unit is photographed while
// it is still in the salesperson's hand, and the payment rarely waits at all.
// The pictures themselves go through the session's /unknown_uploads routes.

const (
	// unknownPairWait is how long the phone's poll is held when nothing has
	// changed: well under a proxy's read timeout.
	unknownPairWait = 15 * time.Second
	// A phone that asked within this long is shown online on the till.
	unknownPhoneOnlineWithin = 40 * time.Second
)

//go:embed unknown_upload_assets/paired.html
var unknownPairedPageHTML []byte

var unknownPairedPage = newPhoneUploadPage(unknownPairedPageHTML)

func newUnknownToken() (string, error) {
	raw := make([]byte, 24)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return hex.EncodeToString(raw), nil
}

func unknownPhoneSeenRecently(pairing db.UnknownItemPairing) bool {
	return pairing.PhoneSeenAt.Valid && time.Since(pairing.PhoneSeenAt.Time) < unknownPhoneOnlineWithin
}

// unknownPhoneOnline: the till's paired phone has the page open right now.
func (server *Server) unknownPhoneOnline(ctx context.Context, branchID, registerID int64) bool {
	pairing, err := server.store.GetUnknownItemPairing(ctx, db.GetUnknownItemPairingParams{BranchID: branchID, RegisterID: registerID})
	return err == nil && unknownPhoneSeenRecently(pairing)
}

// ensureUnknownPairing is the till's pairing, made with a fresh token the
// first time it is asked for.
func (server *Server) ensureUnknownPairing(ctx context.Context, branchID, registerID int64) (db.UnknownItemPairing, error) {
	token, err := newUnknownToken()
	if err != nil {
		return db.UnknownItemPairing{}, err
	}
	return server.store.EnsureUnknownItemPairing(ctx, db.EnsureUnknownItemPairingParams{
		BranchID: branchID, RegisterID: registerID, Token: token,
	})
}

// setUnknownPairingSession points the till's phone at a cart. The pairing is
// made here if the till has none yet, so a phone paired mid-cart still finds
// the cart.
func (server *Server) setUnknownPairingSession(ctx context.Context, branchID, registerID int64, sessionID sql.NullInt64) error {
	if _, err := server.ensureUnknownPairing(ctx, branchID, registerID); err != nil {
		return err
	}
	return server.store.SetUnknownItemPairingSession(ctx, db.SetUnknownItemPairingSessionParams{
		BranchID: branchID, RegisterID: registerID, CurrentSessionID: sessionID,
	})
}

func (server *Server) writeUnknownPairing(ctx *gin.Context, pairing db.UnknownItemPairing) {
	server.writeJSON(ctx, http.StatusOK, envelope{
		"token":        pairing.Token,
		"phone_online": unknownPhoneSeenRecently(pairing),
	})
}

// branchUnknownPairing is the till's pairing token, for the QR code, and
// whether the phone is online.
func (server *Server) branchUnknownPairing(ctx *gin.Context) {
	pairing, err := server.ensureUnknownPairing(ctx, ctx.MustGet(branchIDKey).(int64), ctx.MustGet(registerIDKey).(int64))
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	server.writeUnknownPairing(ctx, pairing)
}

// resetUnknownPairing is "pair another phone": a new token, and the phone
// holding the old one gets a dead link.
func (server *Server) resetUnknownPairing(ctx *gin.Context) {
	token, err := newUnknownToken()
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	pairing, err := server.store.ResetUnknownItemPairing(ctx, db.ResetUnknownItemPairingParams{
		BranchID: ctx.MustGet(branchIDKey).(int64), RegisterID: ctx.MustGet(registerIDKey).(int64), Token: token,
	})
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	server.writeUnknownPairing(ctx, pairing)
}

// idleUnknownPairing: the till's cart was emptied (paid, held or voided), so
// the phone goes back to waiting. A held cart's session is kept as it is.
func (server *Server) idleUnknownPairing(ctx *gin.Context) {
	if err := server.store.SetUnknownItemPairingSession(ctx, db.SetUnknownItemPairingSessionParams{
		BranchID: ctx.MustGet(branchIDKey).(int64), RegisterID: ctx.MustGet(registerIDKey).(int64),
	}); err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	server.writeJSON(ctx, http.StatusOK, envelope{"idle": true})
}

// --- the paired phone (public, token-gated) ---

func (server *Server) unknownPairedPage(ctx *gin.Context) {
	server.servePhoneUploadPage(ctx, unknownPairedPage)
}

// unknownPairView is what the phone shows: the till's current cart, or
// nothing (no upload token). Sig changes whenever anything shown does.
type unknownPairView struct {
	UploadToken string
	Slots       []unknownSlotView
	Complete    bool
	Sig         string
}

func (server *Server) unknownPairCurrent(ctx context.Context, token string) (unknownPairView, error) {
	view := unknownPairView{Slots: []unknownSlotView{}}
	pairing, err := server.store.GetUnknownItemPairingByToken(ctx, token)
	if err != nil {
		return view, err
	}
	if pairing.CurrentSessionID.Valid {
		session, err := server.store.GetUnknownItemSession(ctx, pairing.CurrentSessionID.Int64)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return view, err
		}
		// A sold or expired cart is over: the phone waits for the next one.
		if err == nil && !session.BranchInvoiceID.Valid && time.Now().UTC().Before(session.ExpiresAt) {
			slots, err := server.store.ListUnknownItemSlotsForSession(ctx, session.ID)
			if err != nil {
				return view, err
			}
			view.UploadToken = session.Token
			view.Slots = unknownSlotViews(slots)
			view.Complete = allUnknownSlotsPictured(slots)
		}
	}
	var sig strings.Builder
	sig.WriteString(view.UploadToken)
	for _, slot := range view.Slots {
		sig.WriteString("|" + slot.SlotRef + ":" + strconv.FormatInt(slot.UnitPrice, 10) + ":" + strconv.FormatInt(slot.ImageID, 10))
		if slot.Waived {
			sig.WriteString(":w")
		}
	}
	view.Sig = sig.String()
	return view, nil
}

// unknownPairStatus is the phone's poll. A phone that already shows `since`
// is held here until the till's cart changes (or a few seconds pass), so a
// unit rung up on the till shows on the phone at once.
func (server *Server) unknownPairStatus(ctx *gin.Context) {
	token := ctx.Param("token")
	view, err := server.unknownPairCurrent(ctx, token)
	since, waiting := ctx.GetQuery("since")
	if err == nil && waiting && since == view.Sig {
		_ = server.store.TouchUnknownItemPairing(ctx, token)
		deadline := time.NewTimer(unknownPairWait)
		defer deadline.Stop()
		tick := time.NewTicker(300 * time.Millisecond)
		defer tick.Stop()
	wait:
		for err == nil && view.Sig == since {
			select {
			case <-ctx.Request.Context().Done():
				return
			case <-deadline.C:
				break wait
			case <-tick.C:
				view, err = server.unknownPairCurrent(ctx, token)
			}
		}
	}
	if errors.Is(err, sql.ErrNoRows) {
		server.writeError(ctx, http.StatusNotFound, errors.New("this link is no longer valid"))
		return
	}
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	_ = server.store.TouchUnknownItemPairing(ctx, token)
	server.writeJSON(ctx, http.StatusOK, envelope{
		"upload_token": view.UploadToken,
		"slots":        view.Slots,
		"complete":     view.Complete,
		"sig":          view.Sig,
	})
}
