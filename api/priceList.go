package api

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	db "github.com/mhdna/kashi/db/sqlc"
)

type createPriceListRequest struct {
	Name      string    `json:"name" binding:"required"`
	IsActive  bool      `json:"is_active"`
	IsDefault bool      `json:"is_default"`
	ValidFrom time.Time `json:"valid_from"`
	ValidTo   time.Time `json:"valid_to"`
}

func (server *Server) createPriceList(ctx *gin.Context) {
	var req createPriceListRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	arg := db.CreatePriceListParams{
		Name:      req.Name,
		IsActive:  req.IsActive,
		IsDefault: req.IsDefault,
		ValidFrom: req.ValidFrom,
		ValidTo:   req.ValidTo,
	}

	priceList, err := server.store.CreatePriceList(ctx, arg)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}
	ctx.JSON(http.StatusOK, priceList)
}

type getPriceListRequest struct {
	ID int64 `uri:"id" binding:"required,min=1"`
}

func (server *Server) getPriceList(ctx *gin.Context) {
	var req getPriceListRequest
	if err := ctx.ShouldBindUri(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	priceList, err := server.store.GetPriceList(ctx, req.ID)
	if err != nil {
		if err == sql.ErrNoRows {
			ctx.JSON(http.StatusNotFound, errorResponse(err))
			return
		}

		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, priceList)
}

type listPriceListsRequest struct {
	PageSize int32 `form:"page_size,default=10" binding:"min=5,max=10"`
	PageID   int32 `form:"page_id,default=1" binding:"min=1"`
}

func (server *Server) listPriceLists(ctx *gin.Context) {
	var req listPriceListsRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	arg := db.ListPriceListsParams{
		Limit:  req.PageSize,
		Offset: req.PageID,
	}
	priceLists, err := server.store.ListPriceLists(ctx, arg)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, priceLists)
}
