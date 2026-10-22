package api

import (
	"context"
	"log"
	"time"
)

const clientSpendRefreshEvery = 5 * time.Minute

// RunClientSpendRefresh rebuilds the client_spend cache at start-up and then
// every few minutes. The clients and loyalty pages read it instead of summing
// every invoice per request, so a new sale shows up in the rankings within one
// interval.
func (server *Server) RunClientSpendRefresh(ctx context.Context) {
	ticker := time.NewTicker(clientSpendRefreshEvery)
	defer ticker.Stop()
	for {
		start := time.Now()
		if err := server.store.RefreshClientSpend(ctx); err != nil {
			log.Printf("client spend refresh: %v", err)
		} else {
			log.Printf("client spend refresh: %s", time.Since(start).Round(time.Millisecond))
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
