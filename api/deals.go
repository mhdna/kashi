package api

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	db "github.com/mhdna/kashi/db/sqlc"
)

// Deals (000201): "Buy 2 get 1 free", "3 for $50", "Buy 3 or more, 20% off
// all". A deal is only its terms; a discount list row or rule puts it on
// products, and the tills price it (see the migration for the arithmetic).

const (
	dealBuyGet        = "buy_get"
	dealBundlePrice   = "bundle_price"
	dealVolumePercent = "volume_percent"
)

func (server *Server) listDeals(ctx *gin.Context) {
	deals, err := server.store.ListDeals(ctx)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	server.writeJSON(ctx, http.StatusOK, envelope{"deals": deals})
}

type dealRequest struct {
	Name        string `json:"name"`
	Kind        string `json:"kind" binding:"required"`
	BuyQty      int32  `json:"buy_qty" binding:"min=1,max=100"`
	GetQty      int32  `json:"get_qty" binding:"min=0,max=100"`
	Percent     int16  `json:"percent" binding:"min=0,max=100"`
	BundlePrice int64  `json:"bundle_price" binding:"min=0"`
	// What units outside a set get ("Buy 2 get 1, otherwise 30%"); buy_get only.
	FallbackPercent int16 `json:"fallback_percent" binding:"min=0,max=100"`
}

// params checks the request makes sense for its kind and clears the fields
// the kind doesn't read, so the table's CHECK never has to be the one to say.
func (req dealRequest) params() (db.CreateDealParams, error) {
	p := db.CreateDealParams{
		Name: strings.TrimSpace(req.Name), Kind: req.Kind, BuyQty: req.BuyQty,
	}
	if p.Name == "" {
		return p, errors.New("name is required")
	}
	switch req.Kind {
	case dealBuyGet:
		if req.GetQty < 1 {
			return p, errors.New("say how many units the customer gets")
		}
		if req.Percent < 1 {
			return p, errors.New("say how much off they get (100% is free)")
		}
		p.GetQty, p.Percent, p.FallbackPercent = req.GetQty, req.Percent, req.FallbackPercent
	case dealBundlePrice:
		if req.BuyQty < 2 {
			return p, errors.New("a bundle is at least 2 units")
		}
		if req.BundlePrice <= 0 {
			return p, errors.New("give the bundle's price")
		}
		p.BundlePrice = req.BundlePrice
	case dealVolumePercent:
		if req.BuyQty < 2 {
			return p, errors.New("a volume deal starts at 2 units or more")
		}
		if req.Percent < 1 {
			return p, errors.New("say how much off they get")
		}
		p.Percent = req.Percent
	default:
		return p, fmt.Errorf("unknown deal kind %q", req.Kind)
	}
	return p, nil
}

func (server *Server) createDeal(ctx *gin.Context) {
	var req dealRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	p, err := req.params()
	if err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	deal, err := server.store.CreateDeal(ctx, p)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	server.writeJSON(ctx, http.StatusOK, envelope{"deal": deal})
}

type dealURI struct {
	ID int64 `uri:"id" binding:"required,min=1"`
}

// updateDeal changes a deal's terms everywhere it is used: every list naming
// it is re-sent to the tills (000201's trigger). Sales already made keep the
// terms they were sold under; the tills copy them onto the invoice.
func (server *Server) updateDeal(ctx *gin.Context) {
	var uri dealURI
	if err := ctx.ShouldBindUri(&uri); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	var req dealRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	p, err := req.params()
	if err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	deal, err := server.store.UpdateDeal(ctx, db.UpdateDealParams{
		ID: uri.ID, Name: p.Name, Kind: p.Kind, BuyQty: p.BuyQty, GetQty: p.GetQty,
		Percent: p.Percent, BundlePrice: p.BundlePrice, FallbackPercent: p.FallbackPercent,
	})
	if err != nil {
		server.writeError(ctx, statusForLookup(err), err)
		return
	}
	server.writeJSON(ctx, http.StatusOK, envelope{"deal": deal})
}

func (server *Server) deleteDeal(ctx *gin.Context) {
	var uri dealURI
	if err := ctx.ShouldBindUri(&uri); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	if err := server.store.DeleteDeal(ctx, uri.ID); err != nil {
		if isForeignKeyViolation(err) {
			server.writeError(ctx, http.StatusConflict, errors.New("this deal is still on a discount list or rule; take it off first"))
			return
		}
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	server.writeJSON(ctx, http.StatusOK, envelope{"status": "deleted"})
}

// dealParam is the deal a list row or rule request names: 0 is none.
func (server *Server) dealParam(ctx *gin.Context, dealID int64) (sql.NullInt64, error) {
	if dealID == 0 {
		return sql.NullInt64{}, nil
	}
	if _, err := server.store.GetDeal(ctx, dealID); err != nil {
		return sql.NullInt64{}, err
	}
	return sql.NullInt64{Int64: dealID, Valid: true}, nil
}
