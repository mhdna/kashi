package api

import (
	"database/sql"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	db "github.com/mhdna/kashi/db/sqlc"
)

// The Recent activity details dialog: everything about one feed row, beyond
// the one line the feed shows. A feed row is named by its activity (the
// label HomeActivityFeed stamps on it) and the id in that activity's own
// table, so this looks the row up there and adds what belongs with it: an
// invoice's lines and payments, a shift's counts and its close-shift
// receipt photo, an expense's photos, a settlement's new payments and the
// invoice it changed.

type homeActivityDetailRequest struct {
	Activity string `form:"activity" binding:"required"`
	ID       int64  `form:"id" binding:"required,min=1"`
}

func (server *Server) getHomeActivityDetail(ctx *gin.Context) {
	var req homeActivityDetailRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}

	out, err := server.homeActivityDetail(ctx, req.Activity, req.ID)
	if err != nil {
		switch {
		case err == sql.ErrNoRows:
			server.writeError(ctx, http.StatusNotFound, fmt.Errorf("%s #%d not found", req.Activity, req.ID))
		case err == errUnknownHomeActivity:
			server.writeError(ctx, http.StatusBadRequest, fmt.Errorf("unknown activity %q", req.Activity))
		default:
			server.writeError(ctx, http.StatusInternalServerError, err)
		}
		return
	}
	server.writeJSON(ctx, http.StatusOK, out)
}

var errUnknownHomeActivity = fmt.Errorf("unknown activity")

func (server *Server) homeActivityDetail(ctx *gin.Context, activity string, id int64) (envelope, error) {
	switch activity {
	case "Invoice":
		invoice, err := server.store.GetBranchInvoice(ctx, id)
		if err != nil {
			return nil, err
		}
		items, err := server.store.ListBranchInvoiceItemsNamed(ctx, id)
		if err != nil {
			return nil, err
		}
		payments, err := server.store.ListBranchInvoicePayments(ctx, id)
		if err != nil {
			return nil, err
		}
		return envelope{
			"invoice": invoice, "items": items, "payments": payments,
			"register": server.registerName(ctx, invoice.RegisterID),
		}, nil

	case "Shift closed":
		shift, err := server.store.GetBranchShift(ctx, id)
		if err != nil {
			return nil, err
		}
		// The close-shift receipt photo, when the till sent one.
		var receiptImageID *int64
		if image, err := server.store.GetBranchShiftReceiptImageForShift(ctx, id); err == nil {
			receiptImageID = &image.ID
		} else if err != sql.ErrNoRows {
			return nil, err
		}
		return envelope{
			"shift": shift, "receipt_image_id": receiptImageID,
			"register": server.registerName(ctx, shift.RegisterID),
		}, nil

	case "Expense":
		expense, err := server.store.GetBranchExpense(ctx, id)
		if err != nil {
			return nil, err
		}
		images, err := server.store.ListBranchExpenseImagesForExpense(ctx, id)
		if err != nil {
			return nil, err
		}
		imageIDs := make([]int64, len(images))
		for i, image := range images {
			imageIDs[i] = image.ID
		}
		return envelope{"expense": expense, "image_ids": imageIDs}, nil

	case "Loan":
		loan, err := server.store.GetLoan(ctx, id)
		if err != nil {
			return nil, err
		}
		return envelope{"loan": loan}, nil

	case "Settlement changed":
		settlement, err := server.store.GetBranchInvoiceSettlement(ctx, id)
		if err != nil {
			return nil, err
		}
		payments, err := server.store.ListBranchInvoiceSettlementPayments(ctx, id)
		if err != nil {
			return nil, err
		}
		// The sale it re-settled; null when that invoice never synced.
		var invoice *db.BranchInvoice
		if found, err := server.store.GetBranchInvoiceByClientRef(ctx, db.GetBranchInvoiceByClientRefParams{
			BranchID: settlement.BranchID, ClientRef: settlement.SaleClientRef,
		}); err == nil {
			invoice = &found
		} else if err != sql.ErrNoRows {
			return nil, err
		}
		return envelope{"settlement": settlement, "payments": payments, "invoice": invoice}, nil

	case "Attendance":
		event, err := server.store.GetBranchAttendanceEvent(ctx, id)
		if err != nil {
			return nil, err
		}
		return envelope{"event": event}, nil

	case "Attendance changed":
		change, err := server.store.GetBranchAttendanceChange(ctx, id)
		if err != nil {
			return nil, err
		}
		return envelope{
			"change":   change,
			"register": server.registerName(ctx, change.RegisterID),
		}, nil
	}
	return nil, errUnknownHomeActivity
}

// registerName is the till's name for display, or "" when it can't be
// found -- a missing name shouldn't fail the whole dialog.
func (server *Server) registerName(ctx *gin.Context, registerID int64) string {
	if registerID == 0 {
		return ""
	}
	register, err := server.store.GetBranchRegister(ctx, registerID)
	if err != nil {
		return ""
	}
	if register.Name != "" {
		return register.Name
	}
	return register.Code
}
