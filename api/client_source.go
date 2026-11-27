package api

import (
	"database/sql"
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	db "github.com/mhdna/kashi/db/sqlc"
)

// Client sources are the answers to "where did you hear about us?", asked
// once at the till when a cashier adds a new customer. The list lives here
// and is synced whole to every till (entity "client_sources"); the till sends
// the picked option's uid with the client (branchClientRequest.SourceUID) and
// the first answer a client gets is kept.

var errClientSourceName = errors.New("enter a name for the option")
var errClientSourceInUse = errors.New("customers gave this answer, so it can't be deleted; hide it instead")

type clientSourceRequest struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	SortOrder int32  `json:"sort_order"`
	IsActive  bool   `json:"is_active"`
}

func (server *Server) listClientSources(ctx *gin.Context) {
	rows, err := server.store.ListClientSources(ctx)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	server.writeJSON(ctx, http.StatusOK, envelope{"client_sources": rows})
}

func (server *Server) createClientSource(ctx *gin.Context) {
	var req clientSourceRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		server.writeError(ctx, http.StatusBadRequest, errClientSourceName)
		return
	}
	row, err := server.store.CreateClientSource(ctx, db.CreateClientSourceParams{
		Name: name, SortOrder: req.SortOrder, IsActive: req.IsActive,
	})
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	server.branchHub.broadcastAll(branchWSMessage{Type: "client_sources_updated"})
	server.writeJSON(ctx, http.StatusOK, envelope{"client_source": row})
}

func (server *Server) updateClientSource(ctx *gin.Context) {
	var req clientSourceRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	name := strings.TrimSpace(req.Name)
	if req.ID <= 0 || name == "" {
		server.writeError(ctx, http.StatusBadRequest, errClientSourceName)
		return
	}
	row, err := server.store.UpdateClientSource(ctx, db.UpdateClientSourceParams{
		ID: req.ID, Name: name, SortOrder: req.SortOrder, IsActive: req.IsActive,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			server.writeError(ctx, http.StatusNotFound, err)
			return
		}
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	server.branchHub.broadcastAll(branchWSMessage{Type: "client_sources_updated"})
	server.writeJSON(ctx, http.StatusOK, envelope{"client_source": row})
}

// deleteClientSource removes an option no customer gave. One that customers
// gave is refused (409): deleting it would lose their answers, so it is
// hidden instead.
func (server *Server) deleteClientSource(ctx *gin.Context) {
	var req struct {
		ID int64 `uri:"id" binding:"required,min=1"`
	}
	if err := ctx.ShouldBindUri(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	used, err := server.store.CountClientsWithSource(ctx, sql.NullInt64{Int64: req.ID, Valid: true})
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	if used > 0 {
		server.writeError(ctx, http.StatusConflict, errClientSourceInUse)
		return
	}
	if err := server.store.DeleteClientSource(ctx, req.ID); err != nil {
		if isForeignKeyViolation(err) {
			server.writeError(ctx, http.StatusConflict, errClientSourceInUse)
			return
		}
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	server.branchHub.broadcastAll(branchWSMessage{Type: "client_sources_updated"})
	server.writeJSON(ctx, http.StatusOK, envelope{"deleted": true})
}

// keepTillClientSource records the source the till sent for this client,
// unless the client already has one. An unknown or malformed uid (an option
// deleted meanwhile) is ignored: the client still syncs.
func (server *Server) keepTillClientSource(ctx *gin.Context, outcome branchClientOutcome, req branchClientRequest) {
	if outcome.client.ID == 0 || outcome.client.SourceID.Valid {
		return
	}
	uid, err := uuid.Parse(strings.TrimSpace(req.SourceUID))
	if err != nil {
		return
	}
	if _, err := server.store.SetClientSourceFromTill(ctx, db.SetClientSourceFromTillParams{
		SourceUid: uid, ClientID: outcome.client.ID,
	}); err != nil {
		log.Printf("keep till client source for client %d: %v", outcome.client.ID, err)
	}
}

// clientHeardFrom is the option a client gave when they were added at the
// till, for their profile; blank when nobody answered.
func (server *Server) clientHeardFrom(ctx *gin.Context, client db.Client) string {
	if !client.SourceID.Valid {
		return ""
	}
	name, err := server.store.GetClientSourceName(ctx, client.SourceID.Int64)
	if err != nil {
		return ""
	}
	return name
}
