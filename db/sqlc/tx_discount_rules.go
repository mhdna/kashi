package db

import "context"

// DiscountListRuleInput is one rule as the list page sends it: the rule and
// its option discounts (sale option id -> discount).
type DiscountListRuleInput struct {
	Rule    CreateDiscountListRuleParams
	Options map[int64]int16
}

// ReplaceDiscountListRulesTx makes a list's rules exactly rules, in that
// order. One transaction, so the rows they write (000200, at commit) are
// worked out once from the finished set, never from half of it.
func (store *SQLStore) ReplaceDiscountListRulesTx(ctx context.Context, listID int64, rules []DiscountListRuleInput) error {
	return store.execTx(ctx, func(q *Queries) error {
		if err := q.DeleteDiscountListRules(ctx, listID); err != nil {
			return err
		}
		for i, in := range rules {
			in.Rule.DiscountListID = listID
			in.Rule.Position = int32(i)
			// A nil condition reaches Postgres as NULL; it means "any".
			for _, list := range []*[]string{&in.Rule.CodePrefixes, &in.Rule.ExceptCodePrefixes, &in.Rule.Kinds,
				&in.Rule.Categories, &in.Rule.SubCategories, &in.Rule.Origins, &in.Rule.Brands} {
				if *list == nil {
					*list = []string{}
				}
			}
			rule, err := q.CreateDiscountListRule(ctx, in.Rule)
			if err != nil {
				return err
			}
			for optionID, discount := range in.Options {
				if err := q.CreateDiscountListRuleOption(ctx, CreateDiscountListRuleOptionParams{
					RuleID: rule.ID, SaleOptionID: optionID, Discount: discount,
				}); err != nil {
					return err
				}
			}
		}
		return nil
	})
}
