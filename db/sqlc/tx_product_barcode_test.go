package db

import (
	"context"
	"strconv"
	"testing"

	"github.com/mhdna/kashi/util"
	"github.com/stretchr/testify/require"
)

// A generated barcode that's already taken moves on to the next number: an insert that
// failed would abort the transaction and leave the retry nothing to run in.
func TestCreateProductSkipsATakenBarcode(t *testing.T) {
	ctx := context.Background()
	next, err := testQueries.GetNextBarcodeItemValue(ctx)
	require.NoError(t, err)

	// The next two numbers belong to something else already.
	for _, item := range []int64{next + 1, next + 2} {
		barcode := generateEAN13(item)
		_, err := testStore.CreateProductTx(ctx, CreateProductTxParams{
			Code: util.RandomString(10), Name: util.RandomString(8), Barcode: &barcode,
		})
		require.NoError(t, err)
	}

	result, err := testStore.CreateProductTx(ctx, CreateProductTxParams{
		Code: util.RandomString(10), Name: util.RandomString(8),
	})
	require.NoError(t, err)
	require.Len(t, result.Variants, 1)
	require.Equal(t, strconv.FormatInt(generateEAN13(next+3), 10), result.Variants[0].Barcode)
}

// The label reader's products keep their barcode exactly as printed, a
// leading zero included (a number would drop it and no scan would match),
// and are tagged with where they came from.
func TestCreateProductWithTextBarcodeAndSource(t *testing.T) {
	ctx := context.Background()
	barcode := "0" + strconv.FormatInt(util.RandomInt(100000000000, 999999999999), 10)
	price := int64(1000)
	result, err := testStore.CreateProductTx(ctx, CreateProductTxParams{
		Code: "LBL-" + util.RandomString(6), Name: "cup set 6pcs basic-Glasses",
		BarcodeText: barcode, Price: &price, Source: ProductSourceLabelPhoto,
	})
	require.NoError(t, err)
	require.Equal(t, barcode, result.Variant.Barcode)
	require.Equal(t, ProductSourceLabelPhoto, result.Product.Source)

	stored, err := testQueries.GetProductVariantByBarcode(ctx, barcode)
	require.NoError(t, err)
	require.Equal(t, result.Variant.ID, stored.ID)
	product, err := testQueries.GetProduct(ctx, result.Product.ID)
	require.NoError(t, err)
	require.Equal(t, ProductSourceLabelPhoto, product.Source)

	// A product made by hand has no source.
	plain, err := testStore.CreateProductTx(ctx, CreateProductTxParams{Code: "LBL-" + util.RandomString(6), Name: "plain"})
	require.NoError(t, err)
	require.Empty(t, plain.Product.Source)

	// Like a numeric barcode, a text one can't be spread over several variants.
	_, err = testStore.CreateProductTx(ctx, CreateProductTxParams{
		Code: "LBL-" + util.RandomString(6), Name: "two", BarcodeText: "0" + barcode[1:],
		ColorIDs: []int64{1, 2},
	})
	require.Error(t, err)
}
