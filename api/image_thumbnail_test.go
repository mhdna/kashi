package api

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
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

	dstPath := filepath.Join(dir, ".thumb", "480_orig.jpg")
	require.NoError(t, generateThumbnail(srcPath, dstPath, 480))

	f, err := os.Open(dstPath)
	require.NoError(t, err)
	defer f.Close()
	cfg, format, err := image.DecodeConfig(f)
	require.NoError(t, err)
	require.Equal(t, "jpeg", format)
	require.Equal(t, 480, cfg.Width, "long edge capped to the requested width")
	require.Equal(t, 240, cfg.Height, "aspect ratio preserved")

	orig, err := os.Stat(srcPath)
	require.NoError(t, err)
	thumb, err := os.Stat(dstPath)
	require.NoError(t, err)
	require.Less(t, thumb.Size(), orig.Size(), "thumbnail is smaller than the original")
}

func TestGenerateThumbnail_RejectsNonImage(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "not-an-image.heic")
	require.NoError(t, os.WriteFile(srcPath, []byte("not really an image"), 0o644))

	// Caller treats any error here as "serve the original" -- we only need
	// it to fail rather than panic or write a broken file.
	require.Error(t, generateThumbnail(srcPath, filepath.Join(dir, ".thumb", "x.jpg"), 480))
}
