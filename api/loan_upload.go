package api

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	db "github.com/mhdna/kashi/db/sqlc"
)

// Loan photos: the same one-time QR upload as branch expenses
// (expense_upload.go) -- same page, same limits, same single-use expiring
// token -- for a loan a till reported. Files go under the expense image
// directory in loans/<loan id>, which prod already keeps on its volume.

func newUploadToken() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("generate upload token: %w", err)
	}
	return hex.EncodeToString(raw), nil
}

// createLoanUploadToken mints the link a till shows as a QR code for a
// loan: on the loan's first sync (createBranchLoan) and whenever the till
// asks for a new one (branchLoanUploadLink).
func (server *Server) createLoanUploadToken(ctx *gin.Context, loanID int64) (string, error) {
	token, err := newUploadToken()
	if err != nil {
		return "", err
	}
	if _, err := server.store.CreateLoanUploadToken(ctx, db.CreateLoanUploadTokenParams{
		LoanID:    loanID,
		Token:     token,
		ExpiresAt: time.Now().UTC().Add(expenseUploadTokenTTL),
	}); err != nil {
		return "", err
	}
	return token, nil
}

func (server *Server) lookupValidLoanUploadToken(ctx *gin.Context, token string) (db.LoanUploadToken, error) {
	row, err := server.store.GetLoanUploadTokenByToken(ctx, token)
	if err != nil {
		return db.LoanUploadToken{}, err
	}
	if time.Now().UTC().After(row.ExpiresAt) {
		return db.LoanUploadToken{}, sql.ErrNoRows
	}
	return row, nil
}

func (server *Server) loanUploadStatus(ctx *gin.Context) {
	row, err := server.lookupValidLoanUploadToken(ctx, ctx.Param("token"))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			server.writeError(ctx, http.StatusNotFound, fmt.Errorf("this link is no longer valid"))
			return
		}
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	loan, err := server.store.GetLoan(ctx, row.LoanID)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	server.writeJSON(ctx, http.StatusOK, envelope{
		"description":  loan.Description,
		"amount_label": formatMoneyLabel(loan.Amount, loan.CurrencyCode),
		"used":         row.UsedAt.Valid,
	})
}

// uploadLoanImages mirrors uploadExpenseImages: the token is re-checked
// from scratch, at most maxExpenseUploadImages files, each optimized and
// stored, then the token is spent.
func (server *Server) uploadLoanImages(ctx *gin.Context) {
	row, err := server.lookupValidLoanUploadToken(ctx, ctx.Param("token"))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			server.writeError(ctx, http.StatusNotFound, fmt.Errorf("this link is no longer valid"))
			return
		}
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	if row.UsedAt.Valid {
		server.writeError(ctx, http.StatusGone, fmt.Errorf("this link has already been used"))
		return
	}

	form, err := ctx.MultipartForm()
	if err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	files := form.File["images"]
	if len(files) == 0 {
		server.writeError(ctx, http.StatusBadRequest, fmt.Errorf("no images provided"))
		return
	}
	if len(files) > maxExpenseUploadImages {
		server.writeError(ctx, http.StatusBadRequest, fmt.Errorf("at most %d images allowed", maxExpenseUploadImages))
		return
	}

	loan, err := server.store.GetLoan(ctx, row.LoanID)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	destDir := filepath.Join(server.config.ExpenseImageStorageDir, "loans", fmt.Sprintf("%d", loan.ID))
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	for _, fileHeader := range files {
		if fileHeader.Size > maxExpenseUploadImageBytes {
			server.writeError(ctx, http.StatusBadRequest, fmt.Errorf("each image must be under %dMB", maxExpenseUploadImageBytes>>20))
			return
		}
		contentType := fileHeader.Header.Get("Content-Type")
		ext, ok := allowedExpenseImageContentTypes[contentType]
		if !ok {
			server.writeError(ctx, http.StatusBadRequest, fmt.Errorf("unsupported image type %q", contentType))
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
		if _, err := server.store.CreateLoanImage(ctx, db.CreateLoanImageParams{
			LoanID:      loan.ID,
			FilePath:    destPath,
			ContentType: stored.ContentType,
			SizeBytes:   int64(len(stored.Data)),
		}); err != nil {
			os.Remove(destPath)
			server.writeError(ctx, http.StatusInternalServerError, err)
			return
		}
	}

	if err := server.store.MarkLoanUploadTokenUsed(ctx, row.ID); err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	if loan.BranchID.Valid {
		server.branchHub.notify(loan.BranchID.Int64, branchWSMessage{
			Type:      "loan_images_uploaded",
			ClientRef: loan.ClientRef.String,
		})
	}
	server.writeJSON(ctx, http.StatusOK, envelope{"success": true})
}

