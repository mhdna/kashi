package api

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	_ "golang.org/x/image/webp"
)

func TestParseThumbWidth(t *testing.T) {
	require.Equal(t, 0, parseThumbWidth(""))
	require.Equal(t, 0, parseThumbWidth("nope"))
	require.Equal(t, 0, parseThumbWidth("0"))
	require.Equal(t, 0, parseThumbWidth("-5"))
	require.Equal(t, thumbMinDim, parseThumbWidth("10"))
	require.Equal(t, 480, parseThumbWidth("480"))
	require.Equal(t, thumbMaxDim, parseThumbWidth("99999"))
}

func TestGenerateThumbnail(t *testing.T) {
	dir := t.TempDir()

	// A 2000x1000 JPEG stands in for a full-size phone photo.
	big := image.NewRGBA(image.Rect(0, 0, 2000, 1000))
	for y := 0; y < 1000; y++ {
		for x := 0; x < 2000; x++ {
			big.Set(x, y, color.RGBA{R: uint8(x % 256), G: uint8(y % 256), B: 128, A: 255})
		}
	}
	srcPath := filepath.Join(dir, "orig.jpg")
	var buf bytes.Buffer
	require.NoError(t, jpeg.Encode(&buf, big, &jpeg.Options{Quality: 95}))
	require.NoError(t, os.WriteFile(srcPath, buf.Bytes(), 0o644))

	dstPath := filepath.Join(dir, ".thumb", "480_orig.jpg.webp")
	require.NoError(t, generateThumbnail(srcPath, dstPath, 480, thumbWebPQuality, 0))

	f, err := os.Open(dstPath)
	require.NoError(t, err)
	defer f.Close()
	cfg, format, err := image.DecodeConfig(f)
	require.NoError(t, err)
	require.Equal(t, "webp", format)
	require.Equal(t, 480, cfg.Width, "long edge capped to the requested width")
	require.Equal(t, 240, cfg.Height, "aspect ratio preserved")

	orig, err := os.Stat(srcPath)
	require.NoError(t, err)
	thumb, err := os.Stat(dstPath)
	require.NoError(t, err)
	require.Less(t, thumb.Size(), orig.Size(), "thumbnail is smaller than the original")
}

func TestGenerateThumbnail_Rotated(t *testing.T) {
	dir := t.TempDir()

	// Landscape, with the left half red: a clockwise quarter turn makes it
	// portrait with the red half on top.
	src := image.NewRGBA(image.Rect(0, 0, 400, 200))
	for y := 0; y < 200; y++ {
		for x := 0; x < 400; x++ {
			c := color.RGBA{B: 255, A: 255}
			if x < 200 {
				c = color.RGBA{R: 255, A: 255}
			}
			src.Set(x, y, c)
		}
	}
	srcPath := filepath.Join(dir, "orig.png")
	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, src))
	require.NoError(t, os.WriteFile(srcPath, buf.Bytes(), 0o644))

	dstPath := thumbCachePath(srcPath, 0, 90)
	require.Equal(t, filepath.Join(dir, ".thumb", "r90_full_orig.png.webp"), dstPath)
	require.NoError(t, generateThumbnail(srcPath, dstPath, 1920, rotatedWebPQuality, 90))

	f, err := os.Open(dstPath)
	require.NoError(t, err)
	defer f.Close()
	out, _, err := image.Decode(f)
	require.NoError(t, err)
	require.Equal(t, 200, out.Bounds().Dx())
	require.Equal(t, 400, out.Bounds().Dy())
	top, _, _, _ := out.At(100, 50).RGBA()
	bottom, _, _, _ := out.At(100, 350).RGBA()
	require.Greater(t, top, bottom, "the left (red) half ends up on top")
}

func TestGenerateThumbnail_RejectsNonImage(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "not-an-image.heic")
	require.NoError(t, os.WriteFile(srcPath, []byte("not really an image"), 0o644))

	// Caller treats any error here as "serve the original" -- we only need
	// it to fail rather than panic or write a broken file.
	require.Error(t, generateThumbnail(srcPath, filepath.Join(dir, ".thumb", "x.jpg"), 480, thumbWebPQuality, 0))
}

// A browser holding the photo as it was before a turn must get the turned
// one, not a 304: the ETag names the rotation.
func TestServeStoredImage_RotationChangesETag(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "orig.png")
	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, image.NewRGBA(image.Rect(0, 0, 40, 20))))
	require.NoError(t, os.WriteFile(srcPath, buf.Bytes(), 0o644))

	server := &Server{}
	serve := func(rotation int16, ifNoneMatch string) *httptest.ResponseRecorder {
		recorder := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(recorder)
		ctx.Request = httptest.NewRequest(http.MethodGet, "/file", nil)
		if ifNoneMatch != "" {
			ctx.Request.Header.Set("If-None-Match", ifNoneMatch)
		}
		server.serveStoredImage(ctx, srcPath, "image/png", rotation)
		ctx.Writer.WriteHeaderNow() // the router does this after every handler
		return recorder
	}

	upright := serve(0, "")
	require.Equal(t, http.StatusOK, upright.Code)
	etag := upright.Header().Get("ETag")
	require.Equal(t, "image/png", upright.Header().Get("Content-Type"))

	require.Equal(t, http.StatusNotModified, serve(0, etag).Code)

	turned := serve(90, etag)
	require.Equal(t, http.StatusOK, turned.Code)
	require.NotEqual(t, etag, turned.Header().Get("ETag"))
	require.Equal(t, "image/webp", turned.Header().Get("Content-Type"))
	cfg, _, err := image.DecodeConfig(turned.Body)
	require.NoError(t, err)
	require.Equal(t, 20, cfg.Width)
	require.Equal(t, 40, cfg.Height)
}
