package db

import (
	"context"
	"testing"

	"github.com/mhdna/kashi/util"
	"github.com/stretchr/testify/require"
)

func createRandomCashbox(t *testing.T) Cashbox {
	arg := CreateCashboxParams{
		Code: util.RandomCode(),
		Name: util.RandomName(),
	}

	cashbox, err := testQueries.CreateCashbox(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, cashbox)
	require.Equal(t, cashbox.Code, arg.Code)
	require.Equal(t, cashbox.Name, arg.Name)
	require.Equal(t, cashbox.IsActive, true)

	return cashbox
}

func TestGetCashbox(t *testing.T) {
	cashbox1 := createRandomCashbox(t)

	cashbox2, err := testQueries.GetCashbox(context.Background(), cashbox1.ID)

	require.NoError(t, err)
	require.Equal(t, cashbox1.ID, cashbox2.ID)
	require.Equal(t, cashbox1.Code, cashbox2.Code)
	require.Equal(t, cashbox1.Name, cashbox2.Name)
	require.Equal(t, cashbox1.IsActive, cashbox2.IsActive)
}

func TestCreateCashbox(t *testing.T) {
	createRandomAttributeValue(t)
}

func TestListCashboxes(t *testing.T) {
	for range 10 {
		createRandomCashbox(t)
	}

	limit := 5
	offset := 5

	arg := ListCashboxesParams{
		Limit:  int32(limit),
		Offset: int32(offset),
	}

	cashboxes, err := testQueries.ListCashboxes(context.Background(), arg)
	require.NoError(t, err)
	require.Len(t, cashboxes, limit)
	for _, cashbox := range cashboxes {
		// TODO: complete this
		require.NotEmpty(t, cashbox)
	}
}
