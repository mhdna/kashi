package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	db "github.com/mhdna/kashi/db/sqlc"
	"github.com/mhdna/kashi/util"
)

// registerCodePattern matches what kashi-pos accepts as the code it puts in
// its invoice numbers (SA/<code>/<year>/<n>; see normalizeBranchCode there).
var registerCodePattern = regexp.MustCompile(`^[A-Z0-9]{1,8}$`)

var (
	errInvalidRegisterCode = errors.New("register code must be 1 to 8 letters or digits")
	errRegisterCodeTaken   = errors.New("that register code is already in use")
)

// branchRegisterResponse omits ApiKeyHash -- a bcrypt hash isn't
// reversible, but there's no reason to hand it to every admin-UI client.
type branchRegisterResponse struct {
	ID                 int64           `json:"id"`
	BranchID           int64           `json:"branch_id"`
	Code               string          `json:"code"`
	Name               string          `json:"name"`
	IsActive           bool            `json:"is_active"`
	LastSeenAt         sql.NullTime    `json:"last_seen_at"`
	ReportedSettings   json.RawMessage `json:"reported_settings"`
	ReportedSettingsAt sql.NullTime    `json:"reported_settings_at"`
	CreatedAt          time.Time       `json:"created_at"`
}

func newBranchRegisterResponse(register db.BranchRegister) branchRegisterResponse {
	reported := json.RawMessage("null")
	if register.ReportedSettings.Valid {
		reported = register.ReportedSettings.RawMessage
	}
	return branchRegisterResponse{
		ID:                 register.ID,
		BranchID:           register.BranchID,
		Code:               register.Code,
		Name:               register.Name,
		IsActive:           register.IsActive,
		LastSeenAt:         register.LastSeenAt,
		ReportedSettings:   reported,
		ReportedSettingsAt: register.ReportedSettingsAt,
		CreatedAt:          register.CreatedAt,
	}
}

type branchRegisterIDRequest struct {
	ID int64 `uri:"id" binding:"required,min=1"`
}

func (server *Server) listBranchRegisters(ctx *gin.Context) {
	var uri branchIDRequest
	if err := ctx.ShouldBindUri(&uri); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	registers, err := server.store.ListBranchRegisters(ctx, uri.ID)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	res := make([]branchRegisterResponse, len(registers))
	for i, register := range registers {
		res[i] = newBranchRegisterResponse(register)
	}
	server.writeJSON(ctx, http.StatusOK, envelope{"registers": res})
}

// listAllBranchRegisters backs the admin UI's register labels (e.g. which
// till an invoice or shift came from) without one request per branch.
func (server *Server) listAllBranchRegisters(ctx *gin.Context) {
	registers, err := server.store.ListAllBranchRegisters(ctx)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	res := make([]branchRegisterResponse, len(registers))
	for i, register := range registers {
		res[i] = newBranchRegisterResponse(register)
	}
	server.writeJSON(ctx, http.StatusOK, envelope{"registers": res})
}

// listConnectedRegisters is listConnectedBranches at register granularity.
func (server *Server) listConnectedRegisters(ctx *gin.Context) {
	server.writeJSON(ctx, http.StatusOK, envelope{"register_ids": server.branchHub.connectedRegisterIDs()})
}

type createBranchRegisterRequest struct {
	Code string `json:"code" binding:"required"`
	Name string `json:"name"`
}

