package db

import (
	"context"
	"testing"
	"time"

	"github.com/mhdna/kashi/util"
	"github.com/stretchr/testify/require"
)

func createRandomSalesInvoice(t *testing.T) SalesInvoice {
	client := createRandomClient(t)
	cashbox := createRandomCashbox(t)
	inventory := createRandomInventory(t)
	discount := int16(0)
	grandTotal := util.RandomAmount()
	subTotal := util.RandomAmount()
	discountedTotal := util.RandomAmount()

	arg := CreateSalesInvoiceParams{
		InventoryID:     inventory.ID,
		ClientID:        client.ID,
		CashboxID:       cashbox.ID,
		Discount:        discount,
		Subtotal:        subTotal,
		GrandTotal:      grandTotal,
		DiscountedTotal: discountedTotal,
		Year:            int32(time.Now().Year()),
	}

	order, err := testQueries.CreateSalesInvoice(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, order)
	require.Equal(t, arg.InvoiceIndex, order.InvoiceIndex)
	require.Equal(t, arg.InvoiceCode, order.InvoiceCode)
	require.Equal(t, arg.Year, order.Year)
	require.Equal(t, arg.InventoryID, order.InventoryID)
	require.Equal(t, arg.ClientID, order.ClientID)
	require.Equal(t, arg.Discount, order.Discount)
	require.Equal(t, arg.GrandTotal, order.GrandTotal)
	require.Equal(t, arg.Subtotal, order.Subtotal)
	require.Equal(t, arg.DiscountedTotal, order.DiscountedTotal)

	require.NotZero(t, order.ID)
	require.NotZero(t, order.CreatedAt)

	return order
}

func TestCreateSalesInvoice(t *testing.T) {
	createRandomSalesInvoice(t)
}

func TestAddSalesInvoiceProduct(t *testing.T) {
	salesInvoice := createRandomSalesInvoice(t)
	product := createRandomProduct(t)

	arg := AddSalesInvoiceProductParams{
		InvoiceID: salesInvoice.ID,
		ProductID: product.ID,
		Price:     product.Price,
		Discount:  product.Discount,
		Quantity:  util.RandomQuantity(),
	}

	result, err := testQueries.AddSalesInvoiceProduct(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, result)
	require.Equal(t, arg.InvoiceID, result.InvoiceID)
	require.Equal(t, arg.ProductID, result.ProductID)
	require.Equal(t, arg.Price, result.Price)
	require.Equal(t, arg.Quantity, result.Quantity)
}

func TestListSalesInvoices(t *testing.T) {
	for range 10 {
		createRandomSalesInvoice(t)
	}

	arg := ListSalesInvoicesParams{
		Limit:  5,
		Offset: 5,
	}

	invoices, err := testQueries.ListSalesInvoices(context.Background(), arg)
	require.NoError(t, err)
	require.Len(t, invoices, 5)
	for _, invoice := range invoices {
		require.NotEmpty(t, invoice)
	}
}

func TestNextSalesInvoiceIndexIncrement(t *testing.T) {
	cashbox := createRandomCashbox(t)
	year := int32(time.Now().Year())

	arg := NextSalesInvoiceIndexIncrementParams{
		Year:      year,
		CashboxID: cashbox.ID,
	}

	index1, err := testQueries.NextSalesInvoiceIndexIncrement(context.Background(), arg)
	require.NoError(t, err)
	require.Equal(t, int64(1), index1)

	index2, err := testQueries.NextSalesInvoiceIndexIncrement(context.Background(), arg)
	require.NoError(t, err)
	require.Equal(t, int64(2), index2)

	index3, err := testQueries.NextSalesInvoiceIndexIncrement(context.Background(), arg)
	require.NoError(t, err)
	require.Equal(t, int64(3), index3)
}

