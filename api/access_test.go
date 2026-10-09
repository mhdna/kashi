package api

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/golang/mock/gomock"
	mockdb "github.com/mhdna/kashi/db/mock"
	db "github.com/mhdna/kashi/db/sqlc"
	"github.com/mhdna/kashi/util"
	"github.com/stretchr/testify/require"
)

func TestRuleFor(t *testing.T) {
	cases := []struct {
		path     string
		wantOpen bool
		write    string
		found    bool
	}{
		{"/products/:id", true, "products", true},
		{"/inventories/", true, "inventory", true},
		// Longest prefix wins: targets aren't open like the bare /branches lookup.
		{"/branches/:id/targets", false, "branches", true},
		{"/branches/:id/activate", true, "branches", true},
		{"/products/:id/low_stock_threshold", true, "inventory", true},
		{"/app_settings/default_low_stock_threshold", true, "inventory", true},
		// A prefix must match whole segments.
		{"/productsfoo", false, "", false},
		// Unlisted -> admin-only.
		{"/users/:id", false, "", false},
		{"/roles", false, "", false},
	}
	for _, c := range cases {
		r, found := ruleFor(c.path)
		require.Equal(t, c.found, found, c.path)
		if found {
			require.Equal(t, c.wantOpen, r.open, c.path)
			require.Equal(t, c.write, r.write, c.path)
		}
	}
}

func TestRuleSectionsAreKnown(t *testing.T) {
	for _, r := range routeRules {
		if r.write != "" {
			require.True(t, isAccessSection(r.write), "%s writes to unknown section %q", r.prefix, r.write)
		}
		for _, s := range r.read {
			require.True(t, isAccessSection(s), "%s reads unknown section %q", r.prefix, s)
		}
	}
}

func TestPermissionMiddleware(t *testing.T) {
	productsView := []db.GetUserAccessRow{{
		RoleID: 2, RoleName: "Clerk",
		Section: sqlStr("products"), CanView: sqlBool(true), CanEdit: sqlBool(false),
	}}
	productsEdit := []db.GetUserAccessRow{{
		RoleID: 2, RoleName: "Clerk",
		Section: sqlStr("products"), CanView: sqlBool(true), CanEdit: sqlBool(true),
	}}
	noPerms := []db.GetUserAccessRow{{RoleID: 3, RoleName: "Nobody"}}
	accountingEdit := []db.GetUserAccessRow{{
		RoleID: 4, RoleName: "Accountant",
		Section: sqlStr("accounting"), CanView: sqlBool(true), CanEdit: sqlBool(true),
	}}
	// A page-less section is stored as its section row alone.
	photoRotation := []db.GetUserAccessRow{{
		RoleID: 5, RoleName: "Rotator",
		Section: sqlStr("photo_rotation"), CanView: sqlBool(true), CanEdit: sqlBool(false),
	}}

	cases := []struct {
		name   string
		rows   []db.GetUserAccessRow
		method string
		url    string
		want   int
	}{
		{"viewer reads own section", productsView, http.MethodGet, "/inventories/1/stock", http.StatusForbidden},
		{"viewer reads lookup", productsView, http.MethodGet, "/products/1", http.StatusOK},
		{"viewer can't write", productsView, http.MethodDelete, "/products/1", http.StatusForbidden},
		{"editor can write", productsEdit, http.MethodDelete, "/products/1", http.StatusOK},
		{"no role sections, lookup read", noPerms, http.MethodGet, "/products/1", http.StatusOK},
		{"no role sections, gated read", noPerms, http.MethodGet, "/dashboard/summary", http.StatusForbidden},
		{"non-admin can't list users", productsEdit, http.MethodGet, "/users", http.StatusForbidden},
		{"non-admin can't touch roles", productsEdit, http.MethodGet, "/roles", http.StatusForbidden},
		{"non-admin reads own profile", productsView, http.MethodGet, "/users/me", http.StatusOK},
		{"readonly POST allowed for viewer", productsView, http.MethodPost, "/barcodes/print", http.StatusOK},
		{"section editor can't rotate its photos", accountingEdit, http.MethodPut, "/branch_expense_images/1/rotation", http.StatusForbidden},
		{"rotate photos permission rotates", photoRotation, http.MethodPut, "/branch_expense_images/1/rotation", http.StatusOK},
		{"rotate photos permission rotates transfers", photoRotation, http.MethodPut, "/transfers/1/attachment/rotation", http.StatusOK},
		{"rotate photos permission can't replace", photoRotation, http.MethodPut, "/branch_expense_images/1/replace", http.StatusForbidden},
		{"section editor can't replace its photos", accountingEdit, http.MethodPut, "/branch_expense_images/1/replace", http.StatusForbidden},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			store := mockdb.NewMockStore(ctrl)
			store.EXPECT().GetUserAccess(gomock.Any(), gomock.Any()).Return(c.rows, nil)

			// newTestServer wraps the store to always answer as admin; build the
			// server directly so the mock's role rows are what's enforced.
			server, err := NewServer(util.Config{
				TokenSymmetricKey:   util.RandomString(32),
				AccessTokenDuration: time.Minute,
			}, store)
			require.NoError(t, err)

			// The same middleware chain as setupRoutes, with stub handlers on the
			// real route patterns -- the middleware decides before they'd run.
			router := gin.New()
			guarded := router.Group("/").Use(server.authMiddleware(), server.permissionMiddleware())
			ok := func(ctx *gin.Context) { ctx.Status(http.StatusOK) }
			guarded.GET("/inventories/:id/stock", ok)
			guarded.GET("/products/:id", ok)
			guarded.DELETE("/products/:id", ok)
			guarded.GET("/dashboard/summary", ok)
			guarded.GET("/users", ok)
			guarded.GET("/roles", ok)
			guarded.GET("/users/me", ok)
			guarded.POST("/barcodes/print", ok)
			guarded.PUT("/branch_expense_images/:id/rotation", ok)
			guarded.PUT("/transfers/:id/attachment/rotation", ok)
			guarded.PUT("/branch_expense_images/:id/replace", ok)
			server.router = router

			recorder := httptest.NewRecorder()
			request, err := http.NewRequest(c.method, c.url, nil)
			require.NoError(t, err)
			addAuthorization(t, request, server.tokenMaker, authorizationTypeBearer, "user", time.Minute)
			server.router.ServeHTTP(recorder, request)
			require.Equal(t, c.want, recorder.Code, recorder.Body.String())
		})
	}
}

