package api

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	mockdb "github.com/mhdna/kashi/db/mock"
	db "github.com/mhdna/kashi/db/sqlc"
	"github.com/stretchr/testify/require"
)

func TestListDeliveryState(t *testing.T) {
	now := time.Now()
	changed := now.Add(-time.Minute)
	at := func(when time.Time) sql.NullTime { return sql.NullTime{Time: when, Valid: true} }
	before, after := changed.Add(-time.Hour), changed.Add(time.Second)
	seen := at(now)
	oldTill, right, wrong := tillListView{}, tillListView{known: true, matches: true, reportedAt: after}, tillListView{known: true}

	for name, tc := range map[string]struct {
		row      db.ListBranchSyncStatusRow
		till     tillListView
		want     string
		verified bool
	}{
		// A till that reports what it holds.
		"holds the right rows": {db.ListBranchSyncStatusRow{LastSeenAt: seen, LastPullAt: at(after), SyncedThrough: at(before)}, right, deliveryReached, true},
		"right rows, offline":  {db.ListBranchSyncStatusRow{LastSeenAt: at(now.Add(-time.Hour))}, right, deliveryReached, true},
		// The crocs bug: by its cursor it's up to date, yet it holds something else.
		"cursor passed, wrong rows": {db.ListBranchSyncStatusRow{LastSeenAt: seen, LastPullAt: at(after), SyncedThrough: at(after)}, wrong, deliveryDifferent, false},
		"wrong rows, still coming":  {db.ListBranchSyncStatusRow{LastSeenAt: seen, LastPullAt: at(after), SyncedThrough: at(before), ItemsSent: 2}, wrong, deliverySending, false},
		"wrong rows, not asked yet": {db.ListBranchSyncStatusRow{LastSeenAt: seen, LastPullAt: at(before), SyncedThrough: at(before)}, wrong, deliveryWaiting, false},
		"wrong rows, a failed pull": {db.ListBranchSyncStatusRow{LastSeenAt: seen, LastPullAt: at(after), Error: "boom"}, wrong, deliveryStuck, false},
		// An older till is only judged by its cursor, and says so.
		"hasn't asked yet":         {db.ListBranchSyncStatusRow{LastSeenAt: seen, LastPullAt: at(before), SyncedThrough: at(before)}, oldTill, deliveryWaiting, false},
		"not seen lately":          {db.ListBranchSyncStatusRow{LastSeenAt: at(now.Add(-time.Hour)), LastPullAt: at(before)}, oldTill, deliveryOffline, false},
		"pulled, no word":          {db.ListBranchSyncStatusRow{LastSeenAt: seen, LastPullAt: at(after), SyncedThrough: at(before), ItemsSent: 2}, oldTill, deliverySending, false},
		"pulled and reported":      {db.ListBranchSyncStatusRow{LastSeenAt: seen, LastPullAt: at(after), SyncedThrough: at(before), ItemsSent: 2, ReportedAt: at(after)}, oldTill, deliveryReached, false},
		"pulled, but nothing came": {db.ListBranchSyncStatusRow{LastSeenAt: seen, LastPullAt: at(after), SyncedThrough: at(before), ReportedAt: at(after)}, oldTill, deliveryWaiting, false},
		"its cursor passed it":     {db.ListBranchSyncStatusRow{LastSeenAt: seen, LastPullAt: at(after), SyncedThrough: at(after)}, oldTill, deliveryReached, false},
		"keeps failing":            {db.ListBranchSyncStatusRow{LastSeenAt: seen, LastPullAt: at(after), SyncedThrough: at(before), ItemsSent: 2, Retries: 2}, oldTill, deliveryStuck, false},
		"reports a failure": {
			db.ListBranchSyncStatusRow{LastSeenAt: seen, LastPullAt: at(after), SyncedThrough: at(before), ItemsSent: 2, Error: "boom", ReportedAt: at(after)}, oldTill, deliveryStuck, false,
		},
	} {
		t.Run(name, func(t *testing.T) {
			got, _, verified := listDeliveryState(tc.row, tc.till, changed, now)
			require.Equal(t, tc.want, got)
			require.Equal(t, tc.verified, verified)
		})
	}
}

