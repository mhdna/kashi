package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	db "github.com/mhdna/kashi/db/sqlc"
	"github.com/sqlc-dev/pqtype"
)

// branchSettingsRequest is the per-branch config kashi holds centrally.
// kashi is the authority: an admin edit updates branch_settings and goes to
// every register as an update_settings command. Each register still
// reports the whole set it runs with on every sync tick
// (branchSettingsPayload in kashi-pos); that snapshot is kept on the
// register (for drift warnings and its per-machine fields) and only seeds
// branch_settings when the branch has none yet.
//
// Still till-only, never modelled here: the register's api_key and its
// register name. The theme travels inside display_prefs.
//
// Fields added after the first version are pointers so a snapshot from an
// older POS build that omits them leaves the stored value alone rather than
// zeroing it. A nil pointer on first insert falls back to the column
// default.
type branchSettingsRequest struct {
	BranchName          string          `json:"branch_name"`
	TaxRate             float64         `json:"tax_rate"`
	RoundingMode        string          `json:"rounding_mode"`
	RoundingCurrency    string          `json:"rounding_currency"`
	ExchangeRate        int64           `json:"exchange_rate"`
	ExchangeWindowHours int64           `json:"exchange_window_hours"`
	MarketName          string          `json:"market_name"`
	MarketPhone         string          `json:"market_phone"`
	MarketDescription   string          `json:"market_description"`
	ReturnPolicy        string          `json:"return_policy"`
	Website             string          `json:"website"`
	Instagram           string          `json:"instagram"`
	SocialPlatforms     json.RawMessage `json:"social_platforms"`
	SocialHandles       json.RawMessage `json:"social_handles"`
	// SearchButtonEnabled is the one original field kashi both reads and
	// writes. Defaults to true when a branch omits it (older POS build).
	SearchButtonEnabled *bool `json:"search_button_enabled"`

	// --- behaviour feature flags (POS: feature_flags rows) ---
	CustomItemDiscountsEnabled    *bool `json:"custom_item_discounts_enabled"`
	CustomItemPricesEnabled       *bool `json:"custom_item_prices_enabled"`
	PerUnitItemPricesEnabled      *bool `json:"per_unit_item_prices_enabled"`
	PriceChangeManualOverrideMode *bool `json:"price_change_manual_override_mode"`
	InvoiceKeyboardMode           *bool `json:"invoice_keyboard_mode"`
	ClientRequired                *bool `json:"client_required"`

	// --- non-admin page-access locks (POS: page_unlock_* flags) ---
	PageUnlockClients      *bool `json:"page_unlock_clients"`
	PageUnlockInventory    *bool `json:"page_unlock_inventory"`
	PageUnlockTransfers    *bool `json:"page_unlock_transfers"`
	PageUnlockAttendance   *bool `json:"page_unlock_attendance"`
	PageUnlockSalespersons *bool `json:"page_unlock_salespersons"`

	// --- printing (POS: settings columns) ---
	PrinterSize      *string  `json:"printer_size"`
	ReceiptWidth     *int64   `json:"receipt_width"`
	ReceiptHeight    *int64   `json:"receipt_height"`
	ReceiptEnabled   *bool    `json:"receipt_enabled"`
	ReceiptFont      *string  `json:"receipt_font"`
	ReceiptBodyFont  *string  `json:"receipt_body_font"`
	ReceiptTitleSize *float64 `json:"receipt_title_size"`
	ReceiptBodySize  *float64 `json:"receipt_body_size"`
	ReceiptCutoff    *bool    `json:"receipt_cutoff"`
	// The printed invoice's look: a theme id and one of its layout ids (the
	// till's catalog, receipt_theme.go). Nil or unrecognisable keeps the
	// till's default, Classic layout A.
	ReceiptTheme  *string `json:"receipt_theme"`
	ReceiptLayout *string `json:"receipt_layout"`
	// The cashier/manager sign-off lines on the end-of-shift receipt. Nil
	// (a till that predates it) keeps them on, the till's own default.
	ShiftReportSignatures *bool `json:"shift_report_signatures"`
	// Store and branch name in capitals on the end-of-shift receipt. Nil
	// keeps them as typed, the till's own default.
	ShiftReportUppercaseHeader *bool `json:"shift_report_uppercase_header"`
	// At close shift, swap a negative Cash LL balance into Cash $$. Nil
	// leaves it off, the till's own default.
	AutoCoverNegativeLBP *bool `json:"auto_cover_negative_lbp"`

	// --- per-machine device identifiers (reported for display; the admin
	// UI defaults these to "managed at the branch") ---
	PrinterID      *string `json:"printer_id"`
	ReceiptPrinter *string `json:"receipt_printer"`
	ScreenPort     *string `json:"screen_port"`

	// --- attendance device config (POS: attendance_config row + one flag) ---
	AkuvoxIP                           *string `json:"akuvox_ip"`
	AkuvoxUsername                     *string `json:"akuvox_username"`
	AttendanceEnabled                  *bool   `json:"attendance_enabled"`
	AttendanceDuplicateIntervalSeconds *int64  `json:"attendance_duplicate_interval_seconds"`
	AttendanceCashierHistory           *bool   `json:"attendance_cashier_history"`

	// --- totals split (POS: settings columns) ---
	// Which currency forms the first half of the till's two-currency total,
	// and whether it comes first. The second half is RoundingCurrency, which
	// predates these. Empty/nil leaves the branch on its own default.
	SplitPrimaryCurrency *string `json:"split_primary_currency"`
	SplitPrimaryFirst    *bool   `json:"split_primary_first"`
	// The split's own denomination, independent of the primary currency's
	// cash_rounding_unit (see 000097). Nil/0 leaves the branch on its own
	// default, which is that currency's cash_rounding_unit.
	SplitPrimaryStep *int64 `json:"split_primary_step"`

	// What the till does when a sale needs more stock than it has: allow,
	// warn or block. Nil leaves the stored value (default allow).
	OversellMode *string `json:"oversell_mode" binding:"omitempty,oneof=allow warn block"`

	// Don't close a shift (or print its receipt) until everything the till
	// has queued has reached kashi. Nil leaves the stored value (default off).
	CloseShiftRequiresSync *bool `json:"close_shift_requires_sync"`

	// Let the scan field fall back to the product code when nothing has the
	// entered barcode. Nil (a till that predates it) keeps it on, the
	// till's own default.
	ScanByCodeEnabled *bool `json:"scan_by_code_enabled"`

	// Product search matches codes only and hides names. Nil (a till that
	// predates it) leaves it off, the till's own default.
	ProductSearchCodeOnly *bool `json:"product_search_code_only"`
	// Manual price and discount edits only on the codes in PriceEditCodes
	// (comma-separated). Nil leaves them off/empty, the till's own default.
	PriceEditRestricted *bool   `json:"price_edit_restricted"`
	PriceEditCodes      *string `json:"price_edit_codes"`
	// Salespersons can tap a line's list discount or deal to turn it off for
	// that sale, never change it; DiscountToggleDefaultOff makes new lines
	// start off. Nil leaves them off, the till's own default.
	DiscountToggle           *bool `json:"discount_toggle"`
	DiscountToggleDefaultOff *bool `json:"discount_toggle_default_off"`
	// Barcode of the product the till's "Unknown item" button adds. Nil or
	// empty: the till's built-in placeholder.
	UnknownProductBarcode *string `json:"unknown_product_barcode"`
	// Product search columns the till leaves out (comma-separated names from
	// code, name, brand, color, size, price). Nil or empty: all shown.
	ProductSearchHiddenFields *string `json:"product_search_hidden_fields"`

	// Let the till play its sounds (scan/checkout chimes, error and warning
	// tones, keypad clicks). Nil (a till that predates it) keeps them off,
	// the till's own default.
	SoundsEnabled *bool `json:"sounds_enabled"`

	// Let a manager decide attendance requests (and add missed punches) on
	// the till with the manager password. Nil (a till that predates it)
	// keeps it on, the till's own default.
	AttendanceManagerApprovalEnabled *bool `json:"attendance_manager_approval_enabled"`

	// --- the rest of the till's settings (000115). Nil on first insert
	// falls back to the till's own default for a missing value. ---
	SalespersonPerInvoiceRequired *bool `json:"salesperson_per_invoice_required"`
	NonAdminMonthNavEnabled       *bool `json:"non_admin_month_nav_enabled"`
	OpeningFloatEnabled           *bool `json:"opening_float_enabled"`
	CountOpeningCashOnClose       *bool `json:"count_opening_cash_on_close"`
	OverchangeConfirmation        *bool `json:"overchange_confirmation"`
	PageUnlockProducts            *bool `json:"page_unlock_products"`
	HidePaymentMethodCurrency     *bool `json:"hide_payment_method_currency"`
	AutoAppendLbpZeros            *bool `json:"auto_append_lbp_zeros"`
	AutoCloseStaleShifts          *bool `json:"auto_close_stale_shifts"`
	VisitorCounterEnabled         *bool `json:"visitor_counter_enabled"`
	// 0=off, 1=central, 2=hybrid (default), 3=local
	ShiftReceiptUploadMode *int64  `json:"shift_receipt_upload_mode" binding:"omitempty,min=0,max=3"`
	BranchPhone            *string `json:"branch_phone"`
	MaxOverpaymentCents    *int64  `json:"max_overpayment_cents"`
	DefaultOpeningFloatUsd *int64  `json:"default_opening_float_usd"`
	DefaultOpeningFloatLbp *int64  `json:"default_opening_float_lbp"`
	// The till screen's look (quick access rows, theme, ...): an object the
	// till validates key by key (kashi-pos display_prefs.go).
	DisplayPrefs json.RawMessage `json:"display_prefs"`
}

