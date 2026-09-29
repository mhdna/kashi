package api

import (
	"database/sql"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	db "github.com/mhdna/kashi/db/sqlc"
)

type branchTransferItemRequest struct {
	Barcode    string `json:"barcode"`
	ProductUID string `json:"product_uid"`
	Quantity   int64  `json:"quantity" binding:"required"`
}

type createBranchTransferRequestRequest struct {
	ClientRef   string                      `json:"client_ref" binding:"required"`
	ToBranchRef string                      `json:"to_branch_ref" binding:"required"`
	Note        string                      `json:"note"`
	Items       []branchTransferItemRequest `json:"items" binding:"required,min=1,dive"`
}

// createBranchTransferRequest is the till-facing endpoint a branch posts a
// proposed transfer to: it always moves stock out of that branch's own
// store inventory (branches.inventory_id, see migration 000077) toward
// the inventory of the branch named in ToBranchRef. Nothing about the
// destination's own local ids leaks back -- the till only ever knows the
// other branch by the code it saw in the branches down-sync feed
// (ListBranchesForSync).
//
// A photo, like a close-shift receipt or an expense receipt, is never
// pushed by the till itself -- it's the QR-scan-and-upload flow (see
// transfer_upload.go): this response's upload_token is what kashi-pos
// turns into the QR it shows the cashier.
//
// Idempotent on (branch_id, client_ref), same check-then-create shape as
// createBranchInvoice: a retried post lands the same row (and, on the
// idempotent-return path, the same still-valid upload token) once.
func (server *Server) createBranchTransferRequest(ctx *gin.Context) {
	branchID := ctx.MustGet(branchIDKey).(int64)

	var req createBranchTransferRequestRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}

	if existing, err := server.store.GetBranchTransferByClientRef(ctx, db.GetBranchTransferByClientRefParams{
		BranchID:  sql.NullInt64{Int64: branchID, Valid: true},
		ClientRef: sql.NullString{String: req.ClientRef, Valid: true},
	}); err == nil {
		server.writeJSON(ctx, http.StatusOK, envelope{"transfer": existing})
		return
	} else if err != sql.ErrNoRows {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	toBranch, err := server.store.GetBranchByCodeExcluding(ctx, db.GetBranchByCodeExcludingParams{
		Code: req.ToBranchRef,
		ID:   branchID,
	})
	if err != nil {
		if err == sql.ErrNoRows {
			server.writeError(ctx, http.StatusBadRequest, fmt.Errorf("unknown destination branch %q", req.ToBranchRef))
			return
		}
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	if !toBranch.InventoryID.Valid {
		server.writeError(ctx, http.StatusBadRequest, fmt.Errorf("destination branch has no linked inventory"))
		return
	}

	items := make([]db.BranchTransferItemParams, 0, len(req.Items))
	for _, it := range req.Items {
		items = append(items, db.BranchTransferItemParams{
			Barcode:    it.Barcode,
			ProductUID: parseNullUUID(it.ProductUID),
			Quantity:   it.Quantity,
		})
	}

	result, err := server.store.CreateBranchTransferRequestTx(ctx, db.CreateBranchTransferRequestTxParams{
		BranchID:      branchID,
		ClientRef:     req.ClientRef,
		ToInventoryID: toBranch.InventoryID.Int64,
		Note:          req.Note,
		Items:         items,
	})
	if err != nil {
		if isUniqueViolation(err) {
			if existing, lookupErr := server.store.GetBranchTransferByClientRef(ctx, db.GetBranchTransferByClientRefParams{
				BranchID:  sql.NullInt64{Int64: branchID, Valid: true},
				ClientRef: sql.NullString{String: req.ClientRef, Valid: true},
			}); lookupErr == nil {
				server.writeJSON(ctx, http.StatusOK, envelope{"transfer": existing})
				return
			}
		}
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}

	uploadToken, err := server.createBranchTransferUploadToken(ctx, result.Transfer.ID)
	if err != nil {
		// The transfer itself is already committed -- a QR the cashier can't
		// get isn't worth failing the whole request over. kashi-pos simply
		// won't have a token to show; the transfer still syncs and can be
		// approved with no photo.
		uploadToken = ""
	}

	server.adminHub.broadcastAll(adminWSMessage{
		Type:     "branch_transfer_requested",
		BranchID: branchID,
		Label:    req.ClientRef,
	})

	server.writeJSON(ctx, http.StatusOK, envelope{"transfer": result.Transfer, "items": result.Items, "upload_token": uploadToken})
}

type branchTransferUploadStatusRequest struct {
	ClientRef string `form:"client_ref" binding:"required"`
}

// branchTransferUploadStatus is kashi-pos's poll while its QR dialog is
// open -- it can't watch its own local table because the phone uploaded
// straight to kashi. Mirrors branchShiftUploadStatus/
// branchExpenseUploadStatus.
func (server *Server) branchTransferUploadStatus(ctx *gin.Context) {
	branchID := ctx.MustGet(branchIDKey).(int64)

	var req branchTransferUploadStatusRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}

	transfer, err := server.store.GetBranchTransferByClientRef(ctx, db.GetBranchTransferByClientRefParams{
		BranchID:  sql.NullInt64{Int64: branchID, Valid: true},
		ClientRef: sql.NullString{String: req.ClientRef, Valid: true},
	})
	if err != nil {
		if err == sql.ErrNoRows {
			server.writeJSON(ctx, http.StatusOK, envelope{"image_uploaded": false})
			return
		}
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	server.writeJSON(ctx, http.StatusOK, envelope{"image_uploaded": transfer.AttachmentPath != ""})
}
