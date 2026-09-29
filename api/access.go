package api

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/mhdna/kashi/token"
)

// Access control for admin-site users.
//
// Every user has one role. A role grants, per *section* (a group of related
// pages), either nothing, view (read-only) or edit (view + add/change/delete).
// A role flagged is_admin skips all of this and is the only kind that can
// manage users and roles.
//
// Enforcement is one middleware (permissionMiddleware) rather than a check in
// every handler: it maps the matched route pattern to a section with
// routeRules below and derives view-vs-edit from the HTTP method. A route
// that isn't listed there is admin-only, so forgetting to classify a new
// route fails closed.

// accessSection is one row in the role editor. Keep the keys in sync with
// `meta.section` in ui/src/router/index.js -- that is what hides pages and
// buttons; this file is what actually protects the data.
type accessSection struct {
	Key         string `json:"key"`
	Label       string `json:"label"`
	Description string `json:"description"`
}

var accessSections = []accessSection{
	{"analytics", "Analytics", "Dashboard, income, financials, targets, footfall, sales analysis and other reports"},
	{"sales", "Sales", "Invoices"},
	{"clients", "Clients", "Clients and loyalty"},
	{"branches", "Branches", "Branches, branch settings and sync conflicts"},
	{"pricing", "Pricing", "Price lists, discount lists and coupons"},
	{"products", "Products", "Products, colors, sizes, attributes and barcodes"},
	{"inventory", "Inventory", "Inventory, stock, purchases, transfers, batches and suppliers"},
	{"accounting", "Accounting", "Expenses, loans, currencies, cashboxes and assets"},
	{"hr", "Human resources", "Employees, attendance, shifts and salespersons"},
	{"calendar", "Calendar", "Calendar events"},
	{"system", "System", "App settings, WhatsApp, audit log, dev logs and storage"},
}

func isAccessSection(key string) bool {
	for _, s := range accessSections {
		if s.Key == key {
			return true
		}
	}
	return false
}

// routeRule says who may call the routes under prefix. Writes (any method
// other than GET/HEAD) need edit on `write`; an empty `write` makes them
// admin-only. Reads are open to every signed-in user when `open` is set (shared
// lookups that many pages fill dropdowns from), otherwise they need view on
// any one of `read`, so a page that shows data owned by another section can
// list both.
type routeRule struct {
	prefix string
	open   bool
	read   []string
	write  string
}

func lookup(prefix, write string) routeRule {
	return routeRule{prefix: prefix, open: true, write: write}
}
func rule(prefix, write string, read ...string) routeRule {
	return routeRule{prefix: prefix, write: write, read: read}
}