// createBranchRegister adds a till to a branch. Like createBranch, the
// plaintext API key is returned only here. The new register is queued the
// branch's full settings so it starts in line with the rest of the branch
// the first time it connects.
func (server *Server) createBranchRegister(ctx *gin.Context) {
	var uri branchIDRequest
	if err := ctx.ShouldBindUri(&uri); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	var req createBranchRegisterRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	code := strings.ToUpper(strings.TrimSpace(req.Code))
	if !registerCodePattern.MatchString(code) {
		server.writeError(ctx, http.StatusBadRequest, errInvalidRegisterCode)
		return
	}

	if _, err := server.store.GetBranch(ctx, uri.ID); err != nil {
		if err == sql.ErrNoRows {
			server.writeError(ctx, http.StatusNotFound, err)
			return
		}
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	apiKey, err := generateBranchAPIKey()
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	apiKeyHash, err := util.HashPassword(apiKey)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	register, err := server.store.CreateBranchRegister(ctx, db.CreateBranchRegisterParams{
		BranchID:   uri.ID,
		Code:       code,
		Name:       strings.TrimSpace(req.Name),
		ApiKeyHash: apiKeyHash,
	})
	if err != nil {
		if isUniqueViolation(err) {
			server.writeError(ctx, http.StatusConflict, errRegisterCodeTaken)
			return
		}
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	server.queueBranchSettingsForRegister(ctx, register)

	server.writeJSON(ctx, http.StatusOK, envelope{
		"register": newBranchRegisterResponse(register),
		"api_key":  apiKey,
	})
}

// queueBranchSettingsForRegister sends a register the branch's full
// settings. Best-effort: a branch with no settings yet has nothing to send,
// and a failure here mustn't undo the register the admin just created.
func (server *Server) queueBranchSettingsForRegister(ctx *gin.Context, register db.BranchRegister) {
	settings, err := server.store.GetBranchSettings(ctx, register.BranchID)
	if err != nil {
		if err != sql.ErrNoRows {
			log.Printf("register %d: load branch settings: %v", register.ID, err)
		}
		return
	}
	payload, err := branchWideSettingsPayload(settings)
	if err != nil {
		log.Printf("register %d: build settings payload: %v", register.ID, err)
		return
	}
	userID := server.currentUserID(ctx)
	if !userID.Valid {
		return
	}
	if _, err := server.enqueueBranchCommand(ctx, register.BranchID, register.ID, "update_settings", payload, userID.Int64); err != nil {
		log.Printf("register %d: queue branch settings: %v", register.ID, err)
	}
}

type updateBranchRegisterRequest struct {
	Name string `json:"name"`
}

// updateBranchRegister renames a register. Its code is fixed: it's printed
// in every invoice number the till has already issued.
func (server *Server) updateBranchRegister(ctx *gin.Context) {
	var uri branchRegisterIDRequest
	if err := ctx.ShouldBindUri(&uri); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	var req updateBranchRegisterRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	register, err := server.store.UpdateBranchRegister(ctx, db.UpdateBranchRegisterParams{
		ID:   uri.ID,
		Name: strings.TrimSpace(req.Name),
	})
	if err != nil {
		if err == sql.ErrNoRows {
			server.writeError(ctx, http.StatusNotFound, err)
			return
		}
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	server.writeJSON(ctx, http.StatusOK, envelope{"register": newBranchRegisterResponse(register)})
}

func (server *Server) setBranchRegisterActive(active bool) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var uri branchRegisterIDRequest
		if err := ctx.ShouldBindUri(&uri); err != nil {
			server.writeError(ctx, http.StatusBadRequest, err)
			return
		}
		if _, err := server.store.GetBranchRegister(ctx, uri.ID); err != nil {
			if err == sql.ErrNoRows {
				server.writeError(ctx, http.StatusNotFound, err)
				return
			}
			server.writeError(ctx, http.StatusInternalServerError, err)
			return
		}
		if err := server.store.SetBranchRegisterActive(ctx, db.SetBranchRegisterActiveParams{
			ID:       uri.ID,
			IsActive: active,
		}); err != nil {
			server.writeError(ctx, http.StatusInternalServerError, err)
			return
		}
		server.writeJSON(ctx, http.StatusOK, envelope{"id": uri.ID, "is_active": active})
	}
}

// rotateBranchRegisterKey invalidates a register's current key immediately
// (a straight overwrite of api_key_hash) and returns the new plaintext key
// once, the same way createBranchRegister does.
func (server *Server) rotateBranchRegisterKey(ctx *gin.Context) {
	var uri branchRegisterIDRequest
	if err := ctx.ShouldBindUri(&uri); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	if _, err := server.store.GetBranchRegister(ctx, uri.ID); err != nil {
		if err == sql.ErrNoRows {
			server.writeError(ctx, http.StatusNotFound, err)
			return
		}
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	apiKey, err := generateBranchAPIKey()
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	apiKeyHash, err := util.HashPassword(apiKey)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	if err := server.store.UpdateBranchRegisterAPIKeyHash(ctx, db.UpdateBranchRegisterAPIKeyHashParams{
		ID:         uri.ID,
		ApiKeyHash: apiKeyHash,
	}); err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	server.writeJSON(ctx, http.StatusOK, envelope{"id": uri.ID, "api_key": apiKey})
}
