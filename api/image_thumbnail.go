package api

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/disintegration/imaging"
	"github.com/gin-gonic/gin"
	"github.com/mhdna/kashi/util/imageopt"
)

const (
	thumbMinDim             = 64
	thumbMaxDim             = 1600
	thumbWebPQuality        = 70
	rotatedWebPQuality      = 90
	storedImageCacheControl = "private, no-cache"
)

// serveStoredImage streams an on-disk upload under the admin Bearer gate
// (ctx.File adds Last-Modified / range support on top).
//
// With ?w=<px> it serves a cached, long-edge-capped WebP thumbnail instead
// of the multi-MB original -- the Storage grid requests one per tile so it
// isn't pulling ~50-100MB of full-resolution phone photos to paint
// 120px squares; the full-size preview omits ?w. Anything the shared imageopt
// decoders (JPEG/PNG/WebP) can't read (e.g. HEIC) transparently falls back to
// the original bytes.
//
// rotation is the photo's saved clockwise turn (0/90/180/270). A turned
// photo, thumbnail or full size, is served from its own cached WebP; the
// upload itself is never rewritten. Because rotating changes what this same
// URL returns, the browser must revalidate on every use (no-cache) and the
// ETag names the rotation, so a stale cached copy is never shown.
func (server *Server) serveStoredImage(ctx *gin.Context, filePath, contentType string, rotation int16) {
	w := parseThumbWidth(ctx.Query("w"))
	etag := fmt.Sprintf(`"%s-w%d-r%d"`, filepath.Base(filePath), w, rotation)
	ctx.Header("Cache-Control", storedImageCacheControl)
	ctx.Header("ETag", etag)
	if ctx.GetHeader("If-None-Match") == etag {
		ctx.Status(http.StatusNotModified)
		return
	}

	if w == 0 && rotation == 0 {
		ctx.Header("Content-Type", contentType)
		ctx.File(filePath)
		return
	}

	maxDim, quality := w, thumbWebPQuality
	if w == 0 {
		// A turned full-size photo: re-encoded once, close to the original.
		maxDim, quality = imageopt.MaxDim, rotatedWebPQuality
	}
	cachedPath := thumbCachePath(filePath, w, rotation)
	if _, err := os.Stat(cachedPath); err != nil {
		if genErr := generateThumbnail(filePath, cachedPath, maxDim, quality, rotation); genErr != nil {
			// Undecodable (HEIC) or a transient FS error -- the original is
			// always a correct answer, just not turned.
			ctx.Header("Content-Type", contentType)
			ctx.File(filePath)
			return
		}
	}
	ctx.Header("Content-Type", imageopt.ContentType)
	ctx.File(cachedPath)
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

// thumbCachePath keeps generated thumbnails and turned copies in a .thumb/
// subdir beside the original, keyed by cap width (0 = full size) and
// rotation. Uploads are immutable (uuid filenames, one photo per parent row)
// and a rotation picks a different file, so a cached copy never goes stale.
// An unturned thumbnail keeps the name it had before rotations existed.
func thumbCachePath(filePath string, w int, rotation int16) string {
	name := strconv.Itoa(w) + "_" + filepath.Base(filePath) + ".webp"
	if w == 0 {
		name = "full_" + filepath.Base(filePath) + ".webp"
	}
	if rotation != 0 {
		name = "r" + strconv.Itoa(int(rotation)) + "_" + name
	}
	return filepath.Join(filepath.Dir(filePath), ".thumb", name)
}

func generateThumbnail(srcPath, dstPath string, maxDim, quality int, rotation int16) error {
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

	// imaging rotates counter-clockwise; rotation is clockwise.
	switch rotation {
	case 90:
		img = imaging.Rotate270(img)
	case 180:
		img = imaging.Rotate180(img)
	case 270:
		img = imaging.Rotate90(img)
	}

	encoded, err := imageopt.EncodeWebP(img, quality)
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

// The long edges the till asks for: list rows (96) and its detail view
// (detailPhotoWidth, see kashi-pos). Baking both at upload time means the
// first view of a photo reads a small cached file instead of decoding the
// original on the request.
var warmThumbWidths = []int{96, 1280}

// warmThumbnails generates the common thumbnails for a freshly stored photo
// in the background. Failures are ignored: serveStoredImage generates the same
// file on demand, and falls back to the original if it can't be decoded.
func warmThumbnails(filePath string) {
	go func() {
		for _, w := range warmThumbWidths {
			dst := thumbCachePath(filePath, w, 0)
			if _, err := os.Stat(dst); err == nil {
				continue
			}
			_ = generateThumbnail(filePath, dst, w, thumbWebPQuality, 0)
		}
	}()
}
