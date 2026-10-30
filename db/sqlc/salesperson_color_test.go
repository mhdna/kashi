package db

import (
	"context"
	"database/sql"
	"testing"

	"github.com/mhdna/kashi/util"
	"github.com/stretchr/testify/require"
)

// A new salesperson gets the colour fewest in their branch have (000217),
// so a branch's first twelve are all different.
func TestNewSalespersonGetsAnUnusedColor(t *testing.T) {
	ctx := context.Background()
	branch, err := testQueries.CreateBranch(ctx, CreateBranchParams{Name: util.RandomString(8), Code: util.RandomString(4)})
	require.NoError(t, err)
	seen := map[string]bool{}
	for range 12 {
		p, err := testQueries.CreateBranchSalesperson(ctx, CreateBranchSalespersonParams{
			Name: util.RandomString(6), BranchID: sql.NullInt64{Int64: branch.ID, Valid: true}, IsSalesperson: true,
		})
		require.NoError(t, err)
		require.NotEmpty(t, p.Color)
		require.False(t, seen[p.Color], "colour %s given twice in one branch", p.Color)
		seen[p.Color] = true
	}
	// The thirteenth repeats one, as there are only twelve.
	p, err := testQueries.CreateBranchSalesperson(ctx, CreateBranchSalespersonParams{
		Name: util.RandomString(6), BranchID: sql.NullInt64{Int64: branch.ID, Valid: true},
	})
	require.NoError(t, err)
	require.True(t, seen[p.Color])
}
