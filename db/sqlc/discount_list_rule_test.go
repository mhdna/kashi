package db

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/mhdna/kashi/util"
	"github.com/stretchr/testify/require"
)

// Rules write a list's rows when they are saved, the first match wins, a 0
// rule keeps a product out, option discounts become option rows, products
// follow their attributes, and a hand-set row beats the rules.
func TestDiscountListRules(t *testing.T) {
	ctx := context.Background()
	list, err := testQueries.CreateDiscountList(ctx, CreateDiscountListParams{
		Name: util.RandomString(8), ValidFrom: time.Now(), ValidTo: time.Now().Add(time.Hour),
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = testQueries.DeleteDiscountList(ctx, list.ID) })

	// A kind no other test uses, so the rules only see these products.
	kind := "Kind" + util.RandomString(8)
	setAttr := func(productID, attributeID int64, value string) {
		var valueID int64
		require.NoError(t, testDB.QueryRowContext(ctx, `
			INSERT INTO attributes_values (attribute_id, value) VALUES ($1, $2)
			ON CONFLICT (attribute_id, value) DO UPDATE SET value = EXCLUDED.value RETURNING id`,
			attributeID, value).Scan(&valueID))
		_, err := testDB.ExecContext(ctx, `
			INSERT INTO products_attributes (attribute_id, product_id, attribute_value_id) VALUES ($1, $2, $3)
			ON CONFLICT (attribute_id, product_id) DO UPDATE SET attribute_value_id = EXCLUDED.attribute_value_id`,
			attributeID, productID, valueID)
		require.NoError(t, err)
	}
	for id, name := range map[int64]string{0: "category", 3: "kind", 8: "origin"} {
		_, err := testDB.ExecContext(ctx, `INSERT INTO attributes (id, name) VALUES ($1, $2) ON CONFLICT DO NOTHING`, id, name)
		require.NoError(t, err)
	}

	shirt, bag := createRandomVariant(t), createRandomVariant(t)
	setAttr(shirt.ProductID, 3, kind)
	setAttr(shirt.ProductID, 8, "Turkey")
	setAttr(bag.ProductID, 3, kind)
	setAttr(bag.ProductID, 8, "Turkey")
	setAttr(bag.ProductID, 0, "Accessories")
	summer, err := testQueries.CreateSaleOption(ctx, CreateSaleOptionParams{Name: "Opt" + util.RandomString(6)})
	require.NoError(t, err)

	require.NoError(t, testStore.ReplaceDiscountListRulesTx(ctx, list.ID, []DiscountListRuleInput{
		{Rule: CreateDiscountListRuleParams{Name: "accessories", Categories: []string{"accessories"}, Kinds: []string{kind}}},
		{
			Rule:    CreateDiscountListRuleParams{Name: "turkey", Kinds: []string{kind}, Origins: []string{"TURKEY"}, Discount: 30},
			Options: map[int64]int16{summer.ID: 50},
		},
	}))

	discountOf := func(variantID int64) (int16, bool) {
		row, err := testQueries.GetVariantDiscountFromList(ctx, GetVariantDiscountFromListParams{
			DiscountListID: list.ID, ProductVariantID: variantID,
		})
		if err != nil {
			return 0, false
		}
		return row.Discount, row.RuleID.Valid
	}
	optionRows := func(variantID int64) int {
		var n int
		require.NoError(t, testDB.QueryRowContext(ctx, `SELECT count(*) FROM discount_list_option_items
			WHERE discount_list_id = $1 AND product_variant_id = $2`, list.ID, variantID).Scan(&n))
		return n
	}

	d, fromRule := discountOf(shirt.ID)
	require.Equal(t, int16(30), d)
	require.True(t, fromRule)
	require.Equal(t, 1, optionRows(shirt.ID))
	_, found := discountOf(bag.ID)
	require.False(t, found, "the 0 rule above keeps accessories out")

	rules, err := testQueries.ListDiscountListRules(ctx, list.ID)
	require.NoError(t, err)
	require.Len(t, rules, 2)
	require.Equal(t, int64(1), rules[1].Variants)
	require.Equal(t, int64(0), rules[0].Variants, "an exclusion writes no rows")

	// What each rule decides, the exclusion's too: the bag.
	decided, err := testQueries.CountDiscountListRuleDecided(ctx, list.ID)
	require.NoError(t, err)
	byRule := map[int64]int64{}
	for _, d := range decided {
		byRule[d.RuleID] = d.Variants
	}
	require.Equal(t, int64(1), byRule[rules[0].ID], "the accessories rule keeps the bag out")
	require.Equal(t, int64(1), byRule[rules[1].ID], "the turkey rule has the shirt")

	// The shirt stops being Turkish: it leaves the rule when that commits.
	setAttr(shirt.ProductID, 8, "China")
	_, found = discountOf(shirt.ID)
	require.False(t, found)
	require.Equal(t, 0, optionRows(shirt.ID))

	// Back to Turkey, then set by hand: the hand-set row stays and the rule's
	// option row goes, so the hand-set discount is what the till gives.
	setAttr(shirt.ProductID, 8, "Turkey")
	require.Equal(t, 1, optionRows(shirt.ID))
	_, err = testQueries.CreateDiscountListItem(ctx, CreateDiscountListItemParams{
		DiscountListID: list.ID, ProductVariantID: shirt.ID, Discount: 5,
	})
	require.NoError(t, err)
	d, fromRule = discountOf(shirt.ID)
	require.Equal(t, int16(5), d)
	require.False(t, fromRule)
	require.Equal(t, 0, optionRows(shirt.ID))

	// Removing the rules removes what they wrote, not the hand-set row.
	setAttr(bag.ProductID, 0, "Tops")
	require.NoError(t, testStore.ReplaceDiscountListRulesTx(ctx, list.ID, nil))
	d, fromRule = discountOf(shirt.ID)
	require.Equal(t, int16(5), d)
	require.False(t, fromRule)
	_, found = discountOf(bag.ID)
	require.False(t, found)
}

