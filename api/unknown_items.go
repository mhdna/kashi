package api

import (
	"crypto/rand"
	"database/sql"
	_ "embed"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	db "github.com/mhdna/kashi/db/sqlc"
	"github.com/mhdna/kashi/token"
)

// "Unknown item" sales. A till sells a product it has no code for after a
// phone has taken a picture of every unit; the pictures go straight from the
// phone to kashi and the till waits only until every unit has one. Once the
// sale lands, an admin matches each picture to a real product
// (resolveUnknownItems) and the sold line is relabelled.

const (
	unknownSessionTTL      = 6 * time.Hour
	unknownStatusWait      = 6 * time.Second // under the till's 10s call timeout
	maxUnknownSessionSlots = 60
	maxUnknownImageBytes   = 10 << 20 // 10MB; the phone sends far less
)

//go:embed unknown_upload_assets/upload.html
var unknownUploadPageHTML []byte

var unknownUploadPage = newPhoneUploadPage(unknownUploadPageHTML)

type unknownSlotRequest struct {
	SlotRef   string `json:"slot_ref" binding:"required"`
	Label     string `json:"label"`
	UnitPrice int64  `json:"unit_price"`
	Discount  int16  `json:"discount"`
}

type createUnknownSessionRequest struct {
	ClientRef string               `json:"client_ref" binding:"required"`
	Slots     []unknownSlotRequest `json:"slots" binding:"required,min=1,dive"`
}

// unknownSlotView is one unit as the phone page and the till see it.
type unknownSlotView struct {
	SlotRef   string `json:"slot_ref"`
	Position  int32  `json:"position"`
	Label     string `json:"label"`
	UnitPrice int64  `json:"unit_price"`
	Discount  int16  `json:"discount"`
	ImageID   int64  `json:"image_id,omitempty"`
	HasImage  bool   `json:"has_image"`
	// Waived: a manager let this unit through with no picture.
	Waived bool `json:"waived"`
}

func unknownSlotViews(slots []db.UnknownItemSlot) []unknownSlotView {
	views := make([]unknownSlotView, 0, len(slots))
	for _, slot := range slots {
		view := unknownSlotView{
			SlotRef: slot.SlotRef, Position: slot.Position, Label: slot.Label,
			UnitPrice: slot.UnitPrice, Discount: slot.Discount,
			HasImage: slot.ImageID.Valid,
			Waived:   slot.WaivedAt.Valid,
		}
		if slot.ImageID.Valid {
			view.ImageID = slot.ImageID.Int64
		}
		views = append(views, view)
	}
	return views
}

func countDoneUnknownSlots(slots []db.UnknownItemSlot) int {
	done := 0
	for _, slot := range slots {
		if slot.ImageID.Valid || slot.WaivedAt.Valid {
			done++
		}
	}
	return done
}

// allUnknownSlotsPictured: every unit has a picture or a manager's pass.
func allUnknownSlotsPictured(slots []db.UnknownItemSlot) bool {
	for _, slot := range slots {
		if !slot.ImageID.Valid && !slot.WaivedAt.Valid {
			return false
		}
	}
	return len(slots) > 0
}

// createUnknownSession opens (or reopens, for a retried call) the picture
// session of one checkout and returns the token the phone's QR link carries.
func (server *Server) createUnknownSession(ctx *gin.Context) {
	branchID := ctx.MustGet(branchIDKey).(int64)
	registerID := ctx.MustGet(registerIDKey).(int64)

	var req createUnknownSessionRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	if len(req.Slots) > maxUnknownSessionSlots {
		server.writeError(ctx, http.StatusBadRequest, fmt.Errorf("at most %d unknown items per sale", maxUnknownSessionSlots))
		return
	}

	raw := make([]byte, 24)
	if _, err := rand.Read(raw); err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	session, err := server.store.CreateUnknownItemSession(ctx, db.CreateUnknownItemSessionParams{
		BranchID:   branchID,
		RegisterID: registerID,
		ClientRef:  req.ClientRef,
		Token:      hex.EncodeToString(raw),
		ExpiresAt:  time.Now().UTC().Add(unknownSessionTTL),
	})
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	for i, slot := range req.Slots {
		if err := server.store.CreateUnknownItemSlot(ctx, db.CreateUnknownItemSlotParams{
			SessionID: session.ID, SlotRef: slot.SlotRef, Position: int32(i),
			Label: slot.Label, UnitPrice: slot.UnitPrice, Discount: slot.Discount,
		}); err != nil {
			server.writeError(ctx, http.StatusInternalServerError, err)
			return
		}
	}
	slots, err := server.store.ListUnknownItemSlotsForSession(ctx, session.ID)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	server.writeJSON(ctx, http.StatusOK, envelope{
		"upload_token": session.Token,
		"slots":        unknownSlotViews(slots),
		"complete":     allUnknownSlotsPictured(slots),
	})
}

