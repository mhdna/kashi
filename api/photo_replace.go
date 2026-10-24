package api

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	db "github.com/mhdna/kashi/db/sqlc"
)

// Replacing an uploaded photo with another one -- a blurry receipt, the
// wrong page -- from the admin photo viewer, for a role with the "Replace
// Photos" permission (access.go). The replacement goes through the same
// checks and WebP optimisation as a phone upload and is written under a new
// name beside the old file, so the "uploads are immutable" assumption the
// thumbnail cache and ETags rely on (image_thumbnail.go) still holds. The
// row then points at it, upright, and the old file and its cached
// thumbnails are deleted.

// storeReplacementPhoto validates the multipart "image" and writes it into
// dir. false means it already answered with the error.
func (server *Server) storeReplacementPhoto(ctx *gin.Context, dir string) (string, storedImage, bool) {
	fileHeader, err := ctx.FormFile("image")
	if err != nil {
		server.writeError(ctx, http.StatusBadRequest, fmt.Errorf("no image provided"))
		return "", storedImage{}, false
	}
	if fileHeader.Size > maxExpenseUploadImageBytes {
		server.writeError(ctx, http.StatusBadRequest, fmt.Errorf("the image must be under %dMB", maxExpenseUploadImageBytes>>20))
		return "", storedImage{}, false
	}
	contentType := fileHeader.Header.Get("Content-Type")
	ext, ok := allowedExpenseImageContentTypes[contentType]
	if !ok {
		server.writeError(ctx, http.StatusBadRequest, fmt.Errorf("unsupported image type %q", contentType))
		return "", storedImage{}, false
	}

	stored, err := optimizeUploadedImage(fileHeader, contentType, ext)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return "", storedImage{}, false
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return "", storedImage{}, false
	}
	destPath := filepath.Join(dir, uuid.NewString()+stored.Ext)
	if err := os.WriteFile(destPath, stored.Data, 0o644); err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return "", storedImage{}, false
	}
	warmThumbnails(destPath)
	return destPath, stored, true
}

// removeReplacedPhoto deletes a photo that's no longer referenced, along
// with every cached thumbnail or turned copy of it (thumbCachePath names
// them all "<prefix>_<base>.webp"). Best effort: the row already points at
// the replacement, so a leftover file is only wasted space.
func removeReplacedPhoto(filePath string) {
	if err := os.Remove(filePath); err != nil && !errors.Is(err, os.ErrNotExist) {
		log.Printf("replace photo: remove %s: %v", filePath, err)
	}
	cached, _ := filepath.Glob(filepath.Join(filepath.Dir(filePath), ".thumb", "*_"+filepath.Base(filePath)+".webp"))
	for _, path := range cached {
		os.Remove(path)
	}
}

type replacePhotoURI struct {
	ID int64 `uri:"id" binding:"required,min=1"`
}

func (server *Server) replaceBranchExpenseImage(ctx *gin.Context) {
	var uri replacePhotoURI
	if err := ctx.ShouldBindUri(&uri); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	old, err := server.store.GetBranchExpenseImage(ctx, uri.ID)
	if err != nil {
		server.writeError(ctx, statusForLookup(err), err)
		return
	}

	destPath, stored, ok := server.storeReplacementPhoto(ctx, filepath.Dir(old.FilePath))
	if !ok {
		return
	}
	image, err := server.store.ReplaceBranchExpenseImageFile(ctx, db.ReplaceBranchExpenseImageFileParams{
		ID:          old.ID,
		FilePath:    destPath,
		ContentType: stored.ContentType,
		SizeBytes:   int64(len(stored.Data)),
	})
	if err != nil {
		os.Remove(destPath)
		server.writeError(ctx, statusForLookup(err), err)
		return
	}
	removeReplacedPhoto(old.FilePath)

	if expense, err := server.store.GetBranchExpense(ctx, image.BranchExpenseID); err == nil {
		server.adminHub.broadcastAll(adminWSMessage{Type: "branch_expense_images_uploaded", BranchID: expense.BranchID})
	}
	server.writeJSON(ctx, http.StatusOK, envelope{"image": image})
}

func (server *Server) replaceBranchShiftReceiptImage(ctx *gin.Context) {
	var uri replacePhotoURI
	if err := ctx.ShouldBindUri(&uri); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	old, err := server.store.GetBranchShiftReceiptImage(ctx, uri.ID)
	if err != nil {
		server.writeError(ctx, statusForLookup(err), err)
		return
	}

	destPath, stored, ok := server.storeReplacementPhoto(ctx, filepath.Dir(old.FilePath))
	if !ok {
		return
	}
	image, err := server.store.ReplaceBranchShiftReceiptImageFile(ctx, db.ReplaceBranchShiftReceiptImageFileParams{
		ID:          old.ID,
		FilePath:    destPath,
		ContentType: stored.ContentType,
		SizeBytes:   int64(len(stored.Data)),
	})
	if err != nil {
		os.Remove(destPath)
		server.writeError(ctx, statusForLookup(err), err)
		return
	}
	removeReplacedPhoto(old.FilePath)

	if shift, err := server.store.GetBranchShift(ctx, image.BranchShiftID); err == nil {
		server.adminHub.broadcastAll(adminWSMessage{Type: "branch_shift_receipt_uploaded", BranchID: shift.BranchID})
	}
	server.writeJSON(ctx, http.StatusOK, envelope{"image": image})
}

// replaceTransferAttachment swaps a transfer's one photo. A transfer without
// a photo has nothing to replace (404) -- photos are added from the till's
// QR flow, not here.
func (server *Server) replaceTransferAttachment(ctx *gin.Context) {
	var uri replacePhotoURI
	if err := ctx.ShouldBindUri(&uri); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	transfer, err := server.store.GetTransfer(ctx, uri.ID)
	if err != nil {
		server.writeError(ctx, statusForLookup(err), err)
		return
	}
	if transfer.AttachmentPath == "" {
		server.writeError(ctx, http.StatusNotFound, fmt.Errorf("no photo on this transfer"))
		return
	}

	destPath, _, ok := server.storeReplacementPhoto(ctx, filepath.Dir(transfer.AttachmentPath))
	if !ok {
		return
	}
	// Also resets the turn to upright.
	if err := server.store.SetTransferAttachmentPath(ctx, db.SetTransferAttachmentPathParams{
		ID:             transfer.ID,
		AttachmentPath: destPath,
	}); err != nil {
		os.Remove(destPath)
		if errors.Is(err, sql.ErrNoRows) {
			server.writeError(ctx, http.StatusNotFound, err)
			return
		}
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	removeReplacedPhoto(transfer.AttachmentPath)

	server.adminHub.broadcastAll(adminWSMessage{Type: "branch_transfer_photo_uploaded", BranchID: transfer.BranchID.Int64})
	server.writeJSON(ctx, http.StatusOK, envelope{"success": true})
}
