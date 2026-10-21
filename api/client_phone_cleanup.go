package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	db "github.com/mhdna/kashi/db/sqlc"
)

type normalizeClientPhonesRequest struct {
	// Without apply=true this only reports what it would do -- no writes.
	Apply bool `form:"apply"`
}

// normalizeClientPhonesResult is the full report: every client whose phone
// wasn't already in canonical form, bucketed by what happened to it.
type normalizeClientPhonesResult struct {
	TotalClients int                   `json:"total_clients"`
	Applied      bool                  `json:"applied"`
	AlreadyClean int                   `json:"already_clean"`
	Fixed        []phoneFixEntry       `json:"fixed"`
	Invalid      []phoneInvalidEntry   `json:"invalid"`
	Collisions   []phoneCollisionEntry `json:"collisions"`
}

type phoneFixEntry struct {
	ClientID int64  `json:"client_id"`
	Before   string `json:"before"`
	After    string `json:"after"`
}

type phoneInvalidEntry struct {
	ClientID int64  `json:"client_id"`
	Phone    string `json:"phone"`
}

type phoneCollisionEntry struct {
	Canonical string  `json:"canonical"`
	ClientIDs []int64 `json:"client_ids"`
}

// pendingPhoneUpdate is one row's new phone_valid/phone_country/phone,
// queued for the batched write at the end of normalizeClientPhones.
type pendingPhoneUpdate struct {
	id      int64
	phone   string
	valid   bool
	country string
}

// normalizeClientPhones is a one-off cleanup over every client's phone: it
// rewrites anything that's just a formatting difference (missing/extra
// trunk 0, a +961 or 00961 country code, stray spaces/dashes/quotes) to one
// canonical E164 form -- matching what kashi-pos's own normalizeClientPhone
// already sends for a client created at a till -- and backfills the
// phone_valid/phone_country columns (000190_client_phone_validity) that
// the clients page's country/validity filters and flag icons read.
//
// It never merges two different clients, even when their phones plainly
// mean the same number: two rows that would collide on the same canonical
// value are left exactly as they were (phone text, phone_valid and
// phone_country all untouched) and reported under "collisions" for a human
// to resolve -- which of the two, if either, is the real duplicate is not
// something this can safely guess. A phone that isn't a real number at all
// even after cleanup (too many/few digits, an unknown prefix) gets
// phone_valid=false and phone_country="" written, but its phone text is
// left exactly as typed -- normalizing only ever happens to a number that
// parses, never a guess at fixing a broken one.
//
// Without ?apply=true this only computes and returns the report; nothing
// is written. That's deliberate: run it once to see the scale, then again
// with apply=true once the report looks sane.
func (server *Server) normalizeClientPhones(ctx *gin.Context) {
	var req normalizeClientPhonesRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}

	rows, err := server.store.ListAllClientPhones(ctx)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	result := normalizeClientPhonesResult{TotalClients: len(rows), Applied: req.Apply}
	var pending []pendingPhoneUpdate

	byCanonical := make(map[string][]db.ListAllClientPhonesRow)
	for _, r := range rows {
		canon, _, ok := parseAnyPhone(r.Phone)
		if !ok {
			result.Invalid = append(result.Invalid, phoneInvalidEntry{ClientID: r.ID, Phone: r.Phone})
			pending = append(pending, pendingPhoneUpdate{id: r.ID, phone: r.Phone, valid: false, country: ""})
			continue
		}
		byCanonical[canon] = append(byCanonical[canon], r)
	}

	for canon, group := range byCanonical {
		if len(group) > 1 {
			ids := make([]int64, len(group))
			for i, r := range group {
				ids[i] = r.ID
			}
			result.Collisions = append(result.Collisions, phoneCollisionEntry{Canonical: canon, ClientIDs: ids})
			continue
		}
		r := group[0]
		_, region, _ := parseAnyPhone(r.Phone)
		if r.Phone != canon {
			result.Fixed = append(result.Fixed, phoneFixEntry{ClientID: r.ID, Before: r.Phone, After: canon})
		} else {
			result.AlreadyClean++
		}
		pending = append(pending, pendingPhoneUpdate{id: r.ID, phone: canon, valid: true, country: region})
	}

	// Batched in chunks rather than one UPDATE per row (the first version
	// of this did that and timed out at Cloudflare's gateway past ~100s on
	// 60k+ individual round trips) and rather than one single statement
	// (a transaction holding row locks on tens of thousands of rows for
	// the whole job isn't something to risk against a live table).
	const batchSize = 1000
	if req.Apply {
		for i := 0; i < len(pending); i += batchSize {
			end := i + batchSize
			if end > len(pending) {
				end = len(pending)
			}
			chunk := pending[i:end]
			ids := make([]int64, len(chunk))
			phones := make([]string, len(chunk))
			valids := make([]bool, len(chunk))
			countries := make([]string, len(chunk))
			for j, p := range chunk {
				ids[j], phones[j], valids[j], countries[j] = p.id, p.phone, p.valid, p.country
			}
			if err := server.store.BatchUpdateClientPhones(ctx, db.BatchUpdateClientPhonesParams{
				Ids:            ids,
				Phones:         phones,
				PhoneValids:    valids,
				PhoneCountries: countries,
			}); err != nil {
				server.writeError(ctx, http.StatusInternalServerError, err)
				return
			}
		}
	}

	server.writeJSON(ctx, http.StatusOK, envelope{"report": result})
}

// listClientPhoneCountries feeds the clients page's country filter
// dropdown: only countries actually present among clients' phones, each
// with how many clients have one, rather than offering all ~240 ISO
// countries regardless of whether any would match.
func (server *Server) listClientPhoneCountries(ctx *gin.Context) {
	rows, err := server.store.ListClientPhoneCountries(ctx)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	server.writeJSON(ctx, http.StatusOK, envelope{"countries": rows})
}
