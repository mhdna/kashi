package api

import (
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"
	db "github.com/mhdna/kashi/db/sqlc"
)

// defaultProductCategoryIcon / Color mirror assetCategoryIcon/Color.
const (
	defaultProductCategoryIcon  = "mdi-tag-outline"
	defaultProductCategoryColor = "blue-grey"
)

func productCategoryIcon(icon string) string {
	if icon == "" {
		return defaultProductCategoryIcon
	}
	return icon
}

func productCategoryColor(color string) string {
	if color == "" {
		return defaultProductCategoryColor
	}
	return color
}

type createProductCategoryRequest struct {
	Name     string `json:"name" binding:"required"`
	IsActive bool   `json:"is_active"`
	Icon     string `json:"icon"`
	Color    string `json:"color"`
}

func (server *Server) createProductCategory(ctx *gin.Context) {
	var req createProductCategoryRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}

	category, err := server.store.CreateProductCategory(ctx, db.CreateProductCategoryParams{
		Name:     req.Name,
		IsActive: req.IsActive,
		Icon:     productCategoryIcon(req.Icon),
		Color:    productCategoryColor(req.Color),
	})
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	ctx.JSON(http.StatusOK, category)
}

type getProductCategoryRequest struct {
	ID int64 `uri:"id" binding:"required,min=1"`
}

func (server *Server) getProductCategory(ctx *gin.Context) {
	var req getProductCategoryRequest
	if err := ctx.ShouldBindUri(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}

	category, err := server.store.GetProductCategory(ctx, req.ID)
	if err != nil {
		if err == sql.ErrNoRows {
			server.writeError(ctx, http.StatusNotFound, err)
			return
		}
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	ctx.JSON(http.StatusOK, category)
}

func (server *Server) listProductCategories(ctx *gin.Context) {
	categories, err := server.store.ListProductCategories(ctx)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	ctx.JSON(http.StatusOK, categories)
}

type updateProductCategoryRequest struct {
	ID       int64  `json:"id" binding:"required,min=1"`
	Name     string `json:"name" binding:"required"`
	IsActive bool   `json:"is_active"`
	Icon     string `json:"icon"`
	Color    string `json:"color"`
}

func (server *Server) updateProductCategory(ctx *gin.Context) {
	var req updateProductCategoryRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}

	category, err := server.store.UpdateProductCategory(ctx, db.UpdateProductCategoryParams{
		ID:       req.ID,
		Name:     req.Name,
		IsActive: req.IsActive,
		Icon:     productCategoryIcon(req.Icon),
		Color:    productCategoryColor(req.Color),
	})
	if err != nil {
		if err == sql.ErrNoRows {
			server.writeError(ctx, http.StatusNotFound, err)
			return
		}
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	ctx.JSON(http.StatusOK, category)
}

type deleteProductCategoryRequest struct {
	ID int64 `uri:"id" binding:"required,min=1"`
}

// deleteProductCategory hard-deletes a category. products.category_id has no
// ON DELETE CASCADE, so deleting a category still referenced by a product
// fails on the FK constraint -- surfaced as a 409, same convention as
// deleteAssetCategory.
func (server *Server) deleteProductCategory(ctx *gin.Context) {
	var req deleteProductCategoryRequest
	if err := ctx.ShouldBindUri(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}

	if err := server.store.DeleteProductCategory(ctx, req.ID); err != nil {
		if isForeignKeyViolation(err) {
			server.writeError(ctx, http.StatusConflict, err)
			return
		}
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"deleted": true})
}
