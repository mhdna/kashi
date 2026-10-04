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
		RegisterID:    ctx.MustGet(registerIDKey).(int64),
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

	// The destination branch's own "notification directly after" the
	// request lands -- a nudge-then-pull, same convention as
	// target_updated/cashbox_account_updated: the till reacts by
	// re-fetching the incoming_transfer_requests entity, not by trusting
	// anything carried in this message itself.
	server.branchHub.notify(toBranch.ID, branchWSMessage{Type: "incoming_transfer_requests_updated"})

	server.writeJSON(ctx, http.StatusOK, envelope{"transfer": result.Transfer, "items": result.Items, "upload_token": uploadToken})
}

type confirmBranchTransferItemRequest struct {
	Barcode          string `json:"barcode"`
	ProductUID       string `json:"product_uid"`
	ReceivedQuantity int64  `json:"received_quantity" binding:"required"`
}

type confirmBranchTransferRequest struct {
	Items []confirmBranchTransferItemRequest `json:"items" binding:"required,min=1,dive"`
	Note  string                             `json:"note"`
}

type confirmBranchTransferURI struct {
	ID int64 `uri:"id" binding:"required,min=1"`
}

// confirmBranchTransfer is the destination branch reporting what it
// actually counted on physical receipt -- the second leg
// createBranchTransferRequest's sender started. It never moves stock (see
// ConfirmBranchTransferTx); kashi's admin approve reads these counts back
// once it runs. Re-sendable: a later confirm just overwrites the earlier
// one, same as editing rather than creating.
func (server *Server) confirmBranchTransfer(ctx *gin.Context) {
	branchID := ctx.MustGet(branchIDKey).(int64)

	var uri confirmBranchTransferURI
	if err := ctx.ShouldBindUri(&uri); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	var req confirmBranchTransferRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}

	items := make([]db.ConfirmBranchTransferItemParams, 0, len(req.Items))
	for _, it := range req.Items {
		items = append(items, db.ConfirmBranchTransferItemParams{
			Barcode:          it.Barcode,
			ProductUID:       parseNullUUID(it.ProductUID),
			ReceivedQuantity: it.ReceivedQuantity,
		})
	}

	transfer, err := server.store.ConfirmBranchTransferTx(ctx, db.ConfirmBranchTransferTxParams{
		BranchID:   branchID,
		TransferID: uri.ID,
		Note:       req.Note,
		Items:      items,
	})
	if err != nil {
		if err == sql.ErrNoRows {
			server.writeError(ctx, http.StatusNotFound, err)
			return
		}
		server.writeError(ctx, http.StatusForbidden, err)
		return
	}

	server.adminHub.broadcastAll(adminWSMessage{
		Type:     "branch_transfer_confirmed",
		BranchID: branchID,
		Label:    fmt.Sprintf("Transfer #%d", transfer.ID),
	})
	// Let the sender's own outgoing-transfer view reflect "confirmed by
	// destination" without waiting for kashi's own approve/reject.
	if transfer.BranchID.Valid {
		server.branchHub.notify(transfer.BranchID.Int64, branchWSMessage{Type: "transfer_requests_updated"})
	}

	server.writeJSON(ctx, http.StatusOK, envelope{"transfer": transfer})
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
