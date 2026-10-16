package db

import "context"

// SaveReviewSettingsTx saves the review link settings and every branch's own
// Google place together, so the Reviews tab never half-saves.
func (store *SQLStore) SaveReviewSettingsTx(ctx context.Context, settings SetReviewSettingsParams, branches []SetBranchPlaceIDParams) error {
	return store.execTx(ctx, func(q *Queries) error {
		if err := q.SetReviewSettings(ctx, settings); err != nil {
			return err
		}
		for _, b := range branches {
			if err := q.SetBranchPlaceID(ctx, b); err != nil {
				return err
			}
		}
		return nil
	})
}
