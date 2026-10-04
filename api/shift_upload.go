package api

import (
	"crypto/rand"
	"database/sql"
	_ "embed"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	db "github.com/mhdna/kashi/db/sqlc"
)

// shiftUploadTokenTTL bounds how long the QR shown in kashi-pos stays
// scannable. Longer than the expense equivalent because the cashier may
// step away to fetch a phone right after closing.
const shiftUploadTokenTTL = 2 * time.Hour

//go:embed shift_upload_assets/upload.html
var shiftUploadPageHTML []byte

// createBranchShiftUploadToken mints a one-time, expiring token for a
// freshly-created branch shift close. Called from createBranchShiftClose
// (branch_shift.go) only on the genuinely-new-insert path.
func (server *Server) createBranchShiftUploadToken(ctx *gin.Context, branchShiftID int64) (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("generate upload token: %w", err)
	}
	token := hex.EncodeToString(raw)

	_, err := server.store.CreateBranchShiftUploadToken(ctx, db.CreateBranchShiftUploadTokenParams{
		BranchShiftID: branchShiftID,
		Token:         token,
		ExpiresAt:     time.Now().UTC().Add(shiftUploadTokenTTL),
	})
	if err != nil {
		return "", err
	}
	return token, nil
}

// shiftUploadPage serves the self-contained mobile upload page (identical
// asset to the one kashi-pos's local server embeds), all validity checking
// happens client-side against the status / images endpoints below.
func (server *Server) shiftUploadPage(ctx *gin.Context) {
	ctx.Data(http.StatusOK, "text/html; charset=utf-8", shiftUploadPageHTML)
}

func (server *Server) lookupValidShiftUploadToken(ctx *gin.Context, token string) (db.BranchShiftUploadToken, error) {
	row, err := server.store.GetBranchShiftUploadTokenByToken(ctx, token)
	if err != nil {
		return db.BranchShiftUploadToken{}, err
	}
	if time.Now().UTC().After(row.ExpiresAt) {
		return db.BranchShiftUploadToken{}, sql.ErrNoRows
	}
	return row, nil
}

func (server *Server) shiftUploadStatus(ctx *gin.Context) {
	token := ctx.Param("token")

	row, err := server.lookupValidShiftUploadToken(ctx, token)
	if err != nil {
		if err == sql.ErrNoRows {
			server.writeError(ctx, http.StatusNotFound, fmt.Errorf("this link is no longer valid"))
			return
		}
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	shift, err := server.store.GetBranchShift(ctx, row.BranchShiftID)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	subtitle := fmt.Sprintf("Shift #%d", shift.BranchShiftID)
	if shift.ClosedAt.Valid {
		subtitle += " · closed " + shift.ClosedAt.Time.Format("2006-01-02 15:04")
	}
	if shift.ClosingPersonName != "" {
		subtitle += " · " + shift.ClosingPersonName
	}

	server.writeJSON(ctx, http.StatusOK, envelope{
		"subtitle": subtitle,
		"used":     row.UsedAt.Valid,
	})
}

// uploadShiftReceiptImage is the only write path a phone with no login can
// reach on the central server. It re-validates the token from scratch
// (unexpired, unused) rather than trusting an earlier /status call.
func (server *Server) uploadShiftReceiptImage(ctx *gin.Context) {
	token := ctx.Param("token")

	row, err := server.lookupValidShiftUploadToken(ctx, token)
	if err != nil {
		if err == sql.ErrNoRows {
			server.writeError(ctx, http.StatusNotFound, fmt.Errorf("this link is no longer valid"))
			return
		}
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	if row.UsedAt.Valid {
		server.writeError(ctx, http.StatusGone, fmt.Errorf("a photo has already been uploaded for this shift"))
		return
	}

	shift, err := server.store.GetBranchShift(ctx, row.BranchShiftID)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	fileHeader, err := ctx.FormFile("image")
	if err != nil {
		server.writeError(ctx, http.StatusBadRequest, fmt.Errorf("no image provided"))
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

	// If a photo already landed (e.g. via the POS-forwarded path), treat a
	// second upload as a no-op success rather than a hard error.
	if _, err := server.store.GetBranchShiftReceiptImageForShift(ctx, shift.ID); err == nil {
		_ = server.store.MarkBranchShiftUploadTokenUsed(ctx, row.ID)
		server.writeJSON(ctx, http.StatusOK, envelope{"success": true})
		return
	} else if err != sql.ErrNoRows {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	destDir := filepath.Join(server.shiftReceiptImageStorageDir(), fmt.Sprintf("%d", shift.ID))
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	stored, err := optimizeUploadedImage(fileHeader, contentType, ext)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	destPath := filepath.Join(destDir, uuid.NewString()+stored.Ext)
	if err := os.WriteFile(destPath, stored.Data, 0o644); err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	if _, err := server.store.CreateBranchShiftReceiptImage(ctx, db.CreateBranchShiftReceiptImageParams{
		BranchShiftID: shift.ID,
		FilePath:      destPath,
		ContentType:   stored.ContentType,
		SizeBytes:     int64(len(stored.Data)),
	}); err != nil {
		if isUniqueViolation(err) {
			os.Remove(destPath)
			_ = server.store.MarkBranchShiftUploadTokenUsed(ctx, row.ID)
			server.writeJSON(ctx, http.StatusOK, envelope{"success": true})
			return
		}
		os.Remove(destPath)
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	if err := server.store.MarkBranchShiftUploadTokenUsed(ctx, row.ID); err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	server.adminHub.broadcastAll(adminWSMessage{
		Type:     "branch_shift_receipt_uploaded",
		BranchID: shift.BranchID,
		Label:    shift.ClosingPersonName,
	})

	server.writeJSON(ctx, http.StatusOK, envelope{"success": true})
}

type branchShiftUploadStatusRequest struct {
	ShiftRef string `form:"shift_ref" binding:"required"`
}

// branchShiftUploadStatus is kashi-pos's poll while its QR dialog is open in
// "central" mode -- it can't watch its own local table because the phone
// uploaded straight here. Mirrors branchExpenseUploadStatus.
func (server *Server) branchShiftUploadStatus(ctx *gin.Context) {
	registerID := ctx.MustGet(registerIDKey).(int64)

	var req branchShiftUploadStatusRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}

	shift, err := server.store.GetBranchShiftByClientRef(ctx, db.GetBranchShiftByClientRefParams{
		RegisterID: registerID,
		ClientRef:  req.ShiftRef,
	})
	if err != nil {
		if err == sql.ErrNoRows {
			server.writeJSON(ctx, http.StatusOK, envelope{"images_uploaded": false})
			return
		}
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	if _, err := server.store.GetBranchShiftReceiptImageForShift(ctx, shift.ID); err == nil {
		server.writeJSON(ctx, http.StatusOK, envelope{"images_uploaded": true})
		return
	} else if err != sql.ErrNoRows {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	server.writeJSON(ctx, http.StatusOK, envelope{"images_uploaded": false})
}
