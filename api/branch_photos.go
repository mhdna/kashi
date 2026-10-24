package api

import (
	"database/sql"
	"errors"
	"fmt"
	"mime"
	"net/http"
	"path/filepath"

	"github.com/gin-gonic/gin"
	db "github.com/mhdna/kashi/db/sqlc"
)

// Photos a till's own records carry, served back to that till so its
// Expenses and Transfers pages can show them. The phone uploads them straight
// to kashi (one-time QR links), so the till never has a copy. Every route is
// limited to the calling till's branch; ?w= gives a thumbnail as on the admin
// routes (serveStoredImage).

type branchExpensePhotosRequest struct {
	ClientRef string `form:"client_ref" binding:"required"`
}

// branchExpensePhotos lists the photos of one of this branch's expenses, by
// the till's own client_ref. An expense kashi hasn't received yet has none.
func (server *Server) branchExpensePhotos(ctx *gin.Context) {
	branchID := ctx.MustGet(branchIDKey).(int64)
	var req branchExpensePhotosRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	expense, err := server.store.GetBranchExpenseByClientRef(ctx, db.GetBranchExpenseByClientRefParams{
		BranchID: branchID, ClientRef: req.ClientRef,
	})
	if errors.Is(err, sql.ErrNoRows) {
		server.writeJSON(ctx, http.StatusOK, envelope{"images": []db.BranchExpenseImage{}})
		return
	}
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	images, err := server.store.ListBranchExpenseImagesForExpense(ctx, expense.ID)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	server.writeJSON(ctx, http.StatusOK, envelope{"images": images})
}

// branchExpensePhotoFirst serves the first photo of one of this branch's
// expenses in a single request (the till's list rows used to list ids and then
// fetch the file: two round trips each). No expense or no photo is a 404,
// which the till shows as no picture.
func (server *Server) branchExpensePhotoFirst(ctx *gin.Context) {
	branchID := ctx.MustGet(branchIDKey).(int64)
	var req branchExpensePhotosRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	expense, err := server.store.GetBranchExpenseByClientRef(ctx, db.GetBranchExpenseByClientRefParams{
		BranchID: branchID, ClientRef: req.ClientRef,
	})
	if err != nil {
		server.writeError(ctx, statusForLookup(err), err)
		return
	}
	images, err := server.store.ListBranchExpenseImagesForExpense(ctx, expense.ID)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	if len(images) == 0 {
		server.writeError(ctx, http.StatusNotFound, errors.New("no photo"))
		return
	}
	server.serveStoredImage(ctx, images[0].FilePath, images[0].ContentType, images[0].Rotation)
}

type branchPhotoIDRequest struct {
	ID int64 `uri:"id" binding:"required,min=1"`
}

// branchExpensePhotoFile serves one expense photo, if its expense is this
// branch's.
func (server *Server) branchExpensePhotoFile(ctx *gin.Context) {
	branchID := ctx.MustGet(branchIDKey).(int64)
	var req branchPhotoIDRequest
	if err := ctx.ShouldBindUri(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	image, err := server.store.GetBranchExpenseImage(ctx, req.ID)
	if err != nil {
		server.writeError(ctx, statusForLookup(err), err)
		return
	}
	expense, err := server.store.GetBranchExpense(ctx, image.BranchExpenseID)
	if err != nil || expense.BranchID != branchID {
		server.writeError(ctx, http.StatusNotFound, errors.New("no such photo for this branch"))
		return
	}
	server.serveStoredImage(ctx, image.FilePath, image.ContentType, image.Rotation)
}

type branchTransferPhotoRequest struct {
	// The till's own ref for a transfer it sent, or kashi's id for one it
	// received (the till only knows that one by kashi's id).
	ClientRef  string `form:"client_ref"`
	TransferID int64  `form:"transfer_id"`
}

// branchTransferPhoto serves the photo of a transfer this branch sent or
// receives. No photo is a 404, which the till shows as no picture.
func (server *Server) branchTransferPhoto(ctx *gin.Context) {
	branchID := ctx.MustGet(branchIDKey).(int64)
	var req branchTransferPhotoRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	var transfer db.Transfer
	var err error
	switch {
	case req.ClientRef != "":
		transfer, err = server.store.GetBranchTransferByClientRef(ctx, db.GetBranchTransferByClientRefParams{
			BranchID:  sql.NullInt64{Int64: branchID, Valid: true},
			ClientRef: sql.NullString{String: req.ClientRef, Valid: true},
		})
	case req.TransferID > 0:
		transfer, err = server.store.GetTransfer(ctx, req.TransferID)
	default:
		server.writeError(ctx, http.StatusBadRequest, errors.New("client_ref or transfer_id is required"))
		return
	}
	if err != nil {
		server.writeError(ctx, statusForLookup(err), err)
		return
	}
	if !server.transferTouchesBranch(ctx, transfer, branchID) {
		server.writeError(ctx, http.StatusNotFound, errors.New("no such transfer for this branch"))
		return
	}
	if transfer.AttachmentPath == "" {
		server.writeError(ctx, http.StatusNotFound, fmt.Errorf("no photo on this transfer"))
		return
	}
	contentType := mime.TypeByExtension(filepath.Ext(transfer.AttachmentPath))
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	server.serveStoredImage(ctx, transfer.AttachmentPath, contentType, transfer.AttachmentRotation)
}

// transferTouchesBranch is true for a transfer the branch filed, or one
// moving stock into or out of the branch's store inventory.
func (server *Server) transferTouchesBranch(ctx *gin.Context, transfer db.Transfer, branchID int64) bool {
	if transfer.BranchID.Valid && transfer.BranchID.Int64 == branchID {
		return true
	}
	branch, err := server.store.GetBranch(ctx, branchID)
	if err != nil || !branch.InventoryID.Valid {
		return false
	}
	inv := branch.InventoryID.Int64
	return transfer.FromInventoryID == inv || transfer.ToInventoryID == inv
}