// A rule or group that is off decides nothing (000203): the rule below an
// off rule applies, an off group writes no rows, and switching it back on
// brings them back.
func TestDiscountListRuleSwitches(t *testing.T) {
	ctx := context.Background()
	list, err := testQueries.CreateDiscountList(ctx, CreateDiscountListParams{
		Name: util.RandomString(8), ValidFrom: time.Now(), ValidTo: time.Now().Add(time.Hour),
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = testQueries.DeleteDiscountList(ctx, list.ID) })

	prefix := "SW" + util.RandomString(6)
	variant := createRandomVariant(t)
	_, err = testDB.ExecContext(ctx, `UPDATE products SET code = $1 WHERE id = $2`, prefix+"1", variant.ProductID)
	require.NoError(t, err)

	save := func(firstOn, groupOn bool) {
		require.NoError(t, testStore.ReplaceDiscountListRuleGroupsTx(ctx, list.ID, []DiscountListRuleGroupInput{
			{Name: "Sale", Enabled: groupOn, Rules: []DiscountListRuleInput{
				{Rule: CreateDiscountListRuleParams{Name: "first", CodePrefixes: []string{prefix}, Discount: 30, IsEnabled: firstOn}},
				{Rule: CreateDiscountListRuleParams{Name: "second", CodePrefixes: []string{prefix}, Discount: 10, IsEnabled: true}},
			}},
		}))
	}
	discount := func() (int16, bool) {
		row, err := testQueries.GetVariantDiscountFromList(ctx, GetVariantDiscountFromListParams{
			DiscountListID: list.ID, ProductVariantID: variant.ID,
		})
		if err != nil {
			return 0, false
		}
		return row.Discount, true
	}

	save(true, true)
	d, ok := discount()
	require.True(t, ok)
	require.Equal(t, int16(30), d)

	save(false, true)
	d, ok = discount()
	require.True(t, ok)
	require.Equal(t, int16(10), d, "with the first rule off the second decides")

	save(true, false)
	_, ok = discount()
	require.False(t, ok, "an off group writes nothing")

	groups, err := testQueries.ListDiscountListRuleGroups(ctx, list.ID)
	require.NoError(t, err)
	require.Len(t, groups, 1)
	require.False(t, groups[0].IsEnabled)

	save(true, true)
	d, ok = discount()
	require.True(t, ok)
	require.Equal(t, int16(30), d)
}