func sqlStr(s string) sql.NullString { return sql.NullString{String: s, Valid: true} }
func sqlBool(b bool) sql.NullBool    { return sql.NullBool{Bool: b, Valid: true} }

func TestResolvePermissions(t *testing.T) {
	view := sectionLevel{View: true}
	edit := sectionLevel{View: true, Edit: true}

	t.Run("section-only role inherits onto its pages", func(t *testing.T) {
		perms := resolvePermissions([]permissionRow{{Section: "products", Level: edit}})
		require.True(t, perms.Sections["products"])
		require.Equal(t, edit, perms.Pages["colors-sizes"])
		sections, pages := perms.effective()
		require.Equal(t, edit, sections["products"])
		require.Equal(t, edit, pages["barcodes"])
	})

	t.Run("pages set individually; section is their max", func(t *testing.T) {
		perms := resolvePermissions([]permissionRow{
			{Section: "products", Level: view},
			{Section: "products", Page: "products", Level: view},
			{Section: "products", Page: "barcodes", Level: edit},
			{Section: "products", Page: "attributes"},
		})
		sections, pages := perms.effective()
		require.Equal(t, edit, sections["products"])
		require.Equal(t, view, pages["products"])
		require.Equal(t, edit, pages["barcodes"])
		require.NotContains(t, pages, "attributes")
		// Not stored at all once the section has page rows -> nothing.
		require.NotContains(t, pages, "colors-sizes")
	})

	t.Run("switched-off section keeps page levels but grants nothing", func(t *testing.T) {
		perms := resolvePermissions([]permissionRow{
			{Section: "inventory", Page: "alerts", Level: edit},
		})
		require.False(t, perms.Sections["inventory"])
		require.Equal(t, edit, perms.Pages["alerts"])
		sections, pages := perms.effective()
		require.NotContains(t, sections, "inventory")
		require.NotContains(t, pages, "alerts")
	})

	t.Run("enabled section with every page none grants nothing", func(t *testing.T) {
		rows, err := permissionRows(rolePermissions{Sections: map[string]bool{"hr": true}})
		require.NoError(t, err)
		stored := make([]permissionRow, len(rows))
		for i, r := range rows {
			stored[i] = permissionRow{Section: r.Section, Page: r.Page, Level: sectionLevel{View: r.CanView, Edit: r.CanEdit}}
		}
		sections, pages := resolvePermissions(stored).effective()
		require.NotContains(t, sections, "hr")
		require.NotContains(t, pages, "employees")
	})

	t.Run("pageless section is on or off", func(t *testing.T) {
		sections, _ := resolvePermissions([]permissionRow{{Section: "text_selection", Level: view}}).effective()
		require.Equal(t, sectionLevel{View: true, Edit: true}, sections["text_selection"])
		require.NotContains(t, sections, "photo_rotation")
	})
}

func TestPermissionRowsRejectsUnknownKeys(t *testing.T) {
	_, err := permissionRows(rolePermissions{Sections: map[string]bool{"nope": true}})
	require.Error(t, err)
	_, err = permissionRows(rolePermissions{Pages: map[string]sectionLevel{"nope": {View: true}}})
	require.Error(t, err)
}

func TestPageKeysAreUnique(t *testing.T) {
	seen := map[string]bool{}
	for _, s := range accessSections {
		for _, p := range s.Pages {
			require.False(t, seen[p.Key], "duplicate page key %q", p.Key)
			seen[p.Key] = true
		}
	}
}
