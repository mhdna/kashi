package db

import "context"

type SaveInvoiceDiscountTxParams struct {
	// 0 creates a rule.
	ID        int64
	Create    CreateInvoiceDiscountParams
	BranchIDs []int64
}

// SaveInvoiceDiscountTx writes a rule and the branches it is for together,
// so a till never pulls a rule whose branch list is half written. Saving
// moves updated_at, which re-sends the rule to every till -- a branch taken
// off it learns that way (000212).
func (store *SQLStore) SaveInvoiceDiscountTx(ctx context.Context, arg SaveInvoiceDiscountTxParams) (InvoiceDiscount, error) {
	var out InvoiceDiscount
	err := store.execTx(ctx, func(q *Queries) error {
		var err error
		c := arg.Create
		if arg.ID == 0 {
			out, err = q.CreateInvoiceDiscount(ctx, c)
		} else {
			out, err = q.UpdateInvoiceDiscount(ctx, UpdateInvoiceDiscountParams{
				ID: arg.ID, Name: c.Name, MinTotal: c.MinTotal, Kind: c.Kind, Percent: c.Percent,
				Amount: c.Amount, AllBranches: c.AllBranches, ValidFrom: c.ValidFrom,
				ValidTo: c.ValidTo, IsActive: c.IsActive,
			})
		}
		if err != nil {
			return err
		}
		if err := q.DeleteInvoiceDiscountBranches(ctx, out.ID); err != nil {
			return err
		}
		if c.AllBranches {
			return nil
		}
		for _, branchID := range arg.BranchIDs {
			if err := q.AddInvoiceDiscountBranch(ctx, AddInvoiceDiscountBranchParams{InvoiceDiscountID: out.ID, BranchID: branchID}); err != nil {
				return err
			}
		}
		return nil
	})
	return out, err
}
