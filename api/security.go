package api

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	db "github.com/mhdna/kashi/db/sqlc"
	"github.com/mhdna/kashi/token"
)

// The Security page's machinery: every request to the API passes
// securityGuard.middleware, which
//
//   - refuses banned IPs, and -- when the country filter is on -- IPs located
//     outside the allowed countries (unless listed in always_allowed);
//   - counts the request against its IP (visitor_ips, flushed every few
//     seconds rather than written per request -- the tills alone poll
//     constantly);
//   - raises security_alerts: repeated failed logins (and, past a limit, a
//     temporary ban), probing for well-known attack paths, logins and
//     visits from outside the expected countries.
//
// Requests from the server itself or a private network are never refused.
// Till requests (/branch/...) skip the country filter: they carry their own
// API key, and a till on a link that geolocates abroad must keep syncing.
//
// Only requests that reach the API (Caddy's /api/...) are seen; loading the
// web app's static files doesn't, but nothing can be read or signed into
// without the API.

const (
	securityFlushEvery     = 5 * time.Second
	securityRefreshEvery   = time.Minute
	failedLoginWindow      = 15 * time.Minute
	failedLoginAlertAt     = 5
	foreignVisitAlertAgain = 24 * time.Hour
)

// cloudflareNets are Cloudflare's published edge ranges
// (https://www.cloudflare.com/ips/). The site sits behind Cloudflare, so a
// visitor's request reaches Caddy from one of these, and the visitor's own
// address is in CF-Connecting-IP. Only a request that really came from
// Cloudflare gets that header believed -- anyone can send it straight to
// the server. Cloudflare rarely changes the list; a missing range just
// shows that edge's address instead of the visitor's.
var cloudflareNets = func() []*net.IPNet {
	var nets []*net.IPNet
	for _, cidr := range []string{
		"173.245.48.0/20", "103.21.244.0/22", "103.22.200.0/22", "103.31.4.0/22",
		"141.101.64.0/18", "108.162.192.0/18", "190.93.240.0/20", "188.114.96.0/20",
		"197.234.240.0/22", "198.41.128.0/17", "162.158.0.0/15", "104.16.0.0/13",
		"104.24.0.0/14", "172.64.0.0/13", "131.0.72.0/22",
		"2400:cb00::/32", "2606:4700::/32", "2803:f800::/32", "2405:b500::/32",
		"2405:8100::/32", "2a06:98c0::/29", "2c0f:f248::/32",
	} {
		_, n, _ := net.ParseCIDR(cidr)
		nets = append(nets, n)
	}
	return nets
}()

func fromCloudflare(ip net.IP) bool {
	for _, n := range cloudflareNets {
		if n.Contains(ip) {
			return true
		}
	}
	return false
}

// visitorIP is the address of whoever is on the other end: gin's ClientIP
// (Caddy's X-Forwarded-For, see setupRoutes) unless that's a Cloudflare
// edge, in which case the visitor Cloudflare is relaying for.
func visitorIP(ctx *gin.Context) string {
	ip := ctx.ClientIP()
	if parsed := net.ParseIP(ip); parsed != nil && fromCloudflare(parsed) {
		if real := net.ParseIP(strings.TrimSpace(ctx.GetHeader("CF-Connecting-IP"))); real != nil {
			return real.String()
		}
	}
	return ip
}

// probePath matches requests for things this API never serves but scanners
// look for everywhere: PHP and WordPress files, dotfiles, admin panels,
// path traversal.
var probePath = regexp.MustCompile(`(?i)(\.php|\.asp|\.env|\.git|wp-|wordpress|phpmyadmin|/cgi-bin|/\.well-known/security|\.\./|/etc/passwd|/actuator|/solr|/vendor/|\.sql$|\.bak$|/admin\.)`)

type securityConfig struct {
	filterEnabled  bool
	allowed        map[string]bool
	alwaysAllowed  []*net.IPNet
	autoBan        bool
	autoBanAt      int
	autoBanMinutes int
}

