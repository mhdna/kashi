package db

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/mhdna/kashi/util"
	"github.com/stretchr/testify/require"
)

// A rule can put products on a deal (000201): it writes discount-0 rows that
// name the deal, a hand-set deal row beats it, changing the deal re-sends the
// lists using it, and a deal in use can't be deleted.
func TestDealRules(t *testing.T) {
	ctx := context.Background()
	list, err := testQueries.CreateDiscountList(ctx, CreateDiscountListParams{
		Name: util.RandomString(8), ValidFrom: time.Now(), ValidTo: time.Now().Add(time.Hour),
	})
	require.NoError(t, err)
	b2g1, err := testQueries.CreateDeal(ctx, CreateDealParams{
		Name: "Buy 2 get 1 " + util.RandomString(4), Kind: "buy_get", BuyQty: 2, GetQty: 1, Percent: 100,
	})
	require.NoError(t, err)
	b1g1, err := testQueries.CreateDeal(ctx, CreateDealParams{
		Name: "Buy 1 get 1 " + util.RandomString(4), Kind: "buy_get", BuyQty: 1, GetQty: 1, Percent: 100,
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = testQueries.DeleteDiscountList(ctx, list.ID)
		_ = testQueries.DeleteDeal(ctx, b2g1.ID)
		_ = testQueries.DeleteDeal(ctx, b1g1.ID)
	})

	// A prefix no other product has, so the rule only sees these two.
	prefix := "DL" + util.RandomString(6)
	var shirt, jeans ProductVariant
	for _, v := range []*ProductVariant{&shirt, &jeans} {
		*v = createRandomVariant(t)
		_, err := testDB.ExecContext(ctx, `UPDATE products SET code = $1 WHERE id = $2`,
			prefix+util.RandomString(4), v.ProductID)
		require.NoError(t, err)
	}

	require.NoError(t, testStore.ReplaceDiscountListRulesTx(ctx, list.ID, []DiscountListRuleInput{
		{Rule: CreateDiscountListRuleParams{Name: "all", CodePrefixes: []string{prefix},
			DealID: nullInt64(b2g1.ID)}},
	}))
	row := func(variantID int64) DiscountListItem {
		item, err := testQueries.GetVariantDiscountFromList(ctx, GetVariantDiscountFromListParams{
			DiscountListID: list.ID, ProductVariantID: variantID,
		})
		require.NoError(t, err)
		return item
	}
	for _, v := range []ProductVariant{shirt, jeans} {
		item := row(v.ID)
		require.Equal(t, int16(0), item.Discount)
		require.Equal(t, b2g1.ID, item.DealID.Int64)
		require.True(t, item.RuleID.Valid)
	}

	// Set by hand onto the other deal: the hand-set row stays.
	_, err = testQueries.CreateDiscountListItem(ctx, CreateDiscountListItemParams{
		DiscountListID: list.ID, ProductVariantID: jeans.ID, DealID: nullInt64(b1g1.ID),
	})
	require.NoError(t, err)
	require.Equal(t, b1g1.ID, row(jeans.ID).DealID.Int64)
	require.False(t, row(jeans.ID).RuleID.Valid)

	// The list ships the deals it names.
	deals, err := testQueries.ListDealsForDiscountListSync(ctx, list.ID)
	require.NoError(t, err)
	require.Len(t, deals, 2)

	// A row can't carry both a discount and a deal.
	_, err = testQueries.CreateDiscountListItem(ctx, CreateDiscountListItemParams{
		DiscountListID: list.ID, ProductVariantID: jeans.ID, Discount: 10, DealID: nullInt64(b1g1.ID),
	})
	require.Error(t, err)

	// Changing the deal's terms moves the list's updated_at, so the tills get
	// the new terms with the list.
	before, err := testQueries.GetDiscountList(ctx, list.ID)
	require.NoError(t, err)
	_, err = testQueries.UpdateDeal(ctx, UpdateDealParams{
		ID: b2g1.ID, Name: b2g1.Name, Kind: "buy_get", BuyQty: 2, GetQty: 1, Percent: 50,
	})
	require.NoError(t, err)
	after, err := testQueries.GetDiscountList(ctx, list.ID)
	require.NoError(t, err)
	require.True(t, after.UpdatedAt.After(before.UpdatedAt))

	// In use: no delete.
	require.Error(t, testQueries.DeleteDeal(ctx, b1g1.ID))

	// Without the rule the rule's rows go; the hand-set one stays.
	require.NoError(t, testStore.ReplaceDiscountListRulesTx(ctx, list.ID, nil))
	_, err = testQueries.GetVariantDiscountFromList(ctx, GetVariantDiscountFromListParams{
		DiscountListID: list.ID, ProductVariantID: shirt.ID,
	})
	require.Error(t, err)
	require.Equal(t, b1g1.ID, row(jeans.ID).DealID.Int64)
}

