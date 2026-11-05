package api

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strconv"
	"sync"
	"time"

	db "github.com/mhdna/kashi/db/sqlc"
)

// A fingerprint is a list's row count and a hash of its rows, worked out the same
// way here and on the till (kashi-pos sync_report.go; a test in each repo checks the
// same rows give the same hash), so comparing two says they hold the same thing.
//
// One line per row, sorted, so row order can't change it:
//
//	price list:     "<variant uid>|<price>"
//	discount list:  "<variant uid>|<discount>|<deal uid>|<deal fallback>" (a row
//	                with no deal or fallback leaves those empty), then its option
//	                rows as "o|<variant uid>|<sale option>|<discount>"

type listFingerprint struct {
	Rows int
	Hash string
}

func fingerprintLines(lines []string) listFingerprint {
	sort.Strings(lines)
	h := sha256.New()
	for _, line := range lines {
		h.Write([]byte(line))
		h.Write([]byte{'\n'})
	}
	return listFingerprint{Rows: len(lines), Hash: hex.EncodeToString(h.Sum(nil))}
}

// emptyListFingerprint is what a till holds of a list it was told to stop using.
var emptyListFingerprint = fingerprintLines(nil)

func priceListFingerprint(rows []db.ListPriceListItemsForSyncRow) listFingerprint {
	lines := make([]string, 0, len(rows))
	for _, r := range rows {
		lines = append(lines, fmt.Sprintf("%s|%d", r.ProductVariantUid, r.Price))
	}
	return fingerprintLines(lines)
}

func discountListFingerprint(rows []db.ListDiscountListItemsForSyncRow, options []db.ListDiscountListOptionItemsForSyncRow) listFingerprint {
	lines := make([]string, 0, len(rows)+len(options))
	for _, r := range rows {
		deal, fallback := "", ""
		if r.DealUid.Valid {
			deal = r.DealUid.UUID.String()
			if r.DealFallbackPercent.Valid {
				fallback = strconv.Itoa(int(r.DealFallbackPercent.Int16))
			}
		}
		lines = append(lines, fmt.Sprintf("%s|%d|%s|%s", r.ProductVariantUid, r.Discount, deal, fallback))
	}
	for _, o := range options {
		lines = append(lines, fmt.Sprintf("o|%s|%s|%d", o.ProductVariantUid, o.SaleOption, o.Discount))
	}
	return fingerprintLines(lines)
}

// listFingerprints keeps the last fingerprint per list, as the page asks every few
// seconds while a till catches up and a list only changes when updated_at moves.
var listFingerprints sync.Map // "<kind>:<id>" -> cachedListFingerprint

type cachedListFingerprint struct {
	updatedAt   time.Time
	fingerprint listFingerprint
}

func cachedFingerprint(kind string, id int64, updatedAt time.Time, read func() (listFingerprint, error)) (listFingerprint, error) {
	key := kind + ":" + strconv.FormatInt(id, 10)
	if cached, ok := listFingerprints.Load(key); ok && cached.(cachedListFingerprint).updatedAt.Equal(updatedAt) {
		return cached.(cachedListFingerprint).fingerprint, nil
	}
	fingerprint, err := read()
	if err != nil {
		return listFingerprint{}, err
	}
	listFingerprints.Store(key, cachedListFingerprint{updatedAt, fingerprint})
	return fingerprint, nil
}

func (server *Server) expectedDiscountList(ctx context.Context, list db.DiscountList) (listFingerprint, error) {
	if !list.IsActive {
		return emptyListFingerprint, nil
	}
	return cachedFingerprint("discount", list.ID, list.UpdatedAt, func() (listFingerprint, error) {
		rows, err := server.store.ListDiscountListItemsForSync(ctx, list.ID)
		if err != nil {
			return listFingerprint{}, err
		}
		options, err := server.store.ListDiscountListOptionItemsForSync(ctx, list.ID)
		if err != nil {
			return listFingerprint{}, err
		}
		return discountListFingerprint(rows, options), nil
	})
}

func (server *Server) expectedPriceList(ctx context.Context, list db.PriceList) (listFingerprint, error) {
	if !list.IsActive {
		return emptyListFingerprint, nil
	}
	return cachedFingerprint("price", list.ID, list.UpdatedAt, func() (listFingerprint, error) {
		rows, err := server.store.ListPriceListItemsForSync(ctx, list.ID)
		if err != nil {
			return listFingerprint{}, err
		}
		return priceListFingerprint(rows), nil
	})
}
