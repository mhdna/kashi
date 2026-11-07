package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/golang/mock/gomock"
	mockdb "github.com/mhdna/kashi/db/mock"
	db "github.com/mhdna/kashi/db/sqlc"
	"github.com/stretchr/testify/require"
)

// A shift reprint goes to the register that closed the shift, carrying that
// till's own shift id rather than kashi's row id.
func TestCreateBranchCommand_PrintShiftGoesToTheClosingRegister(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)

	store.EXPECT().GetBranch(gomock.Any(), buBranchID).Return(db.Branch{ID: buBranchID}, nil)
	store.EXPECT().GetBranchShift(gomock.Any(), int64(40)).
		Return(db.BranchShift{ID: 40, BranchID: buBranchID, RegisterID: 23, BranchShiftID: 7}, nil)
	store.EXPECT().ListActiveBranchRegisterIDs(gomock.Any(), gomock.Any()).Times(0)
	store.EXPECT().
		CreateBranchCommand(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, arg db.CreateBranchCommandParams) (db.BranchCommand, error) {
			require.Equal(t, int64(23), arg.RegisterID)
			var p struct {
				BranchShiftID int64 `json:"branch_shift_id"`
			}
			require.NoError(t, json.Unmarshal(arg.Payload, &p))
			require.Equal(t, int64(7), p.BranchShiftID)
			return db.BranchCommand{ID: 1, RegisterID: arg.RegisterID}, nil
		})

	server := newTestServer(t, store)
	rec := adminRequest(t, server, http.MethodPost, fmt.Sprintf("/branches/%d/commands", buBranchID), map[string]any{
		"type":    "print_shift_report",
		"payload": map[string]any{"shift_id": 40},
	})
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
}

func TestCreateBranchCommand_PrintShiftOfAnotherBranchIsRejected(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)

	store.EXPECT().GetBranch(gomock.Any(), buBranchID).Return(db.Branch{ID: buBranchID}, nil)
	store.EXPECT().GetBranchShift(gomock.Any(), int64(40)).
		Return(db.BranchShift{ID: 40, BranchID: buBranchID + 1, RegisterID: 23}, nil)
	store.EXPECT().CreateBranchCommand(gomock.Any(), gomock.Any()).Times(0)

	server := newTestServer(t, store)
	rec := adminRequest(t, server, http.MethodPost, fmt.Sprintf("/branches/%d/commands", buBranchID), map[string]any{
		"type":    "print_shift_report",
		"payload": map[string]any{"shift_id": 40},
	})
	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestCancelBranchCommand_OnlyPending(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	store.EXPECT().CancelBranchCommand(gomock.Any(), int64(9)).Return(db.BranchCommand{}, sql.ErrNoRows)

	server := newTestServer(t, store)
	rec := adminRequest(t, server, http.MethodPost, "/branch_commands/9/cancel", nil)
	require.Equal(t, http.StatusConflict, rec.Code)
}

// A retry queues a new copy for the same register; the failed row stays.
func TestRetryBranchCommand_RequeuesFailedForSameRegister(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)

	store.EXPECT().GetBranchCommand(gomock.Any(), int64(9)).Return(db.BranchCommand{
		ID: 9, BranchID: buBranchID, RegisterID: 23, Type: "sync_now", Payload: json.RawMessage(`{}`), Status: "failed",
	}, nil)
	store.EXPECT().
		CreateBranchCommand(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, arg db.CreateBranchCommandParams) (db.BranchCommand, error) {
			require.Equal(t, int64(23), arg.RegisterID)
			require.Equal(t, "sync_now", arg.Type)
			return db.BranchCommand{ID: 10, RegisterID: arg.RegisterID}, nil
		})

	server := newTestServer(t, store)
	rec := adminRequest(t, server, http.MethodPost, "/branch_commands/9/retry", nil)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
}