func parseSecurityConfig(s db.SecuritySetting) securityConfig {
	cfg := securityConfig{
		filterEnabled:  s.CountryFilterEnabled,
		allowed:        map[string]bool{},
		autoBan:        s.AutoBanEnabled,
		autoBanAt:      int(s.AutoBanFailedLogins),
		autoBanMinutes: int(s.AutoBanMinutes),
	}
	for _, c := range s.AllowedCountries {
		cfg.allowed[strings.ToUpper(c)] = true
	}
	for _, entry := range s.AlwaysAllowed {
		if n := parseIPOrCIDR(entry); n != nil {
			cfg.alwaysAllowed = append(cfg.alwaysAllowed, n)
		}
	}
	return cfg
}

// parseIPOrCIDR reads "1.2.3.4" or "1.2.3.0/24" (IPv6 too); nil if neither.
func parseIPOrCIDR(s string) *net.IPNet {
	s = strings.TrimSpace(s)
	if _, n, err := net.ParseCIDR(s); err == nil {
		return n
	}
	ip := net.ParseIP(s)
	if ip == nil {
		return nil
	}
	bits := 128
	if ip.To4() != nil {
		ip, bits = ip.To4(), 32
	}
	return &net.IPNet{IP: ip, Mask: net.CIDRMask(bits, bits)}
}

func (c securityConfig) isAlwaysAllowed(ip net.IP) bool {
	for _, n := range c.alwaysAllowed {
		if n.Contains(ip) {
			return true
		}
	}
	return false
}

// countryRefused says whether the country filter turns this IP away. An IP
// with no known country is let through: refusing on a missing database
// entry would lock out real people.
func (c securityConfig) countryRefused(ip net.IP, countryCode string) bool {
	return c.filterEnabled && countryCode != "" && !c.allowed[countryCode] && !c.isAlwaysAllowed(ip)
}

// internalIP is the server itself or a private network -- never refused.
func internalIP(ip net.IP) bool {
	return ip == nil || ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsUnspecified()
}

// visitorDelta is what one IP did since the last flush.
type visitorDelta struct {
	seenAt                                              time.Time
	requests, failedLogins, logins, notFound, blocked   int64
	kind, userName, branchName, method, path, userAgent string
	status                                              int
}

type securityGuard struct {
	store db.Store
	geo   *geoIP
	// notify tells admin pages a new alert came in (nil in tests).
	notify func()

	mu       sync.Mutex
	cfg      securityConfig
	bans     map[string]time.Time // zero time = until lifted
	pending  map[string]*visitorDelta
	failures map[string][]time.Time
	foreign  map[string]time.Time // IP -> when its last foreign-visit alert went out
}

func newSecurityGuard(store db.Store, geo *geoIP) *securityGuard {
	return &securityGuard{
		store:    store,
		geo:      geo,
		cfg:      securityConfig{allowed: map[string]bool{"LB": true}, autoBan: true, autoBanAt: 10, autoBanMinutes: 60},
		bans:     map[string]time.Time{},
		pending:  map[string]*visitorDelta{},
		failures: map[string][]time.Time{},
		foreign:  map[string]time.Time{},
	}
}

func (g *securityGuard) config() securityConfig {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.cfg
}

func (g *securityGuard) banned(ip string) bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	until, ok := g.bans[ip]
	if !ok {
		return false
	}
	if !until.IsZero() && time.Now().After(until) {
		delete(g.bans, ip)
		return false
	}
	return true
}

