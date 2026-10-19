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
// Every user has one role. A role grants, per *page*, either nothing, view
// (read-only) or edit (view + add/change/delete). Pages are grouped into
// *sections*, and a section can be switched off as a whole, which hides all
// of its pages without losing their individual levels. A role flagged
// is_admin skips all of this and is the only kind that can manage users and
// roles.
//
// Enforcement is one middleware (permissionMiddleware) rather than a check in
// every handler: it maps the matched route pattern to a section with
// routeRules below and derives view-vs-edit from the HTTP method. A route
// that isn't listed there is admin-only, so forgetting to classify a new
// route fails closed.
//
// The API only knows sections, because most endpoints feed several pages
// (/products backs Products, Barcodes, Inventory, ...). A role's level in a
// section is the highest level it has on any of that section's pages, so the
// per-page split is enforced by the UI (router guard, nav, usePageAccess) and
// the section boundary by the API.

// accessSection is one tab in the role editor, accessPage one row in it. Keep
// the keys in sync with `meta.section` / `meta.page` in ui/src/router/index.js
// -- that is what hides pages and buttons; this file is what actually
// protects the data. Page keys are unique across sections.
type accessSection struct {
	Key         string       `json:"key"`
	Label       string       `json:"label"`
	Description string       `json:"description"`
	Pages       []accessPage `json:"pages"`
}

type accessPage struct {
	Key   string `json:"key"`
	Label string `json:"label"`
}

var accessSections = []accessSection{
	{"analytics", "Analytics", "Dashboards and reports", []accessPage{
		{"dashboard", "Dashboard"},
		{"daily-income", "Daily Income"},
		{"financials", "Financials"},
		{"targets", "Targets"},
		{"footfall", "Footfall"},
		{"busy-hours", "Busy Hours"},
		{"product-revenues", "Product Revenues"},
		{"conversion", "Conversion"},
		{"sales-analytics", "Sales Analytics"},
		{"sales-analysis", "Sales Analysis"},
		{"sales-dashboard", "Sales Dashboard"},
		{"salesperson-performance", "Salesperson Performance"},
	}},
	{"sales", "Sales", "Invoices", []accessPage{
		{"invoices", "Invoices"},
		{"unknown-items", "Unknown Items"},
	}},
	{"clients", "Clients", "Clients, loyalty and reviews", []accessPage{
		{"clients", "Clients"},
		{"loyalty", "Loyalty"},
		{"reviews", "Reviews"},
	}},
	{"branches", "Branches", "Branches, branch settings and sync conflicts", []accessPage{
		{"branches", "Branches"},
		{"branch-settings", "Branch Settings"},
		{"default-branch-settings", "Default Settings"},
		{"branch-commands", "Commands"},
		{"branch-conflicts", "Conflicts"},
	}},
	{"pricing", "Pricing", "Price lists, discount lists and coupons", []accessPage{
		{"pricing", "Coupons"},
		{"price-lists", "Price Lists"},
		{"discount-lists", "Discount Lists"},
	}},
	{"products", "Products", "Products, colors, sizes, attributes and barcodes", []accessPage{
		{"products", "Products"},
		{"colors-sizes", "Colors & Sizes"},
		{"attributes", "Attributes"},
		{"autofill-rules", "Auto-fill Rules"},
		{"barcodes", "Barcodes"},
		{"barcode-printing", "Barcode Printing"},
		{"barcode-designer", "Barcode Designer"},
	}},
	{"inventory", "Inventory", "Stock, purchases, transfers, batches and suppliers", []accessPage{
		{"inventory", "Inventory"},
		{"alerts", "Alerts"},
		{"stock-health", "Stock Health"},
		{"inventory-analysis", "Inventory Analysis"},
		{"transfers", "Transfers"},
		{"stock-movements", "Stock Movements"},
		{"stock-counts", "Stock Counts"},
		{"purchase-invoice", "Purchase Invoices"},
		{"batches", "Batches"},
		{"suppliers", "Suppliers"},
	}},
	{"accounting", "Accounting", "Expenses, loans, currencies, cashboxes and assets", []accessPage{
		{"expenses", "Expenses"},
		{"loans", "Loans"},
		{"currencies", "Currencies & Payment Methods"},
		{"assets", "Assets"},
		{"assets-treemap", "Assets Treemap"},
		{"warehouses-treemap", "Warehouses Treemap"},
	}},
	{"hr", "Human resources", "Employees, attendance, shifts and salespersons", []accessPage{
		{"employees", "Employees"},
		{"attendance", "Attendance"},
		{"shifts", "Shifts"},
	}},
	{"calendar", "Calendar", "Calendar events", []accessPage{
		{"calendar", "Calendar"},
	}},
	{"system", "System", "App settings, WhatsApp, emails, audit log, dev logs and storage", []accessPage{
		{"settings", "App Settings & WhatsApp"},
		{"whatsapp", "WhatsApp"},
		{"emails", "Emails"},
		{"audit-log", "Audit Log"},
		{"logs", "Dev Logs"},
		{"storage", "Storage"},
	}},
	// Own sections rather than pages under System: the API enforces sections,
	// so editing App Settings must not also let a role push releases to the
	// tills or ban IPs.
	{"updates", "App Updates", "Upload till releases and roll them out to branches", []accessPage{
		{"app-updates", "App Updates"},
	}},
	{"security", "Security", "Visitors, alerts and IP bans", []accessPage{
		{"security", "Security"},
	}},
	// No pages: switching the section on is the whole grant.
	{"photo_rotation", "Rotate Photos", "Turn expense, shift-receipt and transfer photos; the turn is saved for everyone (off by default)", nil},
	{"photo_replace", "Replace Photos", "Swap an expense, shift-receipt or transfer photo for another one; the old photo is deleted (off by default)", nil},
	// Checked by createBranchCommand (replace_shift_balance with
	// allow_negative), not by a route rule.
	{"shift_negative_balance", "Negative Balance Transfers", "Move a till's shift balance out of a payment method even when that takes it below zero, e.g. LBP change that was really settled by Whish (off by default)", nil},
	// Checked by the /home/activities/resolve route rule.
	{"resolve_urgent_activity", "Resolve Urgent Activity", "Mark an urgent Recent activity row on Home (a shift closed short) as resolved, which unpins it for everyone (off by default)", nil},
	// Checked by the /unknown_items/waive route rule.
	{"unknown_item_waive", "Allow Unknown Items Without Picture", "Let a till sell an unknown item with no picture taken (the Waiting tab on Unknown items), e.g. the cashier sent it privately (off by default)", nil},
	{"text_selection", "Text Selection", "Select and copy text anywhere in the admin interface (off by default; doesn't gate any data)", nil},
}