// branchUnknownSessionStatus is the till's poll: which units have a picture.
func (server *Server) branchUnknownSessionStatus(ctx *gin.Context) {
	branchID := ctx.MustGet(branchIDKey).(int64)
	session, err := server.store.GetUnknownItemSessionByRef(ctx, db.GetUnknownItemSessionByRefParams{
		BranchID: branchID, ClientRef: ctx.Query("client_ref"),
	})
	if errors.Is(err, sql.ErrNoRows) {
		server.writeError(ctx, http.StatusNotFound, errors.New("no such session"))
		return
	}
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	slots, err := server.store.ListUnknownItemSlotsForSession(ctx, session.ID)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	// Long poll: a till that already knows `known` finished units is held
	// here until that number changes (or a few seconds pass), so a picture
	// reaches the till the moment it lands instead of on its next tick.
	if known, err := strconv.Atoi(ctx.Query("known")); err == nil {
		deadline := time.NewTimer(unknownStatusWait)
		defer deadline.Stop()
		tick := time.NewTicker(250 * time.Millisecond)
		defer tick.Stop()
		for countDoneUnknownSlots(slots) == known {
			select {
			case <-ctx.Request.Context().Done():
				return
			case <-deadline.C:
			case <-tick.C:
				if slots, err = server.store.ListUnknownItemSlotsForSession(ctx, session.ID); err != nil {
					server.writeError(ctx, http.StatusInternalServerError, err)
					return
				}
				continue
			}
			break
		}
	}
	server.writeJSON(ctx, http.StatusOK, envelope{
		"slots":    unknownSlotViews(slots),
		"complete": allUnknownSlotsPictured(slots),
	})
}

// linkUnknownSession ties a landed sale to its picture session, so the
// pictures show up in "Unknown items" only for sales that really happened.
func (server *Server) linkUnknownSession(ctx *gin.Context, branchID, invoiceID int64, sessionRef string) {
	if sessionRef == "" {
		return
	}
	_ = server.store.LinkUnknownItemSession(ctx, db.LinkUnknownItemSessionParams{
		BranchID:        branchID,
		ClientRef:       sessionRef,
		BranchInvoiceID: sql.NullInt64{Int64: invoiceID, Valid: true},
	})
}

// --- the phone, reached by the QR link (public, token-gated) ---

func (server *Server) unknownUploadPage(ctx *gin.Context) {
	server.servePhoneUploadPage(ctx, unknownUploadPage)
}

func (server *Server) lookupUnknownSession(ctx *gin.Context) (db.UnknownItemSession, bool) {
	session, err := server.store.GetUnknownItemSessionByToken(ctx, ctx.Param("token"))
	// Dead once expired, or once the sale it belongs to has landed.
	if errors.Is(err, sql.ErrNoRows) || (err == nil && (session.BranchInvoiceID.Valid || time.Now().UTC().After(session.ExpiresAt))) {
		server.writeError(ctx, http.StatusNotFound, errors.New("this link is no longer valid"))
		return db.UnknownItemSession{}, false
	}
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return db.UnknownItemSession{}, false
	}
	return session, true
}

func (server *Server) unknownUploadStatus(ctx *gin.Context) {
	session, ok := server.lookupUnknownSession(ctx)
	if !ok {
		return
	}
	slots, err := server.store.ListUnknownItemSlotsForSession(ctx, session.ID)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	server.writeJSON(ctx, http.StatusOK, envelope{
		"slots":    unknownSlotViews(slots),
		"complete": allUnknownSlotsPictured(slots),
	})
}

// unknownUploadImageFile serves a session picture back to the phone (a small
// thumbnail), so a reload of the page still shows what was already taken.
func (server *Server) unknownUploadImageFile(ctx *gin.Context) {
	session, ok := server.lookupUnknownSession(ctx)
	if !ok {
		return
	}
	imageID, err := strconv.ParseInt(ctx.Param("image_id"), 10, 64)
	if err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	image, err := server.store.GetUnknownItemImage(ctx, imageID)
	if err != nil || image.SessionID != session.ID {
		server.writeError(ctx, http.StatusNotFound, errors.New("no such picture"))
		return
	}
	server.serveStoredImage(ctx, image.FilePath, image.ContentType, image.Rotation)
}

