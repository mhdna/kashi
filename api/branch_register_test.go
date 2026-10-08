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
	"github.com/lib/pq"
	mockdb "github.com/mhdna/kashi/db/mock"
	db "github.com/mhdna/kashi/db/sqlc"
	"github.com/mhdna/kashi/util"
	"github.com/sqlc-dev/pqtype"
	"github.com/stretchr/testify/require"
)

func TestBranchAuth_RegisterCodeResolvesBranchAndRegister(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	stubBranchAuth(store)

	store.EXPECT().
		ListPendingBranchCommands(gomock.Any(), gomock.Eq(testRegisterID)).
		Return([]db.BranchCommand{}, nil)

	server := newTestServer(t, store)
	rec := httptest.NewRecorder()
	server.router.ServeHTTP(rec, newBranchRequest(t, http.MethodGet, "/branch/commands/pending", nil))
	require.Equal(t, http.StatusOK, rec.Code)
}

func TestBranchAuth_RejectsDisabledRegisterOrBranch(t *testing.T) {
	hash, err := util.HashPassword(testBranchKey)
	require.NoError(t, err)

	cases := map[string]struct {
		register db.BranchRegister
		branch   *db.Branch
	}{
		"disabled register": {
			register: db.BranchRegister{ID: testRegisterID, BranchID: testBranchID, ApiKeyHash: hash, IsActive: false},
		},
		"disabled branch": {
			register: db.BranchRegister{ID: testRegisterID, BranchID: testBranchID, ApiKeyHash: hash, IsActive: true},
			branch:   &db.Branch{ID: testBranchID, IsActive: false},
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			store := mockdb.NewMockStore(ctrl)
			store.EXPECT().GetBranchRegisterByCode(gomock.Any(), testRegisterCode).Return(tc.register, nil)
			if tc.branch != nil {
				store.EXPECT().GetBranch(gomock.Any(), testBranchID).Return(*tc.branch, nil)
			}

			server := newTestServer(t, store)
			rec := httptest.NewRecorder()
			server.router.ServeHTTP(rec, newBranchRequest(t, http.MethodGet, "/branch/commands/pending", nil))
			require.Equal(t, http.StatusUnauthorized, rec.Code)
		})
	}
}

func TestAckBranchCommand_RejectsAnotherRegistersCommand(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	stubBranchAuth(store)

	store.EXPECT().
		GetBranchCommand(gomock.Any(), int64(4)).
		Return(db.BranchCommand{ID: 4, BranchID: testBranchID, RegisterID: testRegisterID + 1}, nil)
	store.EXPECT().CompleteBranchCommand(gomock.Any(), gomock.Any()).Times(0)

	server := newTestServer(t, store)
	rec := httptest.NewRecorder()
	server.router.ServeHTTP(rec, newBranchRequest(t, http.MethodPost, "/branch/commands/4/ack", map[string]any{"status": "success"}))
	require.Equal(t, http.StatusForbidden, rec.Code)
}

func TestCreateBranchCommand_UpdateSettingsAppliesToBranchAndEveryRegister(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)

	store.EXPECT().GetBranch(gomock.Any(), buBranchID).Return(db.Branch{ID: buBranchID}, nil)
	store.EXPECT().
		GetBranchSettings(gomock.Any(), buBranchID).
		Return(db.BranchSetting{BranchID: buBranchID, TaxRate: 11, OversellMode: "allow", MarketName: "Old"}, nil)
	store.EXPECT().
		UpsertBranchSettings(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, arg db.UpsertBranchSettingsParams) (db.BranchSetting, error) {
			require.Equal(t, "warn", arg.OversellMode)
			require.Equal(t, "New", arg.MarketName)
			require.Equal(t, float64(11), arg.TaxRate, "fields absent from the patch keep their value")
			return db.BranchSetting{}, nil
		})
	store.EXPECT().ListActiveBranchRegisterIDs(gomock.Any(), buBranchID).Return([]int64{21, 22}, nil)

	var targets []int64
	store.EXPECT().
		CreateBranchCommand(gomock.Any(), gomock.Any()).
		Times(2).
		DoAndReturn(func(_ context.Context, arg db.CreateBranchCommandParams) (db.BranchCommand, error) {
			targets = append(targets, arg.RegisterID)
			var payload map[string]any
			require.NoError(t, json.Unmarshal(arg.Payload, &payload))
			require.NotContains(t, payload, "printer_id", "per-machine settings never go branch-wide")
			require.Equal(t, "warn", payload["oversell_mode"])
			return db.BranchCommand{ID: arg.RegisterID, RegisterID: arg.RegisterID}, nil
		})

	server := newTestServer(t, store)
	rec := adminRequest(t, server, http.MethodPost, fmt.Sprintf("/branches/%d/commands", buBranchID), map[string]any{
		"type":    "update_settings",
		"payload": map[string]any{"oversell_mode": "warn", "market_name": "New", "printer_id": "EPSON-1"},
	})

	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	require.Equal(t, []int64{21, 22}, targets)
}

