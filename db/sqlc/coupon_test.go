package db

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/mhdna/kashi/util"
	"github.com/stretchr/testify/require"
)

func createRandomCouponType(t *testing.T) CouponType {
	arg := CreateCouponTypeParams{
		Name:         util.RandomString(20),
		ValueCents:   int64(util.RandomNumber()),
		CurrencyCode: createRandomCurrency(t).Code,
	}

	couponType, err := testQueries.CreateCouponType(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, couponType)
	require.Equal(t, arg.Name, couponType.Name)
	require.Equal(t, arg.ValueCents, couponType.ValueCents)

	return couponType
}

func createRandomCoupon(t *testing.T) Coupon {
	client := createRandomClient(t)
	couponType := createRandomCouponType(t)
	arg := CreateCouponParams{
		Code:       util.RandomCode(),
		ClientID:   sql.NullInt64{Int64: client.ID, Valid: true},
		Status:     CouponStatusActive,
		TypeID:     sql.NullInt64{Int64: couponType.ID, Valid: true},
		ValueCents: couponType.ValueCents,
		ValidUntil: sql.NullTime{Time: time.Now().Add(time.Minute * time.Duration(util.RandomNumber())), Valid: true},
	}

	coupon, err := testQueries.CreateCoupon(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, coupon)
	require.Equal(t, arg.Code, coupon.Code)
	require.Equal(t, arg.ClientID, coupon.ClientID)
	require.Equal(t, arg.Status, coupon.Status)
	require.Equal(t, arg.TypeID, coupon.TypeID)
	require.WithinDuration(t, arg.ValidUntil.Time, coupon.ValidUntil.Time, time.Second)

	return coupon
}

func TestCreateCoupon(t *testing.T) {
	createRandomCoupon(t)
}

func TestGetCoupon(t *testing.T) {
	coupon1 := createRandomCoupon(t)
	coupon2, err := testQueries.GetCoupon(context.Background(), coupon1.Code)

	require.NoError(t, err)
	require.NotEmpty(t, coupon2)

	require.Equal(t, coupon1.Code, coupon2.Code)
	require.Equal(t, coupon1.ClientID, coupon2.ClientID)
	require.Equal(t, coupon1.Status, coupon2.Status)
	require.Equal(t, coupon1.TypeID, coupon2.TypeID)
	require.WithinDuration(t, coupon1.ValidUntil.Time, coupon2.ValidUntil.Time, time.Second)
}

func TestDeactivateCoupon(t *testing.T) {
	coupon1 := createRandomCoupon(t)
	err := testQueries.DeactivateCoupon(context.Background(), coupon1.Code)
	require.NoError(t, err)

	coupon2, err := testQueries.GetCoupon(context.Background(), coupon1.Code)
	require.NoError(t, err)
	require.Equal(t, coupon1.Code, coupon2.Code)
	require.Equal(t, coupon2.Status, CouponStatusInactive)
}

func TestListCoupons(t *testing.T) {
	for range 10 {
		createRandomCoupon(t)
	}
	arg := ListCouponsParams{
		PageSize:   5,
		PageOffset: 5,
	}
	coupons, err := testQueries.ListCoupons(context.Background(), arg)
	require.NoError(t, err)
	require.Len(t, coupons, 5)
	for _, coupon := range coupons {
		require.NotEmpty(t, coupon)
	}
}

func TestRedeemCoupon(t *testing.T) {
	coupon1 := createRandomCoupon(t)

	redeemed, err := testQueries.RedeemCoupon(context.Background(), RedeemCouponParams{
		Code:               coupon1.Code,
		RedeemedBranchID:   sql.NullInt64{},
		RedeemedInvoiceRef: sql.NullString{String: util.RandomCode(), Valid: true},
	})
	require.NoError(t, err)
	require.Equal(t, CouponStatusRedeemed, redeemed.Status)
	require.True(t, redeemed.RedeemedAt.Valid)

	// A second redemption attempt is a no-op (the WHERE clause only ever
	// matches an 'active' row) -- surfaced as sql.ErrNoRows, which the API
	// layer treats as "already redeemed", not an error.
	_, err = testQueries.RedeemCoupon(context.Background(), RedeemCouponParams{
		Code:               coupon1.Code,
		RedeemedInvoiceRef: sql.NullString{String: util.RandomCode(), Valid: true},
	})
	require.ErrorIs(t, err, sql.ErrNoRows)
}
