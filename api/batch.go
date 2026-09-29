package api

import (
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"
	db "github.com/mhdna/kashi/db/sqlc"
)

type createBatchRequest struct {
	Code       string `json:"code"`
	PurchaseID int64  `json:"purchase_id" binding:"required"`
	Note       string `json:"note"`
}

// createBatch snapshots a purchase invoice's variant lines into a new
// batch: one batch item per SKU, freezing quantity and cost as bought.
func (server *Server) createBatch(ctx *gin.Context) {
	var req createBatchRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}

	result, err := server.store.CreateBatchTx(ctx, db.CreateBatchTxParams{
		Code:       req.Code,
		PurchaseID: req.PurchaseID,
		Note:       req.Note,
	})
	if err != nil {
		if err == sql.ErrNoRows {
			server.writeError(ctx, http.StatusNotFound, err)
			return
		}
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"batch": result.Batch, "items": result.Items})
}

type batchIDRequest struct {
	ID int64 `uri:"id" binding:"required,min=1"`
}

// getBatch returns the batch header plus its per-product stats: bought,
// sold, revenue and on-hand, joined live off the invoice/inventory data.
func (server *Server) getBatch(ctx *gin.Context) {
	var req batchIDRequest
	if err := ctx.ShouldBindUri(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}

	batch, err := server.store.GetBatch(ctx, req.ID)
	if err != nil {
		if err == sql.ErrNoRows {
			server.writeError(ctx, http.StatusNotFound, err)
			return
		}
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	items, err := server.store.ListBatchItemsStats(ctx, req.ID)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"batch": batch, "items": items})
}

type listBatchesRequest struct {
	PageSize int32  `form:"page_size,default=10" binding:"min=5,max=100"`
	PageID   int32  `form:"page_id,default=0" binding:"min=0"`
	Search   string `form:"search"`
}

func (server *Server) listBatches(ctx *gin.Context) {
	var req listBatchesRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}

	batches, err := server.store.ListBatches(ctx, db.ListBatchesParams{
		Search:     req.Search,
		PageSize:   req.PageSize,
		PageOffset: req.PageID,
	})
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	total, err := server.store.CountBatchesFiltered(ctx, req.Search)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"batches": batches, "total": total})
}

func (server *Server) deleteBatch(ctx *gin.Context) {
	var req batchIDRequest
	if err := ctx.ShouldBindUri(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}

	if err := server.store.DeleteBatch(ctx, req.ID); err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"deleted": true})
}
