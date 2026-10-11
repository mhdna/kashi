package api

import (
	"database/sql"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang/mock/gomock"
	mockdb "github.com/mhdna/kashi/db/mock"
	db "github.com/mhdna/kashi/db/sqlc"
	"github.com/stretchr/testify/require"
)

func TestCountryFilter(t *testing.T) {
	cfg := parseSecurityConfig(db.SecuritySetting{
		CountryFilterEnabled: true,
		AllowedCountries:     []string{"lb"},
		AlwaysAllowed:        []string{"203.0.113.7", "198.51.100.0/24", "nonsense"},
	})
	ip := net.ParseIP

	require.False(t, cfg.countryRefused(ip("198.51.100.7"), "LB"), "an allowed country")
	require.True(t, cfg.countryRefused(ip("8.8.8.8"), "US"), "anywhere else")
	require.False(t, cfg.countryRefused(ip("8.8.8.8"), ""), "an unknown location is let through")
	require.False(t, cfg.countryRefused(ip("203.0.113.7"), "FR"), "an always-allowed IP")
	require.False(t, cfg.countryRefused(ip("198.51.100.42"), "DE"), "inside an always-allowed range")

	cfg.filterEnabled = false
	require.False(t, cfg.countryRefused(ip("8.8.8.8"), "US"), "the filter switched off")
}

func TestInternalIPsAreNeverRefused(t *testing.T) {
	for _, s := range []string{"127.0.0.1", "::1", "10.0.0.5", "192.168.1.10"} {
		require.True(t, internalIP(net.ParseIP(s)), s)
	}
	require.False(t, internalIP(net.ParseIP("198.51.100.7")))
}

func TestProbePaths(t *testing.T) {
	for _, p := range []string{"/wp-login.php", "/.env", "/.git/config", "/phpmyadmin/", "/../../etc/passwd"} {
		require.True(t, probePath.MatchString(p), p)
	}
	for _, p := range []string{"/users/login", "/branch_invoices/daily_income", "/products/12"} {
		require.False(t, probePath.MatchString(p), p)
	}
}

// guardedRouter runs the guard in front of one handler, with requests
// coming from `from`.
func guardedRouter(g *securityGuard) *gin.Engine {
	router := gin.New()
	router.Use(g.middleware())
	router.GET("/products", func(ctx *gin.Context) { ctx.Status(http.StatusOK) })
	router.POST("/users/login", func(ctx *gin.Context) { ctx.Status(http.StatusUnauthorized) })
	return router
}

func serveFrom(router *gin.Engine, method, path, ip string) int {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, nil)
	req.RemoteAddr = ip + ":40000"
	router.ServeHTTP(rec, req)
	return rec.Code
}

func TestGuardRefusesBannedIPsAndCountsRequests(t *testing.T) {
	g := newSecurityGuard(nil, newGeoIP(t.TempDir()))
	g.bans["203.0.113.9"] = time.Now().Add(time.Hour)
	g.bans["203.0.113.10"] = time.Now().Add(-time.Minute) // expired
	router := guardedRouter(g)

	require.Equal(t, http.StatusForbidden, serveFrom(router, "GET", "/products", "203.0.113.9"))
	require.Equal(t, http.StatusOK, serveFrom(router, "GET", "/products", "203.0.113.10"))
	require.Equal(t, http.StatusOK, serveFrom(router, "GET", "/products", "198.51.100.1"))
	require.Equal(t, http.StatusOK, serveFrom(router, "GET", "/products", "198.51.100.1"))

	require.EqualValues(t, 1, g.pending["203.0.113.9"].blocked)
	require.EqualValues(t, 2, g.pending["198.51.100.1"].requests)
	require.Equal(t, "/products", g.pending["198.51.100.1"].path)
}

