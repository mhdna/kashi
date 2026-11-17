package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	db "github.com/mhdna/kashi/db/sqlc"
	"github.com/mhdna/kashi/token"
)

// remotePaymentInput is one payment line the admin entered for a remote
// return, exchange or settlement change, the way the till reads it
// (kashi-pos remote_payments.go remotePayment): the till's payment method by
// name, the amount in USD cents, signed (negative is money given back), and
// for a cash-rounded currency (LBP) the amount as counted in its units.
type remotePaymentInput struct {
	AccountName string `json:"account_name"`
	AmountCents int64  `json:"amount_cents"`
	AmountUnits int64  `json:"amount_units,omitempty"`
}

// checkRemotePayments refuses what the till would refuse anyway, so the admin
// hears it at once: a line without a method or an amount, a coupon (entered
// on the till only), and totals that overflow. It answers the lines' sum and
// whether any is counted in a cash-rounded currency's units.
func checkRemotePayments(payments []remotePaymentInput) (sum int64, counted bool, err error) {
	for _, p := range payments {
		name := strings.TrimSpace(p.AccountName)
		if name == "" {
			return 0, false, errors.New("each payment needs a payment method")
		}
		if strings.EqualFold(name, "On Coupon") {
			return 0, false, errors.New("coupons can't be entered from kashi; do it on the till")
		}
		if p.AmountCents == 0 {
			return 0, false, fmt.Errorf("%s has no amount", name)
		}
		if (p.AmountCents > 0 && sum > math.MaxInt64-p.AmountCents) || (p.AmountCents < 0 && sum < math.MinInt64-p.AmountCents) {
			return 0, false, errors.New("payment total is too large")
		}
		sum += p.AmountCents
		if p.AmountUnits != 0 {
			counted = true
		}
	}
	return sum, counted, nil
}

// remoteReturnPayments checks a remote return's optional payments. A cancel
// takes none: it refunds the whole sale the way it was paid.
func remoteReturnPayments(raw json.RawMessage, cancel bool) error {
	var body struct {
		Payments []remotePaymentInput `json:"payments"`
	}
	if err := json.Unmarshal(raw, &body); err != nil {
		return err
	}
	if len(body.Payments) == 0 {
		return nil
	}
	if cancel {
		return errors.New("a cancel refunds the sale the way it was paid; it takes no payments")
	}
	_, _, err := checkRemotePayments(body.Payments)
	return err
}

// changeInvoiceSettlementPayload is the till's change_invoice_settlement
// (kashi-pos remote_payments.go changeInvoiceSettlementPayload).
type changeInvoiceSettlementPayload struct {
	ClientRef string               `json:"client_ref"`
	Payments  []remotePaymentInput `json:"payments"`
	Reason    string               `json:"reason"`
	EditedBy  string               `json:"edited_by"`
}

// resolveChangeSettlementTarget sends a sale's new settlement to the till that
// rang it up, once it adds up to the sale's total.
func (server *Server) resolveChangeSettlementTarget(ctx *gin.Context, branchID int64, raw json.RawMessage) (int64, json.RawMessage, int, error) {
	var p changeInvoiceSettlementPayload
	if err := json.Unmarshal(raw, &p); err != nil || strings.TrimSpace(p.ClientRef) == "" {
		return 0, nil, http.StatusBadRequest, errors.New("change_invoice_settlement needs the sale's client_ref")
	}
	if len(p.Payments) == 0 {
		return 0, nil, http.StatusBadRequest, errors.New("enter at least one payment")
	}
	sum, counted, err := checkRemotePayments(p.Payments)
	if err != nil {
		return 0, nil, http.StatusBadRequest, err
	}
	invoice, err := server.store.GetBranchInvoiceByClientRef(ctx, db.GetBranchInvoiceByClientRefParams{BranchID: branchID, ClientRef: p.ClientRef})
	if err != nil {
		return 0, nil, statusForLookup(err), err
	}
	if invoice.Historical {
		return 0, nil, http.StatusBadRequest, errors.New("this invoice was imported from the old POS; no till has it")
	}
	if invoice.Kind != "sales" {
		return 0, nil, http.StatusBadRequest, errors.New("only a sale's settlement can be changed")
	}
	if invoice.CancelledAt.Valid {
		return 0, nil, http.StatusBadRequest, errors.New("this sale was cancelled")
	}
	// LBP counted in its units is rounded on the till, which then judges it.
	if !counted && sum != invoice.GrandTotal {
		return 0, nil, http.StatusBadRequest, fmt.Errorf("the payments add up to %s but the sale is %s", formatUSDCents(sum), formatUSDCents(invoice.GrandTotal))
	}
	p.EditedBy = "kashi"
	if tp, ok := ctx.Get(authoizationPayloadKey); ok {
		if tokenPayload, ok := tp.(*token.Payload); ok && tokenPayload.Username != "" {
			p.EditedBy = tokenPayload.Username
		}
	}
	payload, err := json.Marshal(p)
	if err != nil {
		return 0, nil, http.StatusInternalServerError, err
	}
	return invoice.RegisterID, payload, 0, nil
}

func formatUSDCents(cents int64) string {
	sign := ""
	if cents < 0 {
		sign, cents = "-", -cents
	}
	return fmt.Sprintf("%s$%d.%02d", sign, cents/100, cents%100)
}
