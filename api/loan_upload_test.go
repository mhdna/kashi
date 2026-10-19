package api

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	mockdb "github.com/mhdna/kashi/db/mock"
	db "github.com/mhdna/kashi/db/sqlc"
	"github.com/stretchr/testify/require"
)

// A till can get a new photo link for its own expense while it has room for
// more photos, and is told plainly when it has none left or kashi hasn't got
// the expense yet.
func TestBranchExpenseUploadLink(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	stubBranchAuth(store)
	server := newTestServer(t, store)
	byRef := db.GetBranchExpenseByClientRefParams{BranchID: testBranchID, ClientRef: "exp-1"}

	store.EXPECT().GetBranchExpenseByClientRef(gomock.Any(), byRef).Return(db.BranchExpense{ID: 5, BranchID: testBranchID}, nil)
	store.EXPECT().CountBranchExpenseImagesForExpense(gomock.Any(), int64(5)).Return(int64(0), nil)
	store.EXPECT().CreateBranchExpenseUploadToken(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ any, arg db.CreateBranchExpenseUploadTokenParams) (db.BranchExpenseUploadToken, error) {
			require.Equal(t, int64(5), arg.BranchExpenseID)
			require.True(t, arg.ExpiresAt.After(time.Now()))
			return db.BranchExpenseUploadToken{}, nil
		})
	rec := httptest.NewRecorder()
	server.router.ServeHTTP(rec, newBranchRequest(t, http.MethodPost, "/branch/expenses/upload_link", map[string]string{"client_ref": "exp-1"}))
	require.Equal(t, http.StatusOK, rec.Code)
	var body struct {
		UploadToken string `json:"upload_token"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Len(t, body.UploadToken, 64)

	store.EXPECT().GetBranchExpenseByClientRef(gomock.Any(), byRef).Return(db.BranchExpense{ID: 5, BranchID: testBranchID}, nil)
	store.EXPECT().CountBranchExpenseImagesForExpense(gomock.Any(), int64(5)).Return(int64(maxExpenseUploadImages), nil)
	rec = httptest.NewRecorder()
	server.router.ServeHTTP(rec, newBranchRequest(t, http.MethodPost, "/branch/expenses/upload_link", map[string]string{"client_ref": "exp-1"}))
	require.Equal(t, http.StatusConflict, rec.Code)

	store.EXPECT().GetBranchExpenseByClientRef(gomock.Any(), byRef).Return(db.BranchExpense{}, sql.ErrNoRows)
	rec = httptest.NewRecorder()
	server.router.ServeHTTP(rec, newBranchRequest(t, http.MethodPost, "/branch/expenses/upload_link", map[string]string{"client_ref": "exp-1"}))
	require.Equal(t, http.StatusNotFound, rec.Code)

	store.EXPECT().GetLoanUploadTokenByToken(gomock.Any(), "done").
		Return(db.LoanUploadToken{ID: 3, LoanID: 9, ExpiresAt: time.Now().Add(time.Minute), UsedAt: sql.NullTime{Time: time.Now(), Valid: true}}, nil)
	rec = httptest.NewRecorder()
	server.router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/loan_uploads/done/status", nil))
	require.Equal(t, http.StatusNotFound, rec.Code, "an already used link is gone")
}

// The same for a loan, looked up within the calling branch.
func TestBranchLoanUploadLink(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	stubBranchAuth(store)
	server := newTestServer(t, store)

	store.EXPECT().GetBranchLoanByClientRef(gomock.Any(), db.GetBranchLoanByClientRefParams{
		BranchID:  sql.NullInt64{Int64: testBranchID, Valid: true},
		ClientRef: sql.NullString{String: "loan-1", Valid: true},
	}).Return(db.Loan{ID: 9}, nil)
	store.EXPECT().CountLoanImagesForLoan(gomock.Any(), int64(9)).Return(int64(1), nil)
	store.EXPECT().CreateLoanUploadToken(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ any, arg db.CreateLoanUploadTokenParams) (db.LoanUploadToken, error) {
			require.Equal(t, int64(9), arg.LoanID)
			return db.LoanUploadToken{}, nil
		})
	rec := httptest.NewRecorder()
	server.router.ServeHTTP(rec, newBranchRequest(t, http.MethodPost, "/branch/loans/upload_link", map[string]string{"client_ref": "loan-1"}))
	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), "upload_token")
}

// The public status call behind a loan's QR page describes the loan; an
// expired token reads as not found.
func TestLoanUploadStatus(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	server := newTestServer(t, store)

	store.EXPECT().GetLoanUploadTokenByToken(gomock.Any(), "good").
		Return(db.LoanUploadToken{ID: 1, LoanID: 9, ExpiresAt: time.Now().Add(time.Minute)}, nil)
	store.EXPECT().GetLoan(gomock.Any(), int64(9)).
		Return(db.Loan{ID: 9, Description: "rent advance", Amount: 2500, CurrencyCode: "USD"}, nil)
	rec := httptest.NewRecorder()
	server.router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/loan_uploads/good/status", nil))
	require.Equal(t, http.StatusOK, rec.Code)
	require.JSONEq(t, `{"description":"rent advance","amount_label":"25.00 USD","used":false}`, rec.Body.String())

	store.EXPECT().GetLoanUploadTokenByToken(gomock.Any(), "old").
		Return(db.LoanUploadToken{ID: 2, LoanID: 9, ExpiresAt: time.Now().Add(-time.Minute)}, nil)
	rec = httptest.NewRecorder()
	server.router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/loan_uploads/old/status", nil))
	require.Equal(t, http.StatusNotFound, rec.Code)
}