func TestRetryBranchCommand_RejectsSucceeded(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	store.EXPECT().GetBranchCommand(gomock.Any(), int64(9)).Return(db.BranchCommand{ID: 9, Status: "success"}, nil)
	store.EXPECT().CreateBranchCommand(gomock.Any(), gomock.Any()).Times(0)

	server := newTestServer(t, store)
	rec := adminRequest(t, server, http.MethodPost, "/branch_commands/9/retry", nil)
	require.Equal(t, http.StatusConflict, rec.Code)
}

// reprintInvoiceRequest posts a reprint_invoice for clientRef and returns the
// register the command was queued for (0 when none was).
func reprintInvoiceRequest(t *testing.T, invoice db.BranchInvoice, lookupErr error) (int, int64) {
	t.Helper()
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)

	var queuedFor int64
	store.EXPECT().GetBranch(gomock.Any(), buBranchID).Return(db.Branch{ID: buBranchID}, nil)
	store.EXPECT().GetBranchInvoiceByClientRef(gomock.Any(), db.GetBranchInvoiceByClientRefParams{BranchID: buBranchID, ClientRef: "ref-1"}).
		Return(invoice, lookupErr)
	store.EXPECT().ListActiveBranchRegisterIDs(gomock.Any(), gomock.Any()).Times(0)
	store.EXPECT().CreateBranchCommand(gomock.Any(), gomock.Any()).AnyTimes().
		DoAndReturn(func(_ context.Context, arg db.CreateBranchCommandParams) (db.BranchCommand, error) {
			queuedFor = arg.RegisterID
			var p struct {
				ClientRef string `json:"client_ref"`
			}
			require.NoError(t, json.Unmarshal(arg.Payload, &p))
			require.Equal(t, "ref-1", p.ClientRef)
			return db.BranchCommand{ID: 1, RegisterID: arg.RegisterID}, nil
		})

	server := newTestServer(t, store)
	rec := adminRequest(t, server, http.MethodPost, fmt.Sprintf("/branches/%d/commands", buBranchID), map[string]any{
		"type":    "reprint_invoice",
		"payload": map[string]any{"client_ref": "ref-1"},
	})
	return rec.Code, queuedFor
}

// Printing an invoice from kashi goes only to the till that rang it up, for
// every kind: a sale, a return (RS/CT/2026/00068) and an exchange.
func TestCreateBranchCommand_ReprintInvoiceGoesToItsRegister(t *testing.T) {
	for _, kind := range []string{"sales", "return", "exchange"} {
		t.Run(kind, func(t *testing.T) {
			code, register := reprintInvoiceRequest(t, db.BranchInvoice{ID: 5, BranchID: buBranchID, Kind: kind, RegisterID: 3, ClientRef: "ref-1"}, nil)
			require.Equal(t, http.StatusOK, code)
			require.Equal(t, int64(3), register)
		})
	}
}

// A cancelled sale still prints: the till keeps it and its receipt.
func TestCreateBranchCommand_ReprintCancelledInvoice(t *testing.T) {
	code, register := reprintInvoiceRequest(t, db.BranchInvoice{
		ID: 5, BranchID: buBranchID, Kind: "sales", RegisterID: 4, ClientRef: "ref-1",
		CancelledAt: sql.NullTime{Valid: true},
	}, nil)
	require.Equal(t, http.StatusOK, code)
	require.Equal(t, int64(4), register)
}

// An invoice imported from the old POS is on no till: refused at once
// rather than queued for a register that never pairs.
func TestCreateBranchCommand_ReprintImportedInvoiceIsRejected(t *testing.T) {
	code, register := reprintInvoiceRequest(t, db.BranchInvoice{ID: 5, BranchID: buBranchID, Kind: "sales", RegisterID: 9, ClientRef: "ref-1", Historical: true}, nil)
	require.Equal(t, http.StatusBadRequest, code)
	require.Zero(t, register)
}

// An invoice this branch doesn't have (another branch's, or none) is a 404.
func TestCreateBranchCommand_ReprintUnknownInvoiceIsNotFound(t *testing.T) {
	code, register := reprintInvoiceRequest(t, db.BranchInvoice{}, sql.ErrNoRows)
	require.Equal(t, http.StatusNotFound, code)
	require.Zero(t, register)
}
