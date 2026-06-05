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
		Name:      util.RandomName(),
		Code:      util.RandomCode(),
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
		Name: util.RandomName(),
	}

	err := testQueries.UpdateInventory(context.Background(), arg)
	require.NoError(t, err)

	inventory2, err := testQueries.GetInventory(context.Background(), inventory1.ID)

	require.Equal(t, inventory2.ID, arg.ID)
	require.Equal(t, inventory2.Name, arg.Name)
}

func TestAddInventoryProduct(t *testing.T) {
	inventory := createRandomInventory(t)
	product := createRandomProduct(t)
	quantity := util.RandomQuantity()

	arg := AddInventoryProductParams{
		InventoryID: inventory.ID,
		ProductID:   product.ID,
		Quantity:    quantity,
	}

	invProduct, err := testQueries.AddInventoryProduct(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, invProduct)
	require.Equal(t, arg.InventoryID, invProduct.InventoryID)
	require.Equal(t, arg.ProductID, invProduct.ProductID)
	require.Equal(t, arg.Quantity, invProduct.Quantity)
}

func TestAddInventoryProductQuantity(t *testing.T) {
	inventory := createRandomInventory(t)
	product := createRandomProduct(t)
	initialQuantity := util.RandomQuantity()

	_, err := testQueries.AddInventoryProduct(context.Background(), AddInventoryProductParams{
		InventoryID: inventory.ID,
		ProductID:   product.ID,
		Quantity:    initialQuantity,
	})
	require.NoError(t, err)

	additionalQuantity := util.RandomQuantity()
	err = testQueries.AddInventoryProductQuantity(context.Background(), AddInventoryProductQuantityParams{
		InventoryID: inventory.ID,
		ProductID:   product.ID,
		Quantity:    additionalQuantity,
	})
	require.NoError(t, err)

	products, err := testQueries.ListInventoryProducts(context.Background(), inventory.ID)
	require.NoError(t, err)
	require.Len(t, products, 1)
	require.Equal(t, initialQuantity+additionalQuantity, products[0].Quantity)
}

func TestDeleteInventoryProduct(t *testing.T) {
	inventory := createRandomInventory(t)
	product := createRandomProduct(t)

	_, err := testQueries.AddInventoryProduct(context.Background(), AddInventoryProductParams{
		InventoryID: inventory.ID,
		ProductID:   product.ID,
		Quantity:    util.RandomQuantity(),
	})
	require.NoError(t, err)

	err = testQueries.DeleteInventoryProduct(context.Background(), DeleteInventoryProductParams{
		InventoryID: inventory.ID,
		ProductID:   product.ID,
	})
	require.NoError(t, err)

	products, err := testQueries.ListInventoryProducts(context.Background(), inventory.ID)
	require.NoError(t, err)
	require.Len(t, products, 0)
}

func TestListInventoryProducts(t *testing.T) {
	inventory := createRandomInventory(t)

	n := 5
	for range n {
		product := createRandomProduct(t)
		_, err := testQueries.AddInventoryProduct(context.Background(), AddInventoryProductParams{
			InventoryID: inventory.ID,
			ProductID:   product.ID,
			Quantity:    util.RandomQuantity(),
		})
		require.NoError(t, err)
	}

	products, err := testQueries.ListInventoryProducts(context.Background(), inventory.ID)
	require.NoError(t, err)
	require.Len(t, products, n)
	for _, p := range products {
		require.NotEmpty(t, p.ProductID)
		require.NotEmpty(t, p.Name)
		require.Greater(t, p.Quantity, int64(0))
	}
}