var routeRules = []routeRule{
	// Products
	lookup("/products", "products"),
	lookup("/product_variants", "products"),
	lookup("/colors", "products"),
	lookup("/sizes", "products"),
	lookup("/attributes", "products"),
	lookup("/attribute_types", "products"),
	lookup("/attribute_values", "products"),
	rule("/barcodes", "products", "products", "inventory"),
	// Low-stock thresholds and mutes are set from the Alerts page, which is
	// part of Inventory.
	lookup("/products/:id/low_stock_threshold", "inventory"),
	lookup("/products/:id/low_stock_alerts", "inventory"),

	// Inventory
	lookup("/inventories", "inventory"),
	rule("/inventories/:id/stock", "inventory", "inventory"),
	rule("/stock_movements", "inventory", "inventory"),
	rule("/low_stock", "inventory", "inventory"),
	rule("/stock_health", "inventory", "inventory"),
	rule("/stock_counts", "inventory", "inventory"),
	rule("/batches", "inventory", "inventory", "products"),
	rule("/transfers", "inventory", "inventory"),
	rule("/transfer_items", "inventory", "inventory"),
	rule("/purchases", "inventory", "inventory"),
	rule("/purchase_items", "inventory", "inventory"),
	rule("/suppliers", "inventory", "inventory", "accounting"),

	// Sales and clients
	rule("/sales_invoices", "sales", "sales", "analytics"),
	rule("/return_invoices", "sales", "sales", "analytics"),
	rule("/invoices", "sales", "sales", "analytics"),
	rule("/branch_invoices", "sales", "sales", "analytics", "clients"),
	rule("/branch_invoice_settlements", "sales", "sales", "analytics", "accounting"),
	lookup("/invoice_types", "sales"),
	rule("/clients", "clients", "clients", "sales", "analytics"),

	// Pricing
	rule("/price_lists", "pricing", "pricing", "products", "sales"),
	rule("/discount_lists", "pricing", "pricing", "products", "sales"),
	rule("/coupons", "pricing", "pricing", "clients"),
	rule("/coupon_categories", "pricing", "pricing", "clients"),

	// Branches. The longer /branches/:id/... prefixes below win over this.
	lookup("/branches", "branches"),
	lookup("/branches/connected", ""),
	rule("/branches/:id/settings", "branches", "branches"),
	rule("/branches/:id/commands", "branches", "branches"),
	rule("/branches/:id/branch_users", "branches", "branches"),
	rule("/branch_users", "branches", "branches"),
	rule("/branch_sync_conflicts", "branches", "branches"),
	// Targets and salespersons are managed from the Branches page, so writing
	// them takes Branches edit; the Analytics/HR pages only read them.
	rule("/branches/:id/targets", "branches", "branches", "analytics"),
	rule("/branches/:id/target_series", "branches", "branches", "analytics"),
	rule("/branch_targets", "branches", "branches", "analytics"),
	rule("/target_series", "branches", "branches", "analytics"),
	rule("/branches/:id/salespersons", "branches", "branches", "hr", "analytics", "sales"),
	rule("/salespersons", "branches", "branches", "hr", "analytics", "sales"),

	// Analytics
	rule("/dashboard", "", "analytics"),
	rule("/branch_sales_stats", "", "analytics", "sales"),
	rule("/branch_product_revenue", "", "analytics", "inventory"),
	rule("/product_revenue", "", "analytics", "inventory"),
	rule("/branch_visitor_events", "", "analytics"),
	rule("/branch_visitor_stats", "", "analytics"),

	// Accounting
	rule("/expenses", "accounting", "accounting", "analytics"),
	rule("/expense_categories", "accounting", "accounting", "analytics"),
	rule("/recurring_expenses", "accounting", "accounting"),
	rule("/branch_expenses", "accounting", "accounting", "analytics"),
	rule("/branch_expense_images", "accounting", "accounting", "analytics"),
	rule("/loans", "accounting", "accounting", "analytics"),
	rule("/loan_categories", "accounting", "accounting", "analytics"),
	rule("/loan_payments", "accounting", "accounting", "analytics"),
	rule("/entries", "accounting", "accounting"),
	lookup("/currencies", "accounting"),
	rule("/cashboxes", "accounting", "accounting", "sales"),
	rule("/cashbox_accounts", "accounting", "accounting", "sales"),
	rule("/assets", "accounting", "accounting"),
	rule("/asset_categories", "accounting", "accounting"),

	// Human resources
	rule("/employees", "hr", "hr", "analytics"),
	rule("/branch_attendance_events", "hr", "hr"),
	rule("/branch_attendance_changes", "hr", "hr"),
	rule("/shifts", "hr", "hr", "analytics", "accounting"),
	rule("/branch_shifts", "hr", "hr", "analytics", "accounting"),
	rule("/branch_shift_receipt_images", "hr", "hr", "analytics", "accounting"),

	// Calendar, system
	lookup("/calendar_events", "calendar"),
	lookup("/app_settings", "system"),
	lookup("/app_settings/default_low_stock_threshold", "inventory"),
	lookup("/pos_settings", "system"),
	rule("/whatsapp_settings", "system", "system"),
	rule("/audit_logs", "", "system"),
	rule("/logs", "", "system"),

	// Signed-in users may see who else is online and their own access; every
	// other /users route (and all of /roles) is admin-only by omission.
	lookup("/users/online", ""),
	lookup("/users/me", ""),
}

// POST routes that only read (they use POST for a request body), so a
// read-only user still gets them.
var readOnlyPosts = map[string]bool{
	"/barcodes/print": true,
}

// ruleFor returns the most specific rule for a gin route pattern such as
// "/branches/:id/targets", matching whole path segments.
func ruleFor(fullPath string) (routeRule, bool) {
	path := strings.TrimSuffix(fullPath, "/")
	var best routeRule
	found := false
	for _, r := range routeRules {
		if path != r.prefix && !strings.HasPrefix(path, r.prefix+"/") {
			continue
		}
		if !found || len(r.prefix) > len(best.prefix) {
			best, found = r, true
		}
	}
	return best, found
}

