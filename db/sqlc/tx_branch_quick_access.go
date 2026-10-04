package db

import "context"

// QuickAccessItemInput is one card of a branch's Quick Access list, in the
// order the list is given (its index becomes the card's position).
type QuickAccessItemInput struct {
	ProductVariantID int64
	Color            string
	TextColor        string
}

// ReplaceBranchQuickAccessTx saves a branch's whole list, as edited in the
// admin UI. It also marks the list as set up, so a till reporting its own
// list afterwards can't overwrite it (see ImportBranchQuickAccessTx).
func (store *SQLStore) ReplaceBranchQuickAccessTx(ctx context.Context, branchID int64, items []QuickAccessItemInput) error {
	return store.execTx(ctx, func(q *Queries) error {
		if _, err := q.ClaimBranchQuickAccessInit(ctx, branchID); err != nil {
			return err
		}
		return replaceBranchQuickAccessItems(ctx, q, branchID, items)
	})
}

// ImportBranchQuickAccessTx seeds a branch's list from a till's own list --
// only the first time any list is set up for the branch. It reports whether
// it imported; false means the branch already had a list, left untouched.
// An empty list imports nothing and doesn't count as setting one up, so the
// branch's next register (which may have real cards) can still seed it.
func (store *SQLStore) ImportBranchQuickAccessTx(ctx context.Context, branchID int64, items []QuickAccessItemInput) (bool, error) {
	if len(items) == 0 {
		return false, nil
	}
	imported := false
	err := store.execTx(ctx, func(q *Queries) error {
		claimed, err := q.ClaimBranchQuickAccessInit(ctx, branchID)
		if err != nil || claimed == 0 {
			return err
		}
		imported = true
		return replaceBranchQuickAccessItems(ctx, q, branchID, items)
	})
	return imported, err
}

func replaceBranchQuickAccessItems(ctx context.Context, q *Queries, branchID int64, items []QuickAccessItemInput) error {
	if err := q.DeleteBranchQuickAccessItems(ctx, branchID); err != nil {
		return err
	}
	for i, item := range items {
		if err := q.InsertBranchQuickAccessItem(ctx, InsertBranchQuickAccessItemParams{
			BranchID:         branchID,
			ProductVariantID: item.ProductVariantID,
			Position:         int32(i),
			Color:            item.Color,
			TextColor:        item.TextColor,
		}); err != nil {
			return err
		}
	}
	return nil
}
