package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	db "github.com/mhdna/kashi/db/sqlc"
)

// listLatestAdditions backs the home page's "Latest added" feed: newest
// photos, transfers, expenses and purchases in one list. Open to every
// signed-in user, but each kind is included only when their role can view
// the section it belongs to, so nobody sees rows from a page they can't open.
func (server *Server) listLatestAdditions(ctx *gin.Context) {
	var req struct {
		PageSize int32 `form:"page_size,default=20" binding:"min=1,max=100"`
	}
	if err := ctx.ShouldBindQuery(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	access, _ := currentAccess(ctx)

	rows, err := server.store.ListLatestAdditions(ctx, db.ListLatestAdditionsParams{
		IncludeAccounting: access.canView("accounting"),
		IncludeHr:         access.canView("hr"),
		IncludeInventory:  access.canView("inventory"),
		PageSize:          req.PageSize,
	})
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	server.writeJSON(ctx, http.StatusOK, envelope{"items": rows})
}