// perRegisterSettingKeys are settings that describe one machine (its
// printers, its customer screen), never the branch: a branch-wide
// update_settings never carries them.
var perRegisterSettingKeys = []string{"printer_id", "receipt_printer", "screen_port"}

var errNoBranchSettings = errors.New("this branch hasn't reported its settings yet; connect one of its registers first")

// stripPerRegisterSettings removes perRegisterSettingKeys from an
// update_settings payload bound for every register of a branch.
func stripPerRegisterSettings(payload json.RawMessage) (json.RawMessage, error) {
	var patch map[string]json.RawMessage
	if err := json.Unmarshal(payload, &patch); err != nil {
		return nil, err
	}
	for _, key := range perRegisterSettingKeys {
		delete(patch, key)
	}
	return json.Marshal(patch)
}

// applyBranchSettingsPatch merges a branch-wide update_settings payload into
// branch_settings and returns the payload to send to the registers (minus
// per-machine keys). Keys branch_settings doesn't hold (write-only secrets
// such as admin_password) pass through to the registers untouched.
func (server *Server) applyBranchSettingsPatch(ctx *gin.Context, branchID int64, payload json.RawMessage) (json.RawMessage, error) {
	patch, err := stripPerRegisterSettings(payload)
	if err != nil {
		return nil, err
	}
	current, err := server.store.GetBranchSettings(ctx, branchID)
	if err != nil {
		if err == sql.ErrNoRows {
			return patch, errNoBranchSettings
		}
		return nil, err
	}

	merged, err := mergeJSONObjects(current, patch)
	if err != nil {
		return nil, err
	}
	var req branchSettingsRequest
	if err := json.Unmarshal(merged, &req); err != nil {
		return nil, err
	}
	if req.OversellMode != nil && !validOversellMode(*req.OversellMode) {
		return nil, errors.New("oversell_mode must be allow, warn or block")
	}
	if req.ShiftReceiptUploadMode != nil && (*req.ShiftReceiptUploadMode < 0 || *req.ShiftReceiptUploadMode > 3) {
		return nil, errors.New("shift_receipt_upload_mode must be 0 to 3")
	}
	if _, err := server.upsertBranchSettings(ctx, branchID, req); err != nil {
		return nil, err
	}
	return patch, nil
}

