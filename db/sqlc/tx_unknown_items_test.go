package db

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/mhdna/kashi/util"
	"github.com/stretchr/testify/require"
)

func TestResolveUnknownItemsTx_SplitsTheLineAndMovesStock(t *testing.T) {
	ctx := context.Background()
	branch := createTestBranchWithRegister(t)
	variant := createRandomVariant(t)
	_, err := testQueries.AddInventoryStockQuantity(ctx, AddInventoryStockQuantityParams{
		InventoryID: branch.Inventory.ID, VariantID: variant.ID, Quantity: 10,
	})
	require.NoError(t, err)

	sessionRef := uuid.NewString()
	session, err := testQueries.CreateUnknownItemSession(ctx, CreateUnknownItemSessionParams{
		BranchID: branch.Branch.ID, RegisterID: branch.Register.ID, ClientRef: sessionRef,
		Token: util.RandomString(24), ExpiresAt: time.Now().Add(time.Hour),
	})
	require.NoError(t, err)
	for i, ref := range []string{"a", "b"} {
		require.NoError(t, testQueries.CreateUnknownItemSlot(ctx, CreateUnknownItemSlotParams{
			SessionID: session.ID, SlotRef: ref, Position: int32(i), UnitPrice: 1000,
		}))
	}
	image, err := testQueries.CreateUnknownItemImage(ctx, CreateUnknownItemImageParams{
		SessionID: session.ID, FilePath: "x.webp", ContentType: "image/webp", SizeBytes: 1,
	})
	require.NoError(t, err)
	for _, ref := range []string{"a", "b"} {
		require.NoError(t, testQueries.SetUnknownItemSlotImage(ctx, SetUnknownItemSlotImageParams{
			SessionID: session.ID, SlotRef: ref, ImageID: sql.NullInt64{Int64: image.ID, Valid: true},
		}))
	}

	sale, err := testStore.CreateBranchInvoiceTx(ctx, CreateBranchInvoiceTxParams{
		CreateBranchInvoiceParams: CreateBranchInvoiceParams{
			BranchID: branch.Branch.ID, RegisterID: branch.Register.ID, ClientRef: uuid.NewString(),
			Kind: "sales", BranchInvoiceCode: "SA/X/2026/00001", GrandTotal: 2001, OccurredAt: time.Now(),
		},
		Items: []BranchInvoiceItemParams{{Barcode: UnknownItemBarcode, Quantity: 2, UnitPrice: 1000, LineTotal: 2001}},
	})
	require.NoError(t, err)
	require.True(t, sale.Items[0].Unknown)
	require.Equal(t, 1, sale.StockSkippedItems, "an unknown line moves no stock until it is resolved")
	require.NoError(t, testQueries.LinkUnknownItemSession(ctx, LinkUnknownItemSessionParams{
		BranchID: branch.Branch.ID, ClientRef: sessionRef,
		BranchInvoiceID: sql.NullInt64{Int64: sale.Invoice.ID, Valid: true},
	}))

	slots, err := testQueries.ListUnknownItemSlotsForSession(ctx, session.ID)
	require.NoError(t, err)

	// First unit: split off the two-unit line.
	require.NoError(t, testStore.ResolveUnknownItemsTx(ctx, []int64{slots[0].ID}, variant.ID))
	items, err := testQueries.ListBranchInvoiceItems(ctx, sale.Invoice.ID)
	require.NoError(t, err)
	require.Len(t, items, 2)
	var unknownQty, resolvedQty, total int64
	for _, item := range items {
		total += item.LineTotal
		if item.Unknown {
			unknownQty += item.Quantity
		} else {
			resolvedQty += item.Quantity
			require.True(t, item.ProductUid.Valid)
		}
	}
	require.Equal(t, int64(1), unknownQty)
	require.Equal(t, int64(1), resolvedQty)
	require.Equal(t, int64(2001), total, "the invoice total must not move")

	// Second unit: the line's last unit is relabelled in place.
	require.NoError(t, testStore.ResolveUnknownItemsTx(ctx, []int64{slots[1].ID}, variant.ID))
	items, err = testQueries.ListBranchInvoiceItems(ctx, sale.Invoice.ID)
	require.NoError(t, err)
	for _, item := range items {
		require.False(t, item.Unknown)
	}

	stock, err := testQueries.GetInventoryStock(ctx, GetInventoryStockParams{
		InventoryID: branch.Inventory.ID, VariantID: variant.ID,
	})
	require.NoError(t, err)
	require.Equal(t, int64(8), stock.Quantity)

	// Resolving a unit twice is refused.
	require.ErrorIs(t, testStore.ResolveUnknownItemsTx(ctx, []int64{slots[0].ID}, variant.ID), ErrUnknownItemNotResolvable)
}

