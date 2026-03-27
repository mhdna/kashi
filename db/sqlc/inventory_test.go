package db

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/mhdna/kashi/util"
	"github.com/stretchr/testify/require"
)

func createRandomInventory(t *testing.T) Inventory {
	arg := CreateInventoryParams{
		Name:      util.RandomInventoryName(),
		Code:      util.RandomInventoryCode(),
		Latitude:  util.RandomLongitudeLatitude(),
		Longitude: util.RandomLongitudeLatitude(),
	}

	inventory, err := testQueries.CreateInventory(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, inventory)
	require.Equal(t, arg.Name, inventory.Name)
	require.Equal(t, arg.Code, inventory.Code)
	require.Equal(t, arg.Latitude, inventory.Latitude)
	require.Equal(t, arg.Longitude, inventory.Longitude)

	require.NotZero(t, inventory.ID)
	require.NotZero(t, inventory.CreatedAt)

	return inventory
}

func TestCreateInventory(t *testing.T) {
	createRandomInventory(t)
}

func TestGetInventory(t *testing.T) {
	inventory1 := createRandomInventory(t)
	inventory2, err := testQueries.GetInventory(context.Background(), inventory1.ID)
	require.NoError(t, err)
	require.NotEmpty(t, inventory2)

	require.Equal(t, inventory1.ID, inventory2.ID)
	require.Equal(t, inventory1.Code, inventory2.Code)
	require.Equal(t, inventory1.Latitude, inventory2.Latitude)
	require.Equal(t, inventory1.Longitude, inventory2.Longitude)
	require.WithinDuration(t, inventory1.CreatedAt, inventory2.CreatedAt, time.Second)
}

func TestDeleteInventory(t *testing.T) {
	inventory1 := createRandomInventory(t)
	err := testQueries.DeleteInventory(context.Background(), inventory1.ID)

	require.NoError(t, err)
	inventory2, err := testQueries.GetInventory(context.Background(), inventory1.ID)
	require.Error(t, err)
	require.EqualError(t, err, sql.ErrNoRows.Error())
	require.Empty(t, inventory2)
}

func TestListInventories(t *testing.T) {
	for range 10 {
		createRandomInventory(t)
	}
	arg := ListInventoriesParams{
		Limit:  5,
		Offset: 5,
	}

	inventories, err := testQueries.ListInventories(context.Background(), arg)
	require.NoError(t, err)
	require.Len(t, inventories, 5)
	for _, inventory := range inventories {
		require.NotEmpty(t, inventory)
	}
}

func TestUpdateInventory(t *testing.T) {
	inventory1 := createRandomInventory(t)
	arg := UpdateInventoryParams{
		ID:   inventory1.ID,
		Name: util.RandomInventoryName(),
	}

	err := testQueries.UpdateInventory(context.Background(), arg)
	require.NoError(t, err)

	inventory2, err := testQueries.GetInventory(context.Background(), inventory1.ID)

	require.Equal(t, inventory2.ID, arg.ID)
	require.Equal(t, inventory2.Name, arg.Name)
}
