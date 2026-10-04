package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	db "github.com/mhdna/kashi/db/sqlc"
	"github.com/mhdna/kashi/token"
	"github.com/sqlc-dev/pqtype"
)

var (
	errUnknownCommandType      = errors.New("unknown or unsupported command type")
	errCommandRegisterMismatch = errors.New("command does not belong to this register")
	errRegisterBranchMismatch  = errors.New("register does not belong to this branch")
)

// allowedBranchCommandTypes gates what an admin can enqueue -- adding a new
// remote capability means adding it here AND teaching kashi-pos's command
// executor (see commands.go there) how to run it via a real local function,
// never adding a type that has no corresponding, validated execution path.
var allowedBranchCommandTypes = map[string]bool{
	"update_settings":       true,
	"remote_return_invoice": true,
	"set_branch_user_pin":   true,
	// transfer_decision is never enqueued through createBranchCommand (the
	// admin-typed endpoint below) -- only by notifyBranchTransferDecision
	// in api/transfer.go, straight from an approve/cancel. Listed here
	// anyway so this map stays the single place documenting every command
	// type kashi-pos's executor needs to know.
	"transfer_decision": true,
}

type createBranchCommandRequest struct {
	Type    string          `json:"type" binding:"required"`
	Payload json.RawMessage `json:"payload" binding:"required"`
	// RegisterID addresses one register of the branch. 0 lets kashi pick:
	// a remote return goes to the register that made the sale, anything
	// else to every active register.
	RegisterID int64 `json:"register_id"`
}

