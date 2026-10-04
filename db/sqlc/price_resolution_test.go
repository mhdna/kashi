package db

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/mhdna/kashi/util"
	"github.com/stretchr/testify/require"
)

// openList makes a list whose validity window is actually open, which the
// seeded "Default" list has and a hand-made one (valid_to defaults to now())
// does not -- see the note in 000094.
func openPriceList(t *testing.T, name string, isDefault bool) PriceList {
	list, err := testQueries.CreatePriceList(context.Background(), CreatePriceListParams{
		Name:      name,
		IsActive:  true,
		IsDefault: isDefault,
		ValidFrom: time.Now().Add(-time.Hour),
		ValidTo:   time.Now().Add(24 * time.Hour),
	})
	require.NoError(t, err)
	return list
}

func testBranch(t *testing.T) Branch {
	branch, err := testQueries.CreateBranch(context.Background(), CreateBranchParams{
		Name: util.RandomString(10),
		Code: util.RandomString(8),
	})
	require.NoError(t, err)
	return branch
}

// A variant in no list at all has no price. This is the invariant the whole
// change rests on: callers must read "no row" as "not sellable", never as free.
func TestResolveVariantPriceUnlistedVariantHasNoPrice(t *testing.T) {
	variant := createRandomVariant(t)
	branch := testBranch(t)

	_, err := testQueries.ResolveVariantPrice(context.Background(), ResolveVariantPriceParams{
		ProductVariantID: variant.ID,
		BranchID:         branch.ID,
	})
	require.ErrorIs(t, err, sql.ErrNoRows)
}

// With only the default list, every branch is priced from it.
func TestResolveVariantPriceFallsBackToDefaultList(t *testing.T) {
	variant := createRandomVariant(t)
	branch := testBranch(t)

	defaultList, err := testQueries.GetDefaultPriceList(context.Background())
	require.NoError(t, err)

	_, err = testQueries.CreatePriceListItem(context.Background(), CreatePriceListItemParams{
		PriceListID:      defaultList.ID,
		ProductVariantID: variant.ID,
		Price:            1500,
	})
	require.NoError(t, err)

	got, err := testQueries.ResolveVariantPrice(context.Background(), ResolveVariantPriceParams{
		ProductVariantID: variant.ID,
		BranchID:         branch.ID,
	})
	require.NoError(t, err)
	require.Equal(t, int64(1500), got)
}

// A branch assigned its own list is priced from that list, not the default --
// and a *different* branch still gets the default.
func TestResolveVariantPriceBranchListOverridesDefault(t *testing.T) {
	variant := createRandomVariant(t)
	assigned := testBranch(t)
	other := testBranch(t)

	defaultList, err := testQueries.GetDefaultPriceList(context.Background())
	require.NoError(t, err)
	_, err = testQueries.CreatePriceListItem(context.Background(), CreatePriceListItemParams{
		PriceListID:      defaultList.ID,
		ProductVariantID: variant.ID,
		Price:            1000,
	})
	require.NoError(t, err)

	branchList := openPriceList(t, util.RandomString(12), false)
	_, err = testQueries.CreatePriceListItem(context.Background(), CreatePriceListItemParams{
		PriceListID:      branchList.ID,
		ProductVariantID: variant.ID,
		Price:            2000,
	})
	require.NoError(t, err)
	require.NoError(t, testQueries.UpsertPriceListBranch(context.Background(), UpsertPriceListBranchParams{
		BranchID:    assigned.ID,
		PriceListID: branchList.ID,
	}))

	got, err := testQueries.ResolveVariantPrice(context.Background(), ResolveVariantPriceParams{
		ProductVariantID: variant.ID,
		BranchID:         assigned.ID,
	})
	require.NoError(t, err)
	require.Equal(t, int64(2000), got)

	got, err = testQueries.ResolveVariantPrice(context.Background(), ResolveVariantPriceParams{
		ProductVariantID: variant.ID,
		BranchID:         other.ID,
	})
	require.NoError(t, err)
	require.Equal(t, int64(1000), got)
}

// A branch list that doesn't mention a variant falls through to the default
// rather than making the variant unsellable at that branch.
func TestResolveVariantPriceBranchListGapFallsThrough(t *testing.T) {
	variant := createRandomVariant(t)
	assigned := testBranch(t)

	defaultList, err := testQueries.GetDefaultPriceList(context.Background())
	require.NoError(t, err)
	_, err = testQueries.CreatePriceListItem(context.Background(), CreatePriceListItemParams{
		PriceListID:      defaultList.ID,
		ProductVariantID: variant.ID,
		Price:            700,
	})
	require.NoError(t, err)

	// Assigned a list that contains some *other* variant, not this one.
	branchList := openPriceList(t, util.RandomString(12), false)
	require.NoError(t, testQueries.UpsertPriceListBranch(context.Background(), UpsertPriceListBranchParams{
		BranchID:    assigned.ID,
		PriceListID: branchList.ID,
	}))

	got, err := testQueries.ResolveVariantPrice(context.Background(), ResolveVariantPriceParams{
		ProductVariantID: variant.ID,
		BranchID:         assigned.ID,
	})
	require.NoError(t, err)
	require.Equal(t, int64(700), got)
}

// An inactive list prices nothing, even when a branch is assigned to it.
func TestResolveVariantPriceIgnoresInactiveList(t *testing.T) {
	variant := createRandomVariant(t)
	branch := testBranch(t)

	list := openPriceList(t, util.RandomString(12), false)
	_, err := testQueries.CreatePriceListItem(context.Background(), CreatePriceListItemParams{
		PriceListID:      list.ID,
		ProductVariantID: variant.ID,
		Price:            900,
	})
	require.NoError(t, err)
	require.NoError(t, testQueries.UpsertPriceListBranch(context.Background(), UpsertPriceListBranchParams{
		BranchID:    branch.ID,
		PriceListID: list.ID,
	}))
	require.NoError(t, testQueries.UpdatePriceList(context.Background(), UpdatePriceListParams{
		ID:        list.ID,
		Name:      list.Name,
		IsActive:  false,
		IsDefault: false,
		ValidFrom: list.ValidFrom,
		ValidTo:   list.ValidTo,
	}))

	_, err = testQueries.ResolveVariantPrice(context.Background(), ResolveVariantPriceParams{
		ProductVariantID: variant.ID,
		BranchID:         branch.ID,
	})
	require.ErrorIs(t, err, sql.ErrNoRows)
}

// Editing an item bumps the parent list's updated_at, which is what makes a
// repricing visible to the branch sync's "changed since" cursor.
func TestPriceListItemWriteTouchesListUpdatedAt(t *testing.T) {
	variant := createRandomVariant(t)
	list := openPriceList(t, util.RandomString(12), false)

	before, err := testQueries.GetPriceList(context.Background(), list.ID)
	require.NoError(t, err)

	time.Sleep(10 * time.Millisecond)
	_, err = testQueries.CreatePriceListItem(context.Background(), CreatePriceListItemParams{
		PriceListID:      list.ID,
		ProductVariantID: variant.ID,
		Price:            123,
	})
	require.NoError(t, err)

	after, err := testQueries.GetPriceList(context.Background(), list.ID)
	require.NoError(t, err)
	require.True(t, after.UpdatedAt.After(before.UpdatedAt),
		"adding an item must bump the list's updated_at so the sync notices")
}
