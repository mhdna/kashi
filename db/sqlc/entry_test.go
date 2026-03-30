package db

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/mhdna/kashi/util"
	"github.com/stretchr/testify/require"
)

func createRandomEntry(t *testing.T) Entry {
	inventory := createRandomInventory(t)
	purchase := createRandomPurchase(t)
	product := createRandomProduct(t)
	arg := CreateEntryItemParams{
		InventoryID:   inventory.ID,
		ReferenceType: EntryReferenceTypePurchase,
		ReferenceID:   purchase.ID,
		ProductID:     sql.NullInt64{Int64: product.ID, Valid: true},
		Quantity:      util.RandomInt(1, 50),
	}
	entry, err := testQueries.CreateEntryItem(context.Background(), arg)
	require.NoError(t, err)
	require.Equal(t, entry.InventoryID, arg.InventoryID)
	require.Equal(t, entry.ReferenceType, arg.ReferenceType)
	require.Equal(t, entry.ReferenceID, arg.ReferenceID)
	require.Equal(t, entry.ProductID, arg.ProductID)
	require.Equal(t, entry.Quantity, arg.Quantity)

	return entry
}

func TestCreateEntry(t *testing.T) {
	createRandomEntry(t)
}

func TestGetEntry(t *testing.T) {
	entry1 := createRandomEntry(t)

	entry2, err := testQueries.GetEntry(context.Background(), entry1.ID)
	require.NoError(t, err)
	require.Equal(t, entry1.ID, entry2.ID)
	require.Equal(t, entry1.InventoryID, entry2.InventoryID)
	require.Equal(t, entry1.AssetID, entry2.AssetID)
	require.Equal(t, entry1.ProductID, entry2.ProductID)
	require.Equal(t, entry1.ReferenceID, entry2.ReferenceID)
	require.Equal(t, entry1.ReferenceType, entry2.ReferenceType)
	require.Equal(t, entry1.Quantity, entry2.Quantity)
	require.WithinDuration(t, entry1.CreatedAt, entry2.CreatedAt, time.Second)
}

func TestListEntries(t *testing.T) {
	for range 10 {
		createRandomEntry(t)
	}

	arg := ListEntriesParams{
		Limit:  5,
		Offset: 5,
	}

	entries, err := testQueries.ListEntries(context.Background(), arg)
	require.NoError(t, err)
	for entry := range entries {
		require.NotEmpty(t, entry)
	}
}
