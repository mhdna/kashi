package db

import "context"

// ReplaceWhatsAppRulesTx swaps the whole rule list for rules, in order: the
// WhatsApp page saves the list as one edit, like the auto-fill rules.
// Positions are taken from the slice order.
func (store *SQLStore) ReplaceWhatsAppRulesTx(ctx context.Context, rules []CreateWhatsAppRuleParams) ([]WhatsappRule, error) {
	saved := make([]WhatsappRule, 0, len(rules))
	err := store.execTx(ctx, func(q *Queries) error {
		if err := q.DeleteAllWhatsAppRules(ctx); err != nil {
			return err
		}
		for i, rule := range rules {
			rule.Position = int32(i)
			row, err := q.CreateWhatsAppRule(ctx, rule)
			if err != nil {
				return err
			}
			saved = append(saved, row)
		}
		return nil
	})
	return saved, err
}