func (g *securityGuard) middleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		ipText := visitorIP(ctx)
		ip := net.ParseIP(ipText)
		internal := internalIP(ip)
		path := ctx.Request.URL.Path
		// Meta calls the WhatsApp webhook from its own servers abroad; the
		// handler checks Meta's signature instead (whatsapp_webhook.go).
		// Review links are opened from anywhere: link preview fetchers
		// (WhatsApp, Meta, the bridge) and clients abroad (review_link.go).
		metaWebhook := path == whatsAppWebhookPath || isReviewLinkPath(path)
		var loc geoLocation
		if !internal {
			loc = g.geo.lookup(ipText)
		}
		cfg := g.config()

		if !internal {
			refusal := ""
			switch {
			case g.banned(ipText):
				refusal = "banned"
			case !strings.HasPrefix(path, "/branch/") && !metaWebhook && cfg.countryRefused(ip, loc.CountryCode):
				refusal = "country"
			}
			if refusal != "" {
				ctx.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "access denied"})
				g.record(ipText, &visitorDelta{blocked: 1, method: ctx.Request.Method, path: path, status: http.StatusForbidden, userAgent: ctx.Request.UserAgent()})
				if refusal == "country" {
					g.alert("blocked_country", ipText, loc, fmt.Sprintf("Refused %s %s from %s", ctx.Request.Method, path, placeName(loc)))
				}
				return
			}
		}

		ctx.Next()

		status := ctx.Writer.Status()
		d := &visitorDelta{method: ctx.Request.Method, path: path, status: status, userAgent: ctx.Request.UserAgent(), kind: "public"}
		if payload, ok := ctx.Get(authoizationPayloadKey); ok {
			if p, ok := payload.(*token.Payload); ok {
				d.kind, d.userName = "admin", p.Username
			}
		}
		if _, ok := ctx.Get(branchIDKey); ok {
			d.kind, d.branchName = "till", ctx.GetHeader(branchCodeHeaderKey)
		}
		if status == http.StatusNotFound {
			d.notFound = 1
		}
		login := path == "/users/login" && ctx.Request.Method == http.MethodPost
		if login && status == http.StatusUnauthorized {
			d.failedLogins = 1
		}
		if login && status == http.StatusOK {
			d.logins = 1
		}
		g.record(ipText, d)
		if internal {
			return
		}

		if d.failedLogins > 0 {
			g.failedLogin(ipText, loc, cfg)
		}
		if probePath.MatchString(path) {
			g.alert("probe", ipText, loc, fmt.Sprintf("%s %s → %d", ctx.Request.Method, path, status))
		}
		if loc.CountryCode != "" && !cfg.allowed[loc.CountryCode] && !metaWebhook {
			if d.logins > 0 {
				g.alert("login_abroad", ipText, loc, fmt.Sprintf("Signed in from %s", placeName(loc)))
			} else if g.firstForeignVisitToday(ipText) {
				g.alert("foreign_visit", ipText, loc, fmt.Sprintf("Visit from %s (%s)", placeName(loc), loc.Network))
			}
		}
	}
}

func placeName(loc geoLocation) string {
	parts := []string{}
	for _, p := range []string{loc.City, loc.Country} {
		if p != "" {
			parts = append(parts, p)
		}
	}
	if len(parts) == 0 {
		return "an unknown place"
	}
	return strings.Join(parts, ", ")
}

func (g *securityGuard) record(ip string, d *visitorDelta) {
	if d.requests == 0 {
		d.requests = 1
	}
	d.seenAt = time.Now().UTC()
	g.mu.Lock()
	defer g.mu.Unlock()
	cur, ok := g.pending[ip]
	if !ok {
		g.pending[ip] = d
		return
	}
	cur.requests += d.requests
	cur.failedLogins += d.failedLogins
	cur.logins += d.logins
	cur.notFound += d.notFound
	cur.blocked += d.blocked
	cur.seenAt, cur.method, cur.path, cur.status = d.seenAt, d.method, d.path, d.status
	if d.kind != "" {
		cur.kind = d.kind
	}
	for dst, src := range map[*string]string{&cur.userName: d.userName, &cur.branchName: d.branchName, &cur.userAgent: d.userAgent} {
		if src != "" {
			*dst = src
		}
	}
}

