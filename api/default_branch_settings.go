package api

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/mhdna/kashi/token"
)

// defaultBranchSettings is the admin's saved "default settings" template --
// deliberately a subset of branchSettingsRequest. Branch identity (name,
// storefront contact details), per-device fields (printer/COM IDs, Akuvox
// IP/username) and secrets are excluded on purpose: those are specific to
// one branch/machine and must never be bulk-pushed by "Apply to all" (see
// applyDefaultBranchSettingsToAll).
type defaultBranchSettings struct {
	TaxRate                      float64 `json:"tax_rate"`
	RoundingMode                 string  `json:"rounding_mode"`
	RoundingCurrency             string  `json:"rounding_currency"`
	ExchangeRate                 int64   `json:"exchange_rate"`
	ExchangeWindowHours          int64   `json:"exchange_window_hours"`
	SplitPrimaryCurrency         string  `json:"split_primary_currency"`
	SplitPrimaryFirst            bool    `json:"split_primary_first"`
	SplitPrimaryStep             int64   `json:"split_primary_step"`
	SearchButtonEnabled          bool    `json:"search_button_enabled"`
	CustomItemDiscountsEnabled   bool    `json:"custom_item_discounts_enabled"`
	CustomItemPricesEnabled      bool    `json:"custom_item_prices_enabled"`
	PerUnitItemPricesEnabled     bool    `json:"per_unit_item_prices_enabled"`
	PriceChangeManualOverrideMode bool   `json:"price_change_manual_override_mode"`
	InvoiceKeyboardMode          bool    `json:"invoice_keyboard_mode"`
	ClientRequired                bool   `json:"client_required"`
	ShiftReceiptUploadMode        int64  `json:"shift_receipt_upload_mode"`

	PageUnlockClients      bool `json:"page_unlock_clients"`
	PageUnlockInventory    bool `json:"page_unlock_inventory"`
	PageUnlockTransfers    bool `json:"page_unlock_transfers"`
	PageUnlockAttendance   bool `json:"page_unlock_attendance"`
	PageUnlockSalespersons bool `json:"page_unlock_salespersons"`

	PrinterSize      string  `json:"printer_size"`
	ReceiptWidth     int64   `json:"receipt_width"`
	ReceiptHeight    int64   `json:"receipt_height"`
	ReceiptEnabled   bool    `json:"receipt_enabled"`
	ReceiptFont      string  `json:"receipt_font"`
	ReceiptBodyFont  string  `json:"receipt_body_font"`
	ReceiptTitleSize float64 `json:"receipt_title_size"`
	ReceiptBodySize  float64 `json:"receipt_body_size"`
	ReceiptCutoff    bool    `json:"receipt_cutoff"`

	AttendanceEnabled                  bool  `json:"attendance_enabled"`
	AttendanceDuplicateIntervalSeconds int64 `json:"attendance_duplicate_interval_seconds"`
	AttendanceCashierHistory           bool  `json:"attendance_cashier_history"`
}

// getDefaultBranchSettings returns the saved template, or an empty one
// (zero values, not a 404) if nothing's been saved yet -- there's always
// exactly one template, it just starts blank.
func (server *Server) getDefaultBranchSettings(ctx *gin.Context) {
	row, err := server.store.GetDefaultBranchSettings(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			server.writeJSON(ctx, http.StatusOK, envelope{"settings": defaultBranchSettings{}})
			return
		}
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	var settings defaultBranchSettings
	if err := json.Unmarshal(row.Settings, &settings); err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	server.writeJSON(ctx, http.StatusOK, envelope{"settings": settings})
}

// putDefaultBranchSettings saves the template. It never touches any branch
// on its own -- applyDefaultBranchSettingsToAll is the only thing that reads
// it back out to push to branches, and only when an admin explicitly asks.
func (server *Server) putDefaultBranchSettings(ctx *gin.Context) {
	var req defaultBranchSettings
	if err := ctx.ShouldBindJSON(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}

	encoded, err := json.Marshal(req)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	row, err := server.store.UpsertDefaultBranchSettings(ctx, encoded)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	var settings defaultBranchSettings
	if err := json.Unmarshal(row.Settings, &settings); err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	server.writeJSON(ctx, http.StatusOK, envelope{"settings": settings})
}

// applyDefaultBranchSettingsToAll fans the saved template out to every
// branch as an update_settings command, same mechanism as an admin editing
// one branch's settings by hand (see createBranchCommand) -- it does not
// write branch_settings directly, does not touch anything outside this
// template's field list, and each branch applies it on its own next sync
// tick or live push exactly as it would any other command. A branch that's
// managing "device_ids" or "secrets" locally is unaffected either way,
// since neither group's fields are ever part of this template.
func (server *Server) applyDefaultBranchSettingsToAll(ctx *gin.Context) {
	row, err := server.store.GetDefaultBranchSettings(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			server.writeError(ctx, http.StatusBadRequest, sql.ErrNoRows)
			return
		}
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	var settings defaultBranchSettings
	if err := json.Unmarshal(row.Settings, &settings); err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	payload, err := json.Marshal(settings)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	branches, err := server.store.ListBranches(ctx)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	tokenPayload := ctx.MustGet(authoizationPayloadKey).(*token.Payload)
	user, err := server.store.GetUserByUsername(ctx, tokenPayload.Username)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	queued := 0
	for _, branch := range branches {
		if _, err := server.enqueueBranchCommand(ctx, branch.ID, "update_settings", payload, user.ID); err != nil {
			server.writeError(ctx, http.StatusInternalServerError, err)
			return
		}
		queued++
	}

	server.writeJSON(ctx, http.StatusOK, envelope{"branches_queued": queued})
}
