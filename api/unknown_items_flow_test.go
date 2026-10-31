package api

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"image"
	"image/color"
	"image/jpeg"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"strconv"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	db "github.com/mhdna/kashi/db/sqlc"
	"github.com/mhdna/kashi/util"
	"github.com/stretchr/testify/require"
)

// The whole unknown-item journey over HTTP against a real database: the till
// opens a session, a phone (no login) takes one picture and reuses it for the
// second unit, a manager's pass covers the third, and the sale then lands with
// its session ref and shows up for matching.
func TestUnknownItemsFlow(t *testing.T) {
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

	const key = "unknown-flow-key"
	hash, err := util.HashPassword(key)
	require.NoError(t, err)
	branch, err := store.CreateBranchTx(ctx, db.CreateBranchTxParams{
		Name: util.RandomString(10), Code: util.RandomString(6), RegisterAPIKeyHash: hash,
	})
	require.NoError(t, err)

	server := newTestServer(t, store)
	server.config.ExpenseImageStorageDir = t.TempDir()

	do := func(req *http.Request) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		server.router.ServeHTTP(rec, req)
		return rec
	}
	tillRequest := func(method, path string, body any) *http.Request {
		raw, err := json.Marshal(body)
		require.NoError(t, err)
		req, err := http.NewRequest(method, path, bytes.NewReader(raw))
		require.NoError(t, err)
		req.Header.Set(branchCodeHeaderKey, branch.Register.Code)
		req.Header.Set(branchKeyHeaderKey, key)
		req.Header.Set("Content-Type", "application/json")
		return req
	}
	decode := func(rec *httptest.ResponseRecorder, v any) {
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), v))
	}

	// 1. The till opens a session with a slot per unit.
	sessionRef := uuid.NewString()
	var opened struct {
		UploadToken string `json:"upload_token"`
		Complete    bool   `json:"complete"`
	}
	decode(do(tillRequest(http.MethodPost, "/branch/unknown_sessions", map[string]any{
		"client_ref": sessionRef,
		"slots": []map[string]any{
			{"slot_ref": "1", "label": "Unknown item", "unit_price": 1000},
			{"slot_ref": "2", "label": "Unknown item", "unit_price": 1000},
			{"slot_ref": "3", "label": "Unknown item", "unit_price": 1000},
		},
	})), &opened)
	require.NotEmpty(t, opened.UploadToken)
	require.False(t, opened.Complete)
	base := "/unknown_uploads/" + opened.UploadToken

	page := do(httptest.NewRequest(http.MethodGet, base, nil))
	require.Equal(t, http.StatusOK, page.Code)
	require.Contains(t, page.Body.String(), "Unknown items")

	// 2. The phone takes a picture for unit 1...
	var jpg bytes.Buffer
	pic := image.NewRGBA(image.Rect(0, 0, 64, 48))
	for x := 0; x < 64; x++ {
		pic.Set(x, 10, color.RGBA{R: 200, A: 255})
	}
	require.NoError(t, jpeg.Encode(&jpg, pic, nil))
	// The phone page sends the shrunk picture as a multipart image/jpeg part.
	var form bytes.Buffer
	writer := multipart.NewWriter(&form)
	part, err := writer.CreatePart(textproto.MIMEHeader{
		"Content-Disposition": {`form-data; name="image"; filename="item.jpg"`},
		"Content-Type":        {"image/jpeg"},
	})
	require.NoError(t, err)
	_, err = part.Write(jpg.Bytes())
	require.NoError(t, err)
	require.NoError(t, writer.Close())
	upload := httptest.NewRequest(http.MethodPost, base+"/slots/1/image", &form)
	upload.Header.Set("Content-Type", writer.FormDataContentType())
	rec := do(upload)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	// ...and reuses it for unit 2 with "same as #1".
	copyBody, _ := json.Marshal(map[string]string{"from_slot_ref": "1"})
	copyReq := httptest.NewRequest(http.MethodPost, base+"/slots/2/copy", bytes.NewReader(copyBody))
	copyReq.Header.Set("Content-Type", "application/json")
	require.Equal(t, http.StatusOK, do(copyReq).Code)

	status := func() (complete bool, slots []unknownSlotView) {
		var out struct {
			Complete bool              `json:"complete"`
			Slots    []unknownSlotView `json:"slots"`
		}
		decode(do(tillRequest(http.MethodGet, "/branch/unknown_sessions/status?client_ref="+sessionRef, nil)), &out)
		return out.Complete, out.Slots
	}
	complete, slots := status()
	require.False(t, complete, "unit 3 still has no picture")
	require.Equal(t, slots[0].ImageID, slots[1].ImageID, "'same as' shares the picture")

	// Both the original and a thumbnail of the picture can be fetched back.
	for _, query := range []string{"", "?w=480", "?w=160"} {
		pic := do(httptest.NewRequest(http.MethodGet, base+"/images/"+strconv.FormatInt(slots[0].ImageID, 10)+query, nil))
		require.Equal(t, http.StatusOK, pic.Code, "picture %q: %s", query, pic.Body.String())
		require.NotZero(t, pic.Body.Len(), "picture %q", query)
	}

	// 3. The cashier asks for help; a manager's pass covers unit 3.
	decode(do(tillRequest(http.MethodPost, "/branch/unknown_sessions/help", map[string]string{"client_ref": sessionRef})), &struct{}{})
	waiting, err := store.ListWaitingUnknownItems(ctx)
	require.NoError(t, err)
	var third int64
	for _, w := range waiting {
		if w.BranchID == branch.Branch.ID && !w.ImageID.Valid && !w.WaivedAt.Valid {
			third = w.ID
			require.True(t, w.HelpRequestedAt.Valid)
		}
	}
	require.NotZero(t, third)
	n, err := store.WaiveUnknownItemSlots(ctx, db.WaiveUnknownItemSlotsParams{Column1: []int64{third}, WaivedBy: "boss"})
	require.NoError(t, err)
	require.Equal(t, int64(1), n)
	complete, _ = status()
	require.True(t, complete, "every unit now has a picture or a pass")

	// 4. The sale lands carrying the session ref.
	sale := map[string]any{
		"client_ref": uuid.NewString(), "branch_invoice_code": "SA/T/2026/1",
		"branch_cashbox_account_id": 1, "branch_shift_id": 1, "branch_inventory_id": 1,
		"grand_total": 3000, "sub_total": 3000, "discounted_total": 3000,
		"occurred_at":         time.Now().UTC().Format(time.RFC3339),
		"unknown_session_ref": sessionRef,
		"items": []map[string]any{{
			"branch_product_id": 5, "barcode": db.UnknownItemBarcode,
			"unit_price": 1000, "line_total": 3000, "quantity": 3,
		}},
	}
	decode(do(tillRequest(http.MethodPost, "/branch/sales_invoices", sale)), &struct{}{})

	// The phone's link is dead once the sale has landed.
	require.Equal(t, http.StatusNotFound, do(httptest.NewRequest(http.MethodGet, base+"/status", nil)).Code)

	// 5. Two pictured units and one allowed unit now wait to be matched.
	items, err := store.ListUnknownItems(ctx, db.ListUnknownItemsParams{Resolved: false, Limit: 50})
	require.NoError(t, err)
	mine := 0
	for _, item := range items {
		if item.BranchID == branch.Branch.ID {
			mine++
		}
	}
	require.Equal(t, 3, mine)

	// 6. Deleting pictures on the Attachments page: one from a checkout whose
	// sale never came goes, the one still waiting to be matched stays, and the
	// answer names what went so the page can drop just those.
	otherRef := uuid.NewString()
	var other struct {
		UploadToken string `json:"upload_token"`
	}
	decode(do(tillRequest(http.MethodPost, "/branch/unknown_sessions", map[string]any{
		"client_ref": otherRef,
		"slots":      []map[string]any{{"slot_ref": "1", "label": "Unknown item", "unit_price": 1000}},
	})), &other)
	var otherForm bytes.Buffer
	otherWriter := multipart.NewWriter(&otherForm)
	otherPart, err := otherWriter.CreatePart(textproto.MIMEHeader{
		"Content-Disposition": {`form-data; name="image"; filename="item.jpg"`},
		"Content-Type":        {"image/jpeg"},
	})
	require.NoError(t, err)
	_, err = otherPart.Write(jpg.Bytes())
	require.NoError(t, err)
	require.NoError(t, otherWriter.Close())
	otherUpload := httptest.NewRequest(http.MethodPost, "/unknown_uploads/"+other.UploadToken+"/slots/1/image", &otherForm)
	otherUpload.Header.Set("Content-Type", otherWriter.FormDataContentType())
	require.Equal(t, http.StatusOK, do(otherUpload).Code)
	var otherStatus struct {
		Slots []unknownSlotView `json:"slots"`
	}
	decode(do(tillRequest(http.MethodGet, "/branch/unknown_sessions/status?client_ref="+otherRef, nil)), &otherStatus)
	strayID := otherStatus.Slots[0].ImageID
	require.NotZero(t, strayID)

	var deleted struct {
		Deleted    int     `json:"deleted"`
		Kept       int     `json:"kept"`
		DeletedIDs []int64 `json:"deleted_ids"`
	}
	decode(do(adminJSONRequest(t, server, http.MethodPost, "/unknown_items/images/delete",
		map[string]any{"ids": []int64{slots[0].ImageID, strayID}})), &deleted)
	require.Equal(t, 1, deleted.Deleted)
	require.Equal(t, 1, deleted.Kept)
	require.Equal(t, []int64{strayID}, deleted.DeletedIDs)
}

