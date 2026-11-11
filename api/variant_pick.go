package api

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	db "github.com/mhdna/kashi/db/sqlc"
)

// Values of products.variant_pick: the per-item override of the branch's
// "pick colour/size on scan" setting.
const (
	variantPickFollow = ""       // follow the branch setting
	variantPickAlways = "always" // ask which colour/size, whatever the branch says
	variantPickNever  = "never"  // never ask, whatever the branch says
)

type setVariantPickRequest struct {
	// Product codes and/or variant barcodes, as typed or pasted -- the same
	// entries the sale options and discount list dialogs take.
	Entries []string `json:"entries" binding:"required,min=1,max=5000"`
	// "" follows the branch, "always" asks, "never" doesn't.
	Mode string `json:"mode" binding:"omitempty,oneof=always never"`
}

// setVariantPick sets the override on every product the entries name. The
// till gets it on the products' next sync (a trigger moves their variants'
// updated_at, see 000233).
func (server *Server) setVariantPick(ctx *gin.Context) {
	var req setVariantPickRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	entries := cleanEntries(req.Entries)
	if len(entries) == 0 {
		server.writeError(ctx, http.StatusBadRequest, errors.New("no codes given"))
		return
	}
	matched, err := server.store.SetVariantPickForEntries(ctx, db.SetVariantPickForEntriesParams{
		VariantPick: req.Mode, Entries: entries,
	})
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	unmatched, err := server.store.ListUnmatchedDiscountEntries(ctx, entries)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	if unmatched == nil {
		unmatched = []string{}
	}
	server.writeJSON(ctx, http.StatusOK, envelope{"set": matched, "unmatched": unmatched})
}
