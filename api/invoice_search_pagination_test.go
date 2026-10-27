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

func TestListSalesInvoicesSearchPagination(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	store.EXPECT().ListInvoicesPage(gomock.Any(), db.ListInvoicesPageParams{
		Search: "INV-42", PageOffset: 14, PageSize: 14,
	}).Return([]db.Invoice{}, nil)
	store.EXPECT().CountInvoicesFiltered(gomock.Any(), "INV-42").Return(int64(3), nil)

	server := newTestServer(t, store)
	recorder := httptest.NewRecorder()
	request, err := http.NewRequest(http.MethodGet, "/sales_invoices?page_size=14&page_id=14&search=INV-42", nil)
	require.NoError(t, err)
	addAuthorization(t, request, server.tokenMaker, authorizationTypeBearer, "user", time.Minute)

	server.router.ServeHTTP(recorder, request)
	require.Equal(t, http.StatusOK, recorder.Code)
	require.JSONEq(t, `{"sales_invoices":[],"total":3}`, recorder.Body.String())
}

func TestListBranchInvoicesSearchPagination(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	branchID := sql.NullInt64{Int64: 7, Valid: true}
	store.EXPECT().ListBranchInvoicesPage(gomock.Any(), db.ListBranchInvoicesPageParams{
		BranchID: branchID, Search: "Maya", PageOffset: 28, PageSize: 14,
	}).Return([]db.BranchInvoice{}, nil)
	store.EXPECT().CountBranchInvoicesFiltered(gomock.Any(), db.CountBranchInvoicesFilteredParams{
		BranchID: branchID, Search: "Maya",
	}).Return(int64(2), nil)

	server := newTestServer(t, store)
	recorder := httptest.NewRecorder()
	request, err := http.NewRequest(http.MethodGet, "/branch_invoices?page_size=14&page_id=28&branch_id=7&search=Maya", nil)
	require.NoError(t, err)
	addAuthorization(t, request, server.tokenMaker, authorizationTypeBearer, "user", time.Minute)

	server.router.ServeHTTP(recorder, request)
	require.Equal(t, http.StatusOK, recorder.Code)
	require.JSONEq(t, `{"branch_invoices":[],"total":2}`, recorder.Body.String())
}

func TestListBranchInvoicesFilters(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	from := dayStart(time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC))
	to := dayStart(time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)) // 31 Oct is inclusive
	store.EXPECT().ListBranchInvoicesPage(gomock.Any(), db.ListBranchInvoicesPageParams{
		RegisterID: sql.NullInt64{Int64: 3, Valid: true},
		Kind:       sql.NullString{String: "return", Valid: true},
		Status:     sql.NullString{String: "cancelled", Valid: true},
		DateFrom:   sql.NullTime{Time: from, Valid: true},
		DateTo:     sql.NullTime{Time: to, Valid: true},
		PageSize:   14,
	}).Return([]db.BranchInvoice{}, nil)
	store.EXPECT().CountBranchInvoicesFiltered(gomock.Any(), db.CountBranchInvoicesFilteredParams{
		RegisterID: sql.NullInt64{Int64: 3, Valid: true},
		Kind:       sql.NullString{String: "return", Valid: true},
		Status:     sql.NullString{String: "cancelled", Valid: true},
		DateFrom:   sql.NullTime{Time: from, Valid: true},
		DateTo:     sql.NullTime{Time: to, Valid: true},
	}).Return(int64(0), nil)

	server := newTestServer(t, store)
	recorder := httptest.NewRecorder()
	request, err := http.NewRequest(http.MethodGet,
		"/branch_invoices?page_size=14&register_id=3&kind=return&status=cancelled&date_from=2026-10-01&date_to=2026-10-31", nil)
	require.NoError(t, err)
	addAuthorization(t, request, server.tokenMaker, authorizationTypeBearer, "user", time.Minute)

	server.router.ServeHTTP(recorder, request)
	require.Equal(t, http.StatusOK, recorder.Code)
}

func TestExportBranchInvoicesXLSX(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	store.EXPECT().ListBranchInvoicesPage(gomock.Any(), db.ListBranchInvoicesPageParams{PageSize: exportBranchInvoicesLimit}).
		Return([]db.BranchInvoice{{ID: 1, BranchID: 7, RegisterID: 3, Kind: "sales", BranchInvoiceCode: "A-1", GrandTotal: 2500}}, nil)
	store.EXPECT().ListBranches(gomock.Any()).Return([]db.Branch{{ID: 7, Name: "Centro"}}, nil)
	store.EXPECT().ListAllBranchRegisters(gomock.Any()).Return([]db.BranchRegister{{ID: 3, BranchID: 7, Code: "R1"}}, nil)

	server := newTestServer(t, store)
	recorder := httptest.NewRecorder()
	request, err := http.NewRequest(http.MethodGet, "/branch_invoices/export", nil)
	require.NoError(t, err)
	addAuthorization(t, request, server.tokenMaker, authorizationTypeBearer, "user", time.Minute)

	server.router.ServeHTTP(recorder, request)
	require.Equal(t, http.StatusOK, recorder.Code)
	require.Contains(t, recorder.Header().Get("Content-Type"), "spreadsheetml")
	require.Equal(t, "PK", recorder.Body.String()[:2]) // an xlsx is a zip
}
