package db

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/mhdna/kashi/util"
	"github.com/stretchr/testify/require"
)

func backdateVariant(t *testing.T, id int64) time.Time {
	_, err := testDB.ExecContext(context.Background(),
		`UPDATE product_variants SET updated_at = now() - interval '1 day' WHERE id = $1`, id)
	require.NoError(t, err)
	return time.Now().Add(-time.Hour)
}

func feedRow(t *testing.T, uid uuid.UUID) ListProductVariantsForSyncByUIDsRow {
	rows, err := testQueries.ListProductVariantsForSyncByUIDs(context.Background(), []uuid.UUID{uid})
	require.NoError(t, err)
	require.Len(t, rows, 1)
	return rows[0]
}

// The reported case: an item set to "always ask" reaches the till's product
// feed on every variant, and the change moves the variants so a till that
// already has them picks it up.
func TestSetVariantPickForEntries_ReachesTheProductFeed(t *testing.T) {
	ctx := context.Background()
	variant := createRandomVariant(t)
	product, err := testQueries.GetProduct(ctx, variant.ProductID)
	require.NoError(t, err)
	require.Equal(t, "", product.VariantPick, "an item follows the branch until told otherwise")
	require.Equal(t, "", feedRow(t, variant.Uid).VariantPick)
	before := backdateVariant(t, variant.ID)

	n, err := testQueries.SetVariantPickForEntries(ctx, SetVariantPickForEntriesParams{
		VariantPick: "always", Entries: []string{product.Code},
	})
	require.NoError(t, err)
	require.EqualValues(t, 1, n)

	row := feedRow(t, variant.Uid)
	require.Equal(t, "always", row.VariantPick)
	require.True(t, row.UpdatedAt.After(before), "the change has to move the variant for the delta feed")
}

func TestSetVariantPickForEntries_Edges(t *testing.T) {
	ctx := context.Background()

	t.Run("a variant barcode and a code in another case both name the product", func(t *testing.T) {
		a, b := createRandomVariant(t), createRandomVariant(t)
		pa, _ := testQueries.GetProduct(ctx, a.ProductID)
		pb, _ := testQueries.GetProduct(ctx, b.ProductID)
		n, err := testQueries.SetVariantPickForEntries(ctx, SetVariantPickForEntriesParams{
			VariantPick: "never", Entries: []string{a.Barcode, strings.ToLower(pb.Code)},
		})
		require.NoError(t, err)
		require.EqualValues(t, 2, n)
		require.Equal(t, "never", feedRow(t, a.Uid).VariantPick)
		require.Equal(t, "never", feedRow(t, b.Uid).VariantPick)
		_ = pa
	})

	t.Run("every variant of the item carries it, and other items are left alone", func(t *testing.T) {
		first := createRandomVariant(t)
		product, _ := testQueries.GetProduct(ctx, first.ProductID)
		second, err := testQueries.CreateProductVariant(ctx, CreateProductVariantParams{
			ProductID: product.ID, Barcode: util.RandomString(16),
		})
		require.NoError(t, err)
		other := createRandomVariant(t)

		_, err = testQueries.SetVariantPickForEntries(ctx, SetVariantPickForEntriesParams{
			VariantPick: "always", Entries: []string{first.Barcode},
		})
		require.NoError(t, err)
		require.Equal(t, "always", feedRow(t, first.Uid).VariantPick)
		require.Equal(t, "always", feedRow(t, second.Uid).VariantPick)
		require.Equal(t, "", feedRow(t, other.Uid).VariantPick)
	})

	t.Run("an empty mode goes back to following the branch", func(t *testing.T) {
		v := createRandomVariant(t)
		p, _ := testQueries.GetProduct(ctx, v.ProductID)
		for _, mode := range []string{"always", ""} {
			_, err := testQueries.SetVariantPickForEntries(ctx, SetVariantPickForEntriesParams{
				VariantPick: mode, Entries: []string{p.Code},
			})
			require.NoError(t, err)
			require.Equal(t, mode, feedRow(t, v.Uid).VariantPick)
		}
	})

	t.Run("an unknown entry matches nothing", func(t *testing.T) {
		n, err := testQueries.SetVariantPickForEntries(ctx, SetVariantPickForEntriesParams{
			VariantPick: "always", Entries: []string{"NO-SUCH-" + util.RandomString(8)},
		})
		require.NoError(t, err)
		require.Zero(t, n)
	})

	t.Run("setting the same value again doesn't move the variants", func(t *testing.T) {
		v := createRandomVariant(t)
		p, _ := testQueries.GetProduct(ctx, v.ProductID)
		_, err := testQueries.SetVariantPickForEntries(ctx, SetVariantPickForEntriesParams{VariantPick: "always", Entries: []string{p.Code}})
		require.NoError(t, err)
		before := backdateVariant(t, v.ID)
		_, err = testQueries.SetVariantPickForEntries(ctx, SetVariantPickForEntriesParams{VariantPick: "always", Entries: []string{p.Code}})
		require.NoError(t, err)
		require.True(t, feedRow(t, v.Uid).UpdatedAt.Before(before))
	})

	t.Run("the table refuses a value it doesn't know", func(t *testing.T) {
		v := createRandomVariant(t)
		p, _ := testQueries.GetProduct(ctx, v.ProductID)
		_, err := testQueries.SetVariantPickForEntries(ctx, SetVariantPickForEntriesParams{VariantPick: "sometimes", Entries: []string{p.Code}})
		require.Error(t, err)
	})
}

// The branch default is off, and saving it on (then off) sticks.
func TestBranchSettings_VariantPickRequiredDefaultsOffAndSaves(t *testing.T) {
	ctx := context.Background()
	var branchID int64
	require.NoError(t, testDB.QueryRowContext(ctx,
		`INSERT INTO branches (name, code) VALUES ($1, $2) RETURNING id`,
		"vp-"+util.RandomString(6), "VP"+util.RandomString(8)).Scan(&branchID))
	arg := UpsertBranchSettingsParams{
		BranchID:        branchID,
		RoundingMode:    "half_up",
		OversellMode:    "allow",
		SocialPlatforms: []byte(`[]`),
		SocialHandles:   []byte(`{}`),
		DisplayPrefs:    []byte(`{}`),
	}
	saved, err := testQueries.UpsertBranchSettings(ctx, arg)
	require.NoError(t, err)
	require.False(t, saved.VariantPickRequired)

	arg.VariantPickRequired = true
	saved, err = testQueries.UpsertBranchSettings(ctx, arg)
	require.NoError(t, err)
	require.True(t, saved.VariantPickRequired)

	arg.VariantPickRequired = false
	saved, err = testQueries.UpsertBranchSettings(ctx, arg)
	require.NoError(t, err)
	require.False(t, saved.VariantPickRequired)
}