func TestCreateBranchCommand_UpdateSettingsBeforeAnyReportIsConflict(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)

	store.EXPECT().GetBranch(gomock.Any(), buBranchID).Return(db.Branch{ID: buBranchID}, nil)
	store.EXPECT().GetBranchSettings(gomock.Any(), buBranchID).Return(db.BranchSetting{}, sql.ErrNoRows)
	store.EXPECT().CreateBranchCommand(gomock.Any(), gomock.Any()).Times(0)

	server := newTestServer(t, store)
	rec := adminRequest(t, server, http.MethodPost, fmt.Sprintf("/branches/%d/commands", buBranchID), map[string]any{
		"type":    "update_settings",
		"payload": map[string]any{"tax_rate": 5},
	})
	require.Equal(t, http.StatusConflict, rec.Code)
}

func TestCreateBranchCommand_RemoteReturnGoesToTheSellingRegister(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)

	store.EXPECT().GetBranch(gomock.Any(), buBranchID).Return(db.Branch{ID: buBranchID}, nil)
	store.EXPECT().
		GetBranchInvoiceByClientRef(gomock.Any(), db.GetBranchInvoiceByClientRefParams{BranchID: buBranchID, ClientRef: "sale-1"}).
		Return(db.BranchInvoice{ID: 3, BranchID: buBranchID, RegisterID: 22}, nil)
	store.EXPECT().ListActiveBranchRegisterIDs(gomock.Any(), gomock.Any()).Times(0)
	store.EXPECT().
		CreateBranchCommand(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, arg db.CreateBranchCommandParams) (db.BranchCommand, error) {
			require.Equal(t, int64(22), arg.RegisterID)
			return db.BranchCommand{ID: 1, RegisterID: arg.RegisterID}, nil
		})

	server := newTestServer(t, store)
	rec := adminRequest(t, server, http.MethodPost, fmt.Sprintf("/branches/%d/commands", buBranchID), map[string]any{
		"type":    "remote_return_invoice",
		"payload": map[string]any{"branch_invoice_id": 3, "client_ref": "sale-1"},
	})
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
}

