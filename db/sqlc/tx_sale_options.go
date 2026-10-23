package db

import "context"

// SetProductSaleOptionsTx makes the options of every product named by
// entries exactly saleOptionIDs: an empty set clears them. One transaction,
// so a till pulling mid-way never sees a product with its options half
// replaced. Returns how many products it matched.
func (store *SQLStore) SetProductSaleOptionsTx(ctx context.Context, entries []string, saleOptionIDs []int64) (int, error) {
	matched := 0
	err := store.execTx(ctx, func(q *Queries) error {
		if err := q.ClearProductSaleOptionsForEntries(ctx, entries); err != nil {
			return err
		}
		if len(saleOptionIDs) == 0 {
			return nil
		}
		ids, err := q.AddProductSaleOptionsForEntries(ctx, AddProductSaleOptionsForEntriesParams{
			Entries: entries, SaleOptionIds: saleOptionIDs,
		})
		if err != nil {
			return err
		}
		seen := make(map[int64]bool, len(ids))
		for _, id := range ids {
			seen[id] = true
		}
		matched = len(seen)
		return nil
	})
	return matched, err
}