func TestGuardBansAfterRepeatedFailedLogins(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	// Failures from the 5th on raise (or bump) an alert.
	store.EXPECT().RecordSecurityAlert(gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
	banned := make(chan db.UpsertIPBanParams, 1)
	store.EXPECT().UpsertIPBan(gomock.Any(), gomock.Any()).DoAndReturn(func(_ any, arg db.UpsertIPBanParams) error {
		banned <- arg
		return nil
	})

	g := newSecurityGuard(store, newGeoIP(t.TempDir()))
	g.cfg.autoBanAt = 6
	router := guardedRouter(g)

	for i := 0; i < 6; i++ {
		require.Equal(t, http.StatusUnauthorized, serveFrom(router, "POST", "/users/login", "203.0.113.20"))
	}
	require.Equal(t, http.StatusForbidden, serveFrom(router, "GET", "/products", "203.0.113.20"), "banned after the 6th failure")
	require.Equal(t, http.StatusOK, serveFrom(router, "GET", "/products", "203.0.113.21"), "others unaffected")

	select {
	case arg := <-banned:
		require.Equal(t, "203.0.113.20", arg.Ip)
		require.True(t, arg.Until.Valid)
		require.Equal(t, "automatic", arg.CreatedBy)
	case <-time.After(2 * time.Second):
		t.Fatal("the ban was never saved")
	}
	time.Sleep(50 * time.Millisecond) // let the alert goroutines finish
}

func TestVisitorIPBelievesCloudflareOnly(t *testing.T) {
	ipFor := func(remote, cfHeader string) string {
		ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
		ctx.Request = httptest.NewRequest("GET", "/", nil)
		ctx.Request.RemoteAddr = net.JoinHostPort(remote, "443")
		if cfHeader != "" {
			ctx.Request.Header.Set("CF-Connecting-IP", cfHeader)
		}
		return visitorIP(ctx)
	}

	require.Equal(t, "198.51.100.7", ipFor("172.68.10.20", "198.51.100.7"), "relayed by a Cloudflare edge")
	require.Equal(t, "2a02:2788::1", ipFor("2a06:98c1::5", "2a02:2788::1"), "IPv6 edge")
	require.Equal(t, "203.0.113.5", ipFor("203.0.113.5", "198.51.100.7"), "a forged header sent straight to the server is ignored")
	require.Equal(t, "172.68.10.20", ipFor("172.68.10.20", "not-an-ip"), "a garbled header falls back to the edge")
}

func TestSecurityTimesEncodeAsTimeOrNull(t *testing.T) {
	seen := time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC)
	out, err := json.Marshal(alertResponses([]db.SecurityAlert{
		{ID: 1},
		{ID: 2, AcknowledgedAt: sql.NullTime{Time: seen, Valid: true}},
	}))
	require.NoError(t, err)
	require.Contains(t, string(out), `"id":1,`)
	require.Contains(t, string(out), `"acknowledged_at":null`)
	require.Contains(t, string(out), `"acknowledged_at":"2026-10-09T12:00:00Z"`)
	require.NotContains(t, string(out), `"Valid"`)
}

func TestCountOpenSecurityAlerts(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	store.EXPECT().CountOpenSecurityAlerts(gomock.Any()).Return(int64(3), nil)

	server := newTestServer(t, store)
	recorder := httptest.NewRecorder()
	request, err := http.NewRequest(http.MethodGet, "/security/alerts/open_count", nil)
	require.NoError(t, err)
	addAuthorization(t, request, server.tokenMaker, authorizationTypeBearer, "admin", time.Minute)
	server.router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code)
	var body struct {
		OpenAlerts int64 `json:"open_alerts"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
	require.Equal(t, int64(3), body.OpenAlerts)
}

func TestAlertResponsesSeverityAndPin(t *testing.T) {
	seen := sql.NullTime{Time: time.Now(), Valid: true}
	out := alertResponses([]db.SecurityAlert{
		{ID: 1, Kind: "login_abroad"},
		{ID: 2, Kind: "login_abroad", AcknowledgedAt: seen},
		{ID: 3, Kind: "failed_logins"},
		{ID: 4, Kind: "something_new"},
	})
	require.Equal(t, "critical", out[0].Severity)
	require.True(t, out[0].Pinned, "open critical alert is pinned")
	require.False(t, out[1].Pinned, "resolved critical alert is not pinned")
	require.Equal(t, "high", out[2].Severity)
	require.False(t, out[2].Pinned)
	require.Equal(t, "medium", out[3].Severity)
}

// "Mark all seen" (no ids) leaves critical alerts open and records who
// acknowledged.
func TestAcknowledgeAllKeepsCriticalAlerts(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	store.EXPECT().AcknowledgeSecurityAlerts(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ any, arg db.AcknowledgeSecurityAlertsParams) error {
			require.Empty(t, arg.Ids)
			require.Equal(t, []string{"login_abroad"}, arg.KeptKinds)
			require.Equal(t, "admin", arg.AcknowledgedBy)
			return nil
		})

	server := newTestServer(t, store)
	recorder := httptest.NewRecorder()
	request, err := http.NewRequest(http.MethodPost, "/security/alerts/acknowledge", strings.NewReader(`{"ids":[]}`))
	require.NoError(t, err)
	request.Header.Set("Content-Type", "application/json")
	addAuthorization(t, request, server.tokenMaker, authorizationTypeBearer, "admin", time.Minute)
	server.router.ServeHTTP(recorder, request)
	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
}
