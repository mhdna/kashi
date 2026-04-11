package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/mhdna/kashi/util"
)

type Store interface {
	Querier
	// TransferTx(ctx context.Context, arg TransferTxParams) (TransferTxResult, error)
	SalesInvoiceTx(ctx context.Context, arg SalesInvoiceTxParams) (SalesInvoiceTxResult, error)
	ReturnInvoiceTx(ctx context.Context, arg ReturnInvoiceTxParams) (ReturnInvoiceTxResult, error)
}

// provides all the functions to execute SQL queries
type SQLStore struct {
	*Queries
	db *sql.DB
}

func NewStore(db *sql.DB) Store {
	return &SQLStore{
		db:      db,
		Queries: New(db),
	}
}

func (store *SQLStore) execTx(ctx context.Context, fn func(*Queries) error) error {
	tx, err := store.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	q := New(tx)
	err = fn(q)
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("tx err: %v, rb err: %v", err, rbErr)
		}
		return err
	}

	return tx.Commit()
}

// type TransferTxParams struct {
// 	FromInventoryID int64          `json:"from_inventory_id"`
// 	ToInventoryID   int64          `json:"to_inventory_id"`
// 	Items           []TransferItem `json:"items"`
// 	TransferType    TransferType   `json:"type"`
// }

// type TransferTxResult struct {
// 	Transfer      Transfer       `json:"transfer"`
// 	FromInventory Inventory      `json:"from_inventory"`
// 	ToInventory   Inventory      `json:"to_inventory"`
// 	Items         []TransferItem `json:"items"`
// }

// // TransferTx performs an inventory transfer (assets/products) from one inventory to another.
// // It creates a transfer record, adds inventory entries, and updates inventory balance within a single db transaction.
// func (store *SQLStore) TransferTx(ctx context.Context, arg TransferTxParams) (TransferTxResult, error) {
// 	var result TransferTxResult

// 	err := store.execTx(ctx, func(q *Queries) error {
// 		var err error

// 		result.Transfer, err = q.CreateTransfer(ctx, CreateTransferParams{
// 			FromInventoryID: arg.FromInventoryID,
// 			ToInventoryID:   arg.ToInventoryID,
// 			Type:            arg.TransferType,
// 		})
// 		if err != nil {
// 			return err
// 		}

// 		for _, i := range arg.Items {
// 			var productID, assetID sql.NullInt64
// 			switch arg.TransferType {
// 			case TransferTypeAssets:
// 				assetID = i.AssetID
// 			case TransferTypeProducts:
// 				productID = i.ProductID
// 			default:
// 				return fmt.Errorf("unknown transfer type: %s", arg.TransferType)
// 			}

// 			_, err := q.CreateTransferItem(ctx, CreateTransferItemParams{
// 				TransferID: result.Transfer.ID,
// 				ProductID:  productID,
// 				AssetID:    assetID,
// 				Quantity:   i.Quantity,
// 			})
// 			if err != nil {
// 				return err
// 			}
// 			_, err = q.CreateEntryItem(ctx, CreateEntryItemParams{
// 				InventoryID:   arg.FromInventoryID,
// 				ReferenceType: EntryReferenceTypeTransfer,
// 				ReferenceID:   result.Transfer.ID,
// 				ProductID:     productID,
// 				AssetID:       assetID,
// 				Quantity:      i.Quantity,
// 			})
// 			if err != nil {
// 				return err
// 			}
// 			_, err = q.CreateEntryItem(ctx, CreateEntryItemParams{
// 				InventoryID:   arg.FromInventoryID,
// 				ReferenceType: EntryReferenceTypeTransfer,
// 				ReferenceID:   result.Transfer.ID,
// 				ProductID:     productID,
// 				AssetID:       assetID,
// 				Quantity:      -i.Quantity,
// 			})
// 			if err != nil {
// 				return err
// 			}

// 			// TODO: update inventories
// 		}
// 		return nil
// 	})

// 	return result, err
// }

type SalesInvoiceTxParams struct {
	CashBoxID    int64  `json:"cashbox_id"`
	CurrencyCode string `json:"currency_code"`
	InventoryID  int64  `json:"inventory_id"`
	ClientID     int64  `json:"client_id"`
	Amount       int64  `json:"amount"`
	Discount     int16  `json:"discount"`
	Year         int32  `json:"year"`
	// NetAmount & InvoiceCode are calculated and generated
	// automatically inside the trasaction function itself.
}

