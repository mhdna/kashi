package db

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/mhdna/kashi/util"
	"github.com/stretchr/testify/require"
)

func seededSaleOptions(t *testing.T) (summer, winter SaleOption) {
	options, err := testQueries.ListSaleOptions(context.Background())
	require.NoError(t, err)
	for _, o := range options {
		switch o.Name {
		case "Summer":
			summer = o
		case "Winter":
			winter = o
		}
	}
	require.NotZero(t, summer.ID, "000199 seeds Summer")
	require.NotZero(t, winter.ID, "000199 seeds Winter")
	return summer, winter
}

func TestSetProductSaleOptionsTx_ReplacesAndReachesTheProductFeed(t *testing.T) {
	ctx := context.Background()
	summer, winter := seededSaleOptions(t)
	variant := createRandomVariant(t)
	product, err := testQueries.GetProduct(ctx, variant.ProductID)
	require.NoError(t, err)
	// updated_at is kept to the second, so backdate it to see the trigger move it.
	_, err = testDB.ExecContext(ctx, `UPDATE product_variants SET updated_at = now() - interval '1 day' WHERE id = $1`, variant.ID)
	require.NoError(t, err)
	before := time.Now().Add(-time.Hour)

	matched, err := testStore.SetProductSaleOptionsTx(ctx, []string{product.Code}, []int64{winter.ID, summer.ID})
	require.NoError(t, err)
	require.Equal(t, 1, matched)

	rows, err := testQueries.ListProductVariantsForSyncByUIDs(ctx, []uuid.UUID{variant.Uid})
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, []string{"Summer", "Winter"}, rows[0].SaleOptions, "names, in sort_order")
	require.True(t, rows[0].UpdatedAt.After(before), "an options change has to move the variant for the delta feed")

	// Set again by barcode with one option: replaced, not added to.
	_, err = testStore.SetProductSaleOptionsTx(ctx, []string{variant.Barcode}, []int64{winter.ID})
	require.NoError(t, err)
	rows, err = testQueries.ListProductVariantsForSyncByUIDs(ctx, []uuid.UUID{variant.Uid})
	require.NoError(t, err)
	require.Equal(t, []string{"Winter"}, rows[0].SaleOptions)

	// An empty set clears; the feed sends an empty array, not null.
	_, err = testStore.SetProductSaleOptionsTx(ctx, []string{product.Code}, nil)
	require.NoError(t, err)
	rows, err = testQueries.ListProductVariantsForSyncByUIDs(ctx, []uuid.UUID{variant.Uid})
	require.NoError(t, err)
	require.NotNil(t, rows[0].SaleOptions)
	require.Empty(t, rows[0].SaleOptions)
}

func TestDiscountListOptionItems_ListSyncAndDelete(t *testing.T) {
	ctx := context.Background()
	summer, winter := seededSaleOptions(t)
	variant := createRandomVariant(t)
	product, err := testQueries.GetProduct(ctx, variant.ProductID)
	require.NoError(t, err)

	list, err := testQueries.CreateDiscountList(ctx, CreateDiscountListParams{
		Name: util.RandomString(8), IsActive: true,
		ValidFrom: time.Now().Add(-time.Hour), ValidTo: time.Now().Add(time.Hour),
	})
	require.NoError(t, err)

	_, err = testQueries.BulkSetDiscountListItems(ctx, BulkSetDiscountListItemsParams{
		DiscountListID: list.ID, Entries: []string{product.Code}, Discount: 20,
	})
	require.NoError(t, err)
	set, err := testQueries.BulkSetDiscountListOptionItems(ctx, BulkSetDiscountListOptionItemsParams{
		DiscountListID: list.ID, Entries: []string{product.Code}, SaleOptionID: summer.ID, Discount: 50,
	})
	require.NoError(t, err)
	require.Equal(t, []int64{variant.ID}, set)

	// An option row must not leak into plain resolution.
	plain, err := testQueries.GetVariantDiscountFromList(ctx, GetVariantDiscountFromListParams{
		DiscountListID: list.ID, ProductVariantID: variant.ID,
	})
	require.NoError(t, err)
	require.Equal(t, int16(20), plain.Discount)

	items, err := testQueries.ListDiscountListItemsWithProduct(ctx, ListDiscountListItemsWithProductParams{
		DiscountListID: list.ID, PageSize: 10,
	})
	require.NoError(t, err)
	require.Len(t, items, 2)
	require.Equal(t, "", items[0].SaleOption, "the plain row first")
	require.Equal(t, int16(20), items[0].Discount)
	require.Equal(t, "Summer", items[1].SaleOption)
	require.Equal(t, summer.ID, items[1].SaleOptionID)
	require.Equal(t, int16(50), items[1].Discount)
	count, err := testQueries.CountDiscountListItemsWithProduct(ctx, CountDiscountListItemsWithProductParams{DiscountListID: list.ID})
	require.NoError(t, err)
	require.Equal(t, int64(2), count)

	synced, err := testQueries.ListDiscountListOptionItemsForSync(ctx, list.ID)
	require.NoError(t, err)
	require.Len(t, synced, 1)
	require.Equal(t, variant.Uid, synced[0].ProductVariantUid)
	require.Equal(t, "Summer", synced[0].SaleOption)

	// Deleting by (variant, option) takes only that pair.
	_, err = testQueries.BulkSetDiscountListOptionItems(ctx, BulkSetDiscountListOptionItemsParams{
		DiscountListID: list.ID, Entries: []string{product.Code}, SaleOptionID: winter.ID, Discount: 20,
	})
	require.NoError(t, err)
	deleted, err := testQueries.DeleteDiscountListOptionItems(ctx, DeleteDiscountListOptionItemsParams{
		DiscountListID: list.ID, ProductVariantIds: []int64{variant.ID}, SaleOptionIds: []int64{summer.ID},
	})
	require.NoError(t, err)
	require.Equal(t, int64(1), deleted)
	synced, err = testQueries.ListDiscountListOptionItemsForSync(ctx, list.ID)
	require.NoError(t, err)
	require.Len(t, synced, 1)
	require.Equal(t, "Winter", synced[0].SaleOption)
}
