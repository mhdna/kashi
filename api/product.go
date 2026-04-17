package api

import (
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"
	db "github.com/mhdna/kashi/db/sqlc"
)

type createProductRequest struct {
	Code        string `json:"code" binding:"required"`
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	Price       int64  `json:"price"`
	Discount    int64  `json:"discount"`
}

func (server *Server) createProduct(ctx *gin.Context) {
	var req createProductRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	arg := db.CreateProductParams{
		Code:        req.Code,
		Name:        req.Name,
		Description: req.Description,
		Price:       req.Price,
		Discount:    req.Discount,
	}

	product, err := server.store.CreateProduct(ctx, arg)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, product)
}

type getProductRequest struct {
	ID int64 `uri:"id" binding:"required,min=1"`
}

func (server *Server) getProduct(ctx *gin.Context) {
	var req getProductRequest
	if err := ctx.ShouldBindUri(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	product, err := server.store.GetProduct(ctx, req.ID)
	if err != nil {
		if err == sql.ErrNoRows {
			ctx.JSON(http.StatusNotFound, errorResponse(err))
			return
		}

		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, product)
}

type listProductRequest struct {
	PageSize int32 `form:"page_size,default=10" binding:"min=5,max=10"`
	PageID   int32 `form:"page_id,default=1" binding:"min=1"`
}

func (server *Server) listProducts(ctx *gin.Context) {
	var req listProductRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	arg := db.ListProductsParams{
		Limit:  req.PageSize,
		Offset: req.PageID,
	}
	products, err := server.store.ListProducts(ctx, arg)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, products)
}

type deleteProductRequest struct {
	ID int64 `uri:"id" binding:"required,min=1"`
}

func (server *Server) deleteProduct(ctx *gin.Context) {
	var req deleteProductRequest
	if err := ctx.ShouldBindUri(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	err := server.store.DeleteProduct(ctx, req.ID)
	if err != nil {
		if err == sql.ErrNoRows {
			ctx.JSON(http.StatusNotFound, errorResponse(err))
			return
		}

		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "deleted"})
}