// uploadUnknownSlotImage takes one picture for one unit. The phone sends each
// as soon as it is taken (already shrunk), so the till is released the moment
// the last unit has its picture.
func (server *Server) uploadUnknownSlotImage(ctx *gin.Context) {
	session, ok := server.lookupUnknownSession(ctx)
	if !ok {
		return
	}
	slot, err := server.store.GetUnknownItemSlotByRef(ctx, db.GetUnknownItemSlotByRefParams{
		SessionID: session.ID, SlotRef: ctx.Param("slot_ref"),
	})
	if err != nil {
		server.writeError(ctx, http.StatusNotFound, errors.New("no such item"))
		return
	}
	fileHeader, err := ctx.FormFile("image")
	if err != nil {
		server.writeError(ctx, http.StatusBadRequest, errors.New("no image provided"))
		return
	}
	if fileHeader.Size > maxUnknownImageBytes {
		server.writeError(ctx, http.StatusBadRequest, fmt.Errorf("the image must be under %dMB", maxUnknownImageBytes>>20))
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
	dir := filepath.Join(server.config.ExpenseImageStorageDir, "unknown", strconv.FormatInt(session.ID, 10))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	destPath := filepath.Join(dir, uuid.NewString()+stored.Ext)
	if err := os.WriteFile(destPath, stored.Data, 0o644); err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	warmThumbnails(destPath)
	image, err := server.store.CreateUnknownItemImage(ctx, db.CreateUnknownItemImageParams{
		SessionID: session.ID, FilePath: destPath, ContentType: stored.ContentType,
		SizeBytes: int64(len(stored.Data)),
	})
	if err != nil {
		os.Remove(destPath)
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	if err := server.store.SetUnknownItemSlotImage(ctx, db.SetUnknownItemSlotImageParams{
		SessionID: session.ID, SlotRef: slot.SlotRef, ImageID: sql.NullInt64{Int64: image.ID, Valid: true},
	}); err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	server.adminHub.broadcastAll(adminWSMessage{Type: "unknown_item_uploaded", BranchID: session.BranchID})
	// The phone gets the new state back, so it needs no status call after.
	slots, err := server.store.ListUnknownItemSlotsForSession(ctx, session.ID)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	server.writeJSON(ctx, http.StatusOK, envelope{
		"image_id": image.ID,
		"slots":    unknownSlotViews(slots),
		"complete": allUnknownSlotsPictured(slots),
	})
}

type copyUnknownSlotRequest struct {
	FromSlotRef string `json:"from_slot_ref" binding:"required"`
}

// copyUnknownSlotImage is "use picture X": a unit that is the same item as an
// already pictured one shares its picture, with nothing uploaded again.
func (server *Server) copyUnknownSlotImage(ctx *gin.Context) {
	session, ok := server.lookupUnknownSession(ctx)
	if !ok {
		return
	}
	var req copyUnknownSlotRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	target := ctx.Param("slot_ref")
	if _, err := server.store.GetUnknownItemSlotByRef(ctx, db.GetUnknownItemSlotByRefParams{SessionID: session.ID, SlotRef: target}); err != nil {
		server.writeError(ctx, http.StatusNotFound, errors.New("no such item"))
		return
	}
	source, err := server.store.GetUnknownItemSlotByRef(ctx, db.GetUnknownItemSlotByRefParams{SessionID: session.ID, SlotRef: req.FromSlotRef})
	if err != nil || !source.ImageID.Valid {
		server.writeError(ctx, http.StatusBadRequest, errors.New("that item has no picture yet"))
		return
	}
	if err := server.store.SetUnknownItemSlotImage(ctx, db.SetUnknownItemSlotImageParams{
		SessionID: session.ID, SlotRef: target, ImageID: source.ImageID,
	}); err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	slots, err := server.store.ListUnknownItemSlotsForSession(ctx, session.ID)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	server.writeJSON(ctx, http.StatusOK, envelope{
		"image_id": source.ImageID.Int64,
		"slots":    unknownSlotViews(slots),
		"complete": allUnknownSlotsPictured(slots),
	})
}

// --- the admin ---

type listUnknownItemsRequest struct {
	Resolved bool  `form:"resolved"`
	PageSize int32 `form:"page_size,default=30" binding:"min=1,max=100"`
	PageID   int32 `form:"page_id,default=0" binding:"min=0"`
}

func (server *Server) listUnknownItems(ctx *gin.Context) {
	var req listUnknownItemsRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	items, err := server.store.ListUnknownItems(ctx, db.ListUnknownItemsParams{
		Resolved: req.Resolved, Limit: req.PageSize, Offset: req.PageID * req.PageSize,
	})
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	total, err := server.store.CountUnknownItems(ctx, req.Resolved)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	pending, err := server.store.CountUnknownItems(ctx, false)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	server.writeJSON(ctx, http.StatusOK, envelope{"items": items, "total": total, "pending": pending})
}

type listUnknownItemImagesRequest struct {
	PageSize int32 `form:"page_size,default=24" binding:"min=1,max=100"`
	PageID   int32 `form:"page_id,default=0" binding:"min=0"`
}

// listUnknownItemImages feeds the Attachments page: the stored pictures
// themselves, not the units (several units can share one).
func (server *Server) listUnknownItemImages(ctx *gin.Context) {
	var req listUnknownItemImagesRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	images, err := server.store.ListUnknownItemImages(ctx, db.ListUnknownItemImagesParams{
		Limit: req.PageSize, Offset: req.PageID * req.PageSize,
	})
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	total, err := server.store.CountUnknownItemImages(ctx)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	server.writeJSON(ctx, http.StatusOK, envelope{"images": images, "total": total})
}

// getUnknownItemImageFile streams a picture to the admin site (fetched as a
// blob, like the other photos, since <img> can't send the Bearer token).
func (server *Server) getUnknownItemImageFile(ctx *gin.Context) {
	imageID, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	image, err := server.store.GetUnknownItemImage(ctx, imageID)
	if err != nil {
		server.writeError(ctx, http.StatusNotFound, err)
		return
	}
	server.serveStoredImage(ctx, image.FilePath, image.ContentType, image.Rotation)
}

type resolveUnknownItemsRequest struct {
	SlotIDs   []int64 `json:"slot_ids" binding:"required,min=1,dive,min=1"`
	VariantID int64   `json:"variant_id" binding:"required,min=1"`
}

func (server *Server) resolveUnknownItems(ctx *gin.Context) {
	var req resolveUnknownItemsRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	if err := server.store.ResolveUnknownItemsTx(ctx, req.SlotIDs, req.VariantID); err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, db.ErrUnknownItemNotResolvable) || errors.Is(err, sql.ErrNoRows) {
			status = http.StatusConflict
		}
		server.writeError(ctx, status, err)
		return
	}
	server.writeJSON(ctx, http.StatusOK, envelope{"resolved": len(req.SlotIDs)})
}

