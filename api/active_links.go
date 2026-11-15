package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	db "github.com/mhdna/kashi/db/sqlc"
)

// activeLink is one phone link a till handed out that still works: the QR a
// cashier shows for an expense's photos, a shift receipt, a transfer, a loan
// or an unknown item's pictures. The home page lists them so a manager can
// open or share one without the till in front of them.
type activeLink struct {
	Kind         string    `json:"kind"`
	Path         string    `json:"path"`
	BranchID     int64     `json:"branch_id"`
	BranchName   string    `json:"branch_name"`
	RefID        int64     `json:"ref_id"`
	Label        string    `json:"label"`
	Amount       int64     `json:"amount"`
	CurrencyCode string    `json:"currency_code"`
	Units        int64     `json:"units"`
	Done         int64     `json:"done"`
	CreatedAt    time.Time `json:"created_at"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// activeLinkKinds maps each kind to the short path its link uses (the one
// the till prints as a QR, see server.go) and the section a user needs to see
// it, the same section as the page that kind belongs to.
var activeLinkKinds = map[string]struct{ letter, section string }{
	"expense":  {"e", "accounting"},
	"loan":     {"l", "accounting"},
	"shift":    {"s", "accounting"},
	"transfer": {"t", "inventory"},
	"unknown":  {"u", "sales"},
}

// activeLinksFor keeps the rows this user may see and gives each its path.
func activeLinksFor(rows []db.ListActiveUploadLinksRow, canView func(section string) bool) []activeLink {
	links := make([]activeLink, 0, len(rows))
	for _, row := range rows {
		kind, ok := activeLinkKinds[row.Kind]
		if !ok || !canView(kind.section) {
			continue
		}
		links = append(links, activeLink{
			Kind:         row.Kind,
			Path:         "/" + kind.letter + "/" + row.Token,
			BranchID:     row.BranchID,
			BranchName:   row.BranchName,
			RefID:        row.RefID,
			Label:        row.Label,
			Amount:       row.Amount,
			CurrencyCode: row.CurrencyCode,
			Units:        row.Units,
			Done:         row.Done,
			CreatedAt:    row.CreatedAt,
			ExpiresAt:    row.ExpiresAt,
		})
	}
	return links
}

func (server *Server) listActiveLinks(ctx *gin.Context) {
	rows, err := server.store.ListActiveUploadLinks(ctx)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	access, _ := currentAccess(ctx)
	server.writeJSON(ctx, http.StatusOK, envelope{"links": activeLinksFor(rows, access.canView)})
}