type branchUploadLinkRequest struct {
	ClientRef string `json:"client_ref" binding:"required"`
}

// branchExpenseUploadLink gives one of this branch's expenses a fresh
// upload token, for a photo that wasn't sent when the expense was made
// (the first link expired, or the phone never got to it). Earlier tokens
// stay as they were; each is single-use and expires on its own.
func (server *Server) branchExpenseUploadLink(ctx *gin.Context) {
	branchID := ctx.MustGet(branchIDKey).(int64)
	var req branchUploadLinkRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	expense, err := server.store.GetBranchExpenseByClientRef(ctx, db.GetBranchExpenseByClientRefParams{
		BranchID: branchID, ClientRef: req.ClientRef,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			server.writeError(ctx, http.StatusNotFound, errors.New("kashi hasn't received this expense yet"))
			return
		}
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	count, err := server.store.CountBranchExpenseImagesForExpense(ctx, expense.ID)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	if count >= maxExpenseUploadImages {
		server.writeError(ctx, http.StatusConflict, fmt.Errorf("this expense already has %d photos", count))
		return
	}
	token, err := server.createBranchExpenseUploadToken(ctx, expense.ID)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	server.writeJSON(ctx, http.StatusOK, envelope{"upload_token": token})
}

// branchLoanUploadLink is branchExpenseUploadLink for a loan.
func (server *Server) branchLoanUploadLink(ctx *gin.Context) {
	branchID := ctx.MustGet(branchIDKey).(int64)
	var req branchUploadLinkRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	loan, err := server.store.GetBranchLoanByClientRef(ctx, db.GetBranchLoanByClientRefParams{
		BranchID:  sql.NullInt64{Int64: branchID, Valid: true},
		ClientRef: sql.NullString{String: req.ClientRef, Valid: true},
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			server.writeError(ctx, http.StatusNotFound, errors.New("kashi hasn't received this loan yet"))
			return
		}
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	count, err := server.store.CountLoanImagesForLoan(ctx, loan.ID)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	if count >= maxExpenseUploadImages {
		server.writeError(ctx, http.StatusConflict, fmt.Errorf("this loan already has %d photos", count))
		return
	}
	token, err := server.createLoanUploadToken(ctx, loan.ID)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	server.writeJSON(ctx, http.StatusOK, envelope{"upload_token": token})
}

// branchLoanPhotos lists the photos of one of this branch's loans, by the
// till's client_ref, as branchExpensePhotos does for expenses.
func (server *Server) branchLoanPhotos(ctx *gin.Context) {
	branchID := ctx.MustGet(branchIDKey).(int64)
	var req branchExpensePhotosRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	loan, err := server.store.GetBranchLoanByClientRef(ctx, db.GetBranchLoanByClientRefParams{
		BranchID:  sql.NullInt64{Int64: branchID, Valid: true},
		ClientRef: sql.NullString{String: req.ClientRef, Valid: true},
	})
	if errors.Is(err, sql.ErrNoRows) {
		server.writeJSON(ctx, http.StatusOK, envelope{"images": []db.LoanImage{}})
		return
	}
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	images, err := server.store.ListLoanImagesForLoan(ctx, loan.ID)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	server.writeJSON(ctx, http.StatusOK, envelope{"images": images})
}

// branchLoanPhotoFile serves one loan photo, if its loan is this branch's.
func (server *Server) branchLoanPhotoFile(ctx *gin.Context) {
	branchID := ctx.MustGet(branchIDKey).(int64)
	var req branchPhotoIDRequest
	if err := ctx.ShouldBindUri(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	image, err := server.store.GetLoanImage(ctx, req.ID)
	if err != nil {
		server.writeError(ctx, statusForLookup(err), err)
		return
	}
	loan, err := server.store.GetLoan(ctx, image.LoanID)
	if err != nil || !loan.BranchID.Valid || loan.BranchID.Int64 != branchID {
		server.writeError(ctx, http.StatusNotFound, errors.New("no such photo for this branch"))
		return
	}
	server.serveStoredImage(ctx, image.FilePath, image.ContentType, 0)
}

// mintLoanUploadToken is createBranchLoan's call: a failure only costs the
// QR dialog, never the loan.
func (server *Server) mintLoanUploadToken(ctx *gin.Context, loanID int64) string {
	token, err := server.createLoanUploadToken(ctx, loanID)
	if err != nil {
		log.Printf("create upload token for loan %d: %v", loanID, err)
		return ""
	}
	return token
}
