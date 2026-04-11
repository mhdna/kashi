package db

import (
	"context"
	"testing"

	"github.com/mhdna/kashi/util"
	"github.com/stretchr/testify/require"
)

// func TestTransferTX(t *testing.T) {
// 	store := NewStore(testDB)

// 	inventory1 := createRandomInventory(t)
// 	inventory2 := createRandomInventory(t)

// 	errs := make(chan error)
// 	results := make(chan TransferTxResult)

// 	for range 5 {
// 		go func() {
// 			result, err := store.TransferTx(context.Background(), TransferTxParams{
// 				FromInventoryID: inventory1.ID,
// 				ToInventoryID:   inventory2.ID,
// 				TransferType:    TransferTypeProducts,
// 			})
// 			errs <- err
// 			results <- result
// 		}()
// 	}

// 	for range 5 {
// 		err := <-errs
// 		require.NoError(t, err)

// 		result := <-results
// 		require.NotEmpty(t, results)

// 		// check transfer
// 		transfer := result.Transfer

// 		items := result.Items

// 		require.NotEmpty(t, transfer)
// 		require.Equal(t, inventory1.ID, transfer.FromInventoryID)
// 		require.Equal(t, inventory2.ID, transfer.ToInventoryID)
// 		require.NotZero(t, transfer.ID)
// 		require.NotZero(t, transfer.CreatedAt)

// 		for _, i := range items {
// 			require.NotZero(t, i.Quantity)
// 		}
// 	}
// }

// TODO add tests to see if invoices are incrementing properly
// TODO add tests to cover different cashboxes, accounts, etc.
func TestSalesInvoiceTx(t *testing.T) {
	store := NewStore(testDB)

	n := 5

	type txResult struct {
		salesInvoice SalesInvoice
		idx          int
	}
	errs := make(chan error)
	results := make(chan txResult)

	for i := range n {
			txName := fmt.Sprintf("tx %d", i+1)
			txRes, err := store.SalesInvoiceTx(context.WithValue(context.Background(), txKey, txName), SalesInvoiceTxParams{
				CashBoxID:    cashbox.ID,
				CurrencyCode: currency.Code,
				InventoryID:  inventory.ID,
				ClientID:     client.ID,
				Amount:       amount,
				Discount:     discount,
				Year:         int32(time.Now().Year()),
			})
			errs <- err
			results <- txResult{salesInvoice: txRes.SalesInvoice, idx: i}
		}(i)
	}

	for range n {
		err := <-errs
		require.NoError(t, err)

		res := <-results
		salesInvoice := res.salesInvoice
		// i := res.idx

		// require.NotEmpty(t, salesInvoice)
		// require.Equal(t, inventory.ID, salesInvoice.InventoryID)
		// require.Equal(t, cashbox.ID, salesInvoice.CashboxID)
		// require.Equal(t, currency.Code, salesInvoice.CurrencyCode)
		// require.Equal(t, client.ID, salesInvoice.ClientID)
		// require.Equal(t, amounts[i], salesInvoice.Amount)
		// require.Equal(t, discounts[i], salesInvoice.Discount)

		require.NotZero(t, salesInvoice.ID)
		require.NotZero(t, salesInvoice.CreatedAt)

	}
}
