package api

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	db "github.com/mhdna/kashi/db/sqlc"
	"github.com/mhdna/kashi/token"
)

// The home page's "In kashi" feed: who changed what through the admin API.
// Instead of every handler writing its own row, adminActivityMiddleware
// records each successful write after it has run, and describes it from
// its route pattern (describeAdminRoute). A new endpoint shows up in the
// feed with a generic description on its own; adminActivityOverrides is
// only for wording the generic rule gets wrong.

// adminActivityOverrides words routes the generic rule describes badly.
// Keyed "METHOD /route/pattern".
var adminActivityOverrides = map[string]string{
	"POST /branches/:id/commands":                   "sent a command to",
	"PUT /branches/:id/settings/managed_locally":    "changed what's managed locally at",
	"POST /default_branch_settings/apply_to_all":    "applied default settings to every branch",
	"PUT /default_branch_settings":                  "updated default branch settings",
	"PUT /branches/:id/quick_access":                "updated quick access of",
	"PUT /branch_registers/:id/quick_access_mode":   "switched quick access source of register",
	"POST /branch_registers/:id/rotate_key":         "rotated the key of register",
	"PUT /branches/:id/payment_methods/:account_id": "changed a payment method of",
	"POST /cashbox_accounts/balance":                "adjusted a payment method balance",
	"PUT /app_settings":                             "updated app settings",
	"PUT /app_settings/product_autofill_rules":      "updated product auto-fill rules",
	"PUT /whatsapp_settings":                        "updated WhatsApp settings",
	"PUT /whatsapp/rules":                           "updated WhatsApp rules",
	"PUT /whatsapp/cost":                            "updated the WhatsApp message price",
	"POST /barcodes/assign":                         "assigned barcodes",
	"PUT /employees/salaries":                       "updated salaries",
	"POST /branch_sync_conflicts/bulk_resolve":      "resolved several conflicts",
	"POST /inventories/:id/adjustments":             "adjusted stock in inventory",
	"POST /branch_attendance_events/import":         "imported attendance",
	"POST /products/import":                         "imported products",
	"PUT /branch_users/:id/pin":                     "set the PIN of POS user",
	// {branch} is replaced by the branch's name (recordAdminActivity).
	"POST /branches/:id/hide_from_reports":         "hid {branch} from reports",
	"POST /branches/:id/show_in_reports":           "showed {branch} in reports again",
	"POST /branch_registers/:id/hide_from_reports": "hid from reports register",
	"POST /branch_registers/:id/show_in_reports":   "showed in reports again register",
}

// adminResourceNames is the readable singular for a route's first segment
// where dropping a trailing "s" isn't enough.
var adminResourceNames = map[string]string{
	"branch_sync_conflicts":     "conflict",
	"branch_commands":           "command",
	"cashbox_accounts":          "payment method",
	"cashboxes":                 "cashbox",
	"branches":                  "branch",
	"branch_registers":          "register",
	"branch_users":              "POS user",
	"branch_targets":            "target",
	"target_series":             "target series",
	"branch_expenses":           "branch expense",
	"branch_attendance_events":  "attendance punch",
	"categories":                "category",
	"product_categories":        "product category",
	"asset_categories":          "asset category",
	"expense_categories":        "expense category",
	"loan_categories":           "loan category",
	"coupon_categories":         "coupon category",
	"inventories":               "inventory",
	"currencies":                "currency",
	"batches":                   "batch",
	"entries":                   "entry",
	"recurring_expenses":        "recurring expense",
	"calendar_events":           "calendar event",
	"sizes":                     "size",
	"attributes":                "attribute",
	"stock_counts":              "stock count",
	"price_lists":               "price list",
	"discount_lists":            "discount list",
	"coupon_lists":              "coupon list",
	"coupon_types":              "coupon type",
	"invoice_types":             "invoice type",
	"product_variants":          "product variant",
	"transfer_items":            "transfer item",
	"purchase_items":            "purchase item",
	"loan_payments":             "loan payment",
	"sales_invoices":            "sales invoice",
	"return_invoices":           "return invoice",
	"salespersons":              "salesperson",
	"shifts":                    "shift",
	"default_branch_settings":   "default branch settings",
	"branch_attendance_changes": "attendance change",
}

