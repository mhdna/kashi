package db

import (
	"context"
	"database/sql"
	"fmt"
)

type Store struct {
	*Queries
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{
		db:      db,
		Queries: New(db),
	}
}

func (store *Store) execTx(ctx context.Context, fn func(*Queries) error) error {
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

type PTransferTxParams struct {
	FromInventoryID int64           `json:"from_inventory_id"`
	ToInventoryID   int64           `json:"to_inventory_id"`
	Products        []PTransferItem `json:"products"`
}

type PTransferItem struct {
	ProductId int64
	Quantity  int64
}

type PTransferTxResult struct {
	PTransfer     Ptransfer       `json:"ptransfer"`
	FromInventory Inventory       `json:"from_inventory"`
	ToInventory   Inventory       `json:"to_inventory"`
	Products      []PTransferItem `json:"products"`
}

func (store *Store) PTransferTx(ctx context.Context, arg PTransferTxParams) (PTransferTxResult, error) {
	var result PTransferTxResult

	err := store.execTx(ctx, func(q *Queries) error {
		var err error

		result.PTransfer, err = q.CreatePTransfer(ctx, CreatePTransferParams{
			FromInventoryID: arg.FromInventoryID,
			ToInventoryID:   arg.ToInventoryID,
		})
		if err != nil {
			return err
		}
		for _, p := range arg.Products {
			_, err := q.CreatePTransferProduct(ctx, CreatePTransferProductParams{
				TransferID: arg.FromInventoryID,
				ProductID:  p.ProductId,
				Quantity:   -p.Quantity,
			})
			if err != nil {
				return err
			}

			_, err = q.CreatePTransferProduct(ctx, CreatePTransferProductParams{
				TransferID: arg.ToInventoryID,
				ProductID:  p.ProductId,
				Quantity:   p.Quantity,
			})
			if err != nil {
				return err
			}
		}
		return nil
	})

	return result, err
}

type ATransferTxParams struct {
	FromInventoryID int64 `json:"from_inventory_id"`
	ToInventoryID   int64 `json:"to_inventory_id"`
	Assets          []ATransferItem
}

type ATransferItem struct {
	AssetId  int64
	Quantity int64
}

type ATransferTxResult struct {
	ATransfer     Ptransfer `json:"atransfer"`
	FromInventory Inventory `json:"from_inventory"`
	ToInventory   Inventory `json:"to_inventory"`
}
