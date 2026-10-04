package api

import (
	"log"
	"sort"
	"strings"
	"time"

	"github.com/lib/pq"
)

// Postgres announces every committed change to a table a till syncs from
// on the 'kashi_sync' channel (migration 000109): the payload names the
// till sync entities to re-fetch, e.g. "price_lists,branch_stock". This
// forwards them to every connected till as one "sync_changed" message --
// so a change reaches tills however it was made (an API handler, a script,
// a manual SQL fix), without each handler remembering to notify.
//
// Two messages go out:
//
//	sync_changed  Entities=what changed in the last syncNotifyWindow
//	pull_all      the listener lost its connection; changes announced
//	              during the gap are gone, so tills catch up on everything
//
// The payload is still only a pointer: tills re-fetch through the same
// GET /branch/sync/changes they poll, so a duplicate or extra message costs
// one cheap request, never correctness.

const (
	syncNotifyChannel = "kashi_sync"
	// syncNotifyWindow bundles everything announced within it into one
	// message: a 5000-row import, or stock moving on every sale across
	// branches, reaches each till as at most one message per window.
	syncNotifyWindow = time.Second
)

// syncChangeBatch collects entity names until the window closes.
type syncChangeBatch map[string]bool

// add records one notification payload ("a,b,c").
func (b syncChangeBatch) add(payload string) {
	for _, entity := range strings.Split(payload, ",") {
		if entity = strings.TrimSpace(entity); entity != "" {
			b[entity] = true
		}
	}
}

// drain returns the collected entities, sorted, and empties the batch.
func (b syncChangeBatch) drain() []string {
	entities := make([]string, 0, len(b))
	for entity := range b {
		entities = append(entities, entity)
		delete(b, entity)
	}
	sort.Strings(entities)
	return entities
}

// listenForSyncChanges runs for the life of the process (see Start).
func (server *Server) listenForSyncChanges() {
	runSyncNotifyListener(server.config.DBSource, server.branchHub.broadcastAll, nil)
}

// runSyncNotifyListener is listenForSyncChanges with its two outside
// dependencies passed in, so a test can run it against a real Postgres and
// capture what it sends. It returns when stop closes (nil: never).
func runSyncNotifyListener(dbSource string, send func(branchWSMessage), stop <-chan struct{}) {
	listener := pq.NewListener(dbSource, 2*time.Second, time.Minute,
		func(event pq.ListenerEventType, err error) {
			if err != nil {
				log.Printf("sync notify: listener connection: %v", err)
			}
		})
	defer listener.Close()
	if err := listener.Listen(syncNotifyChannel); err != nil {
		log.Printf("sync notify: listen %s: %v", syncNotifyChannel, err)
		return
	}

	batch := syncChangeBatch{}
	var flush <-chan time.Time // nil (blocks forever) while the batch is empty
	// pq recommends pinging an idle listener so a dead connection is noticed
	// (and re-established) without waiting for the next notification.
	ping := time.NewTicker(90 * time.Second)
	defer ping.Stop()

	for {
		select {
		case <-stop:
			return
		case n := <-listener.Notify:
			if n == nil {
				// pq sends nil right after reconnecting: anything announced
				// while it was down never arrived.
				log.Printf("sync notify: listener reconnected, telling tills to pull everything")
				send(branchWSMessage{Type: "pull_all"})
				continue
			}
			batch.add(n.Extra)
			if flush == nil {
				flush = time.After(syncNotifyWindow)
			}
		case <-flush:
			flush = nil
			send(branchWSMessage{Type: "sync_changed", Entities: batch.drain()})
		case <-ping.C:
			if err := listener.Ping(); err != nil {
				log.Printf("sync notify: ping: %v", err)
			}
		}
	}
}