// One picture covers every unit of its cart line: a line of two is done with
// a single upload, while the next line, at the same price, still waits for
// its own.
func TestUnknownItemPictureCoversItsLine(t *testing.T) {
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

	const key = "unknown-line-key"
	hash, err := util.HashPassword(key)
	require.NoError(t, err)
	branch, err := store.CreateBranchTx(ctx, db.CreateBranchTxParams{
		Name: util.RandomString(10), Code: util.RandomString(6), RegisterAPIKeyHash: hash,
	})
	require.NoError(t, err)

	server := newTestServer(t, store)
	server.config.ExpenseImageStorageDir = t.TempDir()
	do := func(req *http.Request) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		server.router.ServeHTTP(rec, req)
		return rec
	}

	clientRef := uuid.NewString()
	raw, _ := json.Marshal(map[string]any{
		"client_ref": clientRef,
		"slots": []map[string]any{
			{"slot_ref": "1", "line_ref": "1", "label": "Unknown item #1", "unit_price": 200},
			{"slot_ref": "2", "line_ref": "1", "label": "Unknown item #1", "unit_price": 200},
			{"slot_ref": "3", "line_ref": "2", "label": "Unknown item #2", "unit_price": 200},
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/branch/unknown_sessions", bytes.NewReader(raw))
	req.Header.Set(branchCodeHeaderKey, branch.Register.Code)
	req.Header.Set(branchKeyHeaderKey, key)
	req.Header.Set("Content-Type", "application/json")
	rec := do(req)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var opened struct {
		UploadToken string `json:"upload_token"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &opened))

	var jpg bytes.Buffer
	require.NoError(t, jpeg.Encode(&jpg, image.NewRGBA(image.Rect(0, 0, 32, 24)), nil))
	var form bytes.Buffer
	writer := multipart.NewWriter(&form)
	part, err := writer.CreatePart(textproto.MIMEHeader{
		"Content-Disposition": {`form-data; name="image"; filename="item.jpg"`},
		"Content-Type":        {"image/jpeg"},
	})
	require.NoError(t, err)
	_, err = part.Write(jpg.Bytes())
	require.NoError(t, err)
	require.NoError(t, writer.Close())
	upload := httptest.NewRequest(http.MethodPost, "/unknown_uploads/"+opened.UploadToken+"/slots/1/image", &form)
	upload.Header.Set("Content-Type", writer.FormDataContentType())
	rec = do(upload)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	var out struct {
		Complete bool              `json:"complete"`
		Slots    []unknownSlotView `json:"slots"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &out))
	require.False(t, out.Complete)
	require.Len(t, out.Slots, 3)
	require.True(t, out.Slots[0].HasImage)
	require.True(t, out.Slots[1].HasImage, "the line's second unit shares the picture")
	require.Equal(t, out.Slots[0].ImageID, out.Slots[1].ImageID)
	require.False(t, out.Slots[2].HasImage, "another line needs its own picture")
	require.Equal(t, "2", out.Slots[2].LineRef)

	// The cashier adds a unit to line 1 and takes line 2 off the cart while
	// the phone is still open: the same link now lists the new set, and the
	// new unit already shares its line's picture.
	resend := func(slots []map[string]any) {
		raw, _ := json.Marshal(map[string]any{"client_ref": clientRef, "slots": slots})
		req := httptest.NewRequest(http.MethodPost, "/branch/unknown_sessions", bytes.NewReader(raw))
		req.Header.Set(branchCodeHeaderKey, branch.Register.Code)
		req.Header.Set(branchKeyHeaderKey, key)
		req.Header.Set("Content-Type", "application/json")
		rec := do(req)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		var again struct {
			UploadToken string `json:"upload_token"`
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &again))
		require.Equal(t, opened.UploadToken, again.UploadToken, "the phone's link stays the same")
	}
	resend([]map[string]any{
		{"slot_ref": "1", "line_ref": "1", "label": "Unknown item #1", "unit_price": 200},
		{"slot_ref": "2", "line_ref": "1", "label": "Unknown item #1", "unit_price": 200},
		{"slot_ref": "4", "line_ref": "1", "label": "Unknown item #1", "unit_price": 200},
	})
	rec = do(httptest.NewRequest(http.MethodGet, "/unknown_uploads/"+opened.UploadToken+"/status", nil))
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &out))
	require.Len(t, out.Slots, 3)
	require.Equal(t, "4", out.Slots[2].SlotRef)
	require.Equal(t, out.Slots[0].ImageID, out.Slots[2].ImageID, "the added unit shares its line's picture")
	require.True(t, out.Complete, "line 2 is gone, so nothing is missing")
}