// The same rows must give the same hash here and on the till. kashi-pos's
// TestListFingerprintMatchesKashi checks these very values.
func TestListFingerprintVector(t *testing.T) {
	a := uuid.MustParse("11111111-1111-4111-8111-111111111111")
	b := uuid.MustParse("22222222-2222-4222-8222-222222222222")
	deal := uuid.NullUUID{UUID: uuid.MustParse("33333333-3333-4333-8333-333333333333"), Valid: true}

	discount := discountListFingerprint(
		[]db.ListDiscountListItemsForSyncRow{
			{ProductVariantUid: b, Discount: 0, DealUid: deal, DealFallbackPercent: sql.NullInt16{Int16: 30, Valid: true}},
			{ProductVariantUid: a, Discount: 21},
		},
		[]db.ListDiscountListOptionItemsForSyncRow{{ProductVariantUid: a, SaleOption: "Winter", Discount: 20}},
	)
	require.Equal(t, 3, discount.Rows)
	require.Equal(t, "e26eb35c5243c924941d9c727c36f816bc8719c8f8f3c93f45cc09b96ebdf9f0", discount.Hash)

	price := priceListFingerprint([]db.ListPriceListItemsForSyncRow{{ProductVariantUid: b, Price: 1900}, {ProductVariantUid: a, Price: 500}})
	require.Equal(t, 2, price.Rows)
	require.Equal(t, "151e829b427adfbd99b1a5c9a150810ba7f380a8f69ba71a3946d69ddc1c16ed", price.Hash)

	require.Equal(t, 0, emptyListFingerprint.Rows)
	require.Equal(t, "e3b0c44298fc1c14", emptyListFingerprint.Hash[:16], "sha256 of nothing")
}

// The endpoint end to end: the crocs bug shows as different, the right rows as
// checked, and a till that reports nothing as reached but not checked.
func TestDiscountListDeliveryComparesWhatTheTillHolds(t *testing.T) {
	const listID, branchID = int64(4), int64(1)
	listUID := uuid.MustParse("44444444-4444-4444-8444-444444444444")
	variant := uuid.MustParse("11111111-1111-4111-8111-111111111111")
	changed := time.Now().Add(-10 * time.Minute)
	rows := []db.ListDiscountListItemsForSyncRow{{ProductVariantUid: variant, Discount: 21}}
	right := discountListFingerprint(rows, nil)
	stale := discountListFingerprint([]db.ListDiscountListItemsForSyncRow{{ProductVariantUid: variant, Discount: 30}}, nil)
	seen := sql.NullTime{Time: time.Now(), Valid: true}
	passed := sql.NullTime{Time: changed.Add(time.Minute), Valid: true}

	for name, tc := range map[string]struct {
		states   []db.BranchRegisterListState
		want     string
		verified bool
	}{
		"holds the old rows":    {[]db.BranchRegisterListState{{RegisterID: 3, ListUid: listUID.String(), Rows: 1, Hash: stale.Hash}}, deliveryDifferent, false},
		"holds the right rows":  {[]db.BranchRegisterListState{{RegisterID: 3, ListUid: listUID.String(), Rows: 1, Hash: right.Hash, ReportedAt: time.Now()}}, deliveryReached, true},
		"doesn't have the list": {[]db.BranchRegisterListState{{RegisterID: 3, ListUid: "another", Rows: 0, Hash: emptyListFingerprint.Hash}}, deliveryDifferent, false},
		"too old to report":     {nil, deliveryReached, false},
	} {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			store := mockdb.NewMockStore(ctrl)
			// A new one per case: the fingerprint is cached by list and stamp.
			stamp := changed.Add(time.Duration(len(name)) * time.Millisecond)
			store.EXPECT().GetDiscountList(gomock.Any(), listID).Return(db.DiscountList{
				ID: listID, Uid: listUID, IsActive: true, UpdatedAt: stamp,
			}, nil)
			store.EXPECT().ListDiscountListBranchIDs(gomock.Any(), listID).Return([]int64{branchID}, nil)
			store.EXPECT().ListDiscountListItemsForSync(gomock.Any(), listID).Return(rows, nil).AnyTimes()
			store.EXPECT().ListDiscountListOptionItemsForSync(gomock.Any(), listID).
				Return([]db.ListDiscountListOptionItemsForSyncRow{}, nil).AnyTimes()
			store.EXPECT().ListBranchRegisterListStates(gomock.Any(), "discount").Return(tc.states, nil)
			store.EXPECT().ListBranchSyncStatus(gomock.Any(), gomock.Any()).Return([]db.ListBranchSyncStatusRow{{
				RegisterID: 3, RegisterCode: "CT", BranchID: branchID, BranchName: "Centro", LastSeenAt: seen,
				Entity: "discount_lists", LastPullAt: passed, SyncedThrough: passed,
			}}, nil)

			server := newTestServer(t, store)
			rec := httptest.NewRecorder()
			req, err := http.NewRequest(http.MethodGet, "/discount_lists/4/delivery", nil)
			require.NoError(t, err)
			addAuthorization(t, req, server.tokenMaker, authorizationTypeBearer, "admin", time.Minute)
			server.router.ServeHTTP(rec, req)
			require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

			var body struct {
				Registers []listDeliveryRegister `json:"registers"`
			}
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
			require.Len(t, body.Registers, 1)
			require.Equal(t, tc.want, body.Registers[0].State)
			require.Equal(t, tc.verified, body.Registers[0].Verified)
		})
	}
}
