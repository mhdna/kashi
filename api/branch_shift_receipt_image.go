package api

import (
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	db "github.com/mhdna/kashi/db/sqlc"
)

const defaultShiftReceiptImageStorageDir = "./storage/shift_receipt_images"

func (server *Server) shiftReceiptImageStorageDir() string {
	if dir := server.config.ShiftReceiptImageStorageDir; dir != "" {
		return dir
	}
	return defaultShiftReceiptImageStorageDir
}

// createBranchShiftReceiptImage is the branch-authenticated endpoint
// kashi-pos forwards a close-shift receipt photo to (see
// sync_shift_receipts.go in that repo). The photo was already captured on
// the POS; this just persists the one it kept retrying to deliver.
//
// Idempotent two ways: a 409 when the shift close itself hasn't synced yet
// (the POS keeps retrying), and a plain 200 no-op when the shift already has
// a photo (a retried forward after the POS missed our first 200).
func (server *Server) createBranchShiftReceiptImage(ctx *gin.Context) {
	branchID := ctx.MustGet(branchIDKey).(int64)

	shiftRef := ctx.PostForm("shift_ref")
	if shiftRef == "" {
		server.writeError(ctx, http.StatusBadRequest, fmt.Errorf("shift_ref is required"))
		return
	}

	shift, err := server.store.GetBranchShiftByClientRef(ctx, db.GetBranchShiftByClientRefParams{
		BranchID:  branchID,
		ClientRef: shiftRef,
	})
	if err != nil {
		if err == sql.ErrNoRows {
			server.writeError(ctx, http.StatusConflict, fmt.Errorf("shift not received yet"))
			return
		}
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	if _, err := server.store.GetBranchShiftReceiptImageForShift(ctx, shift.ID); err == nil {
		server.writeJSON(ctx, http.StatusOK, envelope{"success": true, "already_uploaded": true})
		return
	} else if err != sql.ErrNoRows {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	fileHeader, err := ctx.FormFile("image")
	if err != nil {
		server.writeError(ctx, http.StatusBadRequest, fmt.Errorf("image file is required"))
		return
	}
	if fileHeader.Size > maxExpenseUploadImageBytes {
		server.writeError(ctx, http.StatusBadRequest, fmt.Errorf("the image must be under %dMB", maxExpenseUploadImageBytes>>20))
		return
	}
	contentType := fileHeader.Header.Get("Content-Type")
	ext, ok := allowedExpenseImageContentTypes[contentType]
	if !ok {
		server.writeError(ctx, http.StatusBadRequest, fmt.Errorf("unsupported image type %q", contentType))
		return
	}

	destDir := filepath.Join(server.shiftReceiptImageStorageDir(), fmt.Sprintf("%d", shift.ID))
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	destPath := filepath.Join(destDir, uuid.NewString()+ext)
	if err := saveUploadedExpenseImage(fileHeader, destPath); err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	image, err := server.store.CreateBranchShiftReceiptImage(ctx, db.CreateBranchShiftReceiptImageParams{
		BranchShiftID: shift.ID,
		FilePath:      destPath,
		ContentType:   contentType,
		SizeBytes:     fileHeader.Size,
	})
	if err != nil {
		// A unique-violation race (two forwards landing together) is the
		// same "already have it" outcome as the early check above.
		if isUniqueViolation(err) {
			os.Remove(destPath)
			server.writeJSON(ctx, http.StatusOK, envelope{"success": true, "already_uploaded": true})
			return
		}
		os.Remove(destPath)
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	server.adminHub.broadcastAll(adminWSMessage{
		Type:     "branch_shift_receipt_uploaded",
		BranchID: shift.BranchID,
		Label:    shift.ClosingPersonName,
	})

	server.writeJSON(ctx, http.StatusOK, envelope{"image": image})
}

type listBranchShiftReceiptImagesRequest struct {
	PageSize int32 `form:"page_size,default=20" binding:"min=5,max=100"`
	PageID   int32 `form:"page_id,default=0" binding:"min=0"`
}

// listBranchShiftReceiptImages feeds the admin Storage explorer -- every
// close-shift receipt photo, newest first, with its parent shift and branch
// inlined. Mirrors listBranchExpenseImages.
func (server *Server) listBranchShiftReceiptImages(ctx *gin.Context) {
	var req listBranchShiftReceiptImagesRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}

	images, err := server.store.ListBranchShiftReceiptImages(ctx, db.ListBranchShiftReceiptImagesParams{
		Limit:  req.PageSize,
		Offset: req.PageID * req.PageSize,
	})
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	total, err := server.store.CountBranchShiftReceiptImages(ctx)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	server.writeJSON(ctx, http.StatusOK, envelope{"images": images, "total": total})
}

type branchShiftReceiptImageForShiftRequest struct {
	ID int64 `uri:"id" binding:"required,min=1"`
}

// getBranchShiftReceiptImageForShift returns the receipt photo's metadata
// for one branch_shifts row, or 404 if none was uploaded. The admin Shifts
// page uses ListBranchShifts' has_receipt_image flag for the per-row
// indicator; this is the follow-up when the operator opens one.
func (server *Server) getBranchShiftReceiptImageForShift(ctx *gin.Context) {
	var req branchShiftReceiptImageForShiftRequest
	if err := ctx.ShouldBindUri(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}

	image, err := server.store.GetBranchShiftReceiptImageForShift(ctx, req.ID)
	if err != nil {
		if err == sql.ErrNoRows {
			server.writeError(ctx, http.StatusNotFound, fmt.Errorf("no receipt photo for this shift"))
			return
		}
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	server.writeJSON(ctx, http.StatusOK, envelope{"image": image})
}

type getBranchShiftReceiptImageFileRequest struct {
	ID int64 `uri:"id" binding:"required,min=1"`
}

// getBranchShiftReceiptImageFile streams a receipt photo from disk, under
// the same Bearer-token gate as the rest of the admin API -- an <img> tag
// can't send an Authorization header, so the admin UI fetches this as a blob
// and renders it via an object URL. Same reasoning as
// getBranchExpenseImageFile.
func (server *Server) getBranchShiftReceiptImageFile(ctx *gin.Context) {
	var req getBranchShiftReceiptImageFileRequest
	if err := ctx.ShouldBindUri(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}

	image, err := server.store.GetBranchShiftReceiptImage(ctx, req.ID)
	if err != nil {
		server.writeError(ctx, http.StatusNotFound, err)
		return
	}

	ctx.Header("Content-Type", image.ContentType)
	ctx.File(image.FilePath)
}
