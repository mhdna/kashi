package db

import (
	"context"
	"fmt"
)

type CreateBranchTxParams struct {
	Name string `json:"name"`
	Code string `json:"code"`
	// The branch's first register takes the branch code, so a single-till
	// branch keeps logging in with the code it was created with.
	RegisterAPIKeyHash string `json:"register_api_key_hash"`
}

type CreateBranchTxResult struct {
	Branch    Branch         `json:"branch"`
	Register  BranchRegister `json:"register"`
	Inventory Inventory      `json:"inventory"`
}

// CreateBranchTx creates a branch together with its first register and its
// own store inventory, so branch (POS) sales can decrement central stock for
// that location from day one.
func (store *SQLStore) CreateBranchTx(ctx context.Context, arg CreateBranchTxParams) (CreateBranchTxResult, error) {
	var result CreateBranchTxResult

	err := store.execTx(ctx, func(q *Queries) error {
		branch, err := q.CreateBranch(ctx, CreateBranchParams{
			Name: arg.Name,
			Code: arg.Code,
		})
		if err != nil {
			return err
		}

		register, err := q.CreateBranchRegister(ctx, CreateBranchRegisterParams{
			BranchID:   branch.ID,
			Code:       arg.Code,
			ApiKeyHash: arg.RegisterAPIKeyHash,
		})
		if err != nil {
			return err
		}

		inventory, err := q.CreateInventory(ctx, CreateInventoryParams{
			Name: fmt.Sprintf("Branch: %s", arg.Name),
			Type: InventoryTypeStore,
			Code: fmt.Sprintf("BRANCH-%s", arg.Code),
		})
		if err != nil {
			return err
		}

		if err := q.SetBranchInventory(ctx, SetBranchInventoryParams{
			ID:          branch.ID,
			InventoryID: nullInt64(inventory.ID),
		}); err != nil {
			return err
		}
		branch.InventoryID = nullInt64(inventory.ID)

		result.Branch = branch
		result.Register = register
		result.Inventory = inventory
		return nil
	})

	return result, err
}
