package api

import (
	"context"
	"log"
	"time"

	db "github.com/mhdna/kashi/db/sqlc"
)

const clientSpendRefreshEvery = 5 * time.Minute

// RunClientSpendRefresh (also tops up clients' phone flags) rebuilds the client_spend cache at start-up and then
// every few minutes. The clients and loyalty pages read it instead of summing
// every invoice per request, so a new sale shows up in the rankings within one
// interval.
func (server *Server) RunClientSpendRefresh(ctx context.Context) {
	ticker := time.NewTicker(clientSpendRefreshEvery)
	defer ticker.Stop()
	for {
		if err := server.backfillClientPhoneFlags(ctx); err != nil {
			log.Printf("client phone flags: %v", err)
		}
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

// backfillClientPhoneFlags fills phone_valid/phone_country for clients that
// have no country yet (see ListClientsMissingPhoneCountry), so their flag
// shows. Only rows whose result differs from what's stored are written.
func (server *Server) backfillClientPhoneFlags(ctx context.Context) error {
	rows, err := server.store.ListClientsMissingPhoneCountry(ctx)
	if err != nil {
		return err
	}
	var ids []int64
	var valids []bool
	var countries []string
	for _, r := range rows {
		_, region, valid := parseAnyPhone(r.Phone)
		if valid == r.PhoneValid && region == "" {
			continue
		}
		ids = append(ids, r.ID)
		valids = append(valids, valid)
		countries = append(countries, region)
	}
	const batchSize = 1000
	for i := 0; i < len(ids); i += batchSize {
		end := min(i+batchSize, len(ids))
		if err := server.store.BatchSetClientPhoneFlags(ctx, db.BatchSetClientPhoneFlagsParams{
			Ids:            ids[i:end],
			PhoneValids:    valids[i:end],
			PhoneCountries: countries[i:end],
		}); err != nil {
			return err
		}
	}
	if len(ids) > 0 {
		log.Printf("client phone flags: filled %d clients", len(ids))
	}
	return nil
}
