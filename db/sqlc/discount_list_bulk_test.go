package db

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/mhdna/kashi/util"
	"github.com/stretchr/testify/require"
)

func TestBulkSetDiscountListItems_CodesAndBarcodes(t *testing.T) {
	ctx := context.Background()
	byBarcode := createRandomVariant(t)
	byCode := createRandomVariant(t)
	product, err := testQueries.GetProduct(ctx, byCode.ProductID)
	require.NoError(t, err)

	list, err := testQueries.CreateDiscountList(ctx, CreateDiscountListParams{
		Name: util.RandomString(8), IsActive: true,
		ValidFrom: time.Now().Add(-time.Hour), ValidTo: time.Now().Add(time.Hour),
	})
	require.NoError(t, err)

	entries := []string{byBarcode.Barcode, strings.ToLower(product.Code), "no-such-thing"}
	set, err := testQueries.BulkSetDiscountListItems(ctx, BulkSetDiscountListItemsParams{
		DiscountListID: list.ID, Entries: entries, Discount: 30,
	})
	require.NoError(t, err)
	require.ElementsMatch(t, []int64{byBarcode.ID, byCode.ID}, set, "a barcode and a (case-insensitive) product code both match")

	unmatched, err := testQueries.ListUnmatchedDiscountEntries(ctx, entries)
	require.NoError(t, err)
	require.Equal(t, []string{"no-such-thing"}, unmatched)

	// Running it again with another value changes the discount, not the count.
	_, err = testQueries.BulkSetDiscountListItems(ctx, BulkSetDiscountListItemsParams{
		DiscountListID: list.ID, Entries: []string{byBarcode.Barcode}, Discount: 50,
	})
	require.NoError(t, err)
	item, err := testQueries.GetVariantDiscountFromList(ctx, GetVariantDiscountFromListParams{
		DiscountListID: list.ID, ProductVariantID: byBarcode.ID,
	})
	require.NoError(t, err)
	require.Equal(t, int16(50), item.Discount)
}
