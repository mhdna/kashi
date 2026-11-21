package api

import (
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"
	db "github.com/mhdna/kashi/db/sqlc"
)

// suggestionLimit is how many choices the filter boxes list under what's typed.
const suggestionLimit = 10

type suggestBranchInvoiceRequest struct {
	// field: the filter box the choices are for.
	Field string `form:"field" binding:"required,oneof=salesperson client product"`
	// q: what is typed so far; * is a wildcard, as in the filter itself.
	Q            string `form:"q"`
	BranchID     int64  `form:"branch_id"`
	ProductField string `form:"product_field" binding:"omitempty,oneof=code name barcode"`
}

type invoiceSuggestion struct {
	// Value is what goes into the box when picked; Hint is the other detail
	// shown beside it (a client's phone, a product's name).
	Value string `json:"value"`
	Hint  string `json:"hint"`
}

// suggestBranchInvoiceFilter lists salesperson names, clients or products
// matching what is typed in the Retail invoices filter dialog, so the boxes
// complete as you type. It needs at least a letter typed: an empty box has
// nothing to complete.
func (server *Server) suggestBranchInvoiceFilter(ctx *gin.Context) {
	var req suggestBranchInvoiceRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	out := []invoiceSuggestion{}
	pattern := wildcardPattern(req.Q)
	if pattern == "" || req.Field == "product" && req.ProductField == "barcode" {
		server.writeJSON(ctx, http.StatusOK, envelope{"suggestions": out})
		return
	}

	switch req.Field {
	case "salesperson":
		branch := sql.NullInt64{Int64: req.BranchID, Valid: req.BranchID > 0}
		names, err := server.store.SuggestSalespersonNames(ctx, db.SuggestSalespersonNamesParams{
			Pattern: pattern, BranchID: branch, MaxRows: suggestionLimit,
		})
		if err != nil {
			server.writeError(ctx, http.StatusInternalServerError, err)
			return
		}
		for _, name := range names {
			out = append(out, invoiceSuggestion{Value: name})
		}
	case "client":
		rows, err := server.store.SuggestInvoiceClients(ctx, db.SuggestInvoiceClientsParams{
			ClientPattern: pattern, ClientDigits: phonePattern(req.Q), MaxRows: suggestionLimit,
		})
		if err != nil {
			server.writeError(ctx, http.StatusInternalServerError, err)
			return
		}
		for _, c := range rows {
			out = append(out, invoiceSuggestion{Value: c.Name, Hint: c.Phone})
		}
	default:
		field := req.ProductField
		rows, err := server.store.SuggestInvoiceProducts(ctx, db.SuggestInvoiceProductsParams{
			ProductField: field, Pattern: likePattern(req.Q, "contains"), MaxRows: suggestionLimit,
		})
		if err != nil {
			server.writeError(ctx, http.StatusInternalServerError, err)
			return
		}
		for _, p := range rows {
			if field == "name" {
				out = append(out, invoiceSuggestion{Value: p.Name, Hint: p.Code})
			} else {
				out = append(out, invoiceSuggestion{Value: p.Code, Hint: p.Name})
			}
		}
	}
	server.writeJSON(ctx, http.StatusOK, envelope{"suggestions": out})
}
