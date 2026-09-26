package api

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"math/rand"
	"mime/multipart"
	"testing"

	"github.com/mhdna/kashi/util/imageopt"
	"github.com/stretchr/testify/require"
	_ "golang.org/x/image/webp"
)

// testFileHeader round-trips data through a real multipart body so the
// header behaves exactly as it does in a handler.
func testFileHeader(t *testing.T, data []byte) *multipart.FileHeader {
	t.Helper()
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	part, err := w.CreateFormFile("image", "photo.jpg")
	require.NoError(t, err)
	_, err = part.Write(data)
	require.NoError(t, err)
	require.NoError(t, w.Close())

	form, err := multipart.NewReader(&body, w.Boundary()).ReadForm(1 << 20)
	require.NoError(t, err)
	return form.File["image"][0]
}

// noisyJPEG is photo-like: random noise over a gradient, so it neither
// compresses to nothing nor is trivially flat.
func noisyJPEG(t *testing.T, w, h int) []byte {
	t.Helper()
	rng := rand.New(rand.NewSource(1))
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{
				R: uint8((x*255/w + rng.Intn(40)) % 256),
				G: uint8((y*255/h + rng.Intn(40)) % 256),
				B: uint8(rng.Intn(256)),
				A: 255,
			})
		}
	}
	var buf bytes.Buffer
	require.NoError(t, jpeg.Encode(&buf, img, &jpeg.Options{Quality: 95}))
	return buf.Bytes()
}

// withEXIFOrientation splices a minimal APP1/EXIF segment carrying the given
// orientation right after the JPEG SOI marker.
func withEXIFOrientation(jpegData []byte, orientation byte) []byte {
	exif := []byte{
		'E', 'x', 'i', 'f', 0, 0, // identifier
		'M', 'M', 0, 42, 0, 0, 0, 8, // big-endian TIFF header, IFD0 at offset 8
		0, 1, // one IFD entry
		0x01, 0x12, 0, 3, 0, 0, 0, 1, 0, orientation, 0, 0, // Orientation, SHORT, count 1
		0, 0, 0, 0, // no next IFD
	}
	seg := []byte{0xFF, 0xE1, byte((len(exif) + 2) >> 8), byte(len(exif) + 2)}
	seg = append(seg, exif...)
	out := append([]byte{}, jpegData[:2]...)
	out = append(out, seg...)
	return append(out, jpegData[2:]...)
}

func decodeDims(t *testing.T, data []byte) (int, int, string) {
	t.Helper()
	cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
	require.NoError(t, err)
	return cfg.Width, cfg.Height, format
}

func TestOptimizeUploadedImage_LargePhotoBecomesSmallerWebP(t *testing.T) {
	original := noisyJPEG(t, 3200, 2400)

	got, err := optimizeUploadedImage(testFileHeader(t, original), "image/jpeg", ".jpg")
	require.NoError(t, err)

	require.Equal(t, "image/webp", got.ContentType)
	require.Equal(t, ".webp", got.Ext)
	require.Less(t, len(got.Data), len(original)/2, "expected a substantial size reduction")

	w, h, format := decodeDims(t, got.Data)
	require.Equal(t, "webp", format)
	require.Equal(t, imageopt.MaxDim, w)
	require.Equal(t, 1440, h, "aspect ratio must be preserved")
}

func TestOptimizeUploadedImage_AppliesEXIFOrientation(t *testing.T) {
	// 2400x1200 landscape flagged "rotate 90° CW" must come out portrait.
	original := withEXIFOrientation(noisyJPEG(t, 2400, 1200), 6)

	got, err := optimizeUploadedImage(testFileHeader(t, original), "image/jpeg", ".jpg")
	require.NoError(t, err)

	w, h, _ := decodeDims(t, got.Data)
	require.Greater(t, h, w, "phone photo must not end up sideways")
}

func TestOptimizeUploadedImage_UndecodableIsStoredUnchanged(t *testing.T) {
	junk := []byte("this is not an image, e.g. an unsupported HEIC")

	got, err := optimizeUploadedImage(testFileHeader(t, junk), "image/heic", ".heic")
	require.NoError(t, err)

	require.Equal(t, junk, got.Data)
	require.Equal(t, "image/heic", got.ContentType)
	require.Equal(t, ".heic", got.Ext)
}

func TestOptimizeUploadedImage_NeverGrowsAnAlreadySmallImage(t *testing.T) {
	// A tiny flat JPEG is smaller than any WebP wrapper would justify.
	img := image.NewRGBA(image.Rect(0, 0, 8, 8))
	var buf bytes.Buffer
	require.NoError(t, jpeg.Encode(&buf, img, &jpeg.Options{Quality: 50}))

	got, err := optimizeUploadedImage(testFileHeader(t, buf.Bytes()), "image/jpeg", ".jpg")
	require.NoError(t, err)

	require.LessOrEqual(t, len(got.Data), buf.Len())
}
