package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
	db "github.com/mhdna/kashi/db/sqlc"
	"github.com/mhdna/kashi/util"
	"github.com/stretchr/testify/require"
)

// Seeded options, the same uids as the till's migration 000151.
const (
	sourceSocialUID = "6f1c2a00-5e1d-4c7a-9b00-000000000001"
	sourceSMSUID    = "6f1c2a00-5e1d-4c7a-9b00-000000000002"
	sourceFriendUID = "6f1c2a00-5e1d-4c7a-9b00-000000000003"
)

type syncedSourceRow struct {
	ID       int64  `json:"id"`
	UID      string `json:"uid"`
	Name     string `json:"name"`
	IsActive bool   `json:"is_active"`
}

type listedSourceRow struct {
	ID          int64  `json:"id"`
	UID         string `json:"uid"`
	Name        string `json:"name"`
	ClientCount int64  `json:"client_count"`
	IsActive    bool   `json:"is_active"`
}

// The reported flow: a cashier adds a new customer, picks where they heard
// about the shop, and kashi keeps that answer on the client.
func TestClientSourceFlow(t *testing.T) {
	conn, store := referralDB(t)
	ctx := context.Background()

	const key = "client-source-key"
	hash, err := util.HashPassword(key)
	require.NoError(t, err)
	branch, err := store.CreateBranchTx(ctx, db.CreateBranchTxParams{
		Name: util.RandomString(10), Code: util.RandomString(6), RegisterAPIKeyHash: hash,
	})
	require.NoError(t, err)
	server := newTestServer(t, store)

	till := func(method, path string, body any) *httptest.ResponseRecorder {
		raw, err := json.Marshal(body)
		require.NoError(t, err)
		if body == nil {
			raw = nil
		}
		req, err := http.NewRequest(method, path, bytes.NewReader(raw))
		require.NoError(t, err)
		req.Header.Set(branchCodeHeaderKey, branch.Register.Code)
		req.Header.Set(branchKeyHeaderKey, key)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		server.router.ServeHTTP(rec, req)
		return rec
	}
	var nextLocalID int64 = 1000
	push := func(name, phone, sourceUID string) db.Client {
		nextLocalID++
		rec := till(http.MethodPost, "/branch/clients", map[string]any{
			"branch_client_id": nextLocalID, "name": name, "phone": phone, "source_uid": sourceUID,
		})
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		var body struct {
			Client db.Client `json:"client"`
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		return body.Client
	}
	sourceOf := func(clientID int64) string {
		var name *string
		require.NoError(t, conn.QueryRow(`SELECT s.name FROM clients c LEFT JOIN client_sources s ON s.id = c.source_id WHERE c.id = $1`, clientID).Scan(&name))
		if name == nil {
			return ""
		}
		return *name
	}
	phone := func() string { return fmt.Sprintf("+9617%07d", rand.Int63n(1e7)) }
	synced := func() map[string]syncedSourceRow {
		rec := till(http.MethodGet, "/branch/sync/changes?entity=client_sources", nil)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		var body struct {
			Items []syncedSourceRow `json:"items"`
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		out := map[string]syncedSourceRow{}
		for _, it := range body.Items {
			out[it.UID] = it
		}
		return out
	}

	t.Run("the till gets the whole list", func(t *testing.T) {
		rows := synced()
		require.Equal(t, "Social media", rows[sourceSocialUID].Name)
		require.Equal(t, "SMS", rows[sourceSMSUID].Name)
		require.Equal(t, "A friend", rows[sourceFriendUID].Name)
	})

	t.Run("a new client keeps the answer the till sent", func(t *testing.T) {
		c := push("Maya", phone(), sourceSMSUID)
		require.Equal(t, "SMS", sourceOf(c.ID))
	})

	t.Run("an edit with another answer changes nothing", func(t *testing.T) {
		p := phone()
		nextLocalID++
		id := nextLocalID
		send := func(name, src string) {
			rec := till(http.MethodPost, "/branch/clients", map[string]any{
				"branch_client_id": id, "name": name, "phone": p, "source_uid": src,
			})
			require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		}
		send("Rami", sourceFriendUID)
		send("Rami K", sourceSocialUID)
		var clientID int64
		require.NoError(t, conn.QueryRow(`SELECT id FROM clients WHERE phone = $1`, p).Scan(&clientID))
		require.Equal(t, "A friend", sourceOf(clientID))
	})

	t.Run("a client another till added first keeps their answer", func(t *testing.T) {
		p := phone()
		first := push("Lea", p, sourceSocialUID)
		second := push("Lea", p, sourceSMSUID)
		require.Equal(t, first.ID, second.ID)
		require.Equal(t, "Social media", sourceOf(first.ID))
	})

	t.Run("an existing client with no answer takes the till's", func(t *testing.T) {
		p := phone()
		var id int64
		require.NoError(t, conn.QueryRow(`INSERT INTO clients (name, phone) VALUES ('Nour', $1) RETURNING id`, p).Scan(&id))
		push("Nour", p, sourceFriendUID)
		require.Equal(t, "A friend", sourceOf(id))
	})

	t.Run("no answer, or one kashi doesn't know, still syncs the client", func(t *testing.T) {
		require.Equal(t, "", sourceOf(push("Ali", phone(), "").ID))
		require.Equal(t, "", sourceOf(push("Omar", phone(), uuid.NewString()).ID))
		require.Equal(t, "", sourceOf(push("Hadi", phone(), "not-a-uid").ID))
	})

	t.Run("the batch route keeps the answer too", func(t *testing.T) {
		p := phone()
		rec := till(http.MethodPost, "/branch/clients_batch", map[string]any{"clients": []map[string]any{
			{"branch_client_id": 5001, "name": "Sara", "phone": p, "source_uid": sourceSMSUID},
		}})
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		var id int64
		require.NoError(t, conn.QueryRow(`SELECT id FROM clients WHERE phone = $1`, p).Scan(&id))
		require.Equal(t, "SMS", sourceOf(id))
	})

	t.Run("the profile shows where they heard about us", func(t *testing.T) {
		c := push("Dana", phone(), sourceFriendUID)
		got, err := store.GetClient(ctx, c.ID)
		require.NoError(t, err)
		gctx, _ := gin.CreateTestContext(httptest.NewRecorder())
		gctx.Request = httptest.NewRequest(http.MethodGet, "/", nil)
		require.Equal(t, "A friend", server.clientHeardFrom(gctx, got))
		require.Equal(t, "", server.clientHeardFrom(gctx, db.Client{}))
	})

	t.Run("adding, hiding and deleting options", func(t *testing.T) {
		rec := adminRequest(t, server, http.MethodPost, "/client_sources", map[string]any{"name": "  ", "is_active": true})
		require.Equal(t, http.StatusBadRequest, rec.Code)

		rec = adminRequest(t, server, http.MethodPost, "/client_sources", map[string]any{"name": " Radio ", "sort_order": 9, "is_active": true})
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		var created struct {
			ClientSource syncedSourceRow `json:"client_source"`
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &created))
		radio := created.ClientSource
		require.Equal(t, "Radio", radio.Name)
		require.True(t, synced()[radio.UID].IsActive)

		// Hidden: still sent, so the till drops it from the question.
		rec = adminRequest(t, server, http.MethodPut, "/client_sources", map[string]any{"id": radio.ID, "name": "Radio", "sort_order": 9, "is_active": false})
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.False(t, synced()[radio.UID].IsActive)

		// A till that hasn't heard yet may still send it: the answer is kept.
		// Given by a customer, it can't be deleted.
		c := push("Rita", phone(), radio.UID)
		require.Equal(t, "Radio", sourceOf(c.ID))
		rec = adminRequest(t, server, http.MethodDelete, fmt.Sprintf("/client_sources/%d", radio.ID), nil)
		require.Equal(t, http.StatusConflict, rec.Code)

		rec = adminRequest(t, server, http.MethodPost, "/client_sources", map[string]any{"name": "Billboard", "is_active": true})
		require.Equal(t, http.StatusOK, rec.Code)
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &created))
		rec = adminRequest(t, server, http.MethodDelete, fmt.Sprintf("/client_sources/%d", created.ClientSource.ID), nil)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		_, still := synced()[created.ClientSource.UID]
		require.False(t, still)
	})

	// Hundreds of customers with any answer, repeats and pushes from a second
	// look: each option's count on the settings page equals the customers who
	// gave it first.
	t.Run("counts equal the first answers, any mix", func(t *testing.T) {
		counts := func() map[string]int64 {
			rec := adminRequest(t, server, http.MethodGet, "/client_sources", nil)
			require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
			var body struct {
				ClientSources []listedSourceRow `json:"client_sources"`
			}
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
			out := map[string]int64{}
			for _, r := range body.ClientSources {
				out[r.UID] = r.ClientCount
			}
			return out
		}
		before := counts()
		choices := []string{sourceSocialUID, sourceSMSUID, sourceFriendUID, "", uuid.NewString()}
		want := map[string]int64{}
		rng := rand.New(rand.NewSource(7))
		phones := []string{}
		first := map[string]string{}
		for i := 0; i < 300; i++ {
			var p string
			if len(phones) > 0 && rng.Intn(4) == 0 {
				p = phones[rng.Intn(len(phones))] // the same customer again
			} else {
				p = phone()
				phones = append(phones, p)
			}
			src := choices[rng.Intn(len(choices))]
			push("Customer", p, src)
			known := src == sourceSocialUID || src == sourceSMSUID || src == sourceFriendUID
			if _, has := first[p]; !has && known {
				first[p] = src
				want[src]++
			}
		}
		after := counts()
		for _, uid := range []string{sourceSocialUID, sourceSMSUID, sourceFriendUID} {
			require.Equal(t, want[uid], after[uid]-before[uid], uid)
		}
	})
}