// mergeJSONObjects overlays patch's top-level keys onto base's JSON form.
func mergeJSONObjects(base any, patch json.RawMessage) (json.RawMessage, error) {
	raw, err := json.Marshal(base)
	if err != nil {
		return nil, err
	}
	var merged map[string]json.RawMessage
	if err := json.Unmarshal(raw, &merged); err != nil {
		return nil, err
	}
	if merged == nil { // base was null or absent
		merged = map[string]json.RawMessage{}
	}
	var overlay map[string]json.RawMessage
	if err := json.Unmarshal(patch, &overlay); err != nil {
		return nil, err
	}
	for key, value := range overlay {
		if key == "display_prefs" {
			// A patch names only the prefs it changes; the rest stay.
			prefs, err := mergeJSONObjects(merged[key], value)
			if err != nil {
				return nil, err
			}
			value = prefs
		}
		merged[key] = value
	}
	return json.Marshal(merged)
}

// receiptLookID is a theme or layout id as stored: trimmed, short and made
// of letters, digits, dashes and underscores, else the fallback. Whether the
// id names a real theme is the till's call; it falls back too.
func receiptLookID(p *string, fallback string) string {
	if p == nil {
		return fallback
	}
	id := strings.TrimSpace(*p)
	if id == "" || len(id) > 32 {
		return fallback
	}
	for _, r := range id {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_') {
			return fallback
		}
	}
	return id
}

