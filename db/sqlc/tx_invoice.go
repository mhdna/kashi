package db

import (
	"context"
	"errors"
	"fmt"
	"time"
)

type InvoiceItems struct {
	Product  Product
	Quantity int64
}

// InvoiceCode is generated inside the trasaction function itself.
type SalesInvoiceTxParams struct {
	CashboxID        int64          `json:"cashbox_id"`
	CashboxAccountID int64          `json:"cashbox_account_id"`
	ShiftID          int64          `json:"shift_id"`
	InventoryID      int64          `json:"inventory_id"`
	ClientID         int64          `json:"client_id"`
	Discount         int16          `json:"discount"`
	SubTotal         int64          `json:"sub_total"`
	DiscountedTotal  int64          `json:"discounted_total"`
	GrandTotal       int64          `json:"grand_total"`
	Year             int32          `json:"year"`
	Items            []InvoiceItems `json:"items"`
}

type SalesInvoiceTxResult struct {
	SalesInvoice SalesInvoice          `json:"sales_invoice"`
	Entry        Entry                 `json:"entry"`
	Balance      ShiftsAccountsBalance `json:"balance"`
	// NetAmount    int64        `json:"net_amount"`
	// Shift   Shift          `json:"shift"`
	// TODO: should we return items or inventory here?
}

func (q *Queries) generateInvoiceIndex(ctx context.Context, cashboxID int64, year int32) (int64, error) {
	arg := NextSalesInvoiceIndexIncrementParams{
		CashboxID: cashboxID,
		Year:      year,
	}
	index, err := q.NextSalesInvoiceIndexIncrement(ctx, arg)
	if err != nil {
		return 0, err
	}
	return index, nil
}

// generate invoice number in the format:
// CashboxCode/Type of Invoice/Year/Number of Invoice this Year
// E.g. BR1-SA-2026-00034 is the sales invoice number 34 in 2026 from POS Brooklyn1 that has the code BR1
func (q *Queries) generateInvoiceNumber(ctx context.Context, referenceType EntryReferenceType, invoiceIndex, cashboxID int64, year int32) (string, error) {
	var referenceCode string
	var err error

	// set countedInvoices and cashBox code
	cashbox, err := q.GetCashbox(ctx, cashboxID)
	if err != nil {
		return "", err
	}
	cashboxCode := cashbox.Code

	// set referenceCode and countedInvoices
	switch referenceType {
	case EntryReferenceTypeSalesInvoice:
		referenceCode = "SA"
	case EntryReferenceTypeReturnInvoice:
		referenceCode = "RN"
	default:
		return "", errors.New("Invalid Reference Type")
	}

	return fmt.Sprintf("%s-%s-%d-%05d", cashboxCode, referenceCode, year, invoiceIndex), nil
}

// TODO: see for return & sales & exchange
func validateInvoiceAmounts(items []InvoiceItems, discount int16, grandTotal, subTotal int64) error {
	var calculatedAmount, calculatedNetAmount int64

	for _, item := range items {
		itemPrice := item.Product.Price
		itemDiscountedPrice := itemPrice
		if item.Product.Discount > 0 {
			itemDiscountedPrice = itemPrice * int64(item.Product.Discount) / 100
		}
		calculatedAmount += itemPrice
		calculatedNetAmount += itemDiscountedPrice
	}
	if discount > 0 {
		calculatedAmount = calculatedAmount * int64(discount) / 100
	}

	if calculatedAmount != subTotal {
		return errors.New("Invalid Amount")
	}
	if calculatedNetAmount != grandTotal {
		return errors.New("Invalid Net Amount")
	}
	return nil
}

func (store *SQLStore) SalesInvoiceTx(ctx context.Context, arg SalesInvoiceTxParams) (SalesInvoiceTxResult, error) {
	var result SalesInvoiceTxResult

	err := store.execTx(ctx, func(q *Queries) error {
		var err error

		thisYear := int32(time.Now().Year())

		// txName := ctx.Value(txKey)
		invoiceIndex, err := q.generateInvoiceIndex(ctx, arg.CashboxID, thisYear)
		if err != nil {
			return err
		}

		invoiceCode, err := q.generateInvoiceNumber(ctx, EntryReferenceTypeSalesInvoice, invoiceIndex, arg.CashboxID, thisYear)
		if err != nil {
			return err
		}

		err = validateInvoiceAmounts(arg.Items, arg.Discount, arg.GrandTotal, arg.SubTotal)
		if err != nil {
			return err
		}

		salesInvoice, err := q.CreateSalesInvoice(ctx, CreateSalesInvoiceParams{
			CashboxID:       arg.CashboxID,
			InvoiceIndex:    invoiceIndex,
			InvoiceCode:     invoiceCode,
			Year:            thisYear,
			InventoryID:     arg.InventoryID,
			ClientID:        arg.ClientID,
			Discount:        arg.Discount,
			Subtotal:        arg.SubTotal,
			DiscountedTotal: arg.DiscountedTotal,
			GrandTotal:      arg.GrandTotal,
		})

		if err != nil {
			return err
		}

		entry, err := q.CreateEntryItem(ctx, CreateEntryItemParams{
			CashboxID:     arg.CashboxID,
			InventoryID:   arg.InventoryID,
			ReferenceType: EntryReferenceTypeSalesInvoice,
			ReferenceID:   salesInvoice.ID,
			Amount:        arg.GrandTotal,
		})
		if err != nil {
			return err
		}

		// update account balance
		addAccountBalanceArg := AddCashboxAccountBalanceParams{
			AccountID: arg.CashboxAccountID,
			ShiftID:   arg.ShiftID,
			Amount:    arg.GrandTotal,
		}
		balance, err := q.AddCashboxAccountBalance(ctx, addAccountBalanceArg)
		if err != nil {
			return err
		}

		// update inventory
		for _, i := range arg.Items {
			addInventoryProductQuantityArg := AddInventoryProductQuantityParams{
				InventoryID: arg.InventoryID,
				ProductID:   i.Product.ID,
				Quantity:    -i.Quantity,
			}
			err = q.AddInventoryProductQuantity(ctx, addInventoryProductQuantityArg)
			if err != nil {
				return err
			}
		}

		addPointsArg := AddClientLoyaltyPointsParams{
			ID:                 salesInvoice.ClientID,
			TotalLoyaltyPoints: arg.GrandTotal,
			ValidLoyaltyPoints: arg.GrandTotal,
		}
		err = q.AddClientLoyaltyPoints(ctx, addPointsArg)
		if err != nil {
			return err
		}

		result.SalesInvoice = salesInvoice
		result.Entry = entry
		result.Balance = balance

		return nil
	})

	return result, err
}
