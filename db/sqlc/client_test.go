package db

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/mhdna/kashi/util"
	"github.com/stretchr/testify/require"
)

// TODO: add per client discounts

func createRandomClient(t *testing.T) Client {
	arg := CreateClientParams{
		Name:  util.RandomName(),
		Phone: util.RandomPhone(),
	}

	client, err := testQueries.CreateClient(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, client)
	require.Equal(t, arg.Name, client.Name)
	require.Equal(t, arg.Phone, client.Phone)
	require.Equal(t, client.LoyaltyPoints, int64(0))

	require.NotZero(t, client.ID)
	require.NotZero(t, client.CreatedAt)

	return client
}

func TestCreateClient(t *testing.T) {
	createRandomClient(t)
}

func TestGetClient(t *testing.T) {
	client1 := createRandomClient(t)
	client2, err := testQueries.GetClient(context.Background(), client1.ID)
	require.NoError(t, err)
	require.NotEmpty(t, client2)

	require.Equal(t, client1.ID, client2.ID)
	require.Equal(t, client1.Name, client2.Name)
	require.Equal(t, client1.Phone, client2.Phone)
	require.Equal(t, client1.LoyaltyPoints, client2.LoyaltyPoints)
	require.WithinDuration(t, client1.CreatedAt, client2.CreatedAt, time.Second)
}

func TestListClients(t *testing.T) {
	for range 10 {
		createRandomClient(t)
	}
	arg := ListClientsParams{
		Limit:  5,
		Offset: 5,
	}

	clients, err := testQueries.ListClients(context.Background(), arg)
	require.NoError(t, err)
	require.Len(t, clients, 5)
	for _, client := range clients {
		require.NotEmpty(t, client)
	}
}

func TestUpdateClient(t *testing.T) {
	client := createRandomClient(t)

	arg := UpdateClientParams{
		ID:            client.ID,
		Name:          util.RandomName(),
		Phone:         util.RandomPhone(),
		LoyaltyPoints: util.RandomInt(0, 100),
	}
	err := testQueries.UpdateClient(context.Background(), arg)
	require.NoError(t, err)

	client2, err := testQueries.GetClient(context.Background(), client.ID)
	require.Equal(t, arg.ID, client2.ID)
	require.Equal(t, arg.Name, client2.Name)
	require.Equal(t, arg.Phone, client2.Phone)
	require.Equal(t, arg.LoyaltyPoints, client2.LoyaltyPoints)
}

func TestDeleteClient(t *testing.T) {
	client := createRandomClient(t)

	err := testQueries.DeleteClient(context.Background(), client.ID)
	require.NoError(t, err)

	_, err = testQueries.GetClient(context.Background(), client.ID)
	require.Error(t, err)
	require.EqualError(t, err, sql.ErrNoRows.Error())
}
