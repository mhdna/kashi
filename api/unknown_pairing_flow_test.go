package api

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"image"
	"image/jpeg"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"testing"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	db "github.com/mhdna/kashi/db/sqlc"
	"github.com/mhdna/kashi/util"
	"github.com/stretchr/testify/require"
)

// A phone paired with a till, over HTTP against a real database: it scans the
// till's pairing code once, then sees each cart's unknown units as the till
// rings them up, photographs one, and keeps the picture when the cart changes
// around it; the cart only reaches the Waiting tab at the payment, and a new
// pairing kills the old phone's link.
func TestUnknownItemsPairedPhone(t *testing.T) {
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

	const key = "unknown-pair-key"
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
	till := func(method, path string, body any, out any) {
		raw, err := json.Marshal(body)
		require.NoError(t, err)
		req, err := http.NewRequest(method, path, bytes.NewReader(raw))
		require.NoError(t, err)
		req.Header.Set(branchCodeHeaderKey, branch.Register.Code)
		req.Header.Set(branchKeyHeaderKey, key)
		req.Header.Set("Content-Type", "application/json")
		rec := do(req)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		if out != nil {
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), out))
		}
	}
	type phoneView struct {
		UploadToken string            `json:"upload_token"`
		Slots       []unknownSlotView `json:"slots"`
		Sig         string            `json:"sig"`
	}
	phone := func(token string) (int, phoneView) {
		var view phoneView
		rec := do(httptest.NewRequest(http.MethodGet, "/unknown_pairs/"+token+"/status", nil))
		if rec.Code == http.StatusOK {
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &view))
		}
		return rec.Code, view
	}
	sync := func(ref string, slotRefs ...string) {
		slots := []map[string]any{}
		for _, slotRef := range slotRefs {
			slots = append(slots, map[string]any{"slot_ref": slotRef, "label": "Unknown item", "unit_price": 1000})
		}
		till(http.MethodPost, "/branch/unknown_sessions", map[string]any{"client_ref": ref, "slots": slots, "background": true}, nil)
	}
	waitingSlots := func() int {
		waiting, err := store.ListWaitingUnknownItems(ctx)
		require.NoError(t, err)
		n := 0
		for _, w := range waiting {
			if w.BranchID == branch.Branch.ID {
				n++
			}
		}
		return n
	}

	// 1. The till's pairing code; the phone opens it and waits for a cart.
	var pairing struct {
		Token       string `json:"token"`
		PhoneOnline bool   `json:"phone_online"`
	}
	till(http.MethodGet, "/branch/unknown_pairing", nil, &pairing)
	require.NotEmpty(t, pairing.Token)
	require.False(t, pairing.PhoneOnline)
	page := do(httptest.NewRequest(http.MethodGet, "/unknown_pairs/"+pairing.Token, nil))
	require.Equal(t, http.StatusOK, page.Code)
	code, view := phone(pairing.Token)
	require.Equal(t, http.StatusOK, code)
	require.Empty(t, view.UploadToken)

	// The same code comes back every time, and the phone now shows online.
	var again struct {
		Token       string `json:"token"`
		PhoneOnline bool   `json:"phone_online"`
	}
	till(http.MethodGet, "/branch/unknown_pairing", nil, &again)
	require.Equal(t, pairing.Token, again.Token)
	require.True(t, again.PhoneOnline)

	// 2. The cashier rings up two unknown units: the phone shows them at once,
	// and they are not on the Waiting tab while the cart is being rung up.
	cart := uuid.NewString()
	sync(cart, "a", "b")
	code, view = phone(pairing.Token)
	require.Equal(t, http.StatusOK, code)
	require.NotEmpty(t, view.UploadToken)
	require.Len(t, view.Slots, 2)
	require.Zero(t, waitingSlots())

	// 3. The phone photographs unit a.
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
	upload := httptest.NewRequest(http.MethodPost, "/unknown_uploads/"+view.UploadToken+"/slots/a/image", &form)
	upload.Header.Set("Content-Type", writer.FormDataContentType())
	require.Equal(t, http.StatusOK, do(upload).Code)

	// 4. The cashier takes unit b back out and adds c: a keeps its picture.
	sync(cart, "a", "c")
	_, view = phone(pairing.Token)
	require.Len(t, view.Slots, 2)
	require.Equal(t, "a", view.Slots[0].SlotRef)
	require.True(t, view.Slots[0].HasImage)
	require.Equal(t, "c", view.Slots[1].SlotRef)
	require.False(t, view.Slots[1].HasImage)

	// 5. At the payment c still has no picture: the cart joins the Waiting tab.
	till(http.MethodPost, "/branch/unknown_sessions/checkout", map[string]string{"client_ref": cart}, nil)
	require.Equal(t, 2, waitingSlots())

	// 6. The cart is held: the phone goes back to waiting, and the held cart
	// keeps its picture for when it comes back.
	till(http.MethodPost, "/branch/unknown_pairing/idle", nil, nil)
	_, view = phone(pairing.Token)
	require.Empty(t, view.UploadToken)
	sync(cart, "a", "c")
	_, view = phone(pairing.Token)
	require.True(t, view.Slots[0].HasImage)

	// 7. Pairing another phone kills the old phone's link.
	var reset struct {
		Token string `json:"token"`
	}
	till(http.MethodPost, "/branch/unknown_pairing/reset", nil, &reset)
	require.NotEqual(t, pairing.Token, reset.Token)
	code, _ = phone(pairing.Token)
	require.Equal(t, http.StatusNotFound, code)
	code, view = phone(reset.Token)
	require.Equal(t, http.StatusOK, code)
	require.Len(t, view.Slots, 2, "the new phone finds the cart in progress")
}
