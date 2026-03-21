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
		Name: util.RandomInventory(),
		Code: util.RandomInventoryCode(),
		Latitude: sql.NullFloat64{
			Float64: float64(util.RandomLongitudeLatitude()),
			Valid:   true,
		},
		Longitude: sql.NullFloat64{
			Float64: 40.40,
			Valid:   true,
		},
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
