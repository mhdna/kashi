package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/mhdna/kashi/util"
)

// NetAmount & InvoiceCode are calculated and generated
// automatically inside the trasaction function itself.
type SalesInvoiceTxParams struct {
	CashBoxID        int64  `json:"cashbox_id"`
	CashboxAccountID int64  `json:"cashbox_account_id"`
	ShiftID          int64  `json:"shift_id"`
	CurrencyCode     string `json:"currency_code"`
	InventoryID      int64  `json:"inventory_id"`
	ClientID         int64  `json:"client_id"`
	Amount           int64  `json:"amount"`
	Discount         int16  `json:"discount"`
	Year             int32  `json:"year"`
}

type SalesInvoiceTxResult struct {
	SalesInvoice SalesInvoice `json:"sales_invoice"`
	// NetAmount    int64        `json:"net_amount"`
	Entry   Entry          `json:"entry"`
	Shift   Shift          `json:"shift"`
	Account CashboxAccount `json:"account"`
}

func (q *Queries) generateSalesInvoiceIndex(ctx context.Context, cashboxID int64) (int64, error) {
	thisYear := time.Now().Year()
	arg := NextSalesInvoiceIndexIncrementParams{
		CashboxID: cashboxID,
		Year:      int32(thisYear),
	}
	index, err := q.NextSalesInvoiceIndexIncrement(ctx, arg)
	if err != nil {
		return 0, err
	}
	return index, nil
}

// generate invoice number in the format:
// CashboxCode/Type of Invoice/Year/Number of Invoice this Year
// E.g. BR1/SA/2026/34 is the sales invoice number 34 in 2026 from POS Brooklyn1 that has the code BR1
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

	return fmt.Sprintf("%s/%s/%d/%d", cashboxCode, referenceCode, year, invoiceIndex), nil
}

func (store *SQLStore) SalesInvoiceTx(ctx context.Context, arg SalesInvoiceTxParams) (SalesInvoiceTxResult, error) {
	var result SalesInvoiceTxResult

	err := store.execTx(ctx, func(q *Queries) error {
		var err error

		// txName := ctx.Value(txKey)
		invoiceIndex, err := q.generateSalesInvoiceIndex(ctx, arg.CashBoxID)
		if err != nil {
			return err
		}

		thisYear := int32(time.Now().Year())

		invoiceCode, err := store.generateInvoiceNumber(ctx, EntryReferenceTypeSalesInvoice, invoiceIndex, arg.CashBoxID, thisYear)
		if err != nil {
			return err
		}
		netAmount, err := util.CalculateNetAmount(arg.Amount, arg.Discount)
		if err != nil {
			return err
		}
		result.SalesInvoice, err = q.CreateSalesInvoice(ctx, CreateSalesInvoiceParams{
			CashboxID:    arg.CashBoxID,
			InvoiceIndex: invoiceIndex,
			InvoiceCode:  invoiceCode,
			Year:         thisYear,
			InventoryID:  arg.InventoryID,
			ClientID:     arg.ClientID,
			Amount:       arg.Amount,
			Discount:     arg.Discount,
			NetAmount:    netAmount,
			CurrencyCode: arg.CurrencyCode,
		})
		if err != nil {
			return err
		}

		entry, err := q.CreateEntryItem(ctx, CreateEntryItemParams{
			CashboxID:                  arg.CashBoxID,
			InventoryID:                arg.InventoryID,
			ReferenceType:              EntryReferenceTypeSalesInvoice,
			ReferenceID:                result.SalesInvoice.ID,
			NetAmountInDefaultCurrency: netAmount,
		})
		if err != nil {
			return err
		}

		// update shift total balance
		addShiftBalanceArg := AddToShiftBalanceParams{
			ID:     arg.ShiftID,
			Amount: netAmount,
		}
		shift, err := q.AddToShiftBalance(ctx, addShiftBalanceArg)
		if err != nil {
			return err
		}

		// update account balance
		addAccountBalance := AddAccountBalanceParams{
			ID:     arg.CashboxAccountID,
			Amount: netAmount,
		}
		account, err := q.AddAccountBalance(ctx, addAccountBalance)
		if err != nil {
			return err
		}

		// TODO: Update client points

		result.Entry = entry
		result.Shift = shift
		result.Account = account

		return nil
	})

	return result, err
}