// One deal for the whole shop, with a rule giving its own products a
// different "otherwise" percent (000216): the rows carry it, the tills get it
// in its own field, and a hand-set row drops it.
func TestDealRuleFallback(t *testing.T) {
	ctx := context.Background()
	list, err := testQueries.CreateDiscountList(ctx, CreateDiscountListParams{
		Name: util.RandomString(8), ValidFrom: time.Now(), ValidTo: time.Now().Add(time.Hour),
	})
	require.NoError(t, err)
	b2g1, err := testQueries.CreateDeal(ctx, CreateDealParams{
		Name: "Buy 2 get 1 " + util.RandomString(4), Kind: "buy_get", BuyQty: 2, GetQty: 1, Percent: 100, FallbackPercent: 20,
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = testQueries.DeleteDiscountList(ctx, list.ID)
		_ = testQueries.DeleteDeal(ctx, b2g1.ID)
	})

	prefix := "DF" + util.RandomString(6)
	outlet, other := createRandomVariant(t), createRandomVariant(t)
	for code, v := range map[string]ProductVariant{prefix + "CA" + util.RandomString(3): outlet, prefix + "X" + util.RandomString(3): other} {
		_, err := testDB.ExecContext(ctx, `UPDATE products SET code = $1 WHERE id = $2`, code, v.ProductID)
		require.NoError(t, err)
	}
	rules := []DiscountListRuleInput{
		{Rule: CreateDiscountListRuleParams{Name: "outlet", CodePrefixes: []string{prefix + "CA"},
			DealID: nullInt64(b2g1.ID), DealFallbackPercent: sql.NullInt16{Int16: 30, Valid: true}}},
		{Rule: CreateDiscountListRuleParams{Name: "rest", CodePrefixes: []string{prefix}, DealID: nullInt64(b2g1.ID)}},
	}
	require.NoError(t, testStore.ReplaceDiscountListRulesTx(ctx, list.ID, rules))
	row := func(variantID int64) DiscountListItem {
		item, err := testQueries.GetVariantDiscountFromList(ctx, GetVariantDiscountFromListParams{
			DiscountListID: list.ID, ProductVariantID: variantID,
		})
		require.NoError(t, err)
		return item
	}
	require.Equal(t, b2g1.ID, row(outlet.ID).DealID.Int64)
	require.Equal(t, sql.NullInt16{Int16: 30, Valid: true}, row(outlet.ID).DealFallbackPercent)
	require.Equal(t, int16(0), row(outlet.ID).Discount)
	require.Equal(t, b2g1.ID, row(other.ID).DealID.Int64)
	require.False(t, row(other.ID).DealFallbackPercent.Valid)

	synced, err := testQueries.ListDiscountListItemsForSync(ctx, list.ID)
	require.NoError(t, err)
	require.Len(t, synced, 2)
	for _, s := range synced {
		require.Equal(t, s.ProductVariantUid == outlet.Uid, s.DealFallbackPercent.Valid)
	}

	// Changing only the fallback rewrites the row, so the tills get it.
	before, err := testQueries.GetDiscountList(ctx, list.ID)
	require.NoError(t, err)
	rules[0].Rule.DealFallbackPercent = sql.NullInt16{Int16: 25, Valid: true}
	require.NoError(t, testStore.ReplaceDiscountListRulesTx(ctx, list.ID, rules))
	require.Equal(t, int16(25), row(outlet.ID).DealFallbackPercent.Int16)
	after, err := testQueries.GetDiscountList(ctx, list.ID)
	require.NoError(t, err)
	require.True(t, after.UpdatedAt.After(before.UpdatedAt))

	// Saving the rules again keeps a rule's price limit: the saved rule is
	// updated in place, and the update used to drop it.
	limited := append([]DiscountListRuleInput(nil), rules...)
	limited[1].Rule.MaxPrice = sql.NullInt64{Int64: 1, Valid: true}
	require.NoError(t, testStore.ReplaceDiscountListRulesTx(ctx, list.ID, limited))
	saved, err := testQueries.ListDiscountListRules(ctx, list.ID)
	require.NoError(t, err)
	require.Len(t, saved, 2)
	for i := range limited {
		limited[i].ID = saved[i].ID
	}
	require.NoError(t, testStore.ReplaceDiscountListRulesTx(ctx, list.ID, limited))
	saved, err = testQueries.ListDiscountListRules(ctx, list.ID)
	require.NoError(t, err)
	require.Equal(t, limited[1].ID, saved[1].ID, "the rule is updated in place")
	require.Equal(t, sql.NullInt64{Int64: 1, Valid: true}, saved[1].MaxPrice)
	require.NoError(t, testStore.ReplaceDiscountListRulesTx(ctx, list.ID, rules))

	// Set by hand: the row is the hand's, with the deal's own fallback.
	_, err = testQueries.CreateDiscountListItem(ctx, CreateDiscountListItemParams{
		DiscountListID: list.ID, ProductVariantID: outlet.ID, DealID: nullInt64(b2g1.ID),
	})
	require.NoError(t, err)
	require.False(t, row(outlet.ID).RuleID.Valid)
	require.False(t, row(outlet.ID).DealFallbackPercent.Valid)

	// A fallback needs a deal.
	_, err = testDB.ExecContext(ctx, `UPDATE discount_list_items SET deal_id = NULL, deal_fallback_percent = 30
		WHERE discount_list_id = $1 AND product_variant_id = $2`, list.ID, other.ID)
	require.Error(t, err)
}

// A deal's terms have to read one way only (000201's CHECK).
func TestDealKindChecks(t *testing.T) {
	ctx := context.Background()
	for _, bad := range []CreateDealParams{
		{Name: "no reward", Kind: "buy_get", BuyQty: 2, GetQty: 0, Percent: 100},
		{Name: "no percent", Kind: "buy_get", BuyQty: 2, GetQty: 1},
		{Name: "bundle of one", Kind: "bundle_price", BuyQty: 1, BundlePrice: 500},
		{Name: "bundle no price", Kind: "bundle_price", BuyQty: 3},
		{Name: "volume no percent", Kind: "volume_percent", BuyQty: 3},
		{Name: "unknown", Kind: "spend", BuyQty: 3, Percent: 10},
		{Name: "bundle fallback", Kind: "bundle_price", BuyQty: 3, BundlePrice: 5000, FallbackPercent: 30},
		{Name: "fallback over 100", Kind: "buy_get", BuyQty: 2, GetQty: 1, Percent: 100, FallbackPercent: 101},
	} {
		_, err := testQueries.CreateDeal(ctx, bad)
		require.Error(t, err, bad.Name)
	}
}
