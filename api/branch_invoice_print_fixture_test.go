package api

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	mockdb "github.com/mhdna/kashi/db/mock"
	db "github.com/mhdna/kashi/db/sqlc"
	"github.com/stretchr/testify/require"
)

// The "Print on till" button (ui/src/utils/invoicePrint.js) decides from a
// row of GET /branch_invoices: its client_ref and historical fields. This
// writes real rows, as the API sends them, to a fixture the UI test reads,
// so renaming either field on one side fails a test on the other.
func TestBranchInvoiceRowsFixtureForPrintOnTill(t *testing.T) {
	when := time.Date(2026, 11, 7, 18, 52, 15, 0, time.UTC)
	rows := []db.BranchInvoice{
		// RS/CT/2026/00068, printable.
		{ID: 123302, BranchID: 1, RegisterID: 3, ClientRef: "8912abdf-ed49-46cf-be63-84b63ce7f5f2", Kind: "return",
			BranchInvoiceCode: "RS/CT/2026/00068", Subtotal: -4100, DiscountedTotal: -4100, GrandTotal: -4100, OccurredAt: when, ReceivedAt: when},
		// An old-POS import: never printable.
		{ID: 5000, BranchID: 1, RegisterID: 9, ClientRef: "csv-import:CN-1", Kind: "sales",
			BranchInvoiceCode: "CN-1", Subtotal: 2500, DiscountedTotal: 2500, GrandTotal: 2500, Historical: true, OccurredAt: when, ReceivedAt: when},
	}

	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	store.EXPECT().ListBranchInvoicesPage(gomock.Any(), gomock.Any()).Return(rows, nil)
	store.EXPECT().CountBranchInvoicesFiltered(gomock.Any(), gomock.Any()).Return(int64(len(rows)), nil)

	server := newTestServer(t, store)
	rec := adminRequest(t, server, http.MethodGet, "/branch_invoices?page_id=0&page_size=10", nil)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	var body struct {
		BranchInvoices []map[string]any `json:"branch_invoices"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Len(t, body.BranchInvoices, 2)
	require.Equal(t, rows[0].ClientRef, body.BranchInvoices[0]["client_ref"])
	require.Equal(t, true, body.BranchInvoices[1]["historical"])

	out, err := json.MarshalIndent(body.BranchInvoices, "", "  ")
	require.NoError(t, err)
	path := filepath.Join("..", "ui", "src", "utils", "testdata", "branch_invoice_rows.json")
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, append(out, '\n'), 0o644))
}
