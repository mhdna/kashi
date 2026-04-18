package db

import (
	"context"
	"testing"
	"time"

	"github.com/mhdna/kashi/util"
	"github.com/stretchr/testify/require"
)

func createRandomProduct(t *testing.T) Product {
	arg := CreateProductParams{
		Name:        util.RandomString(20),
		Code:        util.RandomString(8),
		Description: util.RandomString(200),
		Price:       util.RandomNumber(),
		Discount:    util.RandomDiscount(),
	}

	product, err := testQueries.CreateProduct(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, product)
	require.Equal(t, arg.Name, product.Name)
	require.Equal(t, arg.Code, product.Code)
	require.Equal(t, arg.Description, product.Description)
	require.Equal(t, arg.Price, product.Price)

	require.NotZero(t, product.ID)
	require.NotZero(t, product.CreatedAt)

	return product
}

func TestCreateProduct(t *testing.T) {
	createRandomProduct(t)
}

func TestGetProduct(t *testing.T) {
	product1 := createRandomProduct(t)
	product2, err := testQueries.GetProduct(context.Background(), product1.ID)

	require.NoError(t, err)
	require.NotEmpty(t, product2)

	require.Equal(t, product1.Name, product2.Name)
	require.Equal(t, product1.Code, product2.Code)
	require.Equal(t, product1.Description, product2.Description)
	require.Equal(t, product1.Price, product2.Price)
	require.WithinDuration(t, product1.CreatedAt, product2.CreatedAt, time.Second)
}
