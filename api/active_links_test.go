package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	db "github.com/mhdna/kashi/db/sqlc"
	"github.com/mhdna/kashi/util"
	"github.com/stretchr/testify/require"
)

// The home page's "Active links" against a real database: a link shows while
// it still works and drops out once it's used, expired, or (for unknown items)
// every unit has its picture or pass. Each link carries the short path the
// till printed as its QR code.
func TestActiveLinks(t *testing.T) {
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

	branch, err := store.CreateBranchTx(ctx, db.CreateBranchTxParams{
		Name: "Links " + util.RandomString(8), Code: util.RandomString(6), RegisterAPIKeyHash: "x",
	})
	require.NoError(t, err)
	hour := time.Now().UTC().Add(time.Hour)

	// Unknown items: one session still missing a picture (active), one fully
	// pictured (done), one expired.
	session := func(expires time.Time, slots int, pictured int) db.UnknownItemSession {
		s, err := store.CreateUnknownItemSession(ctx, db.CreateUnknownItemSessionParams{
			BranchID: branch.Branch.ID, RegisterID: branch.Register.ID, ClientRef: uuid.NewString(), Token: util.RandomString(16), ExpiresAt: expires,
		})
		require.NoError(t, err)
		for i := 0; i < slots; i++ {
			ref := uuid.NewString()
			require.NoError(t, store.CreateUnknownItemSlot(ctx, db.CreateUnknownItemSlotParams{
				SessionID: s.ID, SlotRef: ref, Position: int32(i), UnitPrice: 500,
			}))
			if i < pictured {
				img, err := store.CreateUnknownItemImage(ctx, db.CreateUnknownItemImageParams{
					SessionID: s.ID, FilePath: filepath.Join(os.TempDir(), "x.jpg"), ContentType: "image/jpeg", SizeBytes: 1,
				})
				require.NoError(t, err)
				require.NoError(t, store.SetUnknownItemSlotImage(ctx, db.SetUnknownItemSlotImageParams{
					SessionID: s.ID, SlotRef: ref, ImageID: sql.NullInt64{Int64: img.ID, Valid: true},
				}))
			}
		}
		return s
	}
	active := session(hour, 3, 1)
	session(hour, 2, 2)
	session(time.Now().UTC().Add(-time.Minute), 1, 0)

	// A shift receipt link that's still open, and one already used.
	shift, err := store.CreateBranchShift(ctx, db.CreateBranchShiftParams{
		BranchID: branch.Branch.ID, ClientRef: uuid.NewString(), BranchShiftID: time.Now().UnixNano() % 1_000_000_000, RegisterID: branch.Register.ID,
	})
	require.NoError(t, err)
	shiftToken := util.RandomString(16)
	_, err = store.CreateBranchShiftUploadToken(ctx, db.CreateBranchShiftUploadTokenParams{BranchShiftID: shift.ID, Token: shiftToken, ExpiresAt: hour})
	require.NoError(t, err)
	used := util.RandomString(16)
	_, err = store.CreateBranchShiftUploadToken(ctx, db.CreateBranchShiftUploadTokenParams{BranchShiftID: shift.ID, Token: used, ExpiresAt: hour})
	require.NoError(t, err)
	_, err = conn.ExecContext(ctx, `UPDATE branch_shift_upload_tokens SET used_at = now() WHERE token = $1`, used)
	require.NoError(t, err)

	rows, err := store.ListActiveUploadLinks(ctx)
	require.NoError(t, err)
	mine := func(links []activeLink) []activeLink {
		var out []activeLink
		for _, l := range links {
			if l.BranchID == branch.Branch.ID {
				out = append(out, l)
			}
		}
		return out
	}
	all := func(string) bool { return true }

	got := mine(activeLinksFor(rows, all))
	require.Len(t, got, 2)
	byKind := map[string]activeLink{}
	for _, l := range got {
		byKind[l.Kind] = l
	}
	require.Equal(t, "/u/"+active.Token, byKind["unknown"].Path)
	require.EqualValues(t, 3, byKind["unknown"].Units)
	require.EqualValues(t, 1, byKind["unknown"].Done)
	require.Equal(t, branch.Branch.Name, byKind["unknown"].BranchName)
	require.Equal(t, "/s/"+shiftToken, byKind["shift"].Path)
	require.Equal(t, shift.ID, byKind["shift"].RefID)

	// Someone who can only see sales gets the unknown item's link, not the shift's.
	salesOnly := mine(activeLinksFor(rows, func(s string) bool { return s == "sales" }))
	require.Len(t, salesOnly, 1)
	require.Equal(t, "unknown", salesOnly[0].Kind)
	require.Empty(t, mine(activeLinksFor(rows, func(string) bool { return false })))

	// A hidden branch's links don't show at all.
	_, err = conn.ExecContext(ctx, `UPDATE branches SET hidden_from_reports = true WHERE id = $1`, branch.Branch.ID)
	require.NoError(t, err)
	rows, err = store.ListActiveUploadLinks(ctx)
	require.NoError(t, err)
	require.Empty(t, mine(activeLinksFor(rows, all)))
	_, err = conn.ExecContext(ctx, `UPDATE branches SET hidden_from_reports = false WHERE id = $1`, branch.Branch.ID)
	require.NoError(t, err)

	// Over HTTP: the route answers with the links.
	server := newTestServer(t, store)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/home/active_links", nil)
	addAuthorization(t, req, server.tokenMaker, authorizationTypeBearer, "admin", time.Minute)
	server.router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var body struct {
		Links []activeLink `json:"links"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Len(t, mine(body.Links), 2)
}
