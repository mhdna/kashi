package api

import (
	"database/sql"
	"os"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"github.com/stretchr/testify/require"
)

func TestSyncChangeBatchMergesAndSorts(t *testing.T) {
	batch := syncChangeBatch{}
	batch.add("price_lists,branch_stock")
	batch.add("branch_stock")
	batch.add("currencies")
	require.Equal(t, []string{"branch_stock", "currencies", "price_lists"}, batch.drain())
	require.Empty(t, batch.drain(), "drain must empty the batch")
}

// End to end against a real Postgres with migration 000109 applied: a write
// to a synced table, made with plain SQL (no API handler involved), reaches
// the send func as one bundled sync_changed. Opt-in, since the rest of the
// api tests run without a database:
//
//	KASHI_SYNC_NOTIFY_TEST_DB=postgresql://... go test ./api -run SyncNotify
func TestSyncNotifyListenerForwardsCommittedWrites(t *testing.T) {
	dsn := os.Getenv("KASHI_SYNC_NOTIFY_TEST_DB")
	if dsn == "" {
		t.Skip("KASHI_SYNC_NOTIFY_TEST_DB not set")
	}
	conn, err := sql.Open("postgres", dsn)
	require.NoError(t, err)
	defer conn.Close()

	sent := make(chan branchWSMessage, 10)
	stop := make(chan struct{})
	defer close(stop)
	go runSyncNotifyListener(dsn, func(m branchWSMessage) { sent <- m }, stop)

	// The listener subscribes asynchronously; keep writing until its first
	// message shows it was listening.
	deadline := time.After(10 * time.Second)
	for {
		tx, err := conn.Begin()
		require.NoError(t, err)
		_, err = tx.Exec(`UPDATE currencies SET name = name`)
		require.NoError(t, err)
		_, err = tx.Exec(`UPDATE price_list_items SET price = price`)
		require.NoError(t, err)
		require.NoError(t, tx.Commit())

		select {
		case m := <-sent:
			require.Equal(t, "sync_changed", m.Type)
			require.Equal(t, []string{"branch_stock", "currencies", "price_lists"}, m.Entities)
			return
		case <-time.After(300 * time.Millisecond):
		case <-deadline:
			t.Fatal("no sync_changed message within 10s")
		}
	}
}
