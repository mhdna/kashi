package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mhdna/kashi/token"
)

// The live cart: the sale open on a till right now, and changes to it from
// the app bar (ui/src/components/LiveCartButton.vue).
//
// Each till reports its open sale on every change and every 20 s
// (kashi-pos live_cart.go). It's a view, not a record, so kashi keeps only
// the latest per register, in memory: a restart just waits for the next
// report. A report older than liveCartStale is treated as gone (the till is
// off or offline).
//
// A change is a cart_override command to the register: every deal off, or a
// line's price, discount or deal; an unknown item's goes by its line ref. The till applies it itself, past the
// branch's custom price/discount and deal settings, to that one sale only.

const liveCartStale = 75 * time.Second

type liveCartEntry struct {
	BranchID   int64
	ReceivedAt time.Time
	Report     json.RawMessage
	// The last cart_override sent to this register, to say how it went.
	LastCommandID int64
}

type liveCartStore struct {
	mu      sync.Mutex
	entries map[int64]*liveCartEntry // by register id
}

func (s *liveCartStore) put(registerID, branchID int64, report json.RawMessage) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.entries == nil {
		s.entries = map[int64]*liveCartEntry{}
	}
	e := s.entries[registerID]
	if e == nil {
		e = &liveCartEntry{}
		s.entries[registerID] = e
	}
	e.BranchID, e.ReceivedAt, e.Report = branchID, time.Now(), report
}

func (s *liveCartStore) get(registerID int64) (liveCartEntry, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e := s.entries[registerID]
	if e == nil {
		return liveCartEntry{}, false
	}
	return *e, true
}

func (s *liveCartStore) setLastCommand(registerID, commandID int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if e := s.entries[registerID]; e != nil {
		e.LastCommandID = commandID
	}
}

// The few fields of a report kashi reads itself; the rest is passed through.
type liveCartReportHead struct {
	SaleRef string `json:"saleRef"`
	Mode    string `json:"mode"`
}

const liveCartMaxReport = 256 << 10

// branchLiveCart takes a till's report of its open sale.
func (server *Server) branchLiveCart(ctx *gin.Context) {
	branchID := ctx.MustGet(branchIDKey).(int64)
	registerID := ctx.MustGet(registerIDKey).(int64)
	var report json.RawMessage
	ctx.Request.Body = http.MaxBytesReader(ctx.Writer, ctx.Request.Body, liveCartMaxReport)
	if err := ctx.ShouldBindJSON(&report); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	var head liveCartReportHead
	if err := json.Unmarshal(report, &head); err != nil || head.SaleRef == "" {
		server.writeError(ctx, http.StatusBadRequest, errors.New("a live cart report needs saleRef"))
		return
	}
	server.liveCarts.put(registerID, branchID, report)
	server.writeJSON(ctx, http.StatusOK, envelope{"status": "ok"})
}

type liveCartURI struct {
	BranchID int64 `uri:"branch_id" binding:"required,min=1"`
}

type liveCartView struct {
	RegisterID   int64           `json:"register_id"`
	RegisterName string          `json:"register_name"`
	Online       bool            `json:"online"`
	ReceivedAt   *time.Time      `json:"received_at"`
	Cart         json.RawMessage `json:"cart"`
	// How the last change sent from here went: pending, success or failed.
	LastCommand *liveCartCommandView `json:"last_command,omitempty"`
}

