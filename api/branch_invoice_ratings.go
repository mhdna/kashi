package api

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	db "github.com/mhdna/kashi/db/sqlc"
)

// How clients rated this branch's sales in the WhatsApp poll, for the till's
// invoices list (kashi-pos invoice_ratings.go): the till asks for the codes
// on screen and shows each answer as an emoji. Only this branch's sales.

const maxInvoiceRatingCodes = 200

type branchInvoiceRating struct {
	// 1-5 stars: the poll's options spread over the scale (whatsapp_poll.go).
	Rating int16 `json:"rating"`
	// The option as the client saw it, e.g. "😍 Loved it".
	Answer string `json:"answer"`
}

func (server *Server) branchInvoiceRatings(ctx *gin.Context) {
	branchID := ctx.MustGet(branchIDKey).(int64)
	var req struct {
		Codes []string `json:"codes"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	codes := make([]string, 0, len(req.Codes))
	for _, code := range req.Codes {
		if code = strings.TrimSpace(code); code != "" && len(codes) < maxInvoiceRatingCodes {
			codes = append(codes, code)
		}
	}
	out := map[string]branchInvoiceRating{}
	if len(codes) > 0 {
		rows, err := server.store.ListBranchInvoiceRatings(ctx, db.ListBranchInvoiceRatingsParams{
			BranchID: branchID, InvoiceCodes: codes,
		})
		if err != nil {
			server.writeError(ctx, http.StatusInternalServerError, err)
			return
		}
		for _, r := range rows {
			out[r.InvoiceCode] = branchInvoiceRating{Rating: r.PollRating, Answer: r.PollAnswer}
		}
	}
	server.writeJSON(ctx, http.StatusOK, envelope{"ratings": out})
}