// adminActionVerbs words a route's trailing action segment
// (/transfers/:id/approve) as a past-tense verb.
var adminActionVerbs = map[string]string{
	"approve":     "approved",
	"cancel":      "cancelled",
	"dispatch":    "dispatched",
	"receive":     "received",
	"resolve":     "resolved",
	"dismiss":     "dismissed",
	"activate":    "activated",
	"deactivate":  "deactivated",
	"close":       "closed",
	"post":        "posted",
	"retry":       "retried",
	"bulk_delete": "deleted several",
	"import":      "imported",
	"print":       "printed",
	"assign":      "assigned",
	"active":      "switched on/off",
	"color":       "recoloured",
	"pin":         "set the PIN of",
}

// describeAdminRoute turns a write's method and route pattern into the
// feed's wording, e.g. ("POST", "/transfers/:id/approve") -> "approved
// transfer". Commands on a branch's sub-resource read "added salesperson
// to" / "updated settings of", and the caller appends the branch's name.
func describeAdminRoute(method, route string) string {
	if wording, ok := adminActivityOverrides[method+" "+route]; ok {
		return wording
	}
	var words []string
	for _, seg := range strings.Split(strings.Trim(route, "/"), "/") {
		if seg != "" && !strings.HasPrefix(seg, ":") {
			words = append(words, seg)
		}
	}
	if len(words) == 0 {
		return strings.ToLower(method) + " " + route
	}

	methodVerb := map[string]string{
		http.MethodPost: "created", http.MethodPut: "updated", http.MethodPatch: "updated", http.MethodDelete: "deleted",
	}[method]

	// /branches/:id/<sub> is a branch's own sub-resource.
	if words[0] == "branches" && len(words) >= 2 {
		if verb, ok := adminActionVerbs[words[1]]; ok {
			return verb + " branch" // /branches/:id/activate
		}
		sub := adminResourceName(words[1])
		if len(words) >= 3 {
			if verb, ok := adminActionVerbs[words[2]]; ok {
				return verb + " " + sub + " of"
			}
		}
		if method == http.MethodPost {
			return "added " + sub + " to"
		}
		return methodVerb + " " + sub + " of"
	}

	resource := adminResourceName(words[0])
	if len(words) >= 2 {
		if verb, ok := adminActionVerbs[words[len(words)-1]]; ok {
			return verb + " " + resource
		}
	}
	if methodVerb == "" {
		methodVerb = strings.ToLower(method)
	}
	return methodVerb + " " + resource
}

func adminResourceName(segment string) string {
	if name, ok := adminResourceNames[segment]; ok {
		return name
	}
	name := strings.ReplaceAll(segment, "_", " ")
	switch {
	case strings.HasSuffix(name, "ies"):
		name = strings.TrimSuffix(name, "ies") + "y"
	case strings.HasSuffix(name, "ss"):
	case strings.HasSuffix(name, "s"):
		name = strings.TrimSuffix(name, "s")
	}
	return name
}

// adminDetailKeys are the request fields worth showing beside a change, in
// order of preference. Only these are ever read from a request body, so
// passwords, PINs and keys can never end up in the feed.
var adminDetailKeys = []string{"name", "title", "full_name", "username", "code", "type", "description"}

func adminDetailFromBody(body []byte) string {
	var fields map[string]json.RawMessage
	if json.Unmarshal(body, &fields) != nil {
		return ""
	}
	for _, key := range adminDetailKeys {
		var value string
		if json.Unmarshal(fields[key], &value) != nil {
			continue
		}
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if key == "type" {
			value = strings.ReplaceAll(value, "_", " ")
		}
		if utf8.RuneCountInString(value) > 80 {
			value = string([]rune(value)[:80]) + "…"
		}
		return value
	}
	return ""
}

