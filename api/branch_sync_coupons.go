package api

import (
	"time"

	db "github.com/mhdna/kashi/db/sqlc"
)

// Coupons and transfer requests carry several nullable central columns
// (redeemed_at, dispatched_at, a coupon's optional client_id...) that the
// till either doesn't need or needs in a plainer shape than
// database/sql's Null* types round-trip through JSON as ({"Int64":..,
// "Valid":..} rather than a bare number or omission). These types flatten
// that the same way branchInvoiceRequest.ClientUID already does for the
// up-sync direction: a zero value (empty string, 0) means "unset", never
// a wrapper object.

type syncCouponType struct {
	ID                  int64  `json:"id"`
	Name                string `json:"name"`
	ValueCents          int64  `json:"value_cents"`
	CurrencyCode        string `json:"currency_code"`
	IsActive            bool   `json:"is_active"`
	ReferenceOnly       bool   `json:"reference_only"`
	CalculateOnCheckout bool   `json:"calculate_on_checkout"`
	CodeOptional        bool   `json:"code_optional"`
}

func newSyncCouponType(t db.CouponType) syncCouponType {
	return syncCouponType{
		ID:                  t.ID,
		Name:                t.Name,
		ValueCents:          t.ValueCents,
		CurrencyCode:        t.CurrencyCode,
		IsActive:            t.IsActive,
		ReferenceOnly:       t.ReferenceOnly,
		CalculateOnCheckout: t.CalculateOnCheckout,
		CodeOptional:        t.CodeOptional,
	}
}

type syncCoupon struct {
	Code       string `json:"code"`
	Status     string `json:"status"`
	TypeID     int64  `json:"type_id"`
	ValueCents int64  `json:"value_cents"`
	// ValidUntil is RFC3339, or "" when the coupon never expires.
	ValidUntil string `json:"valid_until"`
}

func newSyncCoupon(c db.Coupon) syncCoupon {
	out := syncCoupon{
		Code:       c.Code,
		Status:     string(c.Status),
		ValueCents: c.ValueCents,
	}
	if c.TypeID.Valid {
		out.TypeID = c.TypeID.Int64
	}
	if c.ValidUntil.Valid {
		out.ValidUntil = c.ValidUntil.Time.Format(time.RFC3339)
	}
	return out
}

// syncTransferRequest is the till's own poll for what happened to a
// transfer it proposed -- see ListBranchTransfersUpdatedSince. Only what
// the till needs to reflect a decision back to the cashier: its status,
// keyed by the client_ref the till itself generated.
type syncTransferRequest struct {
	ClientRef string `json:"client_ref"`
	Status    string `json:"status"`
}

func newSyncTransferRequest(t db.Transfer) syncTransferRequest {
	out := syncTransferRequest{Status: string(t.Status)}
	if t.ClientRef.Valid {
		out.ClientRef = t.ClientRef.String
	}
	return out
}
