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

// The client box sends client=, trimmed, to the page, its count and the export alike.
func TestListBranchInvoicesClientFilter(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	store.EXPECT().ListBranchInvoicesPage(gomock.Any(), db.ListBranchInvoicesPageParams{
		Client: "Maya", Search: "SA/CT", PageSize: 14,
	}).Return([]db.BranchInvoice{}, nil)
	store.EXPECT().CountBranchInvoicesFiltered(gomock.Any(), db.CountBranchInvoicesFilteredParams{
		Client: "Maya", Search: "SA/CT",
	}).Return(int64(4), nil)

	server := newTestServer(t, store)
	recorder := httptest.NewRecorder()
	request, err := http.NewRequest(http.MethodGet, "/branch_invoices?page_size=14&client=%20Maya%20&search=SA/CT", nil)
	require.NoError(t, err)
	addAuthorization(t, request, server.tokenMaker, authorizationTypeBearer, "user", time.Minute)

	server.router.ServeHTTP(recorder, request)
	require.Equal(t, http.StatusOK, recorder.Code)
	require.JSONEq(t, `{"branch_invoices":[],"total":4}`, recorder.Body.String())
}

func TestExportBranchInvoicesClientFilter(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	store.EXPECT().ListBranchInvoicesPage(gomock.Any(), db.ListBranchInvoicesPageParams{
		Client: "70123456", ClientDigits: "70123456", PageSize: exportBranchInvoicesLimit,
	}).Return([]db.BranchInvoice{}, nil)
	store.EXPECT().ListBranches(gomock.Any()).Return([]db.Branch{}, nil)
	store.EXPECT().ListAllBranchRegisters(gomock.Any()).Return([]db.BranchRegister{}, nil)

	server := newTestServer(t, store)
	recorder := httptest.NewRecorder()
	request, err := http.NewRequest(http.MethodGet, "/branch_invoices/export?client=70123456", nil)
	require.NoError(t, err)
	addAuthorization(t, request, server.tokenMaker, authorizationTypeBearer, "user", time.Minute)

	server.router.ServeHTTP(recorder, request)
	require.Equal(t, http.StatusOK, recorder.Code)
}

// The filter dialog's product box: code/name/barcode, a match mode and a total
// range reach the page, its count and the export alike.
func TestListBranchInvoicesProductAndTotalFilters(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	min, max := sql.NullInt64{Int64: -500, Valid: true}, sql.NullInt64{Int64: 12000, Valid: true}
	store.EXPECT().ListBranchInvoicesPage(gomock.Any(), db.ListBranchInvoicesPageParams{
		ProductPattern: "SH-12%", ProductField: "code", TotalMin: min, TotalMax: max, PageSize: 14,
	}).Return([]db.BranchInvoice{}, nil)
	store.EXPECT().CountBranchInvoicesFiltered(gomock.Any(), db.CountBranchInvoicesFilteredParams{
		ProductPattern: "SH-12%", ProductField: "code", TotalMin: min, TotalMax: max,
	}).Return(int64(1), nil)

	server := newTestServer(t, store)
	recorder := httptest.NewRecorder()
	request, err := http.NewRequest(http.MethodGet,
		"/branch_invoices?page_size=14&product=%20SH-12%20&product_field=code&product_match=starts&total_min=-500&total_max=12000", nil)
	require.NoError(t, err)
	addAuthorization(t, request, server.tokenMaker, authorizationTypeBearer, "user", time.Minute)

	server.router.ServeHTTP(recorder, request)
	require.Equal(t, http.StatusOK, recorder.Code)
	require.JSONEq(t, `{"branch_invoices":[],"total":1}`, recorder.Body.String())
}

func TestExportBranchInvoicesProductFilter(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	store.EXPECT().ListBranchInvoicesPage(gomock.Any(), db.ListBranchInvoicesPageParams{
		ProductPattern: "%shirt%", PageSize: exportBranchInvoicesLimit,
	}).Return([]db.BranchInvoice{}, nil)
	store.EXPECT().ListBranches(gomock.Any()).Return([]db.Branch{}, nil)
	store.EXPECT().ListAllBranchRegisters(gomock.Any()).Return([]db.BranchRegister{}, nil)

	server := newTestServer(t, store)
	recorder := httptest.NewRecorder()
	request, err := http.NewRequest(http.MethodGet, "/branch_invoices/export?product=shirt", nil)
	require.NoError(t, err)
	addAuthorization(t, request, server.tokenMaker, authorizationTypeBearer, "user", time.Minute)

	server.router.ServeHTTP(recorder, request)
	require.Equal(t, http.StatusOK, recorder.Code)
}

// A field or mode the dialog never sends is refused, not silently ignored.
func TestListBranchInvoicesRejectsBadProductOptions(t *testing.T) {
	for _, query := range []string{
		"product=a&product_field=color",
		"product=a&product_match=regex",
		"total_min=ten",
	} {
		ctrl := gomock.NewController(t)
		store := mockdb.NewMockStore(ctrl)
		server := newTestServer(t, store)
		recorder := httptest.NewRecorder()
		request, err := http.NewRequest(http.MethodGet, "/branch_invoices?"+query, nil)
		require.NoError(t, err)
		addAuthorization(t, request, server.tokenMaker, authorizationTypeBearer, "user", time.Minute)
		server.router.ServeHTTP(recorder, request)
		require.Equal(t, http.StatusBadRequest, recorder.Code, query)
	}
}

// A field with no term is no filter at all.
func TestListBranchInvoicesFieldWithoutTerm(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	store.EXPECT().ListBranchInvoicesPage(gomock.Any(), db.ListBranchInvoicesPageParams{PageSize: 14}).Return([]db.BranchInvoice{}, nil)
	store.EXPECT().CountBranchInvoicesFiltered(gomock.Any(), db.CountBranchInvoicesFilteredParams{}).Return(int64(0), nil)

	server := newTestServer(t, store)
	recorder := httptest.NewRecorder()
	request, err := http.NewRequest(http.MethodGet, "/branch_invoices?page_size=14&product=%20%20&product_field=name&product_match=exact", nil)
	require.NoError(t, err)
	addAuthorization(t, request, server.tokenMaker, authorizationTypeBearer, "user", time.Minute)
	server.router.ServeHTTP(recorder, request)
	require.Equal(t, http.StatusOK, recorder.Code)
}

func TestLikePattern(t *testing.T) {
	cases := []struct{ term, match, want string }{
		{"SH12", "", "%SH12%"},
		{"SH12", "contains", "%SH12%"},
		{"SH12", "starts", "SH12%"},
		{"SH12", "ends", "%SH12"},
		{"SH12", "exact", "SH12"},
		{"  SH12 ", "exact", "SH12"},
		{"", "starts", ""},
		{"   ", "contains", ""},
		{"50%", "contains", `%50\%%`},
		{"A_B", "exact", `A\_B`},
		{`C:\x`, "starts", `C:\\x%`},
	}
	for _, c := range cases {
		require.Equal(t, c.want, likePattern(c.term, c.match), "%q %q", c.term, c.match)
	}
}
