package db

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/mhdna/kashi/util"
	"github.com/stretchr/testify/require"
)

// A branch is discounted from its own list while that one is active; once
// it isn't, from the default (or none) -- never the inactive own list.
func TestActiveDiscountListForBranch(t *testing.T) {
	ctx := context.Background()
	branch, err := testQueries.CreateBranch(ctx, CreateBranchParams{Name: util.RandomString(8), Code: util.RandomString(4)})
	require.NoError(t, err)
	own, err := testQueries.CreateDiscountList(ctx, CreateDiscountListParams{
		Name: util.RandomString(8), IsActive: true,
		ValidFrom: time.Now().Add(-time.Hour), ValidTo: time.Now().Add(time.Hour),
	})
	require.NoError(t, err)
	require.NoError(t, testQueries.UpsertDiscountListBranch(ctx, UpsertDiscountListBranchParams{
		BranchID: branch.ID, DiscountListID: own.ID,
	}))

	got, err := testQueries.ActiveDiscountListForBranch(ctx, branch.ID)
	require.NoError(t, err)
	require.Equal(t, own.ID, got.ID)

	require.NoError(t, testQueries.UpdateDiscountList(ctx, UpdateDiscountListParams{
		ID: own.ID, Name: own.Name, IsActive: false, ValidFrom: own.ValidFrom, ValidTo: own.ValidTo,
	}))
	got, err = testQueries.ActiveDiscountListForBranch(ctx, branch.ID)
	if !errors.Is(err, sql.ErrNoRows) {
		require.NoError(t, err)
		require.NotEqual(t, own.ID, got.ID, "an inactive own list must not be the active one")
		require.True(t, got.IsDefault)
	}
}

func TestListVariantsForDiscountEntry(t *testing.T) {
	ctx := context.Background()
	variant := createRandomVariant(t)
	product, err := testQueries.GetProduct(ctx, variant.ProductID)
	require.NoError(t, err)

	byBarcode, err := testQueries.ListVariantsForDiscountEntry(ctx, variant.Barcode)
	require.NoError(t, err)
	require.Len(t, byBarcode, 1)
	require.Equal(t, variant.ID, byBarcode[0].ID)

	byCode, err := testQueries.ListVariantsForDiscountEntry(ctx, product.Code)
	require.NoError(t, err)
	require.NotEmpty(t, byCode)
	require.Equal(t, product.Code, byCode[0].ProductCode)
}
