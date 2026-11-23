package db

import (
	"context"
	"database/sql"
	"math/rand/v2"
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

// unknownSaleForResolve stores a sale of one unknown line, qty units at
// unitPrice and discount percent, with the given cents of deal and invoice
// discount taken off it, and a pictured slot per unit, ready to resolve.
func unknownSaleForResolve(t *testing.T, branch CreateBranchTxResult, unitPrice int64, discount int16, qty, dealDiscount, invoiceDiscount int64) (int64, []UnknownItemSlot) {
	t.Helper()
	ctx := context.Background()
	sessionRef := uuid.NewString()
	session, err := testQueries.CreateUnknownItemSession(ctx, CreateUnknownItemSessionParams{
		BranchID: branch.Branch.ID, RegisterID: branch.Register.ID, ClientRef: sessionRef,
		Token: util.RandomString(24), ExpiresAt: time.Now().Add(time.Hour),
	})
	require.NoError(t, err)
	for i := range qty {
		require.NoError(t, testQueries.CreateUnknownItemSlot(ctx, CreateUnknownItemSlotParams{
			SessionID: session.ID, SlotRef: uuid.NewString(), Position: int32(i), UnitPrice: unitPrice, Discount: discount,
		}))
	}
	gross := unknownLineGross(unitPrice, discount, qty)
	lineTotal := gross - dealDiscount - invoiceDiscount
	sale, err := testStore.CreateBranchInvoiceTx(ctx, CreateBranchInvoiceTxParams{
		CreateBranchInvoiceParams: CreateBranchInvoiceParams{
			BranchID: branch.Branch.ID, RegisterID: branch.Register.ID, ClientRef: uuid.NewString(),
			Kind: "sales", BranchInvoiceCode: "SA/X/2026/" + util.RandomString(6), GrandTotal: lineTotal,
			OccurredAt: time.Now(),
		},
		Items: []BranchInvoiceItemParams{{
			Barcode: UnknownItemBarcode, Quantity: qty, UnitPrice: unitPrice, Discount: discount,
			LineTotal: lineTotal, Deal: dealName(dealDiscount), DealDiscount: dealDiscount, InvoiceDiscount: invoiceDiscount,
		}},
	})
	require.NoError(t, err)
	require.NoError(t, testQueries.LinkUnknownItemSession(ctx, LinkUnknownItemSessionParams{
		BranchID: branch.Branch.ID, ClientRef: sessionRef,
		BranchInvoiceID: sql.NullInt64{Int64: sale.Invoice.ID, Valid: true},
	}))
	slots, err := testQueries.ListUnknownItemSlotsForSession(ctx, session.ID)
	require.NoError(t, err)
	return sale.Invoice.ID, slots
}

func dealName(dealDiscount int64) string {
	if dealDiscount == 0 {
		return ""
	}
	return "2 for 1"
}

// unknownLineGross is a line's price after its percent discount, rounded as
// the till rounds it.
func unknownLineGross(unitPrice int64, discount int16, qty int64) int64 {
	return (unitPrice*qty*int64(100-discount) + 50) / 100
}

// requireSplitLinesAddUp checks what a split must keep: the lines' totals and
// discounts add up to the line that was sold, and each line still reads
// price - its discounts = its total, give or take the cent per unit that
// rounding a percent discount per line instead of per unit can shift.
func requireSplitLinesAddUp(t *testing.T, invoiceID, unitPrice int64, discount int16, qty, dealDiscount, invoiceDiscount int64) {
	t.Helper()
	items, err := testQueries.ListBranchInvoiceItems(context.Background(), invoiceID)
	require.NoError(t, err)
	var units, lines, deals, shares int64
	for _, it := range items {
		units += it.Quantity
		lines += it.LineTotal
		deals += it.DealDiscount
		shares += it.InvoiceDiscount
		gap := unknownLineGross(it.UnitPrice, it.Discount, it.Quantity) - it.DealDiscount - it.InvoiceDiscount - it.LineTotal
		require.LessOrEqual(t, max(gap, -gap), qty, "line %d: price %d x %d, deal %d, invoice discount %d, total %d",
			it.ID, it.UnitPrice, it.Quantity, it.DealDiscount, it.InvoiceDiscount, it.LineTotal)
		require.GreaterOrEqual(t, it.InvoiceDiscount, int64(0))
		require.GreaterOrEqual(t, it.DealDiscount, int64(0))
		require.Equal(t, it.DealDiscount != 0, it.Deal != "", "a line with a deal saving names the deal")
	}
	gross := unknownLineGross(unitPrice, discount, qty)
	require.Equal(t, qty, units)
	require.Equal(t, gross-dealDiscount-invoiceDiscount, lines, "the invoice total must not move")
	require.Equal(t, dealDiscount, deals, "the deal saving must not move")
	require.Equal(t, invoiceDiscount, shares, "the lines' invoice discount shares must still add up to the sale's")
}

// SA/CT/2026/03427: two unknown $35 units with $33.43 of the Independence
// Day deal on them. Matching one unit left all $33.43 on the other line and
// none on the new one, so one line read $35 - $33.43 = $18.29.
func TestResolveUnknownItemsTx_SplitCarriesTheInvoiceDiscountShare(t *testing.T) {
	branch := createTestBranchWithRegister(t)
	variant := createRandomVariant(t)
	invoiceID, slots := unknownSaleForResolve(t, branch, 3500, 0, 2, 0, 3343)

	require.NoError(t, testStore.ResolveUnknownItemsTx(context.Background(), []int64{slots[0].ID}, variant.ID))
	items, err := testQueries.ListBranchInvoiceItems(context.Background(), invoiceID)
	require.NoError(t, err)
	require.Len(t, items, 2)
	for _, it := range items {
		require.Equal(t, int64(3500), it.LineTotal+it.InvoiceDiscount, "line %d", it.ID)
	}
	require.InDelta(t, items[0].InvoiceDiscount, items[1].InvoiceDiscount, 1, "each unit carries half the deal")
	requireSplitLinesAddUp(t, invoiceID, 3500, 0, 2, 0, 3343)

	require.NoError(t, testStore.ResolveUnknownItemsTx(context.Background(), []int64{slots[1].ID}, variant.ID))
	requireSplitLinesAddUp(t, invoiceID, 3500, 0, 2, 0, 3343)
}

// Any unknown line, any discounts, resolved unit by unit in any order: the
// lines keep adding up after every step.
func TestResolveUnknownItemsTx_RandomSplitsAddUp(t *testing.T) {
	branch := createTestBranchWithRegister(t)
	variant := createRandomVariant(t)
	_, err := testQueries.AddInventoryStockQuantity(context.Background(), AddInventoryStockQuantityParams{
		InventoryID: branch.Inventory.ID, VariantID: variant.ID, Quantity: 100000,
	})
	require.NoError(t, err)
	r := rand.New(rand.NewPCG(7, 11))
	for range 300 {
		unitPrice := 50 + r.Int64N(10000)
		discount := []int16{0, 0, 10, 20, 50}[r.IntN(5)]
		qty := 2 + r.Int64N(6)
		gross := unknownLineGross(unitPrice, discount, qty)
		var dealDiscount int64
		if r.IntN(3) == 0 {
			dealDiscount = r.Int64N(gross / 2)
		}
		invoiceDiscount := r.Int64N(gross - dealDiscount + 1)
		invoiceID, slots := unknownSaleForResolve(t, branch, unitPrice, discount, qty, dealDiscount, invoiceDiscount)
		r.Shuffle(len(slots), func(i, j int) { slots[i], slots[j] = slots[j], slots[i] })
		for _, s := range slots {
			require.NoError(t, testStore.ResolveUnknownItemsTx(context.Background(), []int64{s.ID}, variant.ID))
			requireSplitLinesAddUp(t, invoiceID, unitPrice, discount, qty, dealDiscount, invoiceDiscount)
		}
	}
}
