package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang/mock/gomock"
	mockdb "github.com/mhdna/kashi/db/mock"
	db "github.com/mhdna/kashi/db/sqlc"
	"github.com/stretchr/testify/require"
)

// The till asks for its own sales' poll answers by invoice code.
func TestBranchInvoiceRatings(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	stubBranchAuth(store)
	store.EXPECT().ListBranchInvoiceRatings(gomock.Any(), db.ListBranchInvoiceRatingsParams{
		BranchID: testBranchID, InvoiceCodes: []string{"S-1", "S-2"},
	}).Return([]db.ListBranchInvoiceRatingsRow{{InvoiceCode: "S-1", PollRating: 5, PollAnswer: "😍 Loved it"}}, nil)

	server := newTestServer(t, store)
	rec := httptest.NewRecorder()
	server.router.ServeHTTP(rec, newBranchRequest(t, http.MethodPost, "/branch/invoice_ratings",
		map[string]any{"codes": []string{"S-1", " ", "S-2"}}))
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	var out struct {
		Ratings map[string]branchInvoiceRating `json:"ratings"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &out))
	require.Equal(t, map[string]branchInvoiceRating{"S-1": {Rating: 5, Answer: "😍 Loved it"}}, out.Ratings)
}
