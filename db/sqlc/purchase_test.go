package db

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/mhdna/kashi/util"
	"github.com/stretchr/testify/require"
)

func createRandomPurchase(t *testing.T) Purchase {
	supplier := createRandomSupplier(t)
	arg := CreatePurchaseParams{
		SupplierID:  supplier.ID,
		PurchasedAt: time.Now(),
	}

	purchase, err := testQueries.CreatePurchase(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, purchase)
	require.Equal(t, arg.SupplierID, purchase.SupplierID)
	require.WithinDuration(t, arg.PurchasedAt, purchase.PurchasedAt, time.Second)

	require.NotZero(t, purchase.ID)
	return purchase
}

func TestCreatePurchase(t *testing.T) {
	createRandomPurchase(t)
}

func TestGetPurchase(t *testing.T) {
	purchase1 := createRandomPurchase(t)
	purchase2, err := testQueries.GetPurchase(context.Background(), purchase1.ID)
	require.NoError(t, err)
	require.NotEmpty(t, purchase2)

	require.Equal(t, purchase1.ID, purchase2.ID)
	require.Equal(t, purchase1.SupplierID, purchase2.SupplierID)
	require.WithinDuration(t, purchase1.PurchasedAt, purchase2.PurchasedAt, time.Second)
}

func TestListPurchases(t *testing.T) {
	for range 10 {
		createRandomPurchase(t)
	}

	arg := ListPurchasesParams{
		Limit:  5,
		Offset: 5,
	}

	purchases, err := testQueries.ListPurchases(context.Background(), arg)
	require.NoError(t, err)
	require.Len(t, purchases, 5)

	for _, purchase := range purchases {
		require.NotEmpty(t, purchase)
	}
}

func TestAddPurchaseItem(t *testing.T) {
	product := createRandomProduct(t)
	purchase := createRandomPurchase(t)
	currency := createRandomCurrency(t)
	arg := AddPurchaseItemParams{
		PurchaseID: sql.NullInt64{Int64: purchase.ID, Valid: true},
		ProductID:  sql.NullInt64{Int64: product.ID, Valid: true},
		Quantity:   util.RandomQuantity(),
		UnitPrice:  util.RandomMoney(),
		CurrencyID: currency.ID,
	}

	purchased_item, err := testQueries.AddPurchaseItem(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, purchase)
	require.Equal(t, arg.PurchaseID, purchased_item.PurchaseID)
	require.Equal(t, arg.ProductID, purchased_item.ProductID)
	require.Equal(t, arg.Quantity, purchased_item.Quantity)
	require.Equal(t, arg.UnitPrice, purchased_item.UnitPrice)
	require.Equal(t, arg.CurrencyID, purchased_item.CurrencyID)

	require.NotZero(t, purchased_item.ID)
}
