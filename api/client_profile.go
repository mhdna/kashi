package api

import (
	"database/sql"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	db "github.com/mhdna/kashi/db/sqlc"
)

// profileQuerier is a minimal interface for the hand-written profile queries
// so the handler can work without modifying the full Store interface (which
// would require updating MockStore in every test).
type profileQuerier interface {
	GetClientProfileStats(ctx *gin.Context, clientID int64) (db.ClientProfileStats, error)
	GetClientHomeBranch(ctx *gin.Context, clientID int64) (string, error)
	ListClientPaymentMix(ctx *gin.Context, clientID int64) ([]db.ClientPaymentMixRow, error)
	ListClientRecentInvoices(ctx *gin.Context, clientID int64) ([]db.ClientRecentInvoiceRow, error)
	ListClientMonthlySpend(ctx *gin.Context, clientID int64) ([]db.ClientMonthlySpendRow, error)
}

// sqlStoreProfileQuerier wraps an *db.SQLStore and satisfies profileQuerier.
type sqlStoreProfileQuerier struct{ s *db.SQLStore }

func (w *sqlStoreProfileQuerier) GetClientProfileStats(ctx *gin.Context, id int64) (db.ClientProfileStats, error) {
	return w.s.GetClientProfileStats(ctx, id)
}
func (w *sqlStoreProfileQuerier) GetClientHomeBranch(ctx *gin.Context, id int64) (string, error) {
	return w.s.GetClientHomeBranch(ctx, id)
}
func (w *sqlStoreProfileQuerier) ListClientPaymentMix(ctx *gin.Context, id int64) ([]db.ClientPaymentMixRow, error) {
	return w.s.ListClientPaymentMix(ctx, id)
}
func (w *sqlStoreProfileQuerier) ListClientRecentInvoices(ctx *gin.Context, id int64) ([]db.ClientRecentInvoiceRow, error) {
	return w.s.ListClientRecentInvoices(ctx, id)
}
func (w *sqlStoreProfileQuerier) ListClientMonthlySpend(ctx *gin.Context, id int64) ([]db.ClientMonthlySpendRow, error) {
	return w.s.ListClientMonthlySpend(ctx, id)
}

func (server *Server) profileStore() (profileQuerier, bool) {
	sqlStore, ok := server.store.(*db.SQLStore)
	if !ok {
		return nil, false
	}
	return &sqlStoreProfileQuerier{s: sqlStore}, true
}

// getClientProfile returns all data needed to render the client profile page:
// aggregate stats, payment mix, recent invoices, and monthly spend series.
func (server *Server) getClientProfile(ctx *gin.Context) {
	var req getClientRequest
	if err := ctx.ShouldBindUri(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}

	client, err := server.store.GetClient(ctx, req.ID)
	if err != nil {
		if err == sql.ErrNoRows {
			server.writeError(ctx, http.StatusNotFound, err)
			return
		}
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	pq, ok := server.profileStore()
	if !ok {
		server.writeError(ctx, http.StatusInternalServerError, fmt.Errorf("profile queries unavailable"))
		return
	}

	stats, err := pq.GetClientProfileStats(ctx, req.ID)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	homeBranch, err := pq.GetClientHomeBranch(ctx, req.ID)
	if err != nil && err != sql.ErrNoRows {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	paymentMix, err := pq.ListClientPaymentMix(ctx, req.ID)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	recentInvoices, err := pq.ListClientRecentInvoices(ctx, req.ID)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	monthlySpend, err := pq.ListClientMonthlySpend(ctx, req.ID)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	// Points redeemed = total earned minus what's still valid (points used
	// to pay for something). Both are kashi's own counters; branch loyalty
	// points are handled separately via the loyalty_total endpoint.
	pointsRedeemed := client.TotalLoyaltyPoints - client.ValidLoyaltyPoints
	if pointsRedeemed < 0 {
		pointsRedeemed = 0
	}

	// Build payment mix list.
	payMixOut := make([]map[string]any, 0, len(paymentMix))
	for _, row := range paymentMix {
		payMixOut = append(payMixOut, map[string]any{
			"name": row.AccountName, "totalCents": row.TotalCents,
		})
	}

	// Build recent invoices list.
	recentOut := make([]map[string]any, 0, len(recentInvoices))
	for _, row := range recentInvoices {
		methods := []string{}
		for _, m := range strings.Split(row.PaymentMethods, ", ") {
			m = strings.TrimSpace(m)
			if m != "" {
				methods = append(methods, m)
			}
		}
		recentOut = append(recentOut, map[string]any{
			"id": row.ID, "invoiceCode": row.InvoiceCode,
			"grandTotal": row.GrandTotal, "occurredAt": row.OccurredAt,
			"branchName": row.BranchName, "itemCount": row.ItemCount,
			"paymentMethods": methods,
		})
	}

	// Build monthly spend list.
	monthlyOut := make([]map[string]any, 0, len(monthlySpend))
	for _, row := range monthlySpend {
		monthlyOut = append(monthlyOut, map[string]any{
			"month": row.Month, "totalCents": row.TotalCents,
		})
	}

	firstPurchase := ""
	if stats.FirstPurchaseAt != nil {
		firstPurchase = stats.FirstPurchaseAt.Format("2006-01-02T15:04:05Z")
	}

	ctx.JSON(http.StatusOK, gin.H{
		"client": gin.H{
			"id": client.ID, "name": client.Name, "phone": client.Phone,
			"uid": client.Uid, "clientType": client.ClientType,
			"totalLoyaltyPoints": client.TotalLoyaltyPoints,
			"validLoyaltyPoints": client.ValidLoyaltyPoints,
		},
		"totalSpentCents": stats.TotalSpentCents,
		"itemsBought":     stats.ItemsBought,
		"pointsRemaining": client.ValidLoyaltyPoints,
		"pointsRedeemed":  pointsRedeemed,
		"homeBranch":      homeBranch,
		"firstPurchaseAt": firstPurchase,
		"paymentMix":      payMixOut,
		"recentInvoices":  recentOut,
		"monthlySpend":    monthlyOut,
	})
}