type liveCartCommandView struct {
	ID     int64  `json:"id"`
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

// getLiveCarts is every active register of a branch, online says whether it
// is reporting now, and cart is the sale it has open (null when it isn't).
// A register with no name goes by its code.
func (server *Server) getLiveCarts(ctx *gin.Context) {
	var uri liveCartURI
	if err := ctx.ShouldBindUri(&uri); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	registers, err := server.store.ListBranchRegisters(ctx, uri.BranchID)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	out := []liveCartView{}
	for _, r := range registers {
		if !r.IsActive {
			continue
		}
		name := strings.TrimSpace(r.Name)
		if name == "" {
			name = r.Code
		}
		view := liveCartView{RegisterID: r.ID, RegisterName: name}
		if e, ok := server.liveCarts.get(r.ID); ok && e.BranchID == uri.BranchID {
			at := e.ReceivedAt
			view.ReceivedAt = &at
			if time.Since(e.ReceivedAt) < liveCartStale {
				view.Online = true
				view.Cart = e.Report
			}
			if e.LastCommandID > 0 {
				if cmd, err := server.store.GetBranchCommand(ctx, e.LastCommandID); err == nil {
					view.LastCommand = &liveCartCommandView{ID: cmd.ID, Status: cmd.Status, Error: cmd.Error.String}
				}
			}
		}
		out = append(out, view)
	}
	server.writeJSON(ctx, http.StatusOK, envelope{"registers": out})
}

// liveCartLineOverride is one line's change, by barcode. Every unknown item
// shares one barcode, so one is named by its line ref (the till's unknownRef,
// reported as lineRef), and can take a deal choice too.
type liveCartLineOverride struct {
	Barcode     string `json:"barcode" binding:"required"`
	LineRef     string `json:"line_ref,omitempty"`
	PriceCents  *int64 `json:"price_cents,omitempty"`
	Discount    *int64 `json:"discount,omitempty"`
	NoDeal      bool   `json:"no_deal,omitempty"`
	UnknownDeal string `json:"unknown_deal,omitempty"`
}

// The deal choices an unknown item can take (kashi-pos unknown_item_deal.go):
// on the shop's deal at 30% or 20% outside a set, or on none.
var liveCartUnknownDeals = map[string]bool{"b2g1_30": true, "b2g1_20": true, "none": true}

type liveCartOverrideRequest struct {
	RegisterID  int64                  `json:"register_id" binding:"required,min=1"`
	SaleRef     string                 `json:"sale_ref" binding:"required"`
	AllDealsOff bool                   `json:"all_deals_off"`
	Lines       []liveCartLineOverride `json:"lines"`
}

// setLiveCartOverride sends the whole set of changes for the sale open on a
// register; it replaces whatever was sent before, so an empty set undoes
// them all. The till re-checks the sale is still the one on screen.
func (server *Server) setLiveCartOverride(ctx *gin.Context) {
	var uri liveCartURI
	if err := ctx.ShouldBindUri(&uri); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	var req liveCartOverrideRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	register, err := server.store.GetBranchRegister(ctx, req.RegisterID)
	if err != nil || register.BranchID != uri.BranchID {
		server.writeError(ctx, http.StatusBadRequest, errRegisterBranchMismatch)
		return
	}
	e, ok := server.liveCarts.get(req.RegisterID)
	if !ok || time.Since(e.ReceivedAt) >= liveCartStale {
		server.writeError(ctx, http.StatusConflict, errors.New("this till hasn't reported its cart lately; it may be off or offline"))
		return
	}
	var head liveCartReportHead
	_ = json.Unmarshal(e.Report, &head)
	if head.SaleRef != req.SaleRef {
		server.writeError(ctx, http.StatusConflict, errors.New("the till has moved on to another sale; look again"))
		return
	}
	if head.Mode != "sale" {
		server.writeError(ctx, http.StatusConflict, errors.New("the till isn't on a plain sale; only a sale can be changed"))
		return
	}
	for _, l := range req.Lines {
		if l.PriceCents != nil && (*l.PriceCents <= 0 || *l.PriceCents > 99999999) {
			server.writeError(ctx, http.StatusBadRequest, errors.New("a price must be above 0"))
			return
		}
		if l.Discount != nil && (*l.Discount < 0 || *l.Discount > 100) {
			server.writeError(ctx, http.StatusBadRequest, errors.New("a discount must be 0 to 100"))
			return
		}
		if l.UnknownDeal != "" && (l.LineRef == "" || !liveCartUnknownDeals[l.UnknownDeal]) {
			server.writeError(ctx, http.StatusBadRequest, errors.New("a deal choice is 30%, 20% or none, and only for an unknown item"))
			return
		}
	}

	tokenPayload := ctx.MustGet(authoizationPayloadKey).(*token.Payload)
	userID, err := server.callerID(ctx, tokenPayload)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	if req.Lines == nil {
		req.Lines = []liveCartLineOverride{}
	}
	payload, err := json.Marshal(map[string]any{
		"sale_ref":      req.SaleRef,
		"all_deals_off": req.AllDealsOff,
		"lines":         req.Lines,
		"by":            tokenPayload.Username,
	})
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	commands, err := server.enqueueBranchCommand(ctx, uri.BranchID, req.RegisterID, "cart_override", payload, userID)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	if len(commands) > 0 {
		server.liveCarts.setLastCommand(req.RegisterID, commands[0].ID)
	}
	server.writeJSON(ctx, http.StatusOK, envelope{"commands": commands})
}
