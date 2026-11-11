package db

import (
	"context"
	"testing"

	"github.com/mhdna/kashi/util"
	"github.com/stretchr/testify/require"
)

// The till's sync light is quiet by default; a branch can switch on the
// every-step display, and switch it off again.
func TestBranchSettings_SyncVerboseDefaultsOffAndSaves(t *testing.T) {
	ctx := context.Background()
	var branchID int64
	require.NoError(t, testDB.QueryRowContext(ctx,
		`INSERT INTO branches (name, code) VALUES ($1, $2) RETURNING id`,
		"sv-"+util.RandomString(6), "SV"+util.RandomString(8)).Scan(&branchID))
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
	require.False(t, saved.SyncVerbose, "quiet by default")

	arg.SyncVerbose = true
	saved, err = testQueries.UpsertBranchSettings(ctx, arg)
	require.NoError(t, err)
	require.True(t, saved.SyncVerbose)
	require.False(t, saved.VariantPickRequired, "a neighbouring setting is untouched")

	arg.SyncVerbose = false
	saved, err = testQueries.UpsertBranchSettings(ctx, arg)
	require.NoError(t, err)
	require.False(t, saved.SyncVerbose)
}
