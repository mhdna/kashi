package db

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTransferTX(t *testing.T) {
	store := NewStore(testDB)

	inventory1 := createRandomInventory(t)
	inventory2 := createRandomInventory(t)

	errs := make(chan error)
	results := make(chan TransferTxResult)

	for range 5 {
		go func() {
			result, err := store.TransferTx(context.Background(), TransferTxParams{
				FromInventoryID: inventory1.ID,
				ToInventoryID:   inventory2.ID,
			})
			errs <- err
			results <- result
		}()
	}

	for range 5 {
		err := <-errs
		require.NoError(t, err)

		result := <-results
		require.NotEmpty(t, results)

		// check transfer
		transfer := result.Transfer

		// TODO: change those to items that depends on type
		products := result.Products

		require.NotEmpty(t, transfer)
		require.Equal(t, inventory1.ID, transfer.FromInventoryID)
		require.Equal(t, inventory2.ID, transfer.ToInventoryID)
		require.NotZero(t, transfer.ID)
		require.NotZero(t, transfer.CreatedAt)

		for _, p := range products {
			require.NotZero(t, p.Quantity)
			// TODO check every product equal and pass it
		}
	}
}
