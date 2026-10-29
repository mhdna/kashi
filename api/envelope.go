package api

import (
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// envelope wraps every JSON response body in a top-level object keyed by
// name (e.g. {"branch": ...}), rather than returning a bare struct or an
// ad-hoc gin.H per handler. Keeping response shapes predictable this way
// means they can grow later (extra metadata, pagination, etc.) without
// breaking existing clients. Pattern from Alex Edwards' "Let's Go Further".
//
// Scoped to the branch/sync endpoints for now (api/branch.go,
// api/branch_sync.go, and friends) rather than retrofitted across the
// whole api package — that's a separate, larger cleanup.
type envelope map[string]any

func (server *Server) writeJSON(ctx *gin.Context, status int, data envelope) {
	ctx.JSON(status, data)
}

// writeError logs the error server-side whenever it's a 5xx — client errors
// (4xx) are the caller's fault and don't need a server-side trace, but a 500
// means something broke here, and this codebase has no other mechanism that
// records that anywhere today.
//
// 502/503/504 are reserved for the proxy: the UI reads them as "the API is
// being redeployed" and covers the app with the Updating overlay until the
// API answers again (ui/src/composables/useServerStatus.js). A handler
// whose upstream failed (the WhatsApp bridge, a slow query) sends a 500
// instead, so its error shows on the page rather than a stuck overlay.
func (server *Server) writeError(ctx *gin.Context, status int, err error) {
	switch status {
	case http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		status = http.StatusInternalServerError
	}
	if status >= http.StatusInternalServerError {
		log.Printf("error: %v %v: %v", logSafe(ctx.Request.Method), logSafe(ctx.Request.URL.Path), logSafe(err.Error()))
	}
	server.writeJSON(ctx, status, envelope{"error": err.Error()})
}

// logSafe escapes line breaks in request-derived text so it can't forge
// extra log lines.
func logSafe(s string) string {
	s = strings.ReplaceAll(s, "\n", `\n`)
	return strings.ReplaceAll(s, "\r", `\r`)
}

// abortError is writeError for middleware: it also stops the handler chain,
// the same way ctx.AbortWithStatusJSON does.
func (server *Server) abortError(ctx *gin.Context, status int, err error) {
	if status >= http.StatusInternalServerError {
		log.Printf("error: %v %v: %v", logSafe(ctx.Request.Method), logSafe(ctx.Request.URL.Path), logSafe(err.Error()))
	}
	ctx.AbortWithStatusJSON(status, envelope{"error": err.Error()})
}
