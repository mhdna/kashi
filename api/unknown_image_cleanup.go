package api

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	db "github.com/mhdna/kashi/db/sqlc"
)

const (
	unknownImageCleanupEvery = 24 * time.Hour
	// A matched unit's picture is kept this long, for checking a match or
	// settling a dispute; a checkout that never became a sale loses its
	// pictures this long after its link expired.
	unknownResolvedKeep  = 30 * 24 * time.Hour
	unknownAbandonedKeep = 2 * 24 * time.Hour
)

// removeStoredImage deletes a stored picture and the thumbnails and turned
// copies cached beside it (.thumb/<...>_<file name>.webp).
func removeStoredImage(filePath string) {
	_ = os.Remove(filePath)
	cached, _ := filepath.Glob(filepath.Join(filepath.Dir(filePath), ".thumb", "*_"+filepath.Base(filePath)+".webp"))
	for _, path := range cached {
		_ = os.Remove(path)
	}
}

// deleteUnknownImageRows drops the rows first (so nothing points at a file
// that is gone) and then the files. A unit keeps its match; it only loses the
// picture.
func (server *Server) deleteUnknownImageRows(ctx context.Context, ids []int64, paths []string) error {
	if len(ids) == 0 {
		return nil
	}
	if err := server.store.DeleteUnknownItemImages(ctx, ids); err != nil {
		return err
	}
	for _, path := range paths {
		if strings.TrimSpace(path) != "" {
			removeStoredImage(path)
		}
	}
	return nil
}

type deleteUnknownItemImagesRequest struct {
	IDs []int64 `json:"ids" binding:"required,min=1,max=200,dive,min=1"`
}

// deleteUnknownItemImages is the Attachments page's Delete. Pictures that
// still matter (a unit of an arrived sale waiting to be matched) are kept and counted.
func (server *Server) deleteUnknownItemImages(ctx *gin.Context) {
	var req deleteUnknownItemImagesRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	rows, err := server.store.ListDeletableUnknownItemImages(ctx, req.IDs)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	ids := make([]int64, 0, len(rows))
	paths := make([]string, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.ID)
		paths = append(paths, row.FilePath)
	}
	if err := server.deleteUnknownImageRows(ctx, ids, paths); err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	// deleted_ids lets the page drop just those pictures instead of loading
	// the whole grid again.
	server.writeJSON(ctx, http.StatusOK, envelope{"deleted": len(ids), "kept": len(req.IDs) - len(ids), "deleted_ids": ids})
}

// RunUnknownImageCleanup deletes old unknown-item pictures once a day, when
// the app setting unknown_images_auto_delete is on (off by default).
func (server *Server) RunUnknownImageCleanup(ctx context.Context) {
	ticker := time.NewTicker(unknownImageCleanupEvery)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := server.cleanUnknownImages(ctx); err != nil {
				log.Printf("unknown image clean-up: %v", err)
			}
		}
	}
}

func (server *Server) cleanUnknownImages(ctx context.Context) error {
	settings, err := server.store.GetAppSettings(ctx)
	if err != nil {
		return err
	}
	if !settings.UnknownImagesAutoDelete {
		return nil
	}
	now := time.Now().UTC()
	rows, err := server.store.ListAutoDeletableUnknownItemImages(ctx, db.ListAutoDeletableUnknownItemImagesParams{
		ResolvedBefore:  now.Add(-unknownResolvedKeep),
		AbandonedBefore: now.Add(-unknownAbandonedKeep),
	})
	if err != nil {
		return err
	}
	ids := make([]int64, 0, len(rows))
	paths := make([]string, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.ID)
		paths = append(paths, row.FilePath)
	}
	if err := server.deleteUnknownImageRows(ctx, ids, paths); err != nil {
		return errors.Join(errors.New("delete"), err)
	}
	if len(ids) > 0 {
		log.Printf("unknown image clean-up: removed %d pictures", len(ids))
	}
	return nil
}
