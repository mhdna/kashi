package api

import (
	db "github.com/mhdna/kashi/db/sqlc"
)

// syncBranchStockRow is one other-branch's on-hand quantity and resolved
// selling price for a variant -- see ListBranchStockForSync. A plain DTO,
// not the raw sqlc row: price is nullable (no active price list covers
// this variant at that branch) and must serialize as a bare 0, not a
// database/sql {Int64,Valid} wrapper -- same reasoning as
// branch_sync_coupons.go's syncCoupon.
type syncBranchStockRow struct {
	BranchCode string `json:"branch_code"`
	VariantUID string `json:"variant_uid"`
	Barcode    string `json:"barcode"`
	Quantity   int64  `json:"quantity"`
	Price      int64  `json:"price"`
}

func newSyncBranchStockRow(r db.ListBranchStockForSyncRow) syncBranchStockRow {
	return syncBranchStockRow{
		BranchCode: r.BranchCode,
		VariantUID: r.VariantUid.String(),
		Barcode:    r.Barcode,
		Quantity:   r.Quantity,
		Price:      r.Price,
	}
}