func createRandomReturnInvoice(t *testing.T) ReturnInvoice {
	salesInvoice := createRandomSalesInvoice(t)
	cashbox := createRandomCashbox(t)
	client := createRandomClient(t)
	inventory := createRandomInventory(t)

	year := int32(time.Now().Year())

	arg := CreateReturnInvoiceParams{
		CashboxID:       cashbox.ID,
		ShiftID:         0,
		InvoiceCode:     "",
		InvoiceIndex:    0,
		Year:            year,
		ClientID:        client.ID,
		InventoryID:     inventory.ID,
		Discount:        0,
		Subtotal:        util.RandomAmount(),
		DiscountedTotal: util.RandomAmount(),
		GrandTotal:      util.RandomAmount(),
		SalesInvoiceID:  salesInvoice.ID,
	}

	returnInvoice, err := testQueries.CreateReturnInvoice(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, returnInvoice)
	require.Equal(t, arg.CashboxID, returnInvoice.CashboxID)
	require.Equal(t, arg.ClientID, returnInvoice.ClientID)
	require.Equal(t, arg.InventoryID, returnInvoice.InventoryID)
	require.Equal(t, arg.Subtotal, returnInvoice.Subtotal)
	require.Equal(t, arg.DiscountedTotal, returnInvoice.DiscountedTotal)
	require.Equal(t, arg.GrandTotal, returnInvoice.GrandTotal)
	require.Equal(t, arg.SalesInvoiceID, returnInvoice.SalesInvoiceID)

	require.NotZero(t, returnInvoice.ID)
	require.NotZero(t, returnInvoice.CreatedAt)

	return returnInvoice
}

func TestCreateReturnInvoice(t *testing.T) {
	createRandomReturnInvoice(t)
}

func TestGetReturnInvoice(t *testing.T) {
	returnInvoice1 := createRandomReturnInvoice(t)
	returnInvoice2, err := testQueries.GetReturnInvoice(context.Background(), returnInvoice1.ID)
	require.NoError(t, err)
	require.Equal(t, returnInvoice1.ID, returnInvoice2.ID)
	require.Equal(t, returnInvoice1.CashboxID, returnInvoice2.CashboxID)
	require.Equal(t, returnInvoice1.ClientID, returnInvoice2.ClientID)
	require.Equal(t, returnInvoice1.SalesInvoiceID, returnInvoice2.SalesInvoiceID)
}

func TestAddReturnInvoiceProduct(t *testing.T) {
	returnInvoice := createRandomReturnInvoice(t)
	product := createRandomProduct(t)

	arg := AddReturnInvoiceProductParams{
		InvoiceID: returnInvoice.ID,
		ProductID: product.ID,
		Price:     product.Price,
		Discount:  product.Discount,
		Quantity:  util.RandomQuantity(),
	}

	result, err := testQueries.AddReturnInvoiceProduct(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, result)
	require.Equal(t, arg.InvoiceID, result.InvoiceID)
	require.Equal(t, arg.ProductID, result.ProductID)
	require.Equal(t, arg.Price, result.Price)
	require.Equal(t, arg.Quantity, result.Quantity)
}

func TestListReturnInvoices(t *testing.T) {
	for range 10 {
		createRandomReturnInvoice(t)
	}

	arg := ListReturnInvoicesParams{
		Limit:  5,
		Offset: 5,
	}

	invoices, err := testQueries.ListReturnInvoices(context.Background(), arg)
	require.NoError(t, err)
	require.Len(t, invoices, 5)
	for _, invoice := range invoices {
		require.NotEmpty(t, invoice)
	}
}

func TestNextReturnInvoiceIndexIncrement(t *testing.T) {
	cashbox := createRandomCashbox(t)
	year := int32(time.Now().Year())

	arg := NextReturnInvoiceIndexIncrementParams{
		Year:      year,
		CashboxID: cashbox.ID,
	}

	index1, err := testQueries.NextReturnInvoiceIndexIncrement(context.Background(), arg)
	require.NoError(t, err)
	require.Equal(t, int64(1), index1)

	index2, err := testQueries.NextReturnInvoiceIndexIncrement(context.Background(), arg)
	require.NoError(t, err)
	require.Equal(t, int64(2), index2)
}
