package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	db "github.com/mhdna/kashi/db/sqlc"
)

// Turning an uploaded photo the right way up. Phones often send receipts
// sideways, so a role with the "Rotate Photos" permission (access.go) can
// turn one from the admin photo viewer. The turn is saved on the photo's row
// and applied by serveStoredImage wherever the photo is served -- the admin
// pages and the till alike -- while the uploaded file stays untouched.

// rotatePhotoRequest is relative: delta is added to whatever turn the photo
// already has, so two people turning it at once both count instead of one
// overwriting the other.
type rotatePhotoRequest struct {
	Delta int32 `json:"delta" binding:"oneof=-90 90 180"`
}

type rotatePhotoURI struct {
	ID int64 `uri:"id" binding:"required,min=1"`
}

// bindRotatePhoto reads the photo id and delta; false means it already
// answered with the error.
func (server *Server) bindRotatePhoto(ctx *gin.Context) (int64, int32, bool) {
	var uri rotatePhotoURI
	if err := ctx.ShouldBindUri(&uri); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return 0, 0, false
	}
	var req rotatePhotoRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return 0, 0, false
	}
	return uri.ID, req.Delta, true
}

func (server *Server) rotateBranchExpenseImage(ctx *gin.Context) {
	id, delta, ok := server.bindRotatePhoto(ctx)
	if !ok {
		return
	}
	rotation, err := server.store.RotateBranchExpenseImage(ctx, db.RotateBranchExpenseImageParams{Delta: delta, ID: id})
	if err != nil {
		server.writeError(ctx, statusForLookup(err), err)
		return
	}
	server.writeJSON(ctx, http.StatusOK, envelope{"rotation": rotation})
}

func (server *Server) rotateBranchShiftReceiptImage(ctx *gin.Context) {
	id, delta, ok := server.bindRotatePhoto(ctx)
	if !ok {
		return
	}
	rotation, err := server.store.RotateBranchShiftReceiptImage(ctx, db.RotateBranchShiftReceiptImageParams{Delta: delta, ID: id})
	if err != nil {
		server.writeError(ctx, statusForLookup(err), err)
		return
	}
	server.writeJSON(ctx, http.StatusOK, envelope{"rotation": rotation})
}

// rotateTransferAttachment turns a transfer's one photo; a transfer without
// a photo is a 404.
func (server *Server) rotateTransferAttachment(ctx *gin.Context) {
	id, delta, ok := server.bindRotatePhoto(ctx)
	if !ok {
		return
	}
	rotation, err := server.store.RotateTransferAttachment(ctx, db.RotateTransferAttachmentParams{Delta: delta, ID: id})
	if err != nil {
		server.writeError(ctx, statusForLookup(err), err)
		return
	}
	server.writeJSON(ctx, http.StatusOK, envelope{"rotation": rotation})
}
