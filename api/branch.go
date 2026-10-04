package api

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	db "github.com/mhdna/kashi/db/sqlc"
	"github.com/mhdna/kashi/util"
)

// generateBranchAPIKey returns a high-entropy, URL-safe plaintext key.
// crypto/rand is required here (not util.RandomString, which uses
// math/rand and is meant for generating test fixtures, not credentials).
func generateBranchAPIKey() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

type branchResponse struct {
	ID         int64        `json:"id"`
	Name       string       `json:"name"`
	Code       string       `json:"code"`
	IsActive   bool         `json:"is_active"`
	LastSeenAt sql.NullTime `json:"last_seen_at"`
	CreatedAt  time.Time    `json:"created_at"`
}

func newBranchResponse(branch db.Branch) branchResponse {
	return branchResponse{
		ID:         branch.ID,
		Name:       branch.Name,
		Code:       branch.Code,
		IsActive:   branch.IsActive,
		LastSeenAt: branch.LastSeenAt,
		CreatedAt:  branch.CreatedAt,
	}
}

type createBranchRequest struct {
	Name string `json:"name" binding:"required"`
	Code string `json:"code" binding:"required"`
}

// createBranch is called by a human admin (authRoutes, PASETO auth), never
// by a branch itself. It also creates the branch's first register under the
// branch code; that register's plaintext API key is only ever returned here
// — from then on only its bcrypt hash is stored.
func (server *Server) createBranch(ctx *gin.Context) {
	var req createBranchRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
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

	result, err := server.store.CreateBranchTx(ctx, db.CreateBranchTxParams{
		Name:               req.Name,
		Code:               req.Code,
		RegisterAPIKeyHash: apiKeyHash,
	})
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	server.writeJSON(ctx, http.StatusOK, envelope{
		"branch":   newBranchResponse(result.Branch),
		"register": newBranchRegisterResponse(result.Register),
		"api_key":  apiKey,
	})
}

func (server *Server) listBranches(ctx *gin.Context) {
	branches, err := server.store.ListBranches(ctx)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	res := make([]branchResponse, len(branches))
	for i, branch := range branches {
		res[i] = newBranchResponse(branch)
	}

	server.writeJSON(ctx, http.StatusOK, envelope{"branches": res})
}

// listConnectedBranches gives the admin UI an initial value for "which
// branches are live right now" on load -- the live figure after that comes
// from the "branch_connection_changed" push over the admin WebSocket (see
// branchWS in branch_ws.go), same poll-then-push pattern SyncCard already
// uses for recent invoices.
func (server *Server) listConnectedBranches(ctx *gin.Context) {
	server.writeJSON(ctx, http.StatusOK, envelope{"branch_ids": server.branchHub.connectedBranchIDs()})
}

// listSyncingBranches gives the admin UI an initial value for "which
// branches are mid a full sync right now" on load, same poll-then-push
// pairing with "branch_sync_status_changed" that listConnectedBranches has
// with "branch_connection_changed".
func (server *Server) listSyncingBranches(ctx *gin.Context) {
	server.writeJSON(ctx, http.StatusOK, envelope{"branch_ids": server.branchHub.syncingBranchIDs(time.Now())})
}

type branchIDRequest struct {
	ID int64 `uri:"id" binding:"required,min=1"`
}

func (server *Server) setBranchActive(active bool) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var req branchIDRequest
		if err := ctx.ShouldBindUri(&req); err != nil {
			server.writeError(ctx, http.StatusBadRequest, err)
			return
		}

		if _, err := server.store.GetBranch(ctx, req.ID); err != nil {
			if err == sql.ErrNoRows {
				server.writeError(ctx, http.StatusNotFound, err)
				return
			}
			server.writeError(ctx, http.StatusInternalServerError, err)
			return
		}

		if err := server.store.SetBranchActive(ctx, db.SetBranchActiveParams{
			ID:       req.ID,
			IsActive: active,
		}); err != nil {
			server.writeError(ctx, http.StatusInternalServerError, err)
			return
		}

		server.writeJSON(ctx, http.StatusOK, envelope{"id": req.ID, "is_active": active})
	}
}
