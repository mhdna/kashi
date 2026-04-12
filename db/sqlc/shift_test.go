package db

import (
	"context"
	"database/sql"
	"log"
	"testing"
	"time"

	"github.com/mhdna/kashi/util"
	"github.com/stretchr/testify/require"
)

func createRandomShift(t *testing.T) Shift {
	cashbox := createRandomCashbox(t)

	arg := CreateShiftParams{
		CashboxID:           cashbox.ID,
		TotalOpeningBalance: util.RandomAmount(),
		TotalBalance:        util.RandomAmount(),
	}

	shift, err := testQueries.CreateShift(context.Background(), arg)
	require.NoError(t, err)
	require.Equal(t, shift.CashboxID, arg.CashboxID)
	require.Equal(t, shift.TotalOpeningBalance, arg.TotalOpeningBalance)
	require.Equal(t, shift.TotalBalance, arg.TotalBalance)

	return shift
}

func TestCreateShift(t *testing.T) {
	createRandomShift(t)
}

func TestGetShift(t *testing.T) {
	shift1 := createRandomShift(t)

	shift2, err := testQueries.GetShift(context.Background(), shift1.ID)
	require.NoError(t, err)
	require.Equal(t, shift1.ID, shift2.ID)
	require.Equal(t, shift1.CashboxID, shift2.CashboxID)
	require.Equal(t, shift1.TotalOpeningBalance, shift2.TotalOpeningBalance)
	require.Equal(t, shift1.TotalBalance, shift2.TotalBalance)

	require.WithinDuration(t, shift1.OpeningDateTime, shift2.OpeningDateTime, time.Second)
	require.WithinDuration(t, shift1.ClosingDateTime.Time, shift2.ClosingDateTime.Time, time.Second)
}

func TestListShifts(t *testing.T) {
	for range 10 {
		createRandomShift(t)
	}

	arg := ListShiftsParams{
		Limit:  5,
		Offset: 5,
	}

	shifts, err := testQueries.ListShifts(context.Background(), arg)
	require.NoError(t, err)
	for shift := range shifts {
		// FIXME
		require.NotEmpty(t, shift)
	}
}

func TestUpdateShiftBalance(t *testing.T) {
	shift1 := createRandomShift(t)

	arg := AddToShiftBalanceParams{
		ID:     shift1.ID,
		Amount: util.RandomAmount(),
	}

	_, err := testQueries.AddToShiftBalance(context.Background(), arg)
	require.NoError(t, err)

	shift2, err := testQueries.GetShift(context.Background(), shift1.ID)
	if err != nil {
		log.Fatal(err)
	}
	require.NoError(t, err)
	require.Equal(t, shift2.TotalBalance, arg.Amount+shift1.TotalBalance)
}

func TestCloseShift(t *testing.T) {
	shift := createRandomShift(t)

	arg := CloseShiftParams{
		ID:              shift.ID,
		ClosingDateTime: sql.NullTime{Time: time.Now(), Valid: true},
		IsClosed:        true,
	}

	err := testQueries.CloseShift(context.Background(), arg)
	require.NoError(t, err)

	shift2, err := testQueries.GetShift(context.Background(), shift.ID)
	if err != nil {
		log.Fatal(err)
	}
	require.NoError(t, err)
	require.Equal(t, shift2.IsClosed, arg.IsClosed)
	require.WithinDuration(t, shift2.ClosingDateTime.Time, arg.ClosingDateTime.Time, time.Second)
}