type SalesInvoiceTxResult struct {
	SalesInvoice SalesInvoice `json:"sales_invoice"`
	NetAmount    int64        `json:"net_amount"`
}

func (store *SQLStore) generateSalesInvoiceIndex(cashboxID int64) (int64, error) {
	thisYear := time.Now().Year()
	arg := NextSalesInvoiceIndexIncrementParams{
		CashboxID: cashboxID,
		Year:      int32(thisYear),
	}
	index, err := store.NextSalesInvoiceIndexIncrement(context.Background(), arg)
	if err != nil {
		return 0, err
	}
	return index, nil
}

// generate invoice number in the format:
// CashboxCode/Type of Invoice/Year/Number of Invoice this Year
// E.g. BR1/SA/2026/34 is the sales invoice number 34 in 2026 from POS Brooklyn1 that has the code BR1
func (store *SQLStore) generateInvoiceNumber(referenceType EntryReferenceType, invoiceIndex, cashboxID int64, year int32) (string, error) {
	var referenceCode string
	var err error

	// set countedInvoices and cashBox code
	cashbox, err := store.GetCashbox(context.Background(), cashboxID)
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

var txKey = struct{}{}

func (store *SQLStore) SalesInvoiceTx(ctx context.Context, arg SalesInvoiceTxParams) (SalesInvoiceTxResult, error) {
	var result SalesInvoiceTxResult

	err := store.execTx(ctx, func(q *Queries) error {
		var err error

		// txName := ctx.Value(txKey)
		invoiceIndex, err := store.generateSalesInvoiceIndex(arg.CashBoxID)
		if err != nil {
			return err
		}

		thisYear := int32(time.Now().Year())

		invoiceCode, err := store.generateInvoiceNumber(EntryReferenceTypeSalesInvoice, invoiceIndex, arg.CashBoxID, thisYear)
		if err != nil {
			return err
		}
		fmt.Println(invoiceCode)
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

		_, err = q.CreateEntryItem(ctx, CreateEntryItemParams{
			CashboxID:                  arg.CashBoxID,
			InventoryID:                arg.InventoryID,
			ReferenceType:              EntryReferenceTypeSalesInvoice,
			ReferenceID:                result.SalesInvoice.ID,
			NetAmountInDefaultCurrency: result.NetAmount,
		})
		if err != nil {
			return err
		}

		// TODO: update cashbox
		return nil
	})

	return result, err
}

type ReturnInvoiceTxParams struct {
	SalesInvoiceID int64 `json:"sales_invoice_id"`
}

type ReturnInvoiceTxResult struct {
	ReturnInvoice ReturnInvoice `json:"return_invoice"`
}

// TODO: prevent returning unless you get something instead you take something
// with the same amount of money or more instead (you have to refer to the new sales invoice that you open)
func (store *SQLStore) ReturnInvoiceTx(ctx context.Context, arg ReturnInvoiceTxParams) (ReturnInvoiceTxResult, error) {
	var result ReturnInvoiceTxResult

	err := store.execTx(ctx, func(q *Queries) error {
		var err error

		salesInvoice, err := store.GetSalesInvoice(context.Background(), arg.SalesInvoiceID)
		if err != nil {
			return err
		}

		invoiceNumber, err := store.generateInvoiceNumber(EntryReferenceTypeReturnInvoice, salesInvoice.CashboxID)
		if err != nil {
			return err
		}

		result.ReturnInvoice, err = q.CreateReturnInvoice(ctx, CreateReturnInvoiceParams{
			SalesInvoiceID: arg.SalesInvoiceID,
			InvoiceNumber:  invoiceNumber,
		})
		if err != nil {
			return err
		}
		_, err = q.CreateEntryItem(ctx, CreateEntryItemParams{
			CashboxID:                  salesInvoice.CashboxID,
			InventoryID:                salesInvoice.InventoryID,
			ReferenceType:              EntryReferenceTypeSalesInvoice,
			ReferenceID:                result.ReturnInvoice.ID,
			NetAmountInDefaultCurrency: -salesInvoice.Amount,
		})
		if err != nil {
			return err
		}

		return nil
	})

	return result, err
}
