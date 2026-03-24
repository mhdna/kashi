package db

import (
	"context"
	"testing"

	"github.com/mhdna/kashi/util"
	"github.com/stretchr/testify/require"
)

func TestTransferTX(t *testing.T) {
	store := NewStore(testDB)

	inventory1 := createRandomInventory(t)
	inventory2 := createRandomInventory(t)
	products := []PTransferItem{}

	for range 5 {
		p := createRandomProduct(t)
		product := PTransferItem{
			ProductId: p.ID,
			Quantity:  util.RandomInt(1, 10),
		}
		products = append(products, product)
	}

	errs := make(chan error)
	results := make(chan PTransferTxResult)

	for range 5 {
		go func() {
			result, err := store.PTransferTx(context.Background(), PTransferTxParams{
				FromInventoryID: inventory1.ID,
				ToInventoryID:   inventory2.ID,
				Products:        products,
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
		transfer := result.PTransfer

		require.NotEmpty(t, transfer)
		require.Equal(t, inventory1.ID, transfer.FromInventoryID)
		require.Equal(t, inventory2.ID, transfer.ToInventoryID)
		require.NotZero(t, transfer.ID)
		require.NotZero(t, transfer.CreatedAt)

		// for _, p := range products {
		// 	require.NotZero(t, p.Quantity)
		// }
		_, err = store.GetPTransfer(context.Background(), transfer.ID)
		require.NoError(t, err)

		products := result.Products
		require.NotEmpty(t, products)
		for _, p := range products {
			require.NotZero(t, p.ProductId)

			arg := GetPTransferProductParams{
				TransferID: transfer.ID,
				ProductID:  p.ProductId,
			}
			ptransfer, err := store.GetPTransferProduct(context.Background(), arg)
			require.NoError(t, err)
			require.Equal(t, transfer.ID, ptransfer.TransferID)
			require.Equal(t, p.ProductId, ptransfer.ProductID)
			require.Equal(t, p.Quantity, ptransfer.Quantity)
		}
	}
}
