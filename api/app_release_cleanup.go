package api

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// Till builds are ~30 MB each and pile up with every release, so their
// files are cleaned up once they've done their job. The App Updates page's
// "Keep old builds" (app_settings.app_release_keep) decides how many stay:
//
//	 0  none (the default): a build's file goes once every active register
//	    set to it runs it.
//	 N  the newest N builds of each flavor stay as well (for rolling back).
//	-1  every build stays.
//
// A fresh upload nobody has been set to yet stays until it's sent somewhere
// or a newer one is published. The release row is never deleted (the history
// points at it); publishing the same version again brings the file back.

const appReleaseCleanupEvery = 6 * time.Hour

const (
	appReleaseKeepAll = -1
	appReleaseKeepMax = 100
)

var appReleaseCleanupMu sync.Mutex

func (server *Server) RunAppReleaseCleanup(ctx context.Context) {
	server.cleanAppReleases(ctx)
	ticker := time.NewTicker(appReleaseCleanupEvery)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			server.cleanAppReleases(ctx)
		}
	}
}

// cleanAppReleasesSoon runs a clean-up in the background, after something
// that may have freed a build: a till finished updating, targets changed, or
// the setting did.
func (server *Server) cleanAppReleasesSoon() {
	go server.cleanAppReleases(context.Background())
}

func (server *Server) cleanAppReleases(ctx context.Context) {
	appReleaseCleanupMu.Lock()
	defer appReleaseCleanupMu.Unlock()

	settings, err := server.store.GetAppSettings(ctx)
	if err != nil {
		log.Printf("app release clean-up: %v", err)
		return
	}
	if settings.AppReleaseKeep == appReleaseKeepAll {
		return
	}
	rows, err := server.store.ListCleanableAppReleases(ctx, int64(max(settings.AppReleaseKeep, 0)))
	if err != nil {
		log.Printf("app release clean-up: %v", err)
		return
	}
	for _, r := range rows {
		// The row first, so no till is ever sent to a file that is gone.
		if err := server.store.MarkAppReleaseFileRemoved(ctx, r.ID); err != nil {
			log.Printf("app release clean-up: %s %s: %v", r.Flavor, r.Version, err)
			continue
		}
		if err := os.RemoveAll(filepath.Dir(filepath.Join(server.appReleaseStorageDir(), r.FilePath))); err != nil {
			log.Printf("app release clean-up: %s %s: %v", r.Flavor, r.Version, err)
		}
	}
	if len(rows) > 0 {
		log.Printf("app release clean-up: removed %d build file(s)", len(rows))
		server.adminHub.broadcastAll(adminWSMessage{Type: "app_update_changed"})
	}
}

type setAppReleaseKeepRequest struct {
	Keep *int32 `json:"keep" binding:"required"`
}

// setAppReleaseKeep is the App Updates page's "Keep old builds".
func (server *Server) setAppReleaseKeep(ctx *gin.Context) {
	var req setAppReleaseKeepRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	keep := *req.Keep
	if keep < appReleaseKeepAll || keep > appReleaseKeepMax {
		server.writeError(ctx, http.StatusBadRequest, fmt.Errorf("keep must be -1 (all), 0 (none) or up to %d", appReleaseKeepMax))
		return
	}
	if _, err := server.store.SetAppReleaseKeep(ctx, keep); err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	server.cleanAppReleasesSoon()
	server.adminHub.broadcastAll(adminWSMessage{Type: "app_update_changed"})
	server.writeJSON(ctx, http.StatusOK, envelope{"keep": keep})
}
