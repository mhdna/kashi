package db

import "context"

// DiscountListRuleInput is one rule as the list page sends it: the rule and
// its option discounts (sale option id -> discount).
type DiscountListRuleInput struct {
	// ID is the saved rule this is, 0 for a new one.
	ID      int64
	Rule    CreateDiscountListRuleParams
	Options map[int64]int16
}

// DiscountListRuleGroupInput is one named set of rules (000203), switched on
// and off as one.
type DiscountListRuleGroupInput struct {
	// ID is the saved group this is, 0 for a new one.
	ID      int64
	Name    string
	Enabled bool
	Rules   []DiscountListRuleInput
}

// ReplaceDiscountListRuleGroupsTx makes a list's rule groups exactly groups,
// in that order. Rules are numbered straight through the groups, so the
// first rule that is on and matches wins, top to bottom.
//
// Groups and rules that are already saved (their ID) are updated in place
// and keep their ids; only new ones are inserted and only those left out are
// deleted. The rows a rule wrote carry its id, so re-creating every rule
// would have every one of those rows deleted and written again on each save
// -- most of the 13s a save of the Centro list took. One transaction, so the
// rows are worked out once (000200, at commit) from the finished set.
func (store *SQLStore) ReplaceDiscountListRuleGroupsTx(ctx context.Context, listID int64, groups []DiscountListRuleGroupInput) error {
	return store.execTx(ctx, func(q *Queries) error {
		keepGroups := []int64{}
		keepRules := []int64{}
		position := int32(0)
		for g, in := range groups {
			groupID := in.ID
			saved := int64(0)
			if groupID > 0 {
				n, err := q.UpdateDiscountListRuleGroup(ctx, UpdateDiscountListRuleGroupParams{
					ID: groupID, DiscountListID: listID, Position: int32(g), Name: in.Name, IsEnabled: in.Enabled,
				})
				if err != nil {
					return err
				}
				saved = n
			}
			if saved == 0 {
				group, err := q.CreateDiscountListRuleGroup(ctx, CreateDiscountListRuleGroupParams{
					DiscountListID: listID, Position: int32(g), Name: in.Name, IsEnabled: in.Enabled,
				})
				if err != nil {
					return err
				}
				groupID = group.ID
			}
			keepGroups = append(keepGroups, groupID)

			for _, rule := range in.Rules {
				r := rule.Rule
				r.DiscountListID, r.GroupID, r.Position = listID, groupID, position
				position++
				// A nil condition reaches Postgres as NULL; it means "any".
				for _, list := range []*[]string{&r.CodePrefixes, &r.ExceptCodePrefixes, &r.Kinds,
					&r.Categories, &r.SubCategories, &r.Origins, &r.Brands} {
					if *list == nil {
						*list = []string{}
					}
				}
				ruleID := rule.ID
				saved := int64(0)
				if ruleID > 0 {
					n, err := q.UpdateDiscountListRule(ctx, UpdateDiscountListRuleParams{
						ID: ruleID, DiscountListID: listID, Position: r.Position, Name: r.Name,
						CodePrefixes: r.CodePrefixes, ExceptCodePrefixes: r.ExceptCodePrefixes, Kinds: r.Kinds,
						Categories: r.Categories, SubCategories: r.SubCategories, Origins: r.Origins, Brands: r.Brands,
						Discount: r.Discount, DealID: r.DealID, GroupID: r.GroupID, IsEnabled: r.IsEnabled,
					})
					if err != nil {
						return err
					}
					saved = n
				}
				if saved == 0 {
					created, err := q.CreateDiscountListRule(ctx, r)
					if err != nil {
						return err
					}
					ruleID = created.ID
				}
				keepRules = append(keepRules, ruleID)
				if err := q.DeleteDiscountListRuleOptions(ctx, ruleID); err != nil {
					return err
				}
				for optionID, discount := range rule.Options {
					if err := q.CreateDiscountListRuleOption(ctx, CreateDiscountListRuleOptionParams{
						RuleID: ruleID, SaleOptionID: optionID, Discount: discount,
					}); err != nil {
						return err
					}
				}
			}
		}
		// Rules first: one moved out of a group being deleted is already in
		// its new group, so the group's ON DELETE CASCADE can't take it.
		if err := q.DeleteDiscountListRulesExcept(ctx, DeleteDiscountListRulesExceptParams{DiscountListID: listID, Keep: keepRules}); err != nil {
			return err
		}
		return q.DeleteDiscountListRuleGroupsExcept(ctx, DeleteDiscountListRuleGroupsExceptParams{DiscountListID: listID, Keep: keepGroups})
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