func TestPutBranchSettings_StoresReportWithoutOverwritingBranchSettings(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	stubBranchAuth(store)

	store.EXPECT().
		SetBranchRegisterReportedSettings(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, arg db.SetBranchRegisterReportedSettingsParams) error {
			require.Equal(t, testRegisterID, arg.ID)
			require.Contains(t, string(arg.ReportedSettings.RawMessage), "Local Tax Shop")
			return nil
		})
	store.EXPECT().
		GetBranchSettings(gomock.Any(), testBranchID).
		Return(db.BranchSetting{BranchID: testBranchID, MarketName: "Central"}, nil)
	store.EXPECT().UpsertBranchSettings(gomock.Any(), gomock.Any()).Times(0)

	server := newTestServer(t, store)
	rec := httptest.NewRecorder()
	server.router.ServeHTTP(rec, newBranchRequest(t, http.MethodPut, "/branch/settings", map[string]any{
		"market_name": "Local Tax Shop",
	}))
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	var got struct {
		BranchCode   string `json:"branch_code"`
		RegisterCode string `json:"register_code"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	require.Equal(t, testBranchCode, got.BranchCode)
	require.Equal(t, testRegisterCode, got.RegisterCode)
}

func TestPutBranchSettings_FirstReportSeedsBranchSettings(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	stubBranchAuth(store)

	store.EXPECT().SetBranchRegisterReportedSettings(gomock.Any(), gomock.Any()).Return(nil)
	store.EXPECT().GetBranchSettings(gomock.Any(), testBranchID).Return(db.BranchSetting{}, sql.ErrNoRows)
	store.EXPECT().
		UpsertBranchSettings(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, arg db.UpsertBranchSettingsParams) (db.BranchSetting, error) {
			require.Equal(t, "First Shop", arg.MarketName)
			require.Equal(t, "allow", arg.OversellMode)
			return db.BranchSetting{BranchID: testBranchID}, nil
		})

	server := newTestServer(t, store)
	rec := httptest.NewRecorder()
	server.router.ServeHTTP(rec, newBranchRequest(t, http.MethodPut, "/branch/settings", map[string]any{
		"market_name": "First Shop",
	}))
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
}

func TestCreateBranchRegister_QueuesBranchSettingsAndReturnsKeyOnce(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)

	store.EXPECT().GetBranch(gomock.Any(), buBranchID).Return(db.Branch{ID: buBranchID}, nil)
	store.EXPECT().
		CreateBranchRegister(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, arg db.CreateBranchRegisterParams) (db.BranchRegister, error) {
			require.Equal(t, "CT2", arg.Code, "codes are upper-cased")
			require.NotEmpty(t, arg.ApiKeyHash)
			return db.BranchRegister{ID: 22, BranchID: arg.BranchID, Code: arg.Code, IsActive: true}, nil
		})
	store.EXPECT().
		GetBranchSettings(gomock.Any(), buBranchID).
		Return(db.BranchSetting{BranchID: buBranchID, TaxRate: 11, PrinterID: "EPSON-1"}, nil)
	store.EXPECT().
		CreateBranchCommand(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, arg db.CreateBranchCommandParams) (db.BranchCommand, error) {
			require.Equal(t, int64(22), arg.RegisterID)
			require.Equal(t, "update_settings", arg.Type)
			var payload map[string]any
			require.NoError(t, json.Unmarshal(arg.Payload, &payload))
			require.Equal(t, float64(11), payload["tax_rate"])
			require.NotContains(t, payload, "printer_id")
			require.NotContains(t, payload, "branch_id")
			return db.BranchCommand{ID: 1}, nil
		})

	server := newTestServer(t, store)
	rec := adminRequest(t, server, http.MethodPost, fmt.Sprintf("/branches/%d/registers", buBranchID), map[string]any{
		"code": "ct2", "name": "Second till",
	})
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	var got struct {
		Register branchRegisterResponse `json:"register"`
		APIKey   string                 `json:"api_key"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	require.Equal(t, "CT2", got.Register.Code)
	require.NotEmpty(t, got.APIKey)
	require.NotContains(t, rec.Body.String(), "api_key_hash")
}

func TestCreateBranchRegister_RejectsBadOrTakenCode(t *testing.T) {
	t.Run("bad code", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		store := mockdb.NewMockStore(ctrl)
		store.EXPECT().CreateBranchRegister(gomock.Any(), gomock.Any()).Times(0)

		server := newTestServer(t, store)
		rec := adminRequest(t, server, http.MethodPost, fmt.Sprintf("/branches/%d/registers", buBranchID), map[string]any{
			"code": "CT-2",
		})
		require.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("taken code", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		store := mockdb.NewMockStore(ctrl)
		store.EXPECT().GetBranch(gomock.Any(), buBranchID).Return(db.Branch{ID: buBranchID}, nil)
		store.EXPECT().
			CreateBranchRegister(gomock.Any(), gomock.Any()).
			Return(db.BranchRegister{}, &pq.Error{Code: "23505"})

		server := newTestServer(t, store)
		rec := adminRequest(t, server, http.MethodPost, fmt.Sprintf("/branches/%d/registers", buBranchID), map[string]any{
			"code": "CT1",
		})
		require.Equal(t, http.StatusConflict, rec.Code)
	})
}

func TestListBranchRegisters_ReportedSettingsArePlainJSON(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	store.EXPECT().
		ListBranchRegisters(gomock.Any(), buBranchID).
		Return([]db.BranchRegister{
			{ID: 1, BranchID: buBranchID, Code: "CT", ReportedSettings: pqtype.NullRawMessage{RawMessage: []byte(`{"printer_id":"EPSON-A"}`), Valid: true}},
			{ID: 2, BranchID: buBranchID, Code: "CT2"},
		}, nil)

	server := newTestServer(t, store)
	rec := adminRequest(t, server, http.MethodGet, fmt.Sprintf("/branches/%d/registers", buBranchID), nil)
	require.Equal(t, http.StatusOK, rec.Code)

	var got struct {
		Registers []struct {
			Code             string         `json:"code"`
			ReportedSettings map[string]any `json:"reported_settings"`
		} `json:"registers"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	require.Equal(t, "EPSON-A", got.Registers[0].ReportedSettings["printer_id"])
	require.Nil(t, got.Registers[1].ReportedSettings)
}