// adminActivityMiddleware records every successful write made through the
// admin API. It sits after permissionMiddleware, so only allowed requests
// reach it, and records after the handler, so only changes that happened.
func (server *Server) adminActivityMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		method := ctx.Request.Method
		route := ctx.FullPath()
		if method == http.MethodGet || method == http.MethodHead || method == http.MethodOptions ||
			route == "" || readOnlyPosts[route] {
			ctx.Next()
			return
		}

		// Read the JSON body for a detail and put it back for the handler.
		var body []byte
		if strings.HasPrefix(ctx.ContentType(), "application/json") && ctx.Request.Body != nil &&
			ctx.Request.ContentLength >= 0 && ctx.Request.ContentLength <= 1<<20 {
			body, _ = io.ReadAll(io.LimitReader(ctx.Request.Body, 1<<20))
			ctx.Request.Body = io.NopCloser(bytes.NewReader(body))
		}

		ctx.Next()

		if ctx.Writer.Status() >= http.StatusBadRequest || len(ctx.Errors) > 0 {
			return
		}

		var userID int64
		var username string
		if payload, ok := ctx.MustGet(authoizationPayloadKey).(*token.Payload); ok {
			userID, username = payload.UserID, payload.Username
		}
		var entityID sql.NullInt64
		if id, err := strconv.ParseInt(ctx.Param("id"), 10, 64); err == nil {
			entityID = sql.NullInt64{Int64: id, Valid: true}
		}
		summary := describeAdminRoute(method, route)
		detail := adminDetailFromBody(body)
		isBranchRoute := strings.HasPrefix(route, "/branches/:id")

		go server.recordActivity(adminActivity{
			userID: userID, username: username, method: method, route: route,
			entityID: entityID, summary: summary, detail: detail, isBranchRoute: isBranchRoute,
		})
	}
}

// adminActivity is one change for the feed, as the middleware saw it.
type adminActivity struct {
	userID        int64
	username      string
	method        string
	route         string
	entityID      sql.NullInt64
	summary       string
	detail        string
	isBranchRoute bool
}

// recordAdminActivity writes the feed row and nudges open home pages. Runs
// after the response: a failure here is logged, never the request's. It is
// Server.recordActivity unless a test swaps it out.
func (server *Server) recordAdminActivity(a adminActivity) {
	userID, username, method, route := a.userID, a.username, a.method, a.route
	entityID, summary, detail := a.entityID, a.summary, a.detail

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Tokens issued before Payload.UserID existed carry 0.
	if userID == 0 && username != "" {
		if user, err := server.store.GetUserByUsername(ctx, username); err == nil {
			userID = user.ID
		}
	}

	// A branch route reads "updated settings of <branch>": finish the
	// sentence with the branch's name.
	if a.isBranchRoute && entityID.Valid {
		name := "branch #" + strconv.FormatInt(entityID.Int64, 10)
		if branch, err := server.store.GetBranch(ctx, entityID.Int64); err == nil {
			name = branch.Name
		}
		if strings.Contains(summary, "{branch}") {
			summary = strings.ReplaceAll(summary, "{branch}", name)
		} else if strings.HasSuffix(summary, " branch") {
			summary = strings.TrimSuffix(summary, "branch") + "branch " + name
		} else {
			summary += " " + name
		}
		entityID = sql.NullInt64{}
	}

	// The same change again, straight after itself and within the hour,
	// counts up the feed's newest entry instead of adding another.
	user := sql.NullInt64{Int64: userID, Valid: userID > 0}
	_, err := server.store.BumpAdminActivity(ctx, db.BumpAdminActivityParams{
		UserID: user, Method: method, Route: route, EntityID: entityID, Summary: summary, Detail: detail,
	})
	if err == nil {
		server.adminHub.broadcastAll(adminWSMessage{Type: "admin_activity"})
		return
	}
	if !errors.Is(err, sql.ErrNoRows) {
		log.Printf("bump admin activity (%s %s): %v", method, route, err)
	}

	if _, err := server.store.CreateAdminActivity(ctx, db.CreateAdminActivityParams{
		UserID:   user,
		Method:   method,
		Route:    route,
		EntityID: entityID,
		Summary:  summary,
		Detail:   detail,
	}); err != nil {
		log.Printf("record admin activity (%s %s): %v", method, route, err)
		return
	}
	server.adminHub.broadcastAll(adminWSMessage{Type: "admin_activity"})
}

// listAdminActivities backs the home page's "In kashi" feed, newest first,
// page_size rows after the cursor.
func (server *Server) listAdminActivities(ctx *gin.Context) {
	var req struct {
		feedCursor
		PageSize int32 `form:"page_size,default=20" binding:"min=1,max=500"`
	}
	if err := ctx.ShouldBindQuery(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	at, _, id, err := req.args()
	if err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	rows, err := server.store.ListAdminActivities(ctx, db.ListAdminActivitiesParams{
		BeforeAt: at,
		BeforeID: id,
		PageSize: req.PageSize,
	})
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	server.writeJSON(ctx, http.StatusOK, envelope{"activities": rows})
}
