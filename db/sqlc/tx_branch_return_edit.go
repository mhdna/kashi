package db

import (
	"context"
	"database/sql"
	"fmt"
)

// ApplyBranchReturnEditTxParams is a till's edit of a return or exchange it
// already synced (kashi-pos return_edit.go, sent to /branch/return_invoice_edits):
// lines named by position, the new total and the new payments.
type ApplyBranchReturnEditTxParams struct {
	BranchID   int64
	ClientRef  string
	GrandTotal int64
	LbpTotal   sql.NullInt64
	LbpRate    sql.NullInt64
	Lines      []BranchReturnEditLine
	Payments   []BranchInvoicePaymentParams
}

type BranchReturnEditLine struct {
	Position  int
	LineTotal int64
}

// ApplyBranchReturnEditTx sets the edited lines, the totals and the payments
// in one transaction. Setting values makes a retried sync a no-op.
func (store *SQLStore) ApplyBranchReturnEditTx(ctx context.Context, arg ApplyBranchReturnEditTxParams) (BranchInvoice, error) {
	var invoice BranchInvoice
	err := store.execTx(ctx, func(q *Queries) error {
		var err error
		invoice, err = q.GetBranchInvoiceByClientRef(ctx, GetBranchInvoiceByClientRefParams{BranchID: arg.BranchID, ClientRef: arg.ClientRef})
		if err != nil {
			return err
		}
		if invoice.Kind != "return" && invoice.Kind != "exchange" {
			return fmt.Errorf("%s is not a return or an exchange", invoice.BranchInvoiceCode)
		}
		ids, err := q.ListBranchInvoiceItemIDs(ctx, invoice.ID)
		if err != nil {
			return err
		}
		for _, line := range arg.Lines {
			if line.Position < 0 || line.Position >= len(ids) {
				return fmt.Errorf("line %d is not on %s", line.Position+1, invoice.BranchInvoiceCode)
			}
			if err := q.SetBranchInvoiceItemLineTotal(ctx, SetBranchInvoiceItemLineTotalParams{ID: ids[line.Position], LineTotal: line.LineTotal}); err != nil {
				return err
			}
		}
		if err := q.SetBranchReturnTotals(ctx, SetBranchReturnTotalsParams{ID: invoice.ID, Subtotal: arg.GrandTotal, LbpTotal: arg.LbpTotal, LbpRate: arg.LbpRate}); err != nil {
			return err
		}
		if err := q.DeleteBranchInvoicePaymentsForInvoice(ctx, invoice.ID); err != nil {
			return err
		}
		for _, p := range arg.Payments {
			if _, err := q.CreateBranchInvoicePayment(ctx, CreateBranchInvoicePaymentParams{BranchInvoiceID: invoice.ID, AccountName: p.AccountName, Amount: p.Amount}); err != nil {
				return err
			}
		}
		invoice, err = q.GetBranchInvoice(ctx, invoice.ID)
		return err
	})
	return invoice, err
}
