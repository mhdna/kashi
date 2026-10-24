package api

import (
	"bytes"
	"net/http"

	"github.com/gin-gonic/gin"
)

// phoneUploadPage is one embedded phone upload page in its two themes. Both
// are built once at start-up, so choosing between them costs a bool read per
// request: the pages are always light, unless the app setting
// upload_pages_follow_system lets them follow the phone's dark mode (the
// page's dark colours only apply under html.follow-system).
type phoneUploadPage struct {
	light  []byte
	system []byte
}

func newPhoneUploadPage(html []byte) phoneUploadPage {
	return phoneUploadPage{
		light:  html,
		system: bytes.Replace(html, []byte(`<html lang="en">`), []byte(`<html lang="en" class="follow-system">`), 1),
	}
}

// uploadPagesFollowSystem is the cached app setting, read from the database
// the first time a page is served and updated when settings are saved.
func (server *Server) uploadPagesFollowSystem(ctx *gin.Context) bool {
	if !server.uploadThemeLoaded.Load() {
		if settings, err := server.store.GetAppSettings(ctx); err == nil {
			server.uploadFollowSystem.Store(settings.UploadPagesFollowSystem)
			server.uploadThemeLoaded.Store(true)
		}
	}
	return server.uploadFollowSystem.Load()
}

func (server *Server) servePhoneUploadPage(ctx *gin.Context, page phoneUploadPage) {
	body := page.light
	if server.uploadPagesFollowSystem(ctx) {
		body = page.system
	}
	ctx.Data(http.StatusOK, "text/html; charset=utf-8", body)
}
