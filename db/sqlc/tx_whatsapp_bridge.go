package db

import "context"

// SaveWhatsAppSenderTx saves how messages go out and which number each
// branch sends from, together.
func (store *SQLStore) SaveWhatsAppSenderTx(ctx context.Context, settings SetWhatsAppSenderSettingsParams, branches []SetBranchWhatsAppNumberParams) error {
	return store.execTx(ctx, func(q *Queries) error {
		if err := q.SetWhatsAppSenderSettings(ctx, settings); err != nil {
			return err
		}
		for _, b := range branches {
			if err := q.SetBranchWhatsAppNumber(ctx, b); err != nil {
				return err
			}
		}
		return nil
	})
}

// UpdateWhatsAppNumberTx renames a number and, when makeDefault, makes it
// the one number every branch without its own sends from.
func (store *SQLStore) UpdateWhatsAppNumberTx(ctx context.Context, id int64, label string, makeDefault bool) (WhatsappNumber, error) {
	var number WhatsappNumber
	err := store.execTx(ctx, func(q *Queries) error {
		if makeDefault {
			if err := q.ClearDefaultWhatsAppNumber(ctx); err != nil {
				return err
			}
			if err := q.SetDefaultWhatsAppNumber(ctx, id); err != nil {
				return err
			}
		}
		var err error
		number, err = q.UpdateWhatsAppNumberLabel(ctx, UpdateWhatsAppNumberLabelParams{ID: id, Label: label})
		return err
	})
	return number, err
}
