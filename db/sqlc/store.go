package db

import (
	"context"
	"database/sql"
	"fmt"
)

type Store interface {
	Querier
	TransferTx(ctx context.Context, arg TransferTxParams) (TransferTxResult, error)
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

type TransferTxParams struct {
	FromInventoryID int64          `json:"from_inventory_id"`
	ToInventoryID   int64          `json:"to_inventory_id"`
	Items           []TransferItem `json:"items"`
	TransferType    string         `json:"type"`
}

type TransferItem struct {
	ID       int64
	Quantity int64
}

type TransferTxResult struct {
	Transfer      Transfer       `json:"transfer"`
	FromInventory Inventory      `json:"from_inventory"`
	ToInventory   Inventory      `json:"to_inventory"`
	Products      []TransferItem `json:"products"`
}

func (store *SQLStore) TransferTx(ctx context.Context, arg TransferTxParams) (TransferTxResult, error) {
	var result TransferTxResult

	err := store.execTx(ctx, func(q *Queries) error {
		var err error

		result.Transfer, err = q.CreateTransfer(ctx, CreateTransferParams{
			FromInventoryID: arg.FromInventoryID,
			ToInventoryID:   arg.ToInventoryID,
		})
		if err != nil {
			return err
		}
		for _, i := range arg.Items {
			_, err := q.CreateTransferProduct(ctx, CreateTransferProductParams{
				TransferID: result.Transfer.ID,
				ProductID:  i.ID,
				Quantity:   -i.Quantity,
			})
			if err != nil {
				return err
			}

			// _, err = q.CreateTransferProduct(ctx, CreateTransferProductParams{
			// 	TransferID: result.Transfer.ID,
			// 	ProductID:  i.ID,
			// 	Quantity:   i.Quantity,
			// })
			// if err != nil {
			// 	return err
			// }
		}
		return nil
	})

	return result, err
}
