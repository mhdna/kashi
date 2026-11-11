package db

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/lib/pq"
	"github.com/mhdna/kashi/util"
	"github.com/stretchr/testify/require"
)

// listenSync listens on the channel kashi forwards to tills (kashi_sync) and
// returns a function that collects what arrived within the given wait.
func listenSync(t *testing.T) func(wait time.Duration) []string {
	t.Helper()
	config, err := util.LoadConfig("../..")
	require.NoError(t, err)
	dsn := config.DBSource
	if env := os.Getenv("DB_SOURCE"); env != "" {
		dsn = env
	}
	l := pq.NewListener(dsn, time.Second, time.Minute, nil)
	require.NoError(t, l.Listen("kashi_sync"))
	t.Cleanup(func() { l.Close() })
	return func(wait time.Duration) []string {
		var got []string
		timeout := time.After(wait)
		for {
			select {
			case n := <-l.Notify:
				if n != nil {
					got = append(got, n.Extra)
				}
			case <-timeout:
				return got
			}
		}
	}
}

// listenSyncUntil is listenSync for checks that expect a notification: it
// returns as soon as a "branches" one arrives (up to two seconds), so a slow
// delivery doesn't make the test flaky. A check that expects silence still
// uses the fixed window.
func listenSyncUntil(t *testing.T) func() []string {
	t.Helper()
	config, err := util.LoadConfig("../..")
	require.NoError(t, err)
	dsn := config.DBSource
	if env := os.Getenv("DB_SOURCE"); env != "" {
		dsn = env
	}
	l := pq.NewListener(dsn, time.Second, time.Minute, nil)
	require.NoError(t, l.Listen("kashi_sync"))
	t.Cleanup(func() { l.Close() })
	return func() []string {
		var got []string
		timeout := time.After(2 * time.Second)
		for {
			select {
			case n := <-l.Notify:
				if n != nil {
					got = append(got, n.Extra)
					if strings.Contains(n.Extra, "branches") {
						return got
					}
				}
			case <-timeout:
				return got
			}
		}
	}
}

func newBranchID(t *testing.T) int64 {
	var id int64
	require.NoError(t, testDB.QueryRowContext(context.Background(),
		`INSERT INTO branches (name, code) VALUES ($1, $2) RETURNING id`,
		"nb-"+util.RandomString(6), "NB"+util.RandomString(8)).Scan(&id))
	return id
}

func hasBranches(got []string) bool {
	for _, g := range got {
		if strings.Contains(g, "branches") {
			return true
		}
	}
	return false
}

// The bug: every till request stamps last_seen_at, and that told every till
// to pull again, which stamped it again... Stamping it is not news.
func TestBranchesNotify_LastSeenDoesNotWakeTheTills(t *testing.T) {
	ctx := context.Background()
	id := newBranchID(t)
	collect := listenSync(t)
	collect(300 * time.Millisecond) // the insert's own notification

	_, err := testDB.ExecContext(ctx, `UPDATE branches SET last_seen_at = now() WHERE id = $1`, id)
	require.NoError(t, err)
	require.False(t, hasBranches(collect(400*time.Millisecond)), "a last_seen_at stamp must not notify")

	// The real code path does it through the register, in one statement.
	var regID int64
	require.NoError(t, testDB.QueryRowContext(ctx,
		`INSERT INTO branch_registers (branch_id, code, name, api_key_hash) VALUES ($1, $2, 'R', 'x') RETURNING id`,
		id, "R"+util.RandomString(6)).Scan(&regID))
	collect(300 * time.Millisecond)
	require.NoError(t, testQueries.TouchBranchRegister(ctx, regID))
	require.False(t, hasBranches(collect(400*time.Millisecond)), "TouchBranchRegister must not notify")
}

// What a till pulls from branches still reaches it at once.
func TestBranchesNotify_RealChangesStillNotify(t *testing.T) {
	ctx := context.Background()
	id := newBranchID(t)
	wait := listenSyncUntil(t)
	wait() // the insert's own notification

	// In this order: the delete has to come last, or there is nothing to change.
	for _, step := range []struct{ name, stmt string }{
		{"rename", `UPDATE branches SET name = name || 'x' WHERE id = $1`},
		{"deactivate", `UPDATE branches SET is_active = false WHERE id = $1`},
		{"delete", `DELETE FROM branches WHERE id = $1`},
	} {
		_, err := testDB.ExecContext(ctx, step.stmt, id)
		require.NoError(t, err, step.name)
		got := wait()
		require.True(t, hasBranches(got), "%s must notify, got %v", step.name, got)
		require.Contains(t, strings.Join(got, ","), "incoming_transfer_requests", step.name)
	}

	// A new branch too.
	newBranchID(t)
	require.True(t, hasBranches(wait()), "an insert must notify")
}

// A stamp together with a real change is still a change.
func TestBranchesNotify_StampPlusRealChangeNotifies(t *testing.T) {
	id := newBranchID(t)
	wait := listenSyncUntil(t)
	wait()
	_, err := testDB.ExecContext(context.Background(), `UPDATE branches SET last_seen_at = now(), name = name || 'y' WHERE id = $1`, id)
	require.NoError(t, err)
	require.True(t, hasBranches(wait()))
}