// failedLogin tracks failed sign-ins per IP over the last 15 minutes:
// alert at 5, and at the configured limit ban the IP for a while.
func (g *securityGuard) failedLogin(ip string, loc geoLocation, cfg securityConfig) {
	now := time.Now()
	g.mu.Lock()
	recent := []time.Time{}
	for _, t := range g.failures[ip] {
		if now.Sub(t) < failedLoginWindow {
			recent = append(recent, t)
		}
	}
	recent = append(recent, now)
	g.failures[ip] = recent
	n := len(recent)
	ban := cfg.autoBan && cfg.autoBanAt > 0 && n >= cfg.autoBanAt
	var until time.Time
	if ban {
		until = now.Add(time.Duration(cfg.autoBanMinutes) * time.Minute)
		g.bans[ip] = until
		delete(g.failures, ip)
	}
	g.mu.Unlock()

	if n >= failedLoginAlertAt {
		g.alert("failed_logins", ip, loc, fmt.Sprintf("%d failed sign-ins in 15 minutes", n))
	}
	if ban {
		go func() {
			if err := g.store.UpsertIPBan(context.Background(), db.UpsertIPBanParams{
				Ip: ip, Reason: fmt.Sprintf("%d failed sign-ins in 15 minutes", n),
				Until: sql.NullTime{Time: until, Valid: true}, CreatedBy: "automatic",
			}); err != nil {
				log.Printf("security: ban %s: %v", ip, err)
			}
		}()
		g.alert("auto_ban", ip, loc, fmt.Sprintf("Banned for %d minutes after %d failed sign-ins", cfg.autoBanMinutes, n))
	}
}

func (g *securityGuard) firstForeignVisitToday(ip string) bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	if last, ok := g.foreign[ip]; ok && time.Since(last) < foreignVisitAlertAgain {
		return false
	}
	g.foreign[ip] = time.Now()
	return true
}

// alert records (or bumps) a security alert off the request path.
func (g *securityGuard) alert(kind, ip string, loc geoLocation, detail string) {
	go func() {
		if err := g.store.RecordSecurityAlert(context.Background(), db.RecordSecurityAlertParams{
			Kind: kind, Ip: ip, CountryCode: loc.CountryCode, Detail: detail,
		}); err != nil {
			log.Printf("security: alert %s %s: %v", kind, ip, err)
			return
		}
		if g.notify != nil {
			g.notify()
		}
	}()
}

