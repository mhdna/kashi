package api

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	db "github.com/mhdna/kashi/db/sqlc"
)

// Sale options (000199) are the choices a till asks for when it sells a
// product that offers them -- "Summer" or "Winter" for outlet stock whose one
// barcode covers both. They change what a discount list gives the line (see
// discount_list_option_items), nothing else.

func (server *Server) listSaleOptions(ctx *gin.Context) {
	options, err := server.store.ListSaleOptions(ctx)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	server.writeJSON(ctx, http.StatusOK, envelope{"sale_options": options})
}

type createSaleOptionRequest struct {
	Name      string `json:"name" binding:"required"`
	SortOrder int32  `json:"sort_order"`
}

// There is no rename: a till knows an option only by its name, so renaming
// one would orphan every till's copy of the products and lists using it.
func (server *Server) createSaleOption(ctx *gin.Context) {
	var req createSaleOptionRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		server.writeError(ctx, http.StatusBadRequest, errors.New("name is required"))
		return
	}
	option, err := server.store.CreateSaleOption(ctx, db.CreateSaleOptionParams{Name: name, SortOrder: req.SortOrder})
	if err != nil {
		if isUniqueViolation(err) {
			server.writeError(ctx, http.StatusConflict, errors.New("an option with that name already exists"))
			return
		}
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	server.writeJSON(ctx, http.StatusOK, envelope{"sale_option": option})
}

type setProductSaleOptionsRequest struct {
	// Product codes and/or variant barcodes, as typed or pasted -- the same
	// entries the discount list bulk dialog takes.
	Entries []string `json:"entries" binding:"required,min=1,max=5000"`
	// The products' options become exactly these; empty clears them.
	SaleOptionIDs []int64 `json:"sale_option_ids" binding:"dive,min=1"`
}

func (server *Server) setProductSaleOptions(ctx *gin.Context) {
	var req setProductSaleOptionsRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	entries := cleanEntries(req.Entries)
	if len(entries) == 0 {
		server.writeError(ctx, http.StatusBadRequest, errors.New("no codes given"))
		return
	}
	for _, id := range req.SaleOptionIDs {
		if _, err := server.store.GetSaleOption(ctx, id); err != nil {
			server.writeError(ctx, statusForLookup(err), err)
			return
		}
	}
	matched, err := server.store.SetProductSaleOptionsTx(ctx, entries, req.SaleOptionIDs)
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

// cleanEntries trims pasted codes and drops blanks and repeats, keeping the
// order they were given in.
func cleanEntries(raw []string) []string {
	seen := make(map[string]bool, len(raw))
	entries := make([]string, 0, len(raw))
	for _, entry := range raw {
		if entry = strings.TrimSpace(entry); entry != "" && !seen[entry] {
			seen[entry] = true
			entries = append(entries, entry)
		}
	}
	return entries
}