func findAccessSection(key string) (accessSection, bool) {
	for _, s := range accessSections {
		if s.Key == key {
			return s, true
		}
	}
	return accessSection{}, false
}

func isAccessSection(key string) bool {
	_, ok := findAccessSection(key)
	return ok
}

// sectionOfPage returns the section a page key belongs to.
func sectionOfPage(page string) (string, bool) {
	for _, s := range accessSections {
		for _, p := range s.Pages {
			if p.Key == page {
				return s.Key, true
			}
		}
	}
	return "", false
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
	rule("/transfer_attachments", "inventory", "inventory"),
	rule("/transfers/:id/attachment/rotation", "photo_rotation"),
	rule("/transfers/:id/attachment/replace", "photo_replace"),
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
	rule("/branch_commands", "branches", "branches"),
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
	rule("/sales_dashboard", "", "analytics"),
	rule("/busy_hours", "", "analytics"),
	rule("/sales_analysis", "", "analytics"),
	rule("/salesperson_performance", "", "analytics"),
	rule("/financials", "", "analytics"),
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
	rule("/unknown_items", "sales", "sales", "analytics"),
	rule("/unknown_items/waive", "unknown_item_waive"),
	rule("/unknown_items/images/delete", "photo_replace"),
	rule("/branch_expense_images", "accounting", "accounting", "analytics"),
	rule("/branch_expense_images/:id/rotation", "photo_rotation"),
	rule("/branch_expense_images/:id/replace", "photo_replace"),
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
	rule("/branch_shift_receipt_images/:id/rotation", "photo_rotation"),
	rule("/branch_shift_receipt_images/:id/replace", "photo_replace"),

	// Calendar, system
	lookup("/calendar_events", "calendar"),
	lookup("/app_settings", "system"),
	lookup("/app_settings/default_low_stock_threshold", "inventory"),
	// Auto-fill rules are edited from the Products section's own page.
	lookup("/app_settings/product_autofill_rules", "products"),
	lookup("/pos_settings", "system"),
	rule("/whatsapp_settings", "system", "system"),
	// The WhatsApp page: rules, message log and stats.
	rule("/whatsapp", "system", "system"),
	// The Reviews page; its settings hold the Places API key, so changing
	// them is System's.
	rule("/reviews", "clients", "clients"),
	rule("/reviews/settings", "system", "clients", "system"),
	rule("/email_settings", "system", "system", "analytics"),
	// Sending an invoice report is done from the Daily Income pages; its
	// history is the Emails page under System.
	rule("/email_reports", "analytics", "analytics", "system"),
	// Scheduled reports send as whoever set them up could; same rights.
	rule("/email_schedules", "analytics", "analytics", "system"),
	rule("/audit_logs", "", "system"),
	// Who did what in kashi (home page feed). Read-only; written by the
	// activity middleware, never through this route.
	rule("/admin_activities", "", "system"),
	// Home page "Latest added": open to everyone signed in; the handler
	// filters each kind by the user's own section access.
	lookup("/home/latest", ""),
	// The home feeds' "today" counts: same deal, filtered per feed.
	lookup("/home/today", ""),
	// The home page's Recent activity feed: branch sync, same access as the
	// dashboard it was split out of.
	rule("/home/activities", "", "analytics"),
	// Unpinning an urgent row is its own permission, not an analytics edit.
	rule("/home/activities/resolve", "resolve_urgent_activity", "analytics"),
	rule("/logs", "", "system"),
	rule("/app_releases", "updates", "updates"),
	rule("/app_updates", "updates", "updates"),
	rule("/security", "security", "security"),

	// Signed-in users may see who else is online and their own access; every
	// other /users route (and all of /roles) is admin-only by omission.
	lookup("/users/online", ""),
	lookup("/users/me", ""),
}

