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

// transferUploadTokenTTL mirrors the expense equivalent: long enough to
// walk over and scan a QR, short enough to keep an unauthenticated upload
// endpoint's exposure window small. A transfer request has no "offline"
// mode the way a shift close does (proposing one already required the
// till to reach kashi), so there's no longer-lived variant to pick between.
const transferUploadTokenTTL = 30 * time.Minute

const defaultTransferAttachmentStorageDir = "./storage/transfer_attachments"

func (server *Server) transferAttachmentStorageDir() string {
	if dir := server.config.TransferAttachmentStorageDir; dir != "" {
		return dir
	}
	return defaultTransferAttachmentStorageDir
}

//go:embed transfer_upload_assets/upload.html
var transferUploadPageHTML []byte

var transferUploadPage = newPhoneUploadPage(transferUploadPageHTML)

// createBranchTransferUploadToken mints a one-time, expiring token for a
// freshly-created branch transfer request. Called from
// createBranchTransferRequest (branch_transfer.go) only on the
// genuinely-new-insert path.
func (server *Server) createBranchTransferUploadToken(ctx *gin.Context, transferID int64) (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("generate upload token: %w", err)
	}
	token := hex.EncodeToString(raw)

	_, err := server.store.CreateBranchTransferUploadToken(ctx, db.CreateBranchTransferUploadTokenParams{
		TransferID: transferID,
		Token:      token,
		ExpiresAt:  time.Now().UTC().Add(transferUploadTokenTTL),
	})
	if err != nil {
		return "", err
	}
	return token, nil
}

// transferUploadPage serves the self-contained mobile upload page. All
// validity checking happens client-side against the status/images
// endpoints below.
func (server *Server) transferUploadPage(ctx *gin.Context) {
	server.servePhoneUploadPage(ctx, transferUploadPage)
}

func (server *Server) lookupValidTransferUploadToken(ctx *gin.Context, token string) (db.BranchTransferUploadToken, error) {
	row, err := server.store.GetBranchTransferUploadTokenByToken(ctx, token)
	if err != nil {
		return db.BranchTransferUploadToken{}, err
	}
	if time.Now().UTC().After(row.ExpiresAt) {
		return db.BranchTransferUploadToken{}, sql.ErrNoRows
	}
	return row, nil
}

func (server *Server) transferUploadStatus(ctx *gin.Context) {
	token := ctx.Param("token")

	row, err := server.lookupValidTransferUploadToken(ctx, token)
	if err != nil {
		if err == sql.ErrNoRows {
			server.writeError(ctx, http.StatusNotFound, fmt.Errorf("this link is no longer valid"))
			return
		}
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	transfer, err := server.store.GetTransfer(ctx, row.TransferID)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	subtitle := fmt.Sprintf("Transfer #%d", transfer.ID)
	if transfer.Note != "" {
		subtitle += " · " + transfer.Note
	}

	server.writeJSON(ctx, http.StatusOK, envelope{
		"subtitle": subtitle,
		"used":     row.UsedAt.Valid,
	})
}

// uploadTransferAttachment is the only write path a phone with no login
// can reach on the central server. It re-validates the token from scratch
// (unexpired, unused) rather than trusting an earlier /status call.
func (server *Server) uploadTransferAttachment(ctx *gin.Context) {
	token := ctx.Param("token")

	row, err := server.lookupValidTransferUploadToken(ctx, token)
	if err != nil {
		if err == sql.ErrNoRows {
			server.writeError(ctx, http.StatusNotFound, fmt.Errorf("this link is no longer valid"))
			return
		}
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	if row.UsedAt.Valid {
		server.writeError(ctx, http.StatusGone, fmt.Errorf("a photo has already been uploaded for this transfer"))
		return
	}

	transfer, err := server.store.GetTransfer(ctx, row.TransferID)
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

	if transfer.AttachmentPath != "" {
		_ = server.store.MarkBranchTransferUploadTokenUsed(ctx, row.ID)
		server.writeJSON(ctx, http.StatusOK, envelope{"success": true})
		return
	}

	stored, err := optimizeUploadedImage(fileHeader, contentType, ext)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	destDir := filepath.Join(server.transferAttachmentStorageDir(), fmt.Sprintf("%d", transfer.ID))
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	destPath := filepath.Join(destDir, uuid.NewString()+stored.Ext)
	if err := os.WriteFile(destPath, stored.Data, 0o644); err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	warmThumbnails(destPath)

	if err := server.store.SetTransferAttachmentPath(ctx, db.SetTransferAttachmentPathParams{
		ID:             transfer.ID,
		AttachmentPath: destPath,
	}); err != nil {
		os.Remove(destPath)
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	if err := server.store.MarkBranchTransferUploadTokenUsed(ctx, row.ID); err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	server.adminHub.broadcastAll(adminWSMessage{
		Type:     "branch_transfer_photo_uploaded",
		BranchID: transfer.BranchID.Int64,
	})

	server.writeJSON(ctx, http.StatusOK, envelope{"success": true})
}
