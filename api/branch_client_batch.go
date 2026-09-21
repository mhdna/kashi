package api

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// maxBranchClientsBatch caps one clients_batch call. A "sync everything"
// run can queue tens of thousands of clients; the till sends them in chunks
// of this size.
const maxBranchClientsBatch = 500

type branchClientsBatchRequest struct {
	// Deliberately not validated with `dive`/`required` per item: one bad
	// element must fail only itself (see the per-item "invalid" result), not
	// 400 the whole chunk and strand its 499 valid neighbours.
	Clients []branchClientRequest `json:"clients" binding:"required,min=1"`
}

// Per-item result statuses. The till maps them onto its outbox row:
// ok -> synced, invalid -> failed (retrying the same bytes never helps),
// error -> back on the retry/backoff path.
const (
	batchClientOK      = "ok"
	batchClientInvalid = "invalid"
	batchClientError   = "error"
)

type branchClientBatchResult struct {
	BranchClientID int64  `json:"branch_client_id"`
	Status         string `json:"status"`
	ConflictKind   string `json:"conflict_kind,omitempty"`
	Error          string `json:"error,omitempty"`
}

// putBranchClients is putBranchClient for many clients at once: same
// per-client resolution (see resolveBranchClient), one HTTP round trip, and
// -- the part that matters at scale -- a single "client_updated" broadcast
// per call instead of one per client, which would otherwise make every
// other till re-pull the client list tens of thousands of times during a
// bulk backfill.
//
// The response is always 200 with one result per input, in order; a
// failure for one client never fails the others. Only a malformed or
// oversized envelope is a 400.
func (server *Server) putBranchClients(ctx *gin.Context) {
	branchID := ctx.MustGet(branchIDKey).(int64)

	var req branchClientsBatchRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	if len(req.Clients) > maxBranchClientsBatch {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "too many clients in one batch"})
		return
	}

	results := make([]branchClientBatchResult, len(req.Clients))
	changed := false
	for i, c := range req.Clients {
		res := branchClientBatchResult{BranchClientID: c.BranchClientID}
		switch {
		case c.BranchClientID == 0 || strings.TrimSpace(c.Name) == "":
			res.Status = batchClientInvalid
			res.Error = "branch_client_id and name are required"
		case ctx.Request.Context().Err() != nil:
			// The caller went away mid-batch; report the rest as retryable
			// rather than doing work nobody will read.
			res.Status = batchClientError
			res.Error = "request canceled"
		default:
			outcome, err := server.resolveBranchClient(ctx, branchID, c)
			if err != nil {
				res.Status = batchClientError
				res.Error = err.Error()
			} else {
				res.Status = batchClientOK
				res.ConflictKind = outcome.conflictKind
				changed = true
			}
		}
		results[i] = res
	}

	if changed {
		server.branchHub.broadcastAll(branchWSMessage{Type: "client_updated"})
	}
	server.writeJSON(ctx, http.StatusOK, envelope{"results": results})
}
