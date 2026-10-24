package db

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/mhdna/kashi/util"
	"github.com/stretchr/testify/require"
)

func createTestBranchWithRegister(t *testing.T) CreateBranchTxResult {
	code := util.RandomString(6)
	result, err := testStore.CreateBranchTx(context.Background(), CreateBranchTxParams{
		Name:               util.RandomString(10),
		Code:               code,
		RegisterAPIKeyHash: util.RandomString(20),
	})
	require.NoError(t, err)
	require.Equal(t, code, result.Register.Code, "the first register takes the branch code")
	return result
}

func sellAtBranch(t *testing.T, branch CreateBranchTxResult, variant ProductVariant, quantity int64) CreateBranchInvoiceTxResult {
	result, err := testStore.CreateBranchInvoiceTx(context.Background(), CreateBranchInvoiceTxParams{
		CreateBranchInvoiceParams: CreateBranchInvoiceParams{
			BranchID:          branch.Branch.ID,
			RegisterID:        branch.Register.ID,
			ClientRef:         uuid.NewString(),
			Kind:              "sales",
			BranchInvoiceCode: "SA/" + branch.Register.Code + "/2026/00001",
			GrandTotal:        1000,
			OccurredAt:        time.Now(),
		},
		Items: []BranchInvoiceItemParams{{Barcode: variant.Barcode, Quantity: quantity, UnitPrice: 1000, LineTotal: 1000}},
	})
	require.NoError(t, err)
	return result
}

func TestCreateBranchInvoiceTx_OversellRaisesAndReopensAlert(t *testing.T) {
	branch := createTestBranchWithRegister(t)
	variant := createRandomVariant(t)

	// Nothing on hand: the sale still lands, and stock goes negative.
	first := sellAtBranch(t, branch, variant, 3)
	require.Len(t, first.Oversold, 1)
	alert := first.Oversold[0]
	require.Equal(t, "stock", alert.Entity)
	require.Equal(t, "oversold", alert.Kind)
	require.Equal(t, variant.Uid.String(), alert.Ref)
	require.Equal(t, "open", alert.Status)

	var payload OversoldPayload
	require.NoError(t, json.Unmarshal(alert.BranchPayload, &payload))
	require.Equal(t, int64(-3), payload.OnHand)
	require.Equal(t, branch.Register.ID, payload.RegisterID)

	// Selling more of it updates the same alert rather than adding another.
	second := sellAtBranch(t, branch, variant, 1)
	require.Len(t, second.Oversold, 1)
	require.Equal(t, alert.ID, second.Oversold[0].ID)
	require.NoError(t, json.Unmarshal(second.Oversold[0].BranchPayload, &payload))
	require.Equal(t, int64(-4), payload.OnHand)

	// Once an admin resolves it, going negative again reopens it.
	_, err := testQueries.ResolveBranchSyncConflict(context.Background(), ResolveBranchSyncConflictParams{
		ID:     alert.ID,
		Status: "resolved",
	})
	require.NoError(t, err)
	third := sellAtBranch(t, branch, variant, 1)
	require.Len(t, third.Oversold, 1)
	require.Equal(t, alert.ID, third.Oversold[0].ID)
	require.Equal(t, "open", third.Oversold[0].Status)
}

func TestCreateBranchInvoiceTx_NoAlertWhileStockLasts(t *testing.T) {
	branch := createTestBranchWithRegister(t)
	variant := createRandomVariant(t)

	_, err := testQueries.AddInventoryStockQuantity(context.Background(), AddInventoryStockQuantityParams{
		InventoryID: branch.Inventory.ID,
		VariantID:   variant.ID,
		Quantity:    5,
	})
	require.NoError(t, err)

	result := sellAtBranch(t, branch, variant, 5)
	require.Empty(t, result.Oversold)
}

func TestBranchShifts_SameLocalRefFromTwoRegisters(t *testing.T) {
	branch := createTestBranchWithRegister(t)
	second, err := testQueries.CreateBranchRegister(context.Background(), CreateBranchRegisterParams{
		BranchID:   branch.Branch.ID,
		Code:       util.RandomString(6),
		ApiKeyHash: util.RandomString(20),
	})
	require.NoError(t, err)

	for _, registerID := range []int64{branch.Register.ID, second.ID} {
		_, err := testQueries.CreateBranchShift(context.Background(), CreateBranchShiftParams{
			BranchID:      branch.Branch.ID,
			RegisterID:    registerID,
			ClientRef:     "shift-5",
			BranchShiftID: 5,
			OccurredAt:    time.Now(),
		})
		require.NoError(t, err, "each register has its own shift 5")
	}
}

// A line the till priced with a deal keeps the deal's name and saving.
func TestCreateBranchInvoiceTx_KeepsDealLines(t *testing.T) {
	branch := createTestBranchWithRegister(t)
	variant := createRandomVariant(t)
	result, err := testStore.CreateBranchInvoiceTx(context.Background(), CreateBranchInvoiceTxParams{
		CreateBranchInvoiceParams: CreateBranchInvoiceParams{
			BranchID: branch.Branch.ID, RegisterID: branch.Register.ID, ClientRef: uuid.NewString(),
			Kind: "sales", BranchInvoiceCode: "SA/" + branch.Register.Code + "/2026/00002",
			GrandTotal: 2000, OccurredAt: time.Now(),
		},
		Items: []BranchInvoiceItemParams{
			{Barcode: variant.Barcode, Quantity: 2, UnitPrice: 1000, LineTotal: 2000, Deal: "Buy 2 get 1"},
			{Barcode: variant.Barcode, Quantity: 1, UnitPrice: 1000, LineTotal: 0, Deal: "Buy 2 get 1", DealDiscount: 1000},
		},
	})
	require.NoError(t, err)
	items, err := testQueries.ListBranchInvoiceItems(context.Background(), result.Invoice.ID)
	require.NoError(t, err)
	require.Len(t, items, 2)
	require.Equal(t, "Buy 2 get 1", items[1].Deal)
	require.Equal(t, int64(1000), items[1].DealDiscount)
	require.Equal(t, int64(0), items[1].LineTotal)
}
