package db

import (
	"context"
	"log"
	"testing"
	"time"

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
	amount := util.RandomAmount()
	discount := util.RandomDiscount()
	cashbox := createRandomCashbox(t)
	account := createRandomAccount(t)
	inventory := createRandomInventory(t)
	currency := createRandomCurrency(t)
	client := createRandomClient(t)

	n := 5

	errs := make(chan error)
	results := make(chan SalesInvoiceTxResult)

	for range n {
		go func() {
			res, err := store.SalesInvoiceTx(context.Background(), SalesInvoiceTxParams{
				CashBoxID:        cashbox.ID,
				CashboxAccountID: account.ID,
				CurrencyCode:     currency.Code,
				InventoryID:      inventory.ID,
				ClientID:         client.ID,
				Amount:           amount,
				Discount:         discount,
				Year:             int32(time.Now().Year()),
			})
			errs <- err
			results <- res
		}()
	}

	for range n {
		err := <-errs
		require.NoError(t, err)

		res := <-results

		salesInvoice := res.SalesInvoice
		require.NotEmpty(t, salesInvoice)
		require.Equal(t, inventory.ID, salesInvoice.InventoryID)
		require.Equal(t, cashbox.ID, salesInvoice.CashboxID)
		require.Equal(t, currency.Code, salesInvoice.CurrencyCode)
		require.Equal(t, client.ID, salesInvoice.ClientID)
		require.Equal(t, amount, salesInvoice.Amount)
		require.Equal(t, discount, salesInvoice.Discount)
		require.NotZero(t, salesInvoice.ID)
		require.NotZero(t, salesInvoice.CreatedAt)

		entry := res.Entry
		require.Equal(t, cashbox.ID, entry.CashboxID)
		require.Equal(t, inventory.ID, entry.InventoryID)
		netAmount, err := util.CalculateNetAmount(amount, discount)
		if err != nil {
			log.Fatal(err)
		}
		require.Equal(t, netAmount, entry.NetAmountInDefaultCurrency)
		require.Equal(t, salesInvoice.ID, entry.ReferenceID)
		require.Equal(t, EntryReferenceTypeSalesInvoice, entry.ReferenceType)
		require.NotZero(t, entry.CreatedAt)
		require.NotZero(t, entry.ID)

		balance := res.Balance
		_ = balance
	}
}