// flush writes the pending per-IP counts.
func (g *securityGuard) flush(ctx context.Context) {
	g.mu.Lock()
	pending := g.pending
	g.pending = map[string]*visitorDelta{}
	g.mu.Unlock()

	for ip, d := range pending {
		loc := g.geo.lookup(ip)
		kind := d.kind
		if kind == "" {
			kind = "public"
		}
		if err := g.store.UpsertVisitorIP(ctx, db.UpsertVisitorIPParams{
			Ip: ip, SeenAt: d.seenAt, Requests: d.requests, FailedLogins: d.failedLogins,
			Logins: d.logins, NotFound: d.notFound, Blocked: d.blocked,
			LastKind: kind, LastUserName: d.userName, LastBranchName: d.branchName,
			LastMethod: d.method, LastPath: truncate(d.path, 300), LastStatus: int32(d.status),
			UserAgent:   truncate(d.userAgent, 300),
			CountryCode: loc.CountryCode, Country: loc.Country, Region: loc.Region, City: loc.City,
			Asn: int64(loc.ASN), Network: loc.Network,
		}); err != nil {
			log.Printf("security: record %s: %v", ip, err)
		}
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

// reload reads the settings and active bans back from the database.
func (g *securityGuard) reload(ctx context.Context) {
	settings, err := g.store.GetSecuritySettings(ctx)
	if err != nil {
		log.Printf("security: load settings: %v", err)
	}
	bans, banErr := g.store.ListActiveIPBans(ctx)
	if banErr != nil {
		log.Printf("security: load bans: %v", banErr)
	}

	g.mu.Lock()
	defer g.mu.Unlock()
	if err == nil {
		g.cfg = parseSecurityConfig(settings)
	}
	if banErr == nil {
		g.bans = map[string]time.Time{}
		for _, b := range bans {
			g.bans[b.Ip] = b.Until.Time // zero when NULL: until lifted
		}
	}
	// Forget failed-login and foreign-visit bookkeeping that has aged out.
	for ip, times := range g.failures {
		if len(times) == 0 || time.Since(times[len(times)-1]) > failedLoginWindow {
			delete(g.failures, ip)
		}
	}
	for ip, at := range g.foreign {
		if time.Since(at) > foreignVisitAlertAgain {
			delete(g.foreign, ip)
		}
	}
}

// Run keeps the guard's state in step with the database and the IP
// databases up to date, until ctx ends. main starts it.
func (g *securityGuard) Run(ctx context.Context) {
	g.reload(ctx)
	go g.geo.runUpdates(ctx)
	flush := time.NewTicker(securityFlushEvery)
	refresh := time.NewTicker(securityRefreshEvery)
	defer flush.Stop()
	defer refresh.Stop()
	for {
		select {
		case <-ctx.Done():
			g.flush(context.Background())
			return
		case <-flush.C:
			g.flush(ctx)
		case <-refresh.C:
			g.reload(ctx)
		}
	}
}

// ---- Admin endpoints (Security section, see access.go's routeRules) ----

type securityLocation struct {
	CountryCode string `json:"country_code"`
	Country     string `json:"country"`
	Region      string `json:"region"`
	City        string `json:"city"`
	Network     string `json:"network"`
}

func (g *securityGuard) locate(ip string) securityLocation {
	loc := g.geo.lookup(ip)
	return securityLocation{loc.CountryCode, loc.Country, loc.Region, loc.City, loc.Network}
}

func (server *Server) getSecurityOverview(ctx *gin.Context) {
	settings, err := server.store.GetSecuritySettings(ctx)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	alerts, err := server.store.ListSecurityAlerts(ctx, db.ListSecurityAlertsParams{OpenOnly: false, PinnedKinds: criticalAlertKinds, PageSize: 200})
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	openAlerts, err := server.store.CountOpenSecurityAlerts(ctx)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	bans, err := server.store.ListActiveIPBans(ctx)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	countries, err := server.store.VisitorCountries(ctx)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	g := server.security
	g.geo.mu.RLock()
	geoReady := g.geo.city != nil
	g.geo.mu.RUnlock()

	server.writeJSON(ctx, http.StatusOK, envelope{
		"settings":    settings,
		"alerts":      alertResponses(alerts),
		"open_alerts": openAlerts,
		"bans":        banResponses(bans),
		"countries":   countries,
		"geo_ready":   geoReady,
		"my_ip":       visitorIP(ctx),
		"my_location": g.locate(visitorIP(ctx)),
	})
}

// How serious each alert kind is. A sign-in from abroad may be a stolen
// password that worked, so it's the one critical kind: it stays pinned at
// the top of the Security page until someone resolves it by itself ("Mark
// all seen" leaves it). An unknown kind counts as medium.
var alertSeverities = map[string]string{
	"login_abroad":    "critical",
	"failed_logins":   "high",
	"auto_ban":        "high",
	"probe":           "medium",
	"foreign_visit":   "low",
	"blocked_country": "low",
}

var criticalAlertKinds = func() []string {
	var kinds []string
	for kind, severity := range alertSeverities {
		if severity == "critical" {
			kinds = append(kinds, kind)
		}
	}
	return kinds
}()

func alertSeverity(kind string) string {
	if s, ok := alertSeverities[kind]; ok {
		return s
	}
	return "medium"
}

// The alert and ban rows carry sql.NullTime, which encodes as
// {"Time":..., "Valid":...}; the page wants a time or null.
type securityAlertResponse struct {
	db.SecurityAlert
	AcknowledgedAt *time.Time `json:"acknowledged_at"`
	Severity       string     `json:"severity"`
	// Pinned: a critical alert nobody has resolved yet.
	Pinned bool `json:"pinned"`
}

type ipBanResponse struct {
	db.IpBan
	Until *time.Time `json:"until"`
}

func nullTimePtr(t sql.NullTime) *time.Time {
	if !t.Valid {
		return nil
	}
	return &t.Time
}

func alertResponses(rows []db.SecurityAlert) []securityAlertResponse {
	out := make([]securityAlertResponse, len(rows))
	for i, r := range rows {
		severity := alertSeverity(r.Kind)
		out[i] = securityAlertResponse{
			SecurityAlert:  r,
			AcknowledgedAt: nullTimePtr(r.AcknowledgedAt),
			Severity:       severity,
			Pinned:         severity == "critical" && !r.AcknowledgedAt.Valid,
		}
	}
	return out
}

func banResponses(rows []db.IpBan) []ipBanResponse {
	out := make([]ipBanResponse, len(rows))
	for i, r := range rows {
		out[i] = ipBanResponse{IpBan: r, Until: nullTimePtr(r.Until)}
	}
	return out
}

type listVisitorsRequest struct {
	Search   string `form:"search"`
	Country  string `form:"country"`
	PageSize int32  `form:"page_size,default=50" binding:"min=5,max=500"`
	PageID   int32  `form:"page_id,default=0" binding:"min=0"`
}

func (server *Server) listSecurityVisitors(ctx *gin.Context) {
	var req listVisitorsRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	search := strings.TrimSpace(req.Search)
	country := strings.ToUpper(strings.TrimSpace(req.Country))
	visitors, err := server.store.ListVisitorIPs(ctx, db.ListVisitorIPsParams{
		Search: search, CountryCode: country, PageSize: req.PageSize, PageOffset: req.PageID,
	})
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	total, err := server.store.CountVisitorIPs(ctx, db.CountVisitorIPsParams{Search: search, CountryCode: country})
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	server.writeJSON(ctx, http.StatusOK, envelope{"visitors": visitors, "total": total})
}

// countOpenSecurityAlerts backs the Security nav badge: just the number of
// alerts nobody has acknowledged, without the overview's lists.
func (server *Server) countOpenSecurityAlerts(ctx *gin.Context) {
	open, err := server.store.CountOpenSecurityAlerts(ctx)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	server.writeJSON(ctx, http.StatusOK, envelope{"open_alerts": open})
}

type acknowledgeAlertsRequest struct {
	IDs []int64 `json:"ids"` // empty = every open alert but the critical ones
}

func (server *Server) acknowledgeSecurityAlerts(ctx *gin.Context) {
	var req acknowledgeAlertsRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	if req.IDs == nil {
		req.IDs = []int64{}
	}
	by := ""
	if payload, ok := ctx.Get(authoizationPayloadKey); ok {
		if p, ok := payload.(*token.Payload); ok {
			by = p.Username
		}
	}
	if err := server.store.AcknowledgeSecurityAlerts(ctx, db.AcknowledgeSecurityAlertsParams{
		AcknowledgedBy: by, Ids: req.IDs, KeptKinds: criticalAlertKinds,
	}); err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	server.writeJSON(ctx, http.StatusOK, envelope{"ok": true})
}

type updateSecuritySettingsRequest struct {
	CountryFilterEnabled bool     `json:"country_filter_enabled"`
	AllowedCountries     []string `json:"allowed_countries"`
	AlwaysAllowed        []string `json:"always_allowed"`
	AutoBanEnabled       bool     `json:"auto_ban_enabled"`
	AutoBanFailedLogins  int32    `json:"auto_ban_failed_logins" binding:"min=3,max=1000"`
	AutoBanMinutes       int32    `json:"auto_ban_minutes" binding:"min=1,max=525600"`
}

var countryCodeRe = regexp.MustCompile(`^[A-Z]{2}$`)

func (server *Server) updateSecuritySettings(ctx *gin.Context) {
	var req updateSecuritySettingsRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	countries := []string{}
	for _, c := range req.AllowedCountries {
		c = strings.ToUpper(strings.TrimSpace(c))
		if !countryCodeRe.MatchString(c) {
			server.writeError(ctx, http.StatusBadRequest, fmt.Errorf("%q isn't a two-letter country code", c))
			return
		}
		countries = append(countries, c)
	}
	if req.CountryFilterEnabled && len(countries) == 0 {
		server.writeError(ctx, http.StatusBadRequest, errors.New("pick at least one allowed country"))
		return
	}
	always := []string{}
	for _, entry := range req.AlwaysAllowed {
		if entry = strings.TrimSpace(entry); entry == "" {
			continue
		}
		if parseIPOrCIDR(entry) == nil {
			server.writeError(ctx, http.StatusBadRequest, fmt.Errorf("%q isn't an IP address or range", entry))
			return
		}
		always = append(always, entry)
	}

	// Don't let a save lock out the person saving it.
	candidate := parseSecurityConfig(db.SecuritySetting{
		CountryFilterEnabled: req.CountryFilterEnabled, AllowedCountries: countries, AlwaysAllowed: always,
	})
	myIP := visitorIP(ctx)
	if parsed := net.ParseIP(myIP); !internalIP(parsed) {
		if loc := server.security.geo.lookup(myIP); candidate.countryRefused(parsed, loc.CountryCode) {
			server.writeError(ctx, http.StatusBadRequest, fmt.Errorf(
				"this would block you: your IP %s is in %s -- add that country or your IP to the always-allowed list first", myIP, placeName(loc)))
			return
		}
	}

	settings, err := server.store.UpdateSecuritySettings(ctx, db.UpdateSecuritySettingsParams{
		CountryFilterEnabled: req.CountryFilterEnabled,
		AllowedCountries:     countries,
		AlwaysAllowed:        always,
		AutoBanEnabled:       req.AutoBanEnabled,
		AutoBanFailedLogins:  req.AutoBanFailedLogins,
		AutoBanMinutes:       req.AutoBanMinutes,
	})
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	server.security.reload(ctx)
	server.writeJSON(ctx, http.StatusOK, envelope{"settings": settings})
}

type banIPRequest struct {
	IP      string `json:"ip" binding:"required"`
	Reason  string `json:"reason"`
	Minutes int    `json:"minutes" binding:"min=0"` // 0 = until lifted
}

func (server *Server) banIP(ctx *gin.Context) {
	var req banIPRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	ip := strings.TrimSpace(req.IP)
	parsed := net.ParseIP(ip)
	switch {
	case parsed == nil:
		server.writeError(ctx, http.StatusBadRequest, fmt.Errorf("%q isn't an IP address", ip))
		return
	case internalIP(parsed):
		server.writeError(ctx, http.StatusBadRequest, errors.New("the server's own and private-network addresses can't be banned"))
		return
	case ip == visitorIP(ctx):
		server.writeError(ctx, http.StatusBadRequest, errors.New("that's your own IP"))
		return
	}
	var until sql.NullTime
	if req.Minutes > 0 {
		until = sql.NullTime{Time: time.Now().Add(time.Duration(req.Minutes) * time.Minute), Valid: true}
	}
	by := ""
	if payload, ok := ctx.Get(authoizationPayloadKey); ok {
		if p, ok := payload.(*token.Payload); ok {
			by = p.Username
		}
	}
	if err := server.store.UpsertIPBan(ctx, db.UpsertIPBanParams{
		Ip: ip, Reason: strings.TrimSpace(req.Reason), Until: until, CreatedBy: by,
	}); err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	server.security.reload(ctx)
	server.writeJSON(ctx, http.StatusOK, envelope{"ok": true})
}

func (server *Server) unbanIP(ctx *gin.Context) {
	ip := strings.TrimSpace(ctx.Param("ip"))
	if err := server.store.DeleteIPBan(ctx, ip); err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	server.security.reload(ctx)
	server.writeJSON(ctx, http.StatusOK, envelope{"ok": true})
}
