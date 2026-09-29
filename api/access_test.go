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