// sectionLevel is what a role grants in one section. Edit implies view.
type sectionLevel struct {
	View bool `json:"view"`
	Edit bool `json:"edit"`
}

// userAccess is a user's effective access, loaded from their role on each
// request (one indexed query) so a role change or a deleted user takes effect
// immediately instead of when their token next renews.
type userAccess struct {
	RoleID   int64                   `json:"role_id"`
	RoleName string                  `json:"role_name"`
	IsAdmin  bool                    `json:"is_admin"`
	Sections map[string]sectionLevel `json:"sections"`
}

var errNoSuchUser = errors.New("user no longer exists")

func (server *Server) loadUserAccess(ctx *gin.Context, userID int64) (userAccess, error) {
	rows, err := server.store.GetUserAccess(ctx, userID)
	if err != nil {
		return userAccess{}, err
	}
	if len(rows) == 0 {
		return userAccess{}, errNoSuchUser
	}
	access := userAccess{
		RoleID:   rows[0].RoleID,
		RoleName: rows[0].RoleName,
		IsAdmin:  rows[0].IsAdmin,
		Sections: map[string]sectionLevel{},
	}
	for _, row := range rows {
		if !row.Section.Valid {
			continue
		}
		edit := row.CanEdit.Bool
		access.Sections[row.Section.String] = sectionLevel{View: row.CanView.Bool || edit, Edit: edit}
	}
	return access, nil
}

func (a userAccess) canEdit(section string) bool { return a.IsAdmin || a.Sections[section].Edit }
func (a userAccess) canView(section string) bool {
	return a.IsAdmin || a.Sections[section].View || a.Sections[section].Edit
}

func (r routeRule) permits(a userAccess, write bool) bool {
	if a.IsAdmin {
		return true
	}
	if write {
		return r.write != "" && a.canEdit(r.write)
	}
	if r.open {
		return true
	}
	for _, section := range r.read {
		if a.canView(section) {
			return true
		}
	}
	return false
}

// permissionMiddleware must run after authMiddleware (it needs the token
// payload).
func (server *Server) permissionMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		payload, ok := ctx.MustGet(authoizationPayloadKey).(*token.Payload)
		if !ok {
			server.abortError(ctx, http.StatusUnauthorized, errors.New("invalid token payload"))
			return
		}

		access, err := server.loadUserAccess(ctx, payload.UserID)
		if err != nil {
			if errors.Is(err, errNoSuchUser) || err == sql.ErrNoRows {
				server.abortError(ctx, http.StatusUnauthorized, errNoSuchUser)
				return
			}
			server.abortError(ctx, http.StatusInternalServerError, err)
			return
		}
		ctx.Set(accessKey, access)

		if access.IsAdmin {
			ctx.Next()
			return
		}

		method := ctx.Request.Method
		write := method != http.MethodGet && method != http.MethodHead && !readOnlyPosts[ctx.FullPath()]
		r, found := ruleFor(ctx.FullPath())
		if found && r.permits(access, write) {
			ctx.Next()
			return
		}

		verb := "view"
		if write {
			verb = "change"
		}
		what := "this"
		if found {
			what = r.describe(write)
		}
		server.abortError(ctx, http.StatusForbidden, fmt.Errorf("your role (%s) can't %s %s", access.RoleName, verb, what))
	}
}

// describe names the section(s) a denied request needed, for the 403 message.
func (r routeRule) describe(write bool) string {
	if write {
		if r.write == "" {
			return "this"
		}
		return sectionLabel(r.write)
	}
	labels := make([]string, 0, len(r.read))
	for _, s := range r.read {
		labels = append(labels, sectionLabel(s))
	}
	sort.Strings(labels)
	if len(labels) == 0 {
		return "this"
	}
	return strings.Join(labels, " / ")
}

func sectionLabel(key string) string {
	for _, s := range accessSections {
		if s.Key == key {
			return s.Label
		}
	}
	return key
}

const accessKey = "user_access"

// currentAccess reads what permissionMiddleware stored.
func currentAccess(ctx *gin.Context) (userAccess, bool) {
	value, exists := ctx.Get(accessKey)
	if !exists {
		return userAccess{}, false
	}
	a, ok := value.(userAccess)
	return a, ok
}
