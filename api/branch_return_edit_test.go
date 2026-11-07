package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang/mock/gomock"
	mockdb "github.com/mhdna/kashi/db/mock"
	db "github.com/mhdna/kashi/db/sqlc"
	"github.com/stretchr/testify/require"
)

// editReturnRequest queues an edit_return_invoice for invoice and answers the
// status and the register it was queued for (0 when none was).
func editReturnRequest(t *testing.T, invoice db.BranchInvoice, lookupErr error) (int, int64, map[string]json.RawMessage) {
	t.Helper()
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	var queuedFor int64
	var sent map[string]json.RawMessage
	store.EXPECT().GetBranch(gomock.Any(), buBranchID).Return(db.Branch{ID: buBranchID}, nil)
	store.EXPECT().GetBranchInvoiceByClientRef(gomock.Any(), db.GetBranchInvoiceByClientRefParams{BranchID: buBranchID, ClientRef: "ref-68"}).
		Return(invoice, lookupErr)
	store.EXPECT().CreateBranchCommand(gomock.Any(), gomock.Any()).AnyTimes().
		DoAndReturn(func(_ context.Context, arg db.CreateBranchCommandParams) (db.BranchCommand, error) {
			queuedFor = arg.RegisterID
			require.Equal(t, "edit_return_invoice", arg.Type)
			require.NoError(t, json.Unmarshal(arg.Payload, &sent))
			return db.BranchCommand{ID: 1, RegisterID: arg.RegisterID}, nil
		})
	server := newTestServer(t, store)
	rec := adminRequest(t, server, http.MethodPost, fmt.Sprintf("/branches/%d/commands", buBranchID), map[string]any{
		"type": "edit_return_invoice",
		"payload": map[string]any{
			"client_ref": "ref-68",
			"lines":      []map[string]any{{"position": 1, "barcode": "8000000115959", "line_total": -1500}},
			"reason":     "Customer keeps the deal",
		},
	})
	return rec.Code, queuedFor, sent
}

// RS/CT/2026/00068's refund edit goes to the till that rang it up, as typed.
func TestCreateBranchCommand_EditReturnGoesToItsRegister(t *testing.T) {
	for _, kind := range []string{"return", "exchange"} {
		t.Run(kind, func(t *testing.T) {
			code, register, sent := editReturnRequest(t, db.BranchInvoice{ID: 5, BranchID: buBranchID, Kind: kind, RegisterID: 3, ClientRef: "ref-68"}, nil)
			require.Equal(t, http.StatusOK, code)
			require.Equal(t, int64(3), register)
			require.JSONEq(t, `[{"position":1,"barcode":"8000000115959","line_total":-1500}]`, string(sent["lines"]))
			require.JSONEq(t, `"Customer keeps the deal"`, string(sent["reason"]))
		})
	}
}

func TestCreateBranchCommand_EditReturnRefuses(t *testing.T) {
	cases := []struct {
		name    string
		invoice db.BranchInvoice
		err     error
		want    int
	}{
		{"a sale", db.BranchInvoice{ID: 5, BranchID: buBranchID, Kind: "sales", RegisterID: 3}, nil, http.StatusBadRequest},
		{"an imported return", db.BranchInvoice{ID: 5, BranchID: buBranchID, Kind: "return", RegisterID: 3, Historical: true}, nil, http.StatusBadRequest},
		{"no such return", db.BranchInvoice{}, sql.ErrNoRows, http.StatusNotFound},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			code, register, _ := editReturnRequest(t, tc.invoice, tc.err)
			require.Equal(t, tc.want, code)
			require.Zero(t, register)
		})
	}
}

// What the till sends after applying the edit reaches kashi's copy as is.
func TestCreateBranchReturnEdit_AppliesTheTillsEdit(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	stubBranchAuth(store)
	lbp, rate := int64(-3960000), int64(90000)
	store.EXPECT().ApplyBranchReturnEditTx(gomock.Any(), db.ApplyBranchReturnEditTxParams{
		BranchID: testBranchID, ClientRef: "ref-68", GrandTotal: -4400,
		LbpTotal: sql.NullInt64{Int64: lbp, Valid: true}, LbpRate: sql.NullInt64{Int64: rate, Valid: true},
		Lines:    []db.BranchReturnEditLine{{Position: 1, LineTotal: -1500}},
		Payments: []db.BranchInvoicePaymentParams{{AccountName: "Cash $$", Amount: -4400}},
	}).Return(db.BranchInvoice{ID: 5, GrandTotal: -4400, BranchInvoiceCode: "RS/CT/2026/00068"}, nil)

	server := newTestServer(t, store)
	rec := httptest.NewRecorder()
	server.router.ServeHTTP(rec, newBranchRequest(t, http.MethodPost, "/branch/return_invoice_edits", map[string]any{
		"edit_ref": "e1", "client_ref": "ref-68", "grand_total": -4400, "lbp_total": lbp, "lbp_rate": rate,
		"lines":    []map[string]any{{"position": 1, "line_total": -1500}},
		"payments": []map[string]any{{"account_name": "Cash $$", "amount": -4400}},
		"reason":   "Customer keeps the deal", "occurred_at": "2026-11-07T20:00:00Z",
	}))
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
}
