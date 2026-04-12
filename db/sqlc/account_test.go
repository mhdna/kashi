package db

import (
	"context"
	"testing"

	"github.com/mhdna/kashi/util"
	"github.com/stretchr/testify/require"
)

func createRandomAccount(t *testing.T) CashboxAccount {
	shift := createRandomShift(t)
	currency := createRandomCurrency(t)
	balance := util.RandomAmount()

	arg := CreateCashboxAccountParams{
		Type:           util.RandomName(),
		ShiftID:        shift.ID,
		CurrencyCode:   currency.Code,
		OpeningBalance: balance,
		Balance:        balance,
	}

	account, err := testQueries.CreateCashboxAccount(context.Background(), arg)
	require.NoError(t, err)
	require.Equal(t, account.Type, arg.Type)
	require.Equal(t, account.ShiftID, arg.ShiftID)
	require.Equal(t, account.CurrencyCode, arg.CurrencyCode)
	require.Equal(t, account.OpeningBalance, arg.OpeningBalance)
	require.Equal(t, account.Balance, arg.Balance)

	return account
}

func TestCreateAccount(t *testing.T) {
	createRandomAccount(t)
}

func TestGetAccount(t *testing.T) {
	account1 := createRandomAccount(t)

	account2, err := testQueries.GetCashboxAccount(context.Background(), account1.ID)
	require.NoError(t, err)
	require.Equal(t, account1.ID, account2.ID)
	require.Equal(t, account1.Type, account2.Type)
	require.Equal(t, account1.ShiftID, account2.ShiftID)
	require.Equal(t, account1.CurrencyCode, account2.CurrencyCode)
	require.Equal(t, account1.OpeningBalance, account2.OpeningBalance)
	require.Equal(t, account1.Balance, account2.Balance)
}

func TestListAccounts(t *testing.T) {
	for range 10 {
		createRandomAccount(t)
	}

	arg := ListAccountsParams{
		Limit:  5,
		Offset: 5,
	}

	accounts, err := testQueries.ListAccounts(context.Background(), arg)
	require.NoError(t, err)
	require.Len(t, accounts, 5)
	for _, account := range accounts {
		// FIXME
		require.NotEmpty(t, account)
	}
}

func TestAddAccountBalance(t *testing.T) {
	account := createRandomAccount(t)

	arg := AddAccountBalanceParams{
		ID:      account.ID,
		Balance: util.RandomAmount(),
	}

	account2, err := testQueries.AddAccountBalance(context.Background(), arg)
	require.NoError(t, err)
	require.Equal(t, account2.Balance, account.Balance+arg.Balance)
}