func validOversellMode(mode string) bool {
	return mode == "allow" || mode == "warn" || mode == "block"
}

// branchWideSettingsPayload is the full update_settings payload that brings
// a register in line with its branch -- sent to a newly added register.
func branchWideSettingsPayload(settings db.BranchSetting) (json.RawMessage, error) {
	raw, err := json.Marshal(settings)
	if err != nil {
		return nil, err
	}
	var payload map[string]json.RawMessage
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, err
	}
	for _, key := range append([]string{"branch_id", "updated_at", "managed_locally"}, perRegisterSettingKeys...) {
		delete(payload, key)
	}
	return json.Marshal(payload)
}

func orZero[T any](p *T) T {
	if p == nil {
		var zero T
		return zero
	}
	return *p
}

func orDefault[T any](p *T, fallback T) T {
	if p == nil {
		return fallback
	}
	return *p
}

func orTrue(p *bool) bool {
	if p == nil {
		return true
	}
	return *p
}

// putBranchSettings is how a register reports the settings it runs with --
// called on every sync tick by kashi-pos (see runSyncTick/pushSettings in
// the kashi-pos repo), not just on change. The snapshot is stored on the
// register; it only becomes the branch's settings when the branch has none
// yet (its first register reporting), since kashi is the authority after
// that and pushes changes down rather than taking them up.
//
// managed_locally is deliberately not touched here -- it's admin-owned
// state, set only via putBranchSettingsManagedLocally.
func (server *Server) putBranchSettings(ctx *gin.Context) {
	branchID := ctx.MustGet(branchIDKey).(int64)
	registerID := ctx.MustGet(registerIDKey).(int64)

	raw, err := ctx.GetRawData()
	if err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	var req branchSettingsRequest
	if err := json.Unmarshal(raw, &req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}

	if err := server.store.SetBranchRegisterReportedSettings(ctx, db.SetBranchRegisterReportedSettingsParams{
		ID:               registerID,
		ReportedSettings: pqtype.NullRawMessage{RawMessage: raw, Valid: true},
	}); err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	if err := server.adoptReportedAttendanceIDs(ctx, branchID, raw); err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	settings, err := server.store.GetBranchSettings(ctx, branchID)
	if err == sql.ErrNoRows {
		settings, err = server.upsertBranchSettings(ctx, branchID, req)
	}
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	identity, err := server.tillIdentity(ctx)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	// branch_code tells the till which rows of the branch_stock feed are its
	// own branch's: it logs in with its register code, which differs from
	// the branch code once a branch has more than one till.
	server.writeJSON(ctx, http.StatusOK, envelope{
		"settings":      settings,
		"branch_code":   identity.BranchCode,
		"register_code": identity.RegisterCode,
	})
}

// adoptReportedAttendanceIDs takes the device numbers a till has typed in for
// kashi's salespersons ("salesperson_attendance_ids": kashi salesperson id ->
// device number, in the settings report) for every salesperson kashi hasn't
// set one for itself, so kashi shows them without anyone retyping them. A
// salesperson kashi has set is left alone -- the till takes kashi's value.
func (server *Server) adoptReportedAttendanceIDs(ctx *gin.Context, branchID int64, raw []byte) error {
	var report struct {
		SalespersonAttendanceIDs map[string]string `json:"salesperson_attendance_ids"`
	}
	if err := json.Unmarshal(raw, &report); err != nil {
		return nil // the rest of the report already parsed; this part is optional
	}
	for key, attendanceUserID := range report.SalespersonAttendanceIDs {
		id, err := strconv.ParseInt(key, 10, 64)
		attendanceUserID = strings.TrimSpace(attendanceUserID)
		if err != nil || id <= 0 || attendanceUserID == "" {
			continue
		}
		if err := server.store.AdoptSalespersonAttendanceID(ctx, db.AdoptSalespersonAttendanceIDParams{
			ID:               id,
			BranchID:         sql.NullInt64{Int64: branchID, Valid: true},
			AttendanceUserID: attendanceUserID,
		}); err != nil {
			return err
		}
	}
	return nil
}

