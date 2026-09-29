package api

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/mhdna/kashi/util/imageopt"
)

const (
	thumbMinDim             = 64
	thumbMaxDim             = 1600
	thumbWebPQuality        = 70
	storedImageCacheControl = "private, max-age=86400"
)

// serveStoredImage streams an on-disk upload under the admin Bearer gate,
// with browser caching turned on (ctx.File adds Last-Modified / range
// support on top).
//
// With ?w=<px> it serves a cached, long-edge-capped WebP thumbnail instead
// of the multi-MB original -- the Storage grid requests one per tile so it
// isn't pulling ~50-100MB of full-resolution phone photos to paint
// 120px squares; the full-size preview omits ?w. Anything the shared imageopt
// decoders (JPEG/PNG/WebP) can't read (e.g. HEIC) transparently falls back to
// the original bytes.
func (server *Server) serveStoredImage(ctx *gin.Context, filePath, contentType string) {
	ctx.Header("Cache-Control", storedImageCacheControl)

	w := parseThumbWidth(ctx.Query("w"))
	if w == 0 {
		ctx.Header("Content-Type", contentType)
		ctx.File(filePath)
		return
	}

	thumbPath := thumbCachePath(filePath, w)
	if _, err := os.Stat(thumbPath); err != nil {
		if genErr := generateThumbnail(filePath, thumbPath, w); genErr != nil {
			// Undecodable (HEIC) or a transient FS error -- the original is
			// always a correct answer.
			ctx.Header("Content-Type", contentType)
			ctx.File(filePath)
			return
		}
	}
	ctx.Header("Content-Type", imageopt.ContentType)
	ctx.File(thumbPath)
}

// parseThumbWidth returns 0 ("serve the original") for a missing or
// unparseable value, otherwise the requested long-edge cap clamped to a
// sane range.
func parseThumbWidth(raw string) int {
	if raw == "" {
		return 0
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		return 0
	}
	if n < thumbMinDim {
		return thumbMinDim
	}
	if n > thumbMaxDim {
		return thumbMaxDim
	}
	return n
}

// thumbCachePath keeps generated thumbnails in a .thumb/ subdir beside the
// original, keyed by cap width. Uploads are immutable (uuid filenames, one
// photo per parent row), so a cached thumbnail never goes stale.
func thumbCachePath(filePath string, w int) string {
	return filepath.Join(filepath.Dir(filePath), ".thumb", strconv.Itoa(w)+"_"+filepath.Base(filePath)+".webp")
}

func generateThumbnail(srcPath, dstPath string, maxDim int) error {
	data, err := os.ReadFile(srcPath)
	if err != nil {
		return err
	}

	// Decode also applies EXIF orientation and caps the long edge at maxDim.
	img, _, err := imageopt.Decode(data, maxDim)
	if err != nil {
		return err
	}
	if b := img.Bounds(); b.Dx() <= 0 || b.Dy() <= 0 {
		return fmt.Errorf("thumbnail: empty source image")
	}

	encoded, err := imageopt.EncodeWebP(img, thumbWebPQuality)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(dstPath), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(dstPath), "thumb-*.webp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)

	if _, err := tmp.Write(encoded); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, dstPath)
}