// searchUnknownItemVariants feeds the product picker on the Unknown items page.
func (server *Server) searchUnknownItemVariants(ctx *gin.Context) {
	search := strings.TrimSpace(ctx.Query("search"))
	if search == "" {
		server.writeJSON(ctx, http.StatusOK, envelope{"variants": []db.SearchVariantsForUnknownItemsRow{}})
		return
	}
	variants, err := server.store.SearchVariantsForUnknownItems(ctx, search)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	server.writeJSON(ctx, http.StatusOK, envelope{"variants": variants})
}

type unknownHelpRequest struct {
	ClientRef string `json:"client_ref" binding:"required"`
}

// requestUnknownSessionHelp is the cashier's "ask a manager" button: the
// session moves to the top of the Waiting tab.
func (server *Server) requestUnknownSessionHelp(ctx *gin.Context) {
	branchID := ctx.MustGet(branchIDKey).(int64)
	var req unknownHelpRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	if err := server.store.RequestUnknownItemHelp(ctx, db.RequestUnknownItemHelpParams{
		BranchID: branchID, ClientRef: req.ClientRef,
	}); err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	server.adminHub.broadcastAll(adminWSMessage{Type: "unknown_item_help_requested", BranchID: branchID})
	server.writeJSON(ctx, http.StatusOK, envelope{"requested": true})
}

func (server *Server) listWaitingUnknownItems(ctx *gin.Context) {
	items, err := server.store.ListWaitingUnknownItems(ctx)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	server.writeJSON(ctx, http.StatusOK, envelope{"items": items})
}

type waiveUnknownItemsRequest struct {
	SlotIDs []int64 `json:"slot_ids" binding:"required,min=1,dive,min=1"`
}

// waiveUnknownItems lets units through with no picture, on the spot, so the
// till can take payment. It is its own permission (unknown_item_waive) and the
// unit still lands in "To resolve" afterwards, marked with who allowed it.
func (server *Server) waiveUnknownItems(ctx *gin.Context) {
	var req waiveUnknownItemsRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	by := ""
	if payload, ok := ctx.MustGet(authoizationPayloadKey).(*token.Payload); ok {
		by = payload.Username
	}
	waived, err := server.store.WaiveUnknownItemSlots(ctx, db.WaiveUnknownItemSlotsParams{
		Column1: req.SlotIDs, WaivedBy: by,
	})
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	server.writeJSON(ctx, http.StatusOK, envelope{"waived": waived})
}

// branchUnknownBarcode is the barcode this branch named for its "Unknown item"
// product ("" when it uses the till's built-in placeholder, which kashi
// recognises on its own).
func (server *Server) branchUnknownBarcode(ctx *gin.Context, branchID int64) string {
	settings, err := server.store.GetBranchSettings(ctx, branchID)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(settings.UnknownProductBarcode)
}