func (server *Server) upsertBranchSettings(ctx *gin.Context, branchID int64, req branchSettingsRequest) (db.BranchSetting, error) {
	socialPlatforms := req.SocialPlatforms
	if len(socialPlatforms) == 0 {
		socialPlatforms = json.RawMessage("[]")
	}
	socialHandles := req.SocialHandles
	if len(socialHandles) == 0 {
		socialHandles = json.RawMessage("{}")
	}
	displayPrefs := req.DisplayPrefs
	if len(displayPrefs) == 0 || string(displayPrefs) == "null" {
		displayPrefs = json.RawMessage("{}")
	}
	oversellMode := "allow"
	if req.OversellMode != nil && validOversellMode(*req.OversellMode) {
		oversellMode = *req.OversellMode
	}

	return server.store.UpsertBranchSettings(ctx, db.UpsertBranchSettingsParams{
		BranchID:            branchID,
		BranchName:          req.BranchName,
		TaxRate:             req.TaxRate,
		RoundingMode:        req.RoundingMode,
		RoundingCurrency:    req.RoundingCurrency,
		ExchangeRate:        req.ExchangeRate,
		ExchangeWindowHours: req.ExchangeWindowHours,
		MarketName:          req.MarketName,
		MarketPhone:         req.MarketPhone,
		MarketDescription:   req.MarketDescription,
		ReturnPolicy:        req.ReturnPolicy,
		Website:             req.Website,
		Instagram:           req.Instagram,
		SocialPlatforms:     socialPlatforms,
		SocialHandles:       socialHandles,
		SearchButtonEnabled: orTrue(req.SearchButtonEnabled),

		CustomItemDiscountsEnabled:    orZero(req.CustomItemDiscountsEnabled),
		CustomItemPricesEnabled:       orZero(req.CustomItemPricesEnabled),
		PerUnitItemPricesEnabled:      orZero(req.PerUnitItemPricesEnabled),
		PriceChangeManualOverrideMode: orZero(req.PriceChangeManualOverrideMode),
		InvoiceKeyboardMode:           orZero(req.InvoiceKeyboardMode),
		ClientRequired:                orZero(req.ClientRequired),

		PageUnlockClients:      orZero(req.PageUnlockClients),
		PageUnlockInventory:    orZero(req.PageUnlockInventory),
		PageUnlockTransfers:    orZero(req.PageUnlockTransfers),
		PageUnlockAttendance:   orZero(req.PageUnlockAttendance),
		PageUnlockSalespersons: orZero(req.PageUnlockSalespersons),

		PrinterSize:      orZero(req.PrinterSize),
		ReceiptWidth:     orZero(req.ReceiptWidth),
		ReceiptHeight:    orZero(req.ReceiptHeight),
		ReceiptEnabled:   orZero(req.ReceiptEnabled),
		ReceiptFont:      orZero(req.ReceiptFont),
		ReceiptBodyFont:  orZero(req.ReceiptBodyFont),
		ReceiptTitleSize: orZero(req.ReceiptTitleSize),
		ReceiptBodySize:  orZero(req.ReceiptBodySize),
		ReceiptCutoff:    orZero(req.ReceiptCutoff),
		ReceiptTheme:     receiptLookID(req.ReceiptTheme, "classic"),
		ReceiptLayout:    receiptLookID(req.ReceiptLayout, "A"),

		ShiftReportSignatures:      orTrue(req.ShiftReportSignatures),
		ShiftReportUppercaseHeader: orZero(req.ShiftReportUppercaseHeader),
		AutoCoverNegativeLbp:       orZero(req.AutoCoverNegativeLBP),

		PrinterID:      orZero(req.PrinterID),
		ReceiptPrinter: orZero(req.ReceiptPrinter),
		ScreenPort:     orZero(req.ScreenPort),

		AkuvoxIp:                           orZero(req.AkuvoxIP),
		AkuvoxUsername:                     orZero(req.AkuvoxUsername),
		AttendanceEnabled:                  orZero(req.AttendanceEnabled),
		AttendanceDuplicateIntervalSeconds: orZero(req.AttendanceDuplicateIntervalSeconds),
		AttendanceCashierHistory:           orZero(req.AttendanceCashierHistory),
		SplitPrimaryCurrency:               orZero(req.SplitPrimaryCurrency),
		SplitPrimaryFirst:                  orTrue(req.SplitPrimaryFirst),
		SplitPrimaryStep:                   orZero(req.SplitPrimaryStep),
		OversellMode:                       oversellMode,
		CloseShiftRequiresSync:             orZero(req.CloseShiftRequiresSync),
		ScanByCodeEnabled:                  orTrue(req.ScanByCodeEnabled),
		ProductSearchCodeOnly:              orZero(req.ProductSearchCodeOnly),
		PriceEditRestricted:                orZero(req.PriceEditRestricted),
		PriceEditCodes:                     orZero(req.PriceEditCodes),
		DiscountToggle:                     orZero(req.DiscountToggle),
		DiscountToggleDefaultOff:           orZero(req.DiscountToggleDefaultOff),
		UnknownProductBarcode:              strings.TrimSpace(orZero(req.UnknownProductBarcode)),
		ProductSearchHiddenFields:          orZero(req.ProductSearchHiddenFields),
		SoundsEnabled:                      orZero(req.SoundsEnabled),
		AttendanceManagerApprovalEnabled:   orTrue(req.AttendanceManagerApprovalEnabled),

		SalespersonPerInvoiceRequired: orZero(req.SalespersonPerInvoiceRequired),
		NonAdminMonthNavEnabled:       orZero(req.NonAdminMonthNavEnabled),
		OpeningFloatEnabled:           orTrue(req.OpeningFloatEnabled),
		CountOpeningCashOnClose:       orTrue(req.CountOpeningCashOnClose),
		OverchangeConfirmation:        orTrue(req.OverchangeConfirmation),
		PageUnlockProducts:            orZero(req.PageUnlockProducts),
		HidePaymentMethodCurrency:     orZero(req.HidePaymentMethodCurrency),
		AutoAppendLbpZeros:            orZero(req.AutoAppendLbpZeros),
		AutoCloseStaleShifts:          orZero(req.AutoCloseStaleShifts),
		VisitorCounterEnabled:         orZero(req.VisitorCounterEnabled),
		ShiftReceiptUploadMode:        orDefault(req.ShiftReceiptUploadMode, 2),
		BranchPhone:                   orZero(req.BranchPhone),
		MaxOverpaymentCents:           orZero(req.MaxOverpaymentCents),
		DefaultOpeningFloatUsd:        orZero(req.DefaultOpeningFloatUsd),
		DefaultOpeningFloatLbp:        orZero(req.DefaultOpeningFloatLbp),
		DisplayPrefs:                  displayPrefs,
	})
}

