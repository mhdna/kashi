package api

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	db "github.com/mhdna/kashi/db/sqlc"
)

// Quick Access is the row of product shortcut cards on a till's selling
// screen. Each register has a mode (migration 000110):
//
//	till   the till keeps and edits its own list (the default)
//	kashi  the till shows its branch's central list, read-only
//
// Tills read both through the "quick_access" sync entity, flip their own
// mode with POST /branch/quick_access/mode, and seed their branch's list
// once from their own (POST /branch/quick_access/import). Admins edit the
// list and flip any register's mode below. Every change reaches tills
// through the database's change feed (sync_notify.go) -- nothing here
// notifies them by hand.

const (
	quickAccessModeTill  = "till"
	quickAccessModeKashi = "kashi"
	// maxQuickAccessItems bounds a list well past any real selling screen.
	maxQuickAccessItems = 200
)

var errBadQuickAccessMode = errors.New(`mode must be "till" or "kashi"`)

func validQuickAccessMode(mode string) bool {
	return mode == quickAccessModeTill || mode == quickAccessModeKashi
}

// --- till side ---------------------------------------------------------

type syncQuickAccessItem struct {
	ProductUID string `json:"product_uid"`
	Position   int32  `json:"position"`
	Color      string `json:"color"`
	TextColor  string `json:"text_color"`
}

// syncQuickAccess is the one item of the "quick_access" sync entity.
type syncQuickAccess struct {
	Mode          string                `json:"mode"`
	ModeChangedAt time.Time             `json:"mode_changed_at"`
	Items         []syncQuickAccessItem `json:"items"`
}

// quickAccessForRegisterSync answers the "quick_access" sync entity: a
// one-element list (the catch-up endpoint always returns a list) holding
// this register's mode and its branch's list.
func (server *Server) quickAccessForRegisterSync(ctx *gin.Context, branchID, registerID int64) ([]syncQuickAccess, error) {
	register, err := server.store.GetBranchRegister(ctx, registerID)
	if err != nil {
		return nil, err
	}
	rows, err := server.store.ListBranchQuickAccessItems(ctx, branchID)
	if err != nil {
		return nil, err
	}
	items := make([]syncQuickAccessItem, len(rows))
	for i, r := range rows {
		items[i] = syncQuickAccessItem{
			ProductUID: r.Uid.String(), Position: r.Position, Color: r.Color, TextColor: r.TextColor,
		}
	}
	return []syncQuickAccess{{
		Mode: register.QuickAccessMode, ModeChangedAt: register.QuickAccessModeChangedAt, Items: items,
	}}, nil
}

type branchQuickAccessModeRequest struct {
	Mode string `json:"mode" binding:"required"`
	// ChangedAt is when the flip was made on the till -- not when it
	// arrives, which can be much later for a flip made offline.
	ChangedAt time.Time `json:"changed_at" binding:"required"`
}

// branchSetQuickAccessMode is a till flipping its own mode. The later flip
// wins (see SetBranchRegisterQuickAccessMode); either way the answer is the
// mode now in effect, which the till adopts.
func (server *Server) branchSetQuickAccessMode(ctx *gin.Context) {
	registerID := ctx.MustGet(registerIDKey).(int64)
	var req branchQuickAccessModeRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	if !validQuickAccessMode(req.Mode) {
		server.writeError(ctx, http.StatusBadRequest, errBadQuickAccessMode)
		return
	}
	applied, err := server.store.SetBranchRegisterQuickAccessMode(ctx, db.SetBranchRegisterQuickAccessModeParams{
		ID: registerID, Mode: req.Mode, ChangedAt: req.ChangedAt,
	})
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	register, err := server.store.GetBranchRegister(ctx, registerID)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	server.writeJSON(ctx, http.StatusOK, envelope{
		"applied":         applied > 0,
		"mode":            register.QuickAccessMode,
		"mode_changed_at": register.QuickAccessModeChangedAt,
	})
}

type branchQuickAccessImportRequest struct {
	Items []struct {
		ProductUID string `json:"product_uid"`
		Color      string `json:"color"`
		TextColor  string `json:"text_color"`
	} `json:"items"`
}