// Rules saved again with their ids are updated in place: they keep their
// ids, an unchanged save writes no rows (the list isn't re-sent), and a
// changed rule moves the list's updated_at once its rows change.
func TestDiscountListRulesSaveInPlace(t *testing.T) {
	ctx := context.Background()
	list, err := testQueries.CreateDiscountList(ctx, CreateDiscountListParams{
		Name: util.RandomString(8), ValidFrom: time.Now(), ValidTo: time.Now().Add(time.Hour),
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = testQueries.DeleteDiscountList(ctx, list.ID) })
	prefix := "IP" + util.RandomString(6)
	variant := createRandomVariant(t)
	_, err = testDB.ExecContext(ctx, `UPDATE products SET code = $1 WHERE id = $2`, prefix+"1", variant.ProductID)
	require.NoError(t, err)

	save := func(groupID, ruleID int64, discount int16) {
		require.NoError(t, testStore.ReplaceDiscountListRuleGroupsTx(ctx, list.ID, []DiscountListRuleGroupInput{
			{ID: groupID, Name: "Sale", Enabled: true, Rules: []DiscountListRuleInput{
				{ID: ruleID, Rule: CreateDiscountListRuleParams{Name: "r", CodePrefixes: []string{prefix}, Discount: discount, IsEnabled: true}},
			}},
		}))
	}
	updatedAt := func() time.Time {
		l, err := testQueries.GetDiscountList(ctx, list.ID)
		require.NoError(t, err)
		return l.UpdatedAt
	}

	save(0, 0, 30)
	groups, err := testQueries.ListDiscountListRuleGroups(ctx, list.ID)
	require.NoError(t, err)
	rules, err := testQueries.ListDiscountListRules(ctx, list.ID)
	require.NoError(t, err)
	require.Len(t, rules, 1)
	groupID, ruleID := groups[0].ID, rules[0].ID
	before := updatedAt()

	save(groupID, ruleID, 30)
	rules, err = testQueries.ListDiscountListRules(ctx, list.ID)
	require.NoError(t, err)
	require.Equal(t, ruleID, rules[0].ID)
	require.Equal(t, before, updatedAt(), "an unchanged save writes no rows")

	save(groupID, ruleID, 40)
	row, err := testQueries.GetVariantDiscountFromList(ctx, GetVariantDiscountFromListParams{
		DiscountListID: list.ID, ProductVariantID: variant.ID,
	})
	require.NoError(t, err)
	require.Equal(t, int16(40), row.Discount)
	require.Equal(t, ruleID, row.RuleID.Int64)
	require.True(t, updatedAt().After(before), "a changed rule's rows move the list")

	// Left out: deleted, with its rows.
	require.NoError(t, testStore.ReplaceDiscountListRuleGroupsTx(ctx, list.ID, nil))
	_, err = testQueries.GetVariantDiscountFromList(ctx, GetVariantDiscountFromListParams{
		DiscountListID: list.ID, ProductVariantID: variant.ID,
	})
	require.Error(t, err)
}