// POST routes that only read (they use POST for a request body), so a
// read-only user still gets them.
var readOnlyPosts = map[string]bool{
	"/barcodes/print":          true,
	"/barcodes/design/preview": true,
	"/whatsapp/rules/test":     true,
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

// sectionLevel is what a role grants on one page (or, derived, in one
// section). Edit implies view.
type sectionLevel struct {
	View bool `json:"view"`
	Edit bool `json:"edit"`
}

func (l sectionLevel) max(o sectionLevel) sectionLevel {
	return sectionLevel{View: l.View || o.View, Edit: l.Edit || o.Edit}
}

// permissionRow is one stored role_permissions row. Page "" is the section
// row, whose presence means the section is switched on.
type permissionRow struct {
	Section string
	Page    string
	Level   sectionLevel
}

// rolePermissions is a role's grants as the role editor sees them: which
// sections are on, and every page's level whether or not its section is on
// (so switching a section off and on again keeps its pages as they were).
type rolePermissions struct {
	Sections map[string]bool         `json:"sections"`
	Pages    map[string]sectionLevel `json:"pages"`
}

// resolvePermissions reads stored rows into rolePermissions. A section with
// no page rows at all was saved before per-page access existed; its pages
// inherit the section row's level, which is exactly what that role granted.
// Once a section has page rows, a page without one (added to the app since
// the role was saved) gets nothing.
func resolvePermissions(rows []permissionRow) rolePermissions {
	sectionRows := map[string]sectionLevel{}
	pageRows := map[string]map[string]sectionLevel{}
	for _, r := range rows {
		level := sectionLevel{View: r.Level.View || r.Level.Edit, Edit: r.Level.Edit}
		if r.Page == "" {
			sectionRows[r.Section] = level
			continue
		}
		if pageRows[r.Section] == nil {
			pageRows[r.Section] = map[string]sectionLevel{}
		}
		pageRows[r.Section][r.Page] = level
	}

	perms := rolePermissions{Sections: map[string]bool{}, Pages: map[string]sectionLevel{}}
	for _, s := range accessSections {
		sectionLvl, on := sectionRows[s.Key]
		if on {
			perms.Sections[s.Key] = true
		}
		legacy := len(pageRows[s.Key]) == 0
		for _, p := range s.Pages {
			if legacy {
				perms.Pages[p.Key] = sectionLvl
			} else {
				perms.Pages[p.Key] = pageRows[s.Key][p.Key]
			}
		}
	}
	return perms
}

// effective is what the role actually grants: pages of switched-off sections
// count for nothing, and a section's level is the highest of its pages (or,
// for a section without pages, view and edit both, simply by being on).
func (p rolePermissions) effective() (sections, pages map[string]sectionLevel) {
	sections = map[string]sectionLevel{}
	pages = map[string]sectionLevel{}
	for _, s := range accessSections {
		if !p.Sections[s.Key] {
			continue
		}
		if len(s.Pages) == 0 {
			sections[s.Key] = sectionLevel{View: true, Edit: true}
			continue
		}
		var level sectionLevel
		for _, pg := range s.Pages {
			if l := p.Pages[pg.Key]; l.View {
				pages[pg.Key] = l
				level = level.max(l)
			}
		}
		if level.View {
			sections[s.Key] = level
		}
	}
	return sections, pages
}

// userAccess is a user's effective access, loaded from their role on each
// request (one indexed query) so a role change or a deleted user takes effect
// immediately instead of when their token next renews. Sections is what the
// API enforces; Pages is what the UI hides pages and controls by.
type userAccess struct {
	RoleID   int64                   `json:"role_id"`
	RoleName string                  `json:"role_name"`
	IsAdmin  bool                    `json:"is_admin"`
	Sections map[string]sectionLevel `json:"sections"`
	Pages    map[string]sectionLevel `json:"pages"`
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
	}
	perms := make([]permissionRow, 0, len(rows))
	for _, row := range rows {
		if !row.Section.Valid {
			continue
		}
		perms = append(perms, permissionRow{
			Section: row.Section.String,
			Page:    row.Page.String,
			Level:   sectionLevel{View: row.CanView.Bool, Edit: row.CanEdit.Bool},
		})
	}
	access.Sections, access.Pages = resolvePermissions(perms).effective()
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
		// The admin UI recognises this wording to keep it out of its error
		// toasts (isPermissionDenied in ui/src/composables/useApi.js).
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