// branchImportQuickAccess seeds the branch's list from a till's own list,
// once per branch -- a no-op (imported: false) when the branch already has
// one. Cards for products kashi doesn't know are skipped, not rejected: a
// till can hold a product kashi has since deleted.
func (server *Server) branchImportQuickAccess(ctx *gin.Context) {
	branchID := ctx.MustGet(branchIDKey).(int64)
	var req branchQuickAccessImportRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	if len(req.Items) > maxQuickAccessItems {
		req.Items = req.Items[:maxQuickAccessItems]
	}

	uids := make([]uuid.UUID, 0, len(req.Items))
	for _, item := range req.Items {
		if id, err := uuid.Parse(item.ProductUID); err == nil {
			uids = append(uids, id)
		}
	}
	rows, err := server.store.ListProductVariantIDsByUIDs(ctx, uids)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	idByUID := make(map[string]int64, len(rows))
	for _, r := range rows {
		idByUID[r.Uid.String()] = r.ID
	}

	items := make([]db.QuickAccessItemInput, 0, len(req.Items))
	seen := make(map[int64]bool, len(req.Items))
	for _, item := range req.Items {
		id, ok := idByUID[item.ProductUID]
		if !ok || seen[id] {
			continue
		}
		seen[id] = true
		items = append(items, db.QuickAccessItemInput{
			ProductVariantID: id, Color: cleanCardColor(item.Color), TextColor: cleanCardColor(item.TextColor),
		})
	}

	imported, err := server.store.ImportBranchQuickAccessTx(ctx, branchID, items)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	server.writeJSON(ctx, http.StatusOK, envelope{"imported": imported})
}

// cleanCardColor keeps a well-formed #rrggbb and drops anything else to ""
// (the default colour) -- an import shouldn't fail over one odd value.
func cleanCardColor(c string) string {
	if hexColorPattern.MatchString(c) {
		return c
	}
	return ""
}

// --- admin side --------------------------------------------------------

type quickAccessItemResponse struct {
	ProductVariantID int64  `json:"product_variant_id"`
	ProductUID       string `json:"product_uid"`
	Position         int32  `json:"position"`
	Color            string `json:"color"`
	TextColor        string `json:"text_color"`
	Barcode          string `json:"barcode"`
	Code             string `json:"code"`
	Name             string `json:"name"`
	ColorName        string `json:"color_name"`
	SizeName         string `json:"size_name"`
	IsActive         bool   `json:"is_active"`
}

type quickAccessRegisterResponse struct {
	ID            int64     `json:"id"`
	Code          string    `json:"code"`
	Name          string    `json:"name"`
	IsActive      bool      `json:"is_active"`
	Mode          string    `json:"mode"`
	ModeChangedAt time.Time `json:"mode_changed_at"`
}

// getBranchQuickAccess is the admin editor's view: the branch's list and
// each of its registers' modes.
func (server *Server) getBranchQuickAccess(ctx *gin.Context) {
	var uri branchIDRequest
	if err := ctx.ShouldBindUri(&uri); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	if _, err := server.store.GetBranch(ctx, uri.ID); err != nil {
		server.writeStoreError(ctx, err)
		return
	}
	rows, err := server.store.ListBranchQuickAccessItems(ctx, uri.ID)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	items := make([]quickAccessItemResponse, len(rows))
	for i, r := range rows {
		items[i] = quickAccessItemResponse{
			ProductVariantID: r.ProductVariantID, ProductUID: r.Uid.String(), Position: r.Position,
			Color: r.Color, TextColor: r.TextColor, Barcode: r.Barcode, Code: r.Code, Name: r.Name,
			ColorName: r.ColorName.String, SizeName: r.SizeName.String, IsActive: r.IsActive,
		}
	}
	registers, err := server.store.ListBranchRegisters(ctx, uri.ID)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	out := make([]quickAccessRegisterResponse, len(registers))
	for i, r := range registers {
		out[i] = quickAccessRegisterResponse{
			ID: r.ID, Code: r.Code, Name: r.Name, IsActive: r.IsActive,
			Mode: r.QuickAccessMode, ModeChangedAt: r.QuickAccessModeChangedAt,
		}
	}
	server.writeJSON(ctx, http.StatusOK, envelope{"items": items, "registers": out})
}