// createBranchCommand is how a human admin asks a branch to do something --
// the enqueue side of the remote command channel. It never talks to
// kashi-pos directly: it writes pending rows (the audit trail) and nudges
// connected registers over the WebSocket to fetch them immediately. A
// disconnected or slow-to-notice till still picks its command up on its
// next sync tick (see branchPendingCommands).
//
// A branch-wide update_settings is also applied to branch_settings here:
// kashi is the authority for branch-wide settings, and every register
// receives the same change.
func (server *Server) createBranchCommand(ctx *gin.Context) {
	var idReq branchIDRequest
	if err := ctx.ShouldBindUri(&idReq); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	var req createBranchCommandRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	if !allowedBranchCommandTypes[req.Type] {
		server.writeError(ctx, http.StatusBadRequest, errUnknownCommandType)
		return
	}

	if _, err := server.store.GetBranch(ctx, idReq.ID); err != nil {
		if err == sql.ErrNoRows {
			server.writeError(ctx, http.StatusNotFound, err)
			return
		}
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	registerID := req.RegisterID
	if registerID != 0 {
		register, err := server.store.GetBranchRegister(ctx, registerID)
		if err != nil || register.BranchID != idReq.ID {
			server.writeError(ctx, http.StatusBadRequest, errRegisterBranchMismatch)
			return
		}
	} else if req.Type == "remote_return_invoice" {
		var target struct {
			ClientRef string `json:"client_ref"`
		}
		if err := json.Unmarshal(req.Payload, &target); err != nil || target.ClientRef == "" {
			server.writeError(ctx, http.StatusBadRequest, errors.New("remote_return_invoice needs the sale's client_ref"))
			return
		}
		invoice, err := server.store.GetBranchInvoiceByClientRef(ctx, db.GetBranchInvoiceByClientRefParams{
			BranchID:  idReq.ID,
			ClientRef: target.ClientRef,
		})
		if err != nil {
			if err == sql.ErrNoRows {
				server.writeError(ctx, http.StatusNotFound, err)
				return
			}
			server.writeError(ctx, http.StatusInternalServerError, err)
			return
		}
		registerID = invoice.RegisterID
	}

	payload := req.Payload
	if req.Type == "update_settings" && registerID == 0 {
		var err error
		payload, err = server.applyBranchSettingsPatch(ctx, idReq.ID, req.Payload)
		if err != nil {
			if errors.Is(err, errNoBranchSettings) {
				server.writeError(ctx, http.StatusConflict, err)
				return
			}
			server.writeError(ctx, http.StatusBadRequest, err)
			return
		}
	}

	// The PASETO payload only carries the operator's username (see
	// token.Payload) -- resolve it to a real user id so the audit trail
	// (branch_commands.issued_by) is a durable FK, not a name that could
	// later be renamed or reused.
	tokenPayload := ctx.MustGet(authoizationPayloadKey).(*token.Payload)
	user, err := server.store.GetUserByUsername(ctx, tokenPayload.Username)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	commands, err := server.enqueueBranchCommand(ctx, idReq.ID, registerID, req.Type, payload, user.ID)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	var first any
	if len(commands) > 0 {
		first = commands[0]
	}
	server.writeJSON(ctx, http.StatusOK, envelope{"command": first, "commands": commands})
}

// enqueueBranchCommand is the shared write side of the remote command
// channel: it writes one pending row per target register (the audit trail)
// and nudges each over the WebSocket if it's connected. registerID 0 fans
// the command out to every active register of the branch, so each till
// acks its own copy. A disconnected or slow-to-notice till still picks its
// command up on its next sync tick (see branchPendingCommands).
func (server *Server) enqueueBranchCommand(ctx *gin.Context, branchID, registerID int64, cmdType string, payload json.RawMessage, issuedBy int64) ([]db.BranchCommand, error) {
	registerIDs := []int64{registerID}
	if registerID == 0 {
		var err error
		registerIDs, err = server.store.ListActiveBranchRegisterIDs(ctx, branchID)
		if err != nil {
			return nil, err
		}
	}

	commands := make([]db.BranchCommand, 0, len(registerIDs))
	for _, id := range registerIDs {
		command, err := server.store.CreateBranchCommand(ctx, db.CreateBranchCommandParams{
			BranchID:   branchID,
			RegisterID: id,
			Type:       cmdType,
			Payload:    payload,
			IssuedBy:   issuedBy,
		})
		if err != nil {
			return commands, err
		}
		commands = append(commands, command)
		server.branchHub.notifyRegister(id, branchWSMessage{Type: "command", CommandID: command.ID})
	}
	return commands, nil
}

// listBranchCommands is the admin-facing audit trail for one branch --
// every command ever issued to any of its registers, its payload, and its
// final result (or that it's still pending).
func (server *Server) listBranchCommands(ctx *gin.Context) {
	var idReq branchIDRequest
	if err := ctx.ShouldBindUri(&idReq); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	var req struct {
		PageSize int32 `form:"page_size,default=10" binding:"min=5,max=100"`
		PageID   int32 `form:"page_id,default=0" binding:"min=0"`
	}
	if err := ctx.ShouldBindQuery(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}

	commands, err := server.store.ListBranchCommands(ctx, db.ListBranchCommandsParams{
		BranchID: idReq.ID,
		Limit:    req.PageSize,
		Offset:   req.PageID,
	})
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	server.writeJSON(ctx, http.StatusOK, envelope{"branch_commands": commands})
}

// branchPendingCommands is the branch-facing poll side -- called every
// sync tick regardless of whether a WebSocket push was received, the same
// dual-path guarantee every other pushed entity gets (see
// branch_catchup.go). A branch that was offline when a command was issued
// simply finds it here the next time it's back.
func (server *Server) branchPendingCommands(ctx *gin.Context) {
	registerID := ctx.MustGet(registerIDKey).(int64)

	commands, err := server.store.ListPendingBranchCommands(ctx, registerID)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	server.writeJSON(ctx, http.StatusOK, envelope{"commands": commands})
}

type ackBranchCommandRequest struct {
	Status string          `json:"status" binding:"required,oneof=success failed"`
	Result json.RawMessage `json:"result"`
	Error  string          `json:"error"`
}

// ackBranchCommand is how a branch reports what happened after it executed
// a command -- kashi never re-derives or double-checks the outcome, it
// records exactly what kashi-pos's real service function returned, the
// same "branch is decisive" principle already applied to invoices.
func (server *Server) ackBranchCommand(ctx *gin.Context) {
	var req branchInvoiceIDRequest // {id} — reused, same uri shape
	if err := ctx.ShouldBindUri(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	var body ackBranchCommandRequest
	if err := ctx.ShouldBindJSON(&body); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}

	registerID := ctx.MustGet(registerIDKey).(int64)
	command, err := server.store.GetBranchCommand(ctx, req.ID)
	if err != nil {
		if err == sql.ErrNoRows {
			server.writeError(ctx, http.StatusNotFound, err)
			return
		}
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	if command.RegisterID != registerID {
		server.writeError(ctx, http.StatusForbidden, errCommandRegisterMismatch)
		return
	}

	var errText sql.NullString
	if body.Error != "" {
		errText = sql.NullString{String: body.Error, Valid: true}
	}
	var result pqtype.NullRawMessage
	if len(body.Result) > 0 {
		result = pqtype.NullRawMessage{RawMessage: body.Result, Valid: true}
	}

	updated, err := server.store.CompleteBranchCommand(ctx, db.CompleteBranchCommandParams{
		ID:     req.ID,
		Status: body.Status,
		Result: result,
		Error:  errText,
	})
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	server.writeJSON(ctx, http.StatusOK, envelope{"command": updated})
}
