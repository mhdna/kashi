package db

import (
	"context"
	"testing"
	"time"

	"github.com/mhdna/kashi/util"
	"github.com/stretchr/testify/require"
)

func createRandomEntry(t *testing.T) Entry {
	inventory := createRandomInventory(t)
	purchase := createRandomPurchase(t)
	cashbox := createRandomCashbox(t)
	arg := CreateEntryItemParams{
		CashboxID:     cashbox.ID,
		InventoryID:   inventory.ID,
		ReferenceType: EntryReferenceTypePurchase,
		ReferenceID:   purchase.ID,
		NetAmount:     util.RandomMoneyAmount(),
	}
	entry, err := testQueries.CreateEntryItem(context.Background(), arg)
	require.NoError(t, err)
	require.Equal(t, entry.CashboxID, arg.CashboxID)
	require.Equal(t, entry.InventoryID, arg.InventoryID)
	require.Equal(t, entry.ReferenceType, arg.ReferenceType)
	require.Equal(t, entry.ReferenceID, arg.ReferenceID)
	require.Equal(t, entry.NetAmount, arg.NetAmount)

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
	require.Equal(t, entry1.CashboxID, entry2.CashboxID)
	require.Equal(t, entry1.InventoryID, entry2.InventoryID)
	require.Equal(t, entry1.ReferenceID, entry2.ReferenceID)
	require.Equal(t, entry1.ReferenceType, entry2.ReferenceType)
	require.Equal(t, entry1.NetAmount, entry2.NetAmount)

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