// getBranchSettings is the admin-facing read side, used by kashi's UI to
// show a branch's current settings snapshot.
func (server *Server) getBranchSettings(ctx *gin.Context) {
	var req branchIDRequest
	if err := ctx.ShouldBindUri(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}

	settings, err := server.store.GetBranchSettings(ctx, req.ID)
	if err != nil {
		if err == sql.ErrNoRows {
			server.writeError(ctx, http.StatusNotFound, err)
			return
		}
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	server.writeJSON(ctx, http.StatusOK, envelope{"settings": settings})
}

type putBranchSettingsManagedLocallyRequest struct {
	// The set of group keys the branch owns locally. The admin UI renders
	// those groups read-only and never puts them in an update_settings
	// command payload. Known groups: "secrets", "device_ids".
	ManagedLocally []string `json:"managed_locally"`
}

// putBranchSettingsManagedLocally records which settings groups the admin
// has handed back to the branch. Admin-only; the branch never sets this.
func (server *Server) putBranchSettingsManagedLocally(ctx *gin.Context) {
	var idReq branchIDRequest
	if err := ctx.ShouldBindUri(&idReq); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	var body putBranchSettingsManagedLocallyRequest
	if err := ctx.ShouldBindJSON(&body); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	if body.ManagedLocally == nil {
		body.ManagedLocally = []string{}
	}

	settings, err := server.store.SetBranchSettingsManagedLocally(ctx, db.SetBranchSettingsManagedLocallyParams{
		BranchID:       idReq.ID,
		ManagedLocally: body.ManagedLocally,
	})
	if err != nil {
		if err == sql.ErrNoRows {
			server.writeError(ctx, http.StatusNotFound, err)
			return
		}
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	server.writeJSON(ctx, http.StatusOK, envelope{"settings": settings})
}
