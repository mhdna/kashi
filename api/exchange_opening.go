package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	db "github.com/mhdna/kashi/db/sqlc"
	"github.com/mhdna/kashi/token"
)

// maxExchangeOpeningDays caps how far ahead a sale can be opened for exchange.
const maxExchangeOpeningDays = 366

// openInvoiceExchangePayload is the till's open_invoice_exchange (kashi-pos
// exchange_opening.go): the sale can be exchanged on the till until the end
// of UntilDate, the shop's day, past the branch's exchange window.
type openInvoiceExchangePayload struct {
	ClientRef string `json:"client_ref"`
	UntilDate string `json:"until_date"`
	Reason    string `json:"reason"`
	OpenedBy  string `json:"opened_by"`
}

// checkExchangeOpeningDate accepts a YYYY-MM-DD from yesterday (the admin's
// browser may sit in a zone behind this server) up to a year ahead. The till
// judges "today" by its own clock.
func checkExchangeOpeningDate(value string, now time.Time) error {
	day, err := time.Parse("2006-01-02", strings.TrimSpace(value))
	if err != nil {
		return errors.New("until_date must be a date like 2026-11-22")
	}
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	if day.Before(today.AddDate(0, 0, -1)) {
		return errors.New("that date has already passed")
	}
	if day.After(today.AddDate(0, 0, maxExchangeOpeningDays)) {
		return errors.New("a sale can be opened for exchange up to a year ahead")
	}
	return nil
}

// resolveOpenExchangeTarget sends the opening to the till that rang the sale up.
func (server *Server) resolveOpenExchangeTarget(ctx *gin.Context, branchID int64, raw json.RawMessage) (int64, json.RawMessage, int, error) {
	var p openInvoiceExchangePayload
	if err := json.Unmarshal(raw, &p); err != nil || strings.TrimSpace(p.ClientRef) == "" {
		return 0, nil, http.StatusBadRequest, errors.New("open_invoice_exchange needs the sale's client_ref")
	}
	if err := checkExchangeOpeningDate(p.UntilDate, time.Now()); err != nil {
		return 0, nil, http.StatusBadRequest, err
	}
	p.UntilDate = strings.TrimSpace(p.UntilDate)
	invoice, err := server.store.GetBranchInvoiceByClientRef(ctx, db.GetBranchInvoiceByClientRefParams{BranchID: branchID, ClientRef: p.ClientRef})
	if err != nil {
		return 0, nil, statusForLookup(err), err
	}
	if invoice.Historical {
		return 0, nil, http.StatusBadRequest, errors.New("this invoice was imported from the old POS; no till has it")
	}
	if invoice.Kind != "sales" {
		return 0, nil, http.StatusBadRequest, errors.New("only a sale can be opened for exchange")
	}
	if invoice.CancelledAt.Valid {
		return 0, nil, http.StatusBadRequest, errors.New("this sale was cancelled")
	}
	p.OpenedBy = "kashi"
	if tp, ok := ctx.Get(authoizationPayloadKey); ok {
		if tokenPayload, ok := tp.(*token.Payload); ok && tokenPayload.Username != "" {
			p.OpenedBy = tokenPayload.Username
		}
	}
	payload, err := json.Marshal(p)
	if err != nil {
		return 0, nil, http.StatusInternalServerError, err
	}
	return invoice.RegisterID, payload, 0, nil
}
