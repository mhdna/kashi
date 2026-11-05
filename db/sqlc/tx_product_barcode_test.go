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
