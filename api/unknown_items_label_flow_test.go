package api

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"image"
	"image/jpeg"
	"math/rand"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	db "github.com/mhdna/kashi/db/sqlc"
	"github.com/mhdna/kashi/util"
	"github.com/stretchr/testify/require"
)

// fakeLabelReader answers every picture with the same reading; gate, when
// set, holds the answer until it's closed (to land the sale first).
type fakeLabelReader struct {
	mu      sync.Mutex
	reading labelReading
	err     error
	gate    chan struct{}
	calls   int
}

func (f *fakeLabelReader) Read(ctx context.Context, _ []byte, _ string) (labelReading, error) {
	f.mu.Lock()
	gate := f.gate
	f.calls++
	f.mu.Unlock()
	if gate != nil {
		<-gate
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.reading, f.err
}

// The label reader end to end, against a real database: the till sells
// unknown items, the phone sends their pictures, the model's reading (faked
// here) comes back, and once the sale lands each pictured line is a real
// product, found by the next till that scans its barcode.
func TestUnknownItemsLabelFlow(t *testing.T) {
	config, err := util.LoadConfig("..")
	require.NoError(t, err)
	conn, err := sql.Open(config.DBDriver, config.DBSource)
	require.NoError(t, err)
	defer conn.Close()
	if err := conn.Ping(); err != nil {
		t.Skipf("no database: %v", err)
	}
	store := db.NewStore(conn)
	ctx := context.Background()

	const key = "label-flow-key"
	hash, err := util.HashPassword(key)
	require.NoError(t, err)
	branch, err := store.CreateBranchTx(ctx, db.CreateBranchTxParams{
		Name: util.RandomString(10), Code: util.RandomString(6), RegisterAPIKeyHash: hash,
	})
	require.NoError(t, err)

	reader := &fakeLabelReader{}
	server := newTestServer(t, store)
	server.config.ExpenseImageStorageDir = t.TempDir()
	server.labelReader = reader

	do := func(req *http.Request) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		server.router.ServeHTTP(rec, req)
		return rec
	}
	till := func(method, path string, body any) *httptest.ResponseRecorder {
		raw, err := json.Marshal(body)
		require.NoError(t, err)
		req, err := http.NewRequest(method, path, bytes.NewReader(raw))
		require.NoError(t, err)
		req.Header.Set(branchCodeHeaderKey, branch.Register.Code)
		req.Header.Set(branchKeyHeaderKey, key)
		req.Header.Set("Content-Type", "application/json")
		rec := do(req)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		return rec
	}
	var jpg bytes.Buffer
	require.NoError(t, jpeg.Encode(&jpg, image.NewRGBA(image.Rect(0, 0, 32, 24)), nil))
	picture := func(token, slotRef string) {
		var form bytes.Buffer
		w := multipart.NewWriter(&form)
		part, err := w.CreatePart(textproto.MIMEHeader{
			"Content-Disposition": {`form-data; name="image"; filename="label.jpg"`},
			"Content-Type":        {"image/jpeg"},
		})
		require.NoError(t, err)
		_, _ = part.Write(jpg.Bytes())
		require.NoError(t, w.Close())
		req := httptest.NewRequest(http.MethodPost, "/unknown_uploads/"+token+"/slots/"+slotRef+"/image", &form)
		req.Header.Set("Content-Type", w.FormDataContentType())
		rec := do(req)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	}
	copyPicture := func(token, from, to string) {
		body, _ := json.Marshal(map[string]string{"from_slot_ref": from})
		req := httptest.NewRequest(http.MethodPost, "/unknown_uploads/"+token+"/slots/"+to+"/copy", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		require.Equal(t, http.StatusOK, do(req).Code)
	}
	// open starts a sale's picture session; units on one line share line_ref.
	open := func(sessionRef string, slots []map[string]any) string {
		var out struct {
			UploadToken string `json:"upload_token"`
		}
		require.NoError(t, json.Unmarshal(till(http.MethodPost, "/branch/unknown_sessions", map[string]any{
			"client_ref": sessionRef, "slots": slots,
		}).Body.Bytes(), &out))
		return out.UploadToken
	}
	land := func(sessionRef string, unitPrice, qty int64) {
		total := unitPrice * qty
		till(http.MethodPost, "/branch/sales_invoices", map[string]any{
			"client_ref": uuid.NewString(), "branch_invoice_code": "SA/L/" + util.RandomString(6),
			"branch_cashbox_account_id": 1, "branch_shift_id": 1, "branch_inventory_id": 1,
			"grand_total": total, "sub_total": total, "discounted_total": total,
			"occurred_at":         time.Now().UTC().Format(time.RFC3339),
			"unknown_session_ref": sessionRef,
			"items": []map[string]any{{
				"branch_product_id": 5, "barcode": db.UnknownItemBarcode,
				"unit_price": unitPrice, "line_total": total, "quantity": qty,
			}},
		})
	}
	session := func(sessionRef string) db.UnknownItemSession {
		s, err := store.GetUnknownItemSessionByRef(ctx, db.GetUnknownItemSessionByRefParams{BranchID: branch.Branch.ID, ClientRef: sessionRef})
		require.NoError(t, err)
		return s
	}
	slotsOf := func(sessionRef string) []db.UnknownItemSlot {
		slots, err := store.ListUnknownItemSlotsForSession(ctx, session(sessionRef).ID)
		require.NoError(t, err)
		return slots
	}
	waitResolved := func(sessionRef string) []db.UnknownItemSlot {
		var slots []db.UnknownItemSlot
		require.Eventually(t, func() bool {
			slots = slotsOf(sessionRef)
			for _, s := range slots {
				if !s.ResolvedAt.Valid {
					return false
				}
			}
			return true
		}, 5*time.Second, 20*time.Millisecond)
		return slots
	}
	waitRead := func(sessionRef string) {
		require.Eventually(t, func() bool {
			for _, s := range slotsOf(sessionRef) {
				img, err := store.GetUnknownItemImage(ctx, s.ImageID.Int64)
				if err != nil || img.LabelStatus == "" {
					return false
				}
			}
			return true
		}, 5*time.Second, 20*time.Millisecond)
	}
	defaultPrice := func(variantID int64) int64 {
		var price int64
		require.NoError(t, conn.QueryRow(`SELECT pli.price FROM price_list_items pli
			JOIN price_lists pl ON pl.id = pli.price_list_id
			WHERE pli.product_variant_id = $1 AND pl.is_default`, variantID).Scan(&price))
		return price
	}
	unknownLines := func(sessionRef string) int {
		var n int
		require.NoError(t, conn.QueryRow(`SELECT COUNT(*) FROM branch_invoice_items
			WHERE branch_invoice_id = $1 AND unknown`, session(sessionRef).BranchInvoiceID.Int64).Scan(&n))
		return n
	}

	// A fresh barcode and code each run: the database outlives the test.
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	barcode := withCheckDigit(fmt.Sprintf("80%010d", rng.Int63n(1e10)))
	code := "CS" + util.RandomString(5) + "-"

	// 1. The user's label: two units on one line, one picture, the reading
	// back before the sale lands. The cashier charged $10, the label says $12.
	reader.reading = labelReading{Name: "cup set 6pcs basic-Glasses", Code: code, Price: floatPtr(12), Barcode: barcode}
	first := uuid.NewString()
	token := open(first, []map[string]any{
		{"slot_ref": "1", "label": "Unknown item", "unit_price": 1000, "line_ref": "a"},
		{"slot_ref": "2", "label": "Unknown item", "unit_price": 1000, "line_ref": "a"},
	})
	picture(token, "1")
	copyPicture(token, "1", "2")
	waitRead(first)
	require.Zero(t, slotsOf(first)[0].ResolvedVariantID.Int64, "nothing is made before the sale lands")
	land(first, 1000, 2)

	slots := waitResolved(first)
	require.Equal(t, slots[0].ResolvedVariantID, slots[1].ResolvedVariantID, "one picture, one product")
	for _, s := range slots {
		require.True(t, s.AutoResolved)
	}
	variant, err := store.GetProductVariantByBarcode(ctx, barcode)
	require.NoError(t, err)
	require.Equal(t, variant.ID, slots[0].ResolvedVariantID.Int64)
	product, err := store.GetProductByCode(ctx, code)
	require.NoError(t, err)
	require.Equal(t, "cup set 6pcs basic-Glasses", product.Name)
	require.Equal(t, db.ProductSourceLabelPhoto, product.Source, "tagged From photo")
	require.Equal(t, int64(1000), defaultPrice(variant.ID), "the cashier's price, not the label's")
	require.Zero(t, unknownLines(first), "the sold line is now the real product")

	// The Unknown items page sees the reading and can warn about the price.
	items, err := store.ListUnknownItems(ctx, db.ListUnknownItemsParams{Resolved: true, Limit: 200})
	require.NoError(t, err)
	found := false
	for _, item := range items {
		if item.SessionID == session(first).ID {
			found = true
			require.Equal(t, "read", item.LabelStatus)
			require.Equal(t, barcode, item.LabelBarcode)
			require.Equal(t, int64(1200), item.LabelPrice.Int64)
			require.Equal(t, db.ProductSourceLabelPhoto, item.ResolvedSource)
			require.True(t, item.AutoResolved)
			require.True(t, unknownItemViews([]db.ListUnknownItemsRow{item})[0].LabelPriceMismatch)
		}
	}
	require.True(t, found)

	// 2. Another till scans the same barcode, the reading lands only after
	// the sale (held back), and a different name is read: the unit goes to
	// the product that already has the barcode, no second product.
	gate := make(chan struct{})
	reader.mu.Lock()
	reader.gate = gate
	reader.reading = labelReading{Name: "glasses", Code: "OTHER-" + util.RandomString(4), Price: floatPtr(10), Barcode: ""}
	reader.mu.Unlock()
	second := uuid.NewString()
	token = open(second, []map[string]any{
		{"slot_ref": "1", "label": "Unknown item", "unit_price": 1000, "line_ref": "b", "barcode": barcode},
	})
	picture(token, "1")
	land(second, 1000, 1)
	close(gate)
	slots = waitResolved(second)
	require.Equal(t, variant.ID, slots[0].ResolvedVariantID.Int64)
	_, err = store.GetProductByCode(ctx, reader.reading.Code)
	require.ErrorIs(t, err, sql.ErrNoRows, "no second product for a known barcode")

	// 3. A label with no barcode, and nothing scanned: left for a person.
	reader.mu.Lock()
	reader.gate = nil
	reader.reading = labelReading{Name: "plate", Code: "PL-" + util.RandomString(4), Price: floatPtr(3)}
	reader.mu.Unlock()
	third := uuid.NewString()
	token = open(third, []map[string]any{{"slot_ref": "1", "label": "Unknown item", "unit_price": 300}})
	picture(token, "1")
	waitRead(third)
	land(third, 300, 1)
	time.Sleep(200 * time.Millisecond)
	require.False(t, slotsOf(third)[0].ResolvedAt.Valid)
	require.Equal(t, 1, unknownLines(third))

	// 4. The model fails: the picture says so and the unit waits for a person.
	reader.mu.Lock()
	reader.err = fmt.Errorf("groq: 503 Service Unavailable")
	reader.mu.Unlock()
	fourth := uuid.NewString()
	token = open(fourth, []map[string]any{{"slot_ref": "1", "label": "Unknown item", "unit_price": 500}})
	picture(token, "1")
	waitRead(fourth)
	img, err := store.GetUnknownItemImage(ctx, slotsOf(fourth)[0].ImageID.Int64)
	require.NoError(t, err)
	require.Equal(t, "failed", img.LabelStatus)
	require.Contains(t, img.LabelError, "503")
	land(fourth, 500, 1)
	time.Sleep(200 * time.Millisecond)
	require.False(t, slotsOf(fourth)[0].ResolvedAt.Valid)

	// 5. The label's code is already another product's: the barcode becomes
	// the code, the product is still made.
	barcode5 := withCheckDigit(fmt.Sprintf("80%010d", rng.Int63n(1e10)))
	reader.mu.Lock()
	reader.err = nil
	reader.reading = labelReading{Name: "bowl", Code: code, Barcode: barcode5}
	reader.mu.Unlock()
	fifth := uuid.NewString()
	token = open(fifth, []map[string]any{{"slot_ref": "1", "label": "Unknown item", "unit_price": 700}})
	picture(token, "1")
	waitRead(fifth)
	land(fifth, 700, 1)
	waitResolved(fifth)
	bowl, err := store.GetProductByCode(ctx, barcode5)
	require.NoError(t, err)
	require.Equal(t, "bowl", bowl.Name)
}
