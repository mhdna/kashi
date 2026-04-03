package db

import (
	"context"
	"fmt"
	"strconv"
	"testing"

	"github.com/mhdna/kashi/util"
	"github.com/stretchr/testify/require"
)

func createRandomSalesInvoice(t *testing.T) SalesInvoice {
	client := createRandomClient(t)
	inventory := createRandomInventory(t)
	amount := util.RandomMoneyAmount()
	discount := int64(0)
	a, _ := strconv.ParseFloat(amount, 64)
	netAmount := fmt.Sprintf("%.2f", a*(1-float64(discount)/100))
	arg := CreateSalesInvoiceParams{
		InvoiceNumber: string(util.RandomInt(0, 100)),
		InventoryID:   inventory.ID,
		ClientID:      client.ID,
		Amount:        amount, // TODO: check Random amount if redundant
		Discount:      discount,
		NetAmount:     netAmount,
	}

	order, err := testQueries.CreateSalesInvoice(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, order)
	require.Equal(t, arg.InvoiceNumber, order.InvoiceNumber)
	require.Equal(t, arg.InventoryID, order.InventoryID)
	require.Equal(t, arg.ClientID, order.ClientID)
	require.Equal(t, arg.Amount, order.Amount)
	require.Equal(t, arg.Discount, order.Discount)
	require.Equal(t, arg.NetAmount, order.NetAmount)

	require.NotZero(t, order.ID)
	require.NotZero(t, order.CreatedAt)

	return order
}

func TestCreateOrder(t *testing.T) {
	createRandomSalesInvoice(t)
}
