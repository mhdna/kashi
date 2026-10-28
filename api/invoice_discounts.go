package api

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	db "github.com/mhdna/kashi/db/sqlc"
)

// Invoice discounts (000212): "invoices of $150 or more get 15% off". A rule
// is its threshold, its discount and the branches it is for; the tills pull
// the rules that apply to them and take the best one off a sale on top of
// its item discounts and deals.

const (
	invoiceDiscountPercent = "percent"
	invoiceDiscountAmount  = "amount"
)

type invoiceDiscountResponse struct {
	db.InvoiceDiscount
	BranchIDs []int64 `json:"branch_ids"`
}

func (server *Server) listInvoiceDiscounts(ctx *gin.Context) {
	rules, err := server.store.ListInvoiceDiscounts(ctx)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	links, err := server.store.ListInvoiceDiscountBranchIDs(ctx)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	branches := make(map[int64][]int64)
	for _, l := range links {
		branches[l.InvoiceDiscountID] = append(branches[l.InvoiceDiscountID], l.BranchID)
	}
	out := make([]invoiceDiscountResponse, len(rules))
	for i, r := range rules {
		out[i] = invoiceDiscountResponse{InvoiceDiscount: r, BranchIDs: branches[r.ID]}
		if out[i].BranchIDs == nil {
			out[i].BranchIDs = []int64{}
		}
	}
	server.writeJSON(ctx, http.StatusOK, envelope{"invoice_discounts": out})
}

type invoiceDiscountRequest struct {
	Name     string `json:"name"`
	MinTotal int64  `json:"min_total" binding:"min=0"`
	Kind     string `json:"kind" binding:"required"`
	Percent  int16  `json:"percent" binding:"min=0,max=100"`
	Amount   int64  `json:"amount" binding:"min=0"`
	// Empty with all_branches false is a rule for no branch at all, which
	// the request refuses rather than save a rule that does nothing.
	AllBranches bool       `json:"all_branches"`
	BranchIDs   []int64    `json:"branch_ids"`
	ValidFrom   *time.Time `json:"valid_from"`
	ValidTo     *time.Time `json:"valid_to"`
	IsActive    bool       `json:"is_active"`
}

// params checks the request and clears the field its kind doesn't read, so
// the table's CHECK never has to be the one to say.
func (req invoiceDiscountRequest) params() (db.SaveInvoiceDiscountTxParams, error) {
	c := db.CreateInvoiceDiscountParams{
		Name: strings.TrimSpace(req.Name), MinTotal: req.MinTotal, Kind: req.Kind,
		AllBranches: req.AllBranches, IsActive: req.IsActive,
	}
	if c.Name == "" {
		return db.SaveInvoiceDiscountTxParams{}, errors.New("name is required")
	}
	switch req.Kind {
	case invoiceDiscountPercent:
		if req.Percent < 1 {
			return db.SaveInvoiceDiscountTxParams{}, errors.New("say how much off the invoice gets")
		}
		c.Percent = req.Percent
	case invoiceDiscountAmount:
		if req.Amount <= 0 {
			return db.SaveInvoiceDiscountTxParams{}, errors.New("say how much comes off the invoice")
		}
		c.Amount = req.Amount
	default:
		return db.SaveInvoiceDiscountTxParams{}, fmt.Errorf("unknown invoice discount kind %q", req.Kind)
	}
	if !req.AllBranches && len(req.BranchIDs) == 0 {
		return db.SaveInvoiceDiscountTxParams{}, errors.New("pick the branches it is for, or all branches")
	}
	if req.ValidFrom != nil {
		c.ValidFrom = sql.NullTime{Time: *req.ValidFrom, Valid: true}
	}
	if req.ValidTo != nil {
		c.ValidTo = sql.NullTime{Time: *req.ValidTo, Valid: true}
	}
	if c.ValidFrom.Valid && c.ValidTo.Valid && !c.ValidTo.Time.After(c.ValidFrom.Time) {
		return db.SaveInvoiceDiscountTxParams{}, errors.New("the end date has to be after the start date")
	}
	return db.SaveInvoiceDiscountTxParams{Create: c, BranchIDs: req.BranchIDs}, nil
}

func (server *Server) createInvoiceDiscount(ctx *gin.Context) {
	server.saveInvoiceDiscount(ctx, 0)
}

type invoiceDiscountURI struct {
	ID int64 `uri:"id" binding:"required,min=1"`
}

// updateInvoiceDiscount changes a rule on every till it reaches. Sales
// already made keep the discount they were sold with.
func (server *Server) updateInvoiceDiscount(ctx *gin.Context) {
	var uri invoiceDiscountURI
	if err := ctx.ShouldBindUri(&uri); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	server.saveInvoiceDiscount(ctx, uri.ID)
}

func (server *Server) saveInvoiceDiscount(ctx *gin.Context, id int64) {
	var req invoiceDiscountRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	p, err := req.params()
	if err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	p.ID = id
	rule, err := server.store.SaveInvoiceDiscountTx(ctx, p)
	if err != nil {
		if isForeignKeyViolation(err) {
			server.writeError(ctx, http.StatusBadRequest, errors.New("one of the branches doesn't exist"))
			return
		}
		server.writeError(ctx, statusForLookup(err), err)
		return
	}
	server.writeJSON(ctx, http.StatusOK, envelope{"invoice_discount": rule})
}

func (server *Server) deleteInvoiceDiscount(ctx *gin.Context) {
	var uri invoiceDiscountURI
	if err := ctx.ShouldBindUri(&uri); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	if err := server.store.DeleteInvoiceDiscount(ctx, uri.ID); err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	server.writeJSON(ctx, http.StatusOK, envelope{"status": "deleted"})
}

// syncInvoiceDiscount is a rule as a till pulls it. A rule that doesn't
// apply to the branch (deleted, switched off, or not for it) comes down
// with applies false, and the till drops it.
type syncInvoiceDiscount struct {
	UID       uuid.UUID  `json:"uid"`
	Name      string     `json:"name"`
	MinTotal  int64      `json:"min_total"`
	Kind      string     `json:"kind"`
	Percent   int16      `json:"percent"`
	Amount    int64      `json:"amount"`
	ValidFrom *time.Time `json:"valid_from"`
	ValidTo   *time.Time `json:"valid_to"`
	Applies   bool       `json:"applies"`
	UpdatedAt time.Time  `json:"updated_at"`
}

func (server *Server) invoiceDiscountsForBranchSync(ctx *gin.Context, branchID int64, since time.Time) ([]syncInvoiceDiscount, error) {
	rows, err := server.store.ListInvoiceDiscountsForBranchSync(ctx, db.ListInvoiceDiscountsForBranchSyncParams{
		BranchID: branchID, Since: since,
	})
	if err != nil {
		return nil, err
	}
	out := make([]syncInvoiceDiscount, len(rows))
	for i, r := range rows {
		out[i] = syncInvoiceDiscount{
			UID: r.Uid, Name: r.Name, MinTotal: r.MinTotal, Kind: r.Kind, Percent: r.Percent,
			Amount: r.Amount, Applies: r.Applies, UpdatedAt: r.UpdatedAt,
		}
		if r.ValidFrom.Valid {
			out[i].ValidFrom = &r.ValidFrom.Time
		}
		if r.ValidTo.Valid {
			out[i].ValidTo = &r.ValidTo.Time
		}
	}
	return out, nil
}
