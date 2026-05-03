package db

import (
	"context"
	"log"
	"testing"
	"time"

	"github.com/mhdna/kashi/util"
	"github.com/stretchr/testify/require"
)

func createRandomSalesInvoice(t *testing.T) SalesInvoice {
	client := createRandomClient(t)
	cashbox := createRandomCashbox(t)
	inventory := createRandomInventory(t)
	amount := util.RandomAmount()
	discount := int16(0)
	netAmount, err := util.CalculateNetAmount(amount, discount)
	if err != nil {
		log.Fatal(err)
	}
	arg := CreateSalesInvoiceParams{
		InventoryID: inventory.ID,
		ClientID:    client.ID,
		CashboxID:   cashbox.ID,
		Amount:      amount,
		Discount:    discount,
		NetAmount:   netAmount,
		Year:        int32(time.Now().Year()),
	}

	order, err := testQueries.CreateSalesInvoice(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, order)
	require.Equal(t, arg.InvoiceIndex, order.InvoiceIndex)
	require.Equal(t, arg.InvoiceCode, order.InvoiceCode)
	require.Equal(t, arg.Year, order.Year)
	require.Equal(t, arg.InventoryID, order.InventoryID)
	require.Equal(t, arg.ClientID, order.ClientID)
	require.Equal(t, arg.Amount, order.Amount)
	require.Equal(t, arg.Discount, order.Discount)
	require.Equal(t, arg.NetAmount, order.NetAmount)

	require.NotZero(t, order.ID)
	require.NotZero(t, order.CreatedAt)

	return order
}

func TestCreateSalesInvoice(t *testing.T) {
	createRandomSalesInvoice(t)
}