// A rule with a price limit takes only products selling for at most that
// ("toys up to $1.11: no discount"), judged by the price in the active price
// lists, and follows a price change; a product with no price is never taken.
func TestDiscountListRuleMaxPrice(t *testing.T) {
	ctx := context.Background()
	list, err := testQueries.CreateDiscountList(ctx, CreateDiscountListParams{
		Name: util.RandomString(8), ValidFrom: time.Now(), ValidTo: time.Now().Add(time.Hour),
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = testQueries.DeleteDiscountList(ctx, list.ID) })
	prices, err := testQueries.CreatePriceList(ctx, CreatePriceListParams{
		Name: util.RandomString(8), IsActive: true, ValidFrom: time.Now(), ValidTo: time.Now().Add(time.Hour),
	})
	require.NoError(t, err)
	t.Cleanup(func() { _, _ = testDB.ExecContext(ctx, `DELETE FROM price_lists WHERE id = $1`, prices.ID) })

	kind := "Kind" + util.RandomString(8)
	_, err = testDB.ExecContext(ctx, `INSERT INTO attributes (id, name) VALUES (3, 'kind') ON CONFLICT DO NOTHING`)
	require.NoError(t, err)
	var valueID int64
	require.NoError(t, testDB.QueryRowContext(ctx, `
		INSERT INTO attributes_values (attribute_id, value) VALUES (3, $1)
		ON CONFLICT (attribute_id, value) DO UPDATE SET value = EXCLUDED.value RETURNING id`, kind).Scan(&valueID))
	cheap, dear, unpriced := createRandomVariant(t), createRandomVariant(t), createRandomVariant(t)
	for _, v := range []ProductVariant{cheap, dear, unpriced} {
		_, err := testDB.ExecContext(ctx, `
			INSERT INTO products_attributes (attribute_id, product_id, attribute_value_id) VALUES (3, $1, $2)`, v.ProductID, valueID)
		require.NoError(t, err)
	}
	setPrice := func(v ProductVariant, cents int64) {
		_, err := testQueries.CreatePriceListItem(ctx, CreatePriceListItemParams{
			PriceListID: prices.ID, ProductVariantID: v.ID, Price: cents,
		})
		require.NoError(t, err)
	}
	setPrice(cheap, 111)
	setPrice(dear, 500)

	// The cheap ones are kept out (0) above the rule that gives the kind 30%.
	require.NoError(t, testStore.ReplaceDiscountListRulesTx(ctx, list.ID, []DiscountListRuleInput{
		{Rule: CreateDiscountListRuleParams{Name: "cheap", Kinds: []string{kind}, MaxPrice: sql.NullInt64{Int64: 111, Valid: true}}},
		{Rule: CreateDiscountListRuleParams{Name: "kind", Kinds: []string{kind}, Discount: 30}},
	}))

	has := func(v ProductVariant) bool {
		_, err := testQueries.GetVariantDiscountFromList(ctx, GetVariantDiscountFromListParams{
			DiscountListID: list.ID, ProductVariantID: v.ID,
		})
		return err == nil
	}
	require.False(t, has(cheap), "$1.11 is at the limit: kept out")
	require.True(t, has(dear), "$5 is over it: 30%")
	require.True(t, has(unpriced), "no price is never taken by a price limit")

	// The page's count says the same.
	decided, err := testQueries.CountDiscountListRuleDecided(ctx, list.ID)
	require.NoError(t, err)
	counts := map[string]int64{}
	rules, err := testQueries.ListDiscountListRules(ctx, list.ID)
	require.NoError(t, err)
	for _, r := range rules {
		for _, d := range decided {
			if d.RuleID == r.ID {
				counts[r.Name] = d.Variants
			}
		}
	}
	require.Equal(t, int64(1), counts["cheap"])
	require.Equal(t, int64(2), counts["kind"])

	// A price change moves a product across the limit, in both directions.
	setPrice(dear, 100)
	require.False(t, has(dear), "now $1: kept out")
	setPrice(cheap, 999)
	require.True(t, has(cheap), "now $9.99: 30%")
}
