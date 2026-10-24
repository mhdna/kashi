package db

import "context"

// DiscountListRuleInput is one rule as the list page sends it: the rule and
// its option discounts (sale option id -> discount).
type DiscountListRuleInput struct {
	Rule    CreateDiscountListRuleParams
	Options map[int64]int16
}

// DiscountListRuleGroupInput is one named set of rules (000203), switched on
// and off as one.
type DiscountListRuleGroupInput struct {
	Name    string
	Enabled bool
	Rules   []DiscountListRuleInput
}

// ReplaceDiscountListRuleGroupsTx makes a list's rule groups exactly groups,
// in that order. Rules are numbered straight through the groups, so the
// first rule that is on and matches wins, top to bottom. One transaction, so
// the rows the rules write (000200, at commit) are worked out once from the
// finished set, never from half of it.
func (store *SQLStore) ReplaceDiscountListRuleGroupsTx(ctx context.Context, listID int64, groups []DiscountListRuleGroupInput) error {
	return store.execTx(ctx, func(q *Queries) error {
		if err := q.DeleteDiscountListRuleGroups(ctx, listID); err != nil {
			return err
		}
		if err := q.DeleteDiscountListRules(ctx, listID); err != nil {
			return err
		}
		position := int32(0)
		for g, in := range groups {
			group, err := q.CreateDiscountListRuleGroup(ctx, CreateDiscountListRuleGroupParams{
				DiscountListID: listID, Position: int32(g), Name: in.Name, IsEnabled: in.Enabled,
			})
			if err != nil {
				return err
			}
			for _, rule := range in.Rules {
				rule.Rule.DiscountListID = listID
				rule.Rule.GroupID = group.ID
				rule.Rule.Position = position
				position++
				// A nil condition reaches Postgres as NULL; it means "any".
				for _, list := range []*[]string{&rule.Rule.CodePrefixes, &rule.Rule.ExceptCodePrefixes, &rule.Rule.Kinds,
					&rule.Rule.Categories, &rule.Rule.SubCategories, &rule.Rule.Origins, &rule.Rule.Brands} {
					if *list == nil {
						*list = []string{}
					}
				}
				created, err := q.CreateDiscountListRule(ctx, rule.Rule)
				if err != nil {
					return err
				}
				for optionID, discount := range rule.Options {
					if err := q.CreateDiscountListRuleOption(ctx, CreateDiscountListRuleOptionParams{
						RuleID: created.ID, SaleOptionID: optionID, Discount: discount,
					}); err != nil {
						return err
					}
				}
			}
		}
		return nil
	})
}

// ReplaceDiscountListRulesTx makes a list's rules exactly rules, all on, in
// one group named "Rules" -- a list's rules as they were before groups.
func (store *SQLStore) ReplaceDiscountListRulesTx(ctx context.Context, listID int64, rules []DiscountListRuleInput) error {
	for i := range rules {
		rules[i].Rule.IsEnabled = true
	}
	var groups []DiscountListRuleGroupInput
	if len(rules) > 0 {
		groups = []DiscountListRuleGroupInput{{Name: "Rules", Enabled: true, Rules: rules}}
	}
	return store.ReplaceDiscountListRuleGroupsTx(ctx, listID, groups)
}
