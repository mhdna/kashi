package db

import (
	"context"
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCreateInventory(t *testing.T) {
	arg := CreateInventoryParams{
		Name: "Inventory 1",
		Code: "IN",
		Latitude: sql.NullFloat64{
			Float64: 30.30,
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
}