func TestWaiveUnknownItemSlots_OnlyUnpicturedUnits(t *testing.T) {
	ctx := context.Background()
	branch := createTestBranchWithRegister(t)
	session, err := testQueries.CreateUnknownItemSession(ctx, CreateUnknownItemSessionParams{
		BranchID: branch.Branch.ID, RegisterID: branch.Register.ID, ClientRef: uuid.NewString(),
		Token: util.RandomString(24), ExpiresAt: time.Now().Add(time.Hour),
	})
	require.NoError(t, err)
	for i, ref := range []string{"a", "b"} {
		require.NoError(t, testQueries.CreateUnknownItemSlot(ctx, CreateUnknownItemSlotParams{
			SessionID: session.ID, SlotRef: ref, Position: int32(i), UnitPrice: 500,
		}))
	}
	image, err := testQueries.CreateUnknownItemImage(ctx, CreateUnknownItemImageParams{
		SessionID: session.ID, FilePath: "y.webp", ContentType: "image/webp", SizeBytes: 1,
	})
	require.NoError(t, err)
	require.NoError(t, testQueries.SetUnknownItemSlotImage(ctx, SetUnknownItemSlotImageParams{
		SessionID: session.ID, SlotRef: "a", ImageID: sql.NullInt64{Int64: image.ID, Valid: true},
	}))
	slots, err := testQueries.ListUnknownItemSlotsForSession(ctx, session.ID)
	require.NoError(t, err)

	waiting, err := testQueries.ListWaitingUnknownItems(ctx)
	require.NoError(t, err)
	var shownIDs, missingIDs []int64
	for _, w := range waiting {
		if w.SessionID == session.ID {
			shownIDs = append(shownIDs, w.ID)
			if !w.ImageID.Valid && !w.WaivedAt.Valid {
				missingIDs = append(missingIDs, w.ID)
			}
		}
	}
	require.Equal(t, []int64{slots[0].ID, slots[1].ID}, shownIDs, "the pictured unit is shown next to the one still missing")
	require.Equal(t, []int64{slots[1].ID}, missingIDs, "only the unit with no picture is missing")

	n, err := testQueries.WaiveUnknownItemSlots(ctx, WaiveUnknownItemSlotsParams{Column1: []int64{slots[0].ID, slots[1].ID}, WaivedBy: "boss"})
	require.NoError(t, err)
	require.Equal(t, int64(1), n, "a pictured unit is never waived")
}

func TestCreateBranchInvoiceTx_NamedUnknownProductMovesNoStock(t *testing.T) {
	ctx := context.Background()
	branch := createTestBranchWithRegister(t)
	variant := createRandomVariant(t)
	_, err := testQueries.AddInventoryStockQuantity(ctx, AddInventoryStockQuantityParams{
		InventoryID: branch.Inventory.ID, VariantID: variant.ID, Quantity: 5,
	})
	require.NoError(t, err)

	// The branch named a real product (variant.Barcode) as its unknown item.
	sale, err := testStore.CreateBranchInvoiceTx(ctx, CreateBranchInvoiceTxParams{
		CreateBranchInvoiceParams: CreateBranchInvoiceParams{
			BranchID: branch.Branch.ID, RegisterID: branch.Register.ID, ClientRef: uuid.NewString(),
			Kind: "sales", BranchInvoiceCode: "SA/N/2026/00001", GrandTotal: 700, OccurredAt: time.Now(),
		},
		Items: []BranchInvoiceItemParams{{Barcode: variant.Barcode, Unknown: true, Quantity: 1, UnitPrice: 700, LineTotal: 700}},
	})
	require.NoError(t, err)
	require.True(t, sale.Items[0].Unknown)
	require.False(t, sale.Items[0].ProductUid.Valid)

	stock, err := testQueries.GetInventoryStock(ctx, GetInventoryStockParams{InventoryID: branch.Inventory.ID, VariantID: variant.ID})
	require.NoError(t, err)
	require.Equal(t, int64(5), stock.Quantity, "an unknown line doesn't touch the named product's stock")
}
