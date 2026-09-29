package api

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	mockdb "github.com/mhdna/kashi/db/mock"
	db "github.com/mhdna/kashi/db/sqlc"
	"github.com/stretchr/testify/require"
)

func TestCreateBranchTransferRequest_ResolvesDestinationAndCreates(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	store := mockdb.NewMockStore(ctrl)
	stubBranchAuth(store)

	store.EXPECT().
		GetBranchTransferByClientRef(gomock.Any(), gomock.Any()).
		Return(db.Transfer{}, sql.ErrNoRows)

	store.EXPECT().
		GetBranchByCodeExcluding(gomock.Any(), db.GetBranchByCodeExcludingParams{Code: "BR02", ID: testBranchID}).
		Return(db.Branch{ID: 9, Code: "BR02", InventoryID: sql.NullInt64{Int64: 42, Valid: true}}, nil)

	store.EXPECT().
		CreateBranchTransferRequestTx(gomock.Any(), db.CreateBranchTransferRequestTxParams{
			BranchID:      testBranchID,
			ClientRef:     "ref-1",
			ToInventoryID: 42,
			Note:          "restock",
			Items: []db.BranchTransferItemParams{
				{Barcode: "12345", Quantity: 3},
			},
		}).
		Return(db.CreateBranchTransferRequestTxResult{
			Transfer: db.Transfer{ID: 1, Status: db.TransferStatusRequested},
		}, nil)

	store.EXPECT().
		CreateBranchTransferUploadToken(gomock.Any(), gomock.Any()).
		Return(db.BranchTransferUploadToken{Token: "tok"}, nil)

	server := newTestServer(t, store)
	rec := httptest.NewRecorder()
	server.router.ServeHTTP(rec, newBranchRequest(t, http.MethodPost, "/branch/transfer_requests", map[string]any{
		"client_ref":    "ref-1",
		"to_branch_ref": "BR02",
		"note":          "restock",
		"items": []map[string]any{
			{"barcode": "12345", "quantity": 3},
		},
	}))

	require.Equal(t, http.StatusOK, rec.Code)
}

// A destination branch can't be itself -- GetBranchByCodeExcluding's own
// query already refuses that (WHERE id != $2), a retried/mistaken request
// naming this branch's own code just surfaces as "unknown destination".
func TestCreateBranchTransferRequest_UnknownDestination(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	store := mockdb.NewMockStore(ctrl)
	stubBranchAuth(store)

	store.EXPECT().
		GetBranchTransferByClientRef(gomock.Any(), gomock.Any()).
		Return(db.Transfer{}, sql.ErrNoRows)
	store.EXPECT().
		GetBranchByCodeExcluding(gomock.Any(), gomock.Any()).
		Return(db.Branch{}, sql.ErrNoRows)

	server := newTestServer(t, store)
	rec := httptest.NewRecorder()
	server.router.ServeHTTP(rec, newBranchRequest(t, http.MethodPost, "/branch/transfer_requests", map[string]any{
		"client_ref":    "ref-1",
		"to_branch_ref": testBranchCode,
		"items": []map[string]any{
			{"barcode": "12345", "quantity": 1},
		},
	}))

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

// A retried post (the till never saw the first 200) is idempotent: the
// existing row is returned, and no new transfer is created.
func TestCreateBranchTransferRequest_IdempotentRetry(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	store := mockdb.NewMockStore(ctrl)
	stubBranchAuth(store)

	store.EXPECT().
		GetBranchTransferByClientRef(gomock.Any(), gomock.Any()).
		Return(db.Transfer{ID: 5, Status: db.TransferStatusRequested}, nil)

	server := newTestServer(t, store)
	rec := httptest.NewRecorder()
	server.router.ServeHTTP(rec, newBranchRequest(t, http.MethodPost, "/branch/transfer_requests", map[string]any{
		"client_ref":    "ref-1",
		"to_branch_ref": "BR02",
		"items": []map[string]any{
			{"barcode": "12345", "quantity": 1},
		},
	}))

	require.Equal(t, http.StatusOK, rec.Code)
}

func TestApproveTransfer_MovesStockAndNotifiesBranch(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	store := mockdb.NewMockStore(ctrl)

	approved := db.Transfer{
		ID:        3,
		Status:    db.TransferStatusReceived,
		BranchID:  sql.NullInt64{Int64: testBranchID, Valid: true},
		ClientRef: sql.NullString{String: "ref-1", Valid: true},
	}

	store.EXPECT().
		TransferApproveTx(gomock.Any(), gomock.Any()).
		Return(db.TransferStageTxResult{Transfer: approved}, nil)
	store.EXPECT().
		CreateBranchCommand(gomock.Any(), gomock.Any()).
		Return(db.BranchCommand{ID: 1}, nil)

	server := newTestServer(t, store)
	rec := httptest.NewRecorder()
	req, err := http.NewRequest(http.MethodPost, "/transfers/3/approve", nil)
	require.NoError(t, err)
	addAuthorization(t, req, server.tokenMaker, authorizationTypeBearer, "user", time.Minute)
	server.router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
}
