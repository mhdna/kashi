package api

import (
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"
	db "github.com/mhdna/kashi/db/sqlc"
)

type createSalesInvoiceRequest struct {
	CashboxID        int64 `json:"cashbox_id" binding:"required"`
	CashboxAccountID int64 `json:"cashbox_account_id" binding:"required"`
	InventoryID      int64 `json:"inventory_id" binding:"required"`
	Year             int32 `json:"year" binding:"required"`
	ClientID         int64 `json:"client_id" binding:"required"`
	Amount           int64 `json:"amount" binding:"required"`
	// TODO: add netamount at run time instead of calculating it
	// NetAmount    int64  `json:"net_amount" binding:"required"`
	Discount int16 `json:"discount" binding:"required"`
}

func (server *Server) createSalesInvoice(ctx *gin.Context) {
	var req createSalesInvoiceRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	arg := db.SalesInvoiceTxParams{
		CashboxID:        req.CashboxID,
		CashboxAccountID: req.CashboxAccountID,
		InventoryID:      req.InventoryID,
		Year:             req.Year,
		ClientID:         req.ClientID,
		Amount:           req.Amount,
		CurrencyCode:     req.CurrencyCode,
		Discount:         req.Discount,
	}

	salesInvoice, err := server.store.SalesInvoiceTx(ctx, arg)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}
	ctx.JSON(http.StatusOK, salesInvoice)
}

type getSalesInvoiceRequest struct {
	ID int64 `uri:"id" binding:"required,min=1"`
}

func (server *Server) getSalesInvoice(ctx *gin.Context) {
	var req getSalesInvoiceRequest
	if err := ctx.ShouldBindUri(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	salesInvoice, err := server.store.GetSalesInvoice(ctx, req.ID)
	if err != nil {
		if err == sql.ErrNoRows {
			ctx.JSON(http.StatusNotFound, errorResponse(err))
			return
		}

		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, salesInvoice)
}

type listSalesInvoiceRequest struct {
	PageSize int32 `form:"page_size,default=10" binding:"min=5,max=10"`
	PageID   int32 `form:"page_id,default=1" binding:"min=1"`
}

func (server *Server) listSalesInvoices(ctx *gin.Context) {
	var req listSalesInvoiceRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	arg := db.ListSalesInvoicesParams{
		Limit:  req.PageSize,
		Offset: req.PageID,
	}
	salesInvoice, err := server.store.ListSalesInvoices(ctx, arg)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, salesInvoice)
}