type putBranchQuickAccessRequest struct {
	Items []struct {
		ProductVariantID int64  `json:"product_variant_id" binding:"required,min=1"`
		Color            string `json:"color"`
		TextColor        string `json:"text_color"`
	} `json:"items" binding:"dive"`
}

// putBranchQuickAccess replaces a branch's whole list, in the given order.
func (server *Server) putBranchQuickAccess(ctx *gin.Context) {
	var uri branchIDRequest
	if err := ctx.ShouldBindUri(&uri); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	var req putBranchQuickAccessRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	if len(req.Items) > maxQuickAccessItems {
		server.writeError(ctx, http.StatusBadRequest, fmt.Errorf("at most %d quick access items", maxQuickAccessItems))
		return
	}
	if _, err := server.store.GetBranch(ctx, uri.ID); err != nil {
		server.writeStoreError(ctx, err)
		return
	}

	items := make([]db.QuickAccessItemInput, len(req.Items))
	seen := make(map[int64]bool, len(req.Items))
	for i, item := range req.Items {
		if seen[item.ProductVariantID] {
			server.writeError(ctx, http.StatusBadRequest, fmt.Errorf("product variant %d is listed twice", item.ProductVariantID))
			return
		}
		seen[item.ProductVariantID] = true
		for _, c := range []string{item.Color, item.TextColor} {
			if c != "" && !hexColorPattern.MatchString(c) {
				server.writeError(ctx, http.StatusBadRequest, fmt.Errorf("colour %q must be #rrggbb", c))
				return
			}
		}
		items[i] = db.QuickAccessItemInput{
			ProductVariantID: item.ProductVariantID, Color: item.Color, TextColor: item.TextColor,
		}
	}

	if err := server.store.ReplaceBranchQuickAccessTx(ctx, uri.ID, items); err != nil {
		if isForeignKeyViolation(err) {
			server.writeError(ctx, http.StatusBadRequest, errors.New("unknown product variant in the list"))
			return
		}
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	server.getBranchQuickAccess(ctx)
}

type putRegisterQuickAccessModeRequest struct {
	Mode string `json:"mode" binding:"required"`
}

// putRegisterQuickAccessMode is an admin flipping one register's mode. An
// admin's flip is always "now", but never older than the stored one: a till
// whose clock runs ahead could otherwise make an admin's flip look stale.
func (server *Server) putRegisterQuickAccessMode(ctx *gin.Context) {
	var uri branchRegisterIDRequest
	if err := ctx.ShouldBindUri(&uri); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	var req putRegisterQuickAccessModeRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	if !validQuickAccessMode(req.Mode) {
		server.writeError(ctx, http.StatusBadRequest, errBadQuickAccessMode)
		return
	}
	register, err := server.store.GetBranchRegister(ctx, uri.ID)
	if err != nil {
		server.writeStoreError(ctx, err)
		return
	}
	changedAt := time.Now()
	if !changedAt.After(register.QuickAccessModeChangedAt) {
		changedAt = register.QuickAccessModeChangedAt.Add(time.Millisecond)
	}
	if _, err := server.store.SetBranchRegisterQuickAccessMode(ctx, db.SetBranchRegisterQuickAccessModeParams{
		ID: uri.ID, Mode: req.Mode, ChangedAt: changedAt,
	}); err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	server.writeJSON(ctx, http.StatusOK, envelope{"mode": req.Mode, "mode_changed_at": changedAt})
}

// writeStoreError maps a lookup's sql.ErrNoRows to 404, anything else to 500.
func (server *Server) writeStoreError(ctx *gin.Context, err error) {
	if errors.Is(err, sql.ErrNoRows) {
		server.writeError(ctx, http.StatusNotFound, err)
		return
	}
	server.writeError(ctx, http.StatusInternalServerError, err)
}
