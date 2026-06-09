package api

import (
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"
	db "github.com/mhdna/kashi/db/sqlc"
)

type createReturnInvoiceRequest struct {
	CashboxID        int64 `json:"cashbox_id" binding:"required"`
	CashboxAccountID int64 `json:"cashbox_account_id" binding:"required"`
	ShiftID          int64 `json:"shift_id" binding:"required"`
	InventoryID      int64 `json:"inventory_id" binding:"required"`
	Year             int32 `json:"year" binding:"required"`
	ClientID         int64 `json:"client_id" binding:"required"`
	SalesInvoiceID   int64 `json:"sales_invoice_id" binding:"required"`
	Discount         int16 `json:"discount" binding:"required"`
	GrandTotal       int64 `json:"grand_total" binding:"required"`
	Subtotal         int64 `json:"sub_total" binding:"required"`
	DiscountedTotal  int64 `json:"discounted_total" binding:"required"`
}

func (server *Server) createReturnInvoice(ctx *gin.Context) {
	var req createReturnInvoiceRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	arg := db.ReturnInvoiceTxParams{
		CashboxID:        req.CashboxID,
		CashboxAccountID: req.CashboxAccountID,
		ShiftID:          req.ShiftID,
		InventoryID:      req.InventoryID,
		Year:             req.Year,
		ClientID:         req.ClientID,
		SalesInvoiceID:   req.SalesInvoiceID,
		Discount:         req.Discount,
		GrandTotal:       req.GrandTotal,
		SubTotal:         req.Subtotal,
		DiscountedTotal:  req.DiscountedTotal,
	}

	returnInvoice, err := server.store.ReturnInvoiceTx(ctx, arg)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}
	ctx.JSON(http.StatusOK, returnInvoice)
}

type getReturnInvoiceRequest struct {
	ID int64 `uri:"id" binding:"required,min=1"`
}

func (server *Server) getReturnInvoice(ctx *gin.Context) {
	var req getReturnInvoiceRequest
	if err := ctx.ShouldBindUri(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	invoice, err := server.store.GetInvoice(ctx, req.ID)
	if err != nil {
		if err == sql.ErrNoRows {
			ctx.JSON(http.StatusNotFound, errorResponse(err))
			return
		}

		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, invoice)
}

type listReturnInvoiceRequest struct {
	PageSize int32 `form:"page_size,default=10" binding:"min=5,max=10"`
	PageID   int32 `form:"page_id,default=1" binding:"min=1"`
}

func (server *Server) listReturnInvoices(ctx *gin.Context) {
	var req listReturnInvoiceRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	arg := db.ListInvoicesParams{
		Limit:  req.PageSize,
		Offset: req.PageID,
	}
	invoices, err := server.store.ListInvoices(ctx, arg)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, invoices)
}
