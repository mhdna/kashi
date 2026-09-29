// Package imageopt turns arbitrary uploaded photos into small, good-looking
// WebP files. It is shared by the upload handlers (api) and the one-off
// backfill command (cmd/optimize_images) so both produce identical output.
package imageopt

import (
	"bytes"
	"fmt"
	"image"
	_ "image/jpeg" // register the JPEG decoder for image.Decode
	_ "image/png"  // register the PNG decoder for image.Decode
	"log"

	"github.com/disintegration/imaging"
	"github.com/gen2brain/webp"
	_ "golang.org/x/image/webp" // register the WebP decoder for image.Decode
)

const (
	// MaxDim caps the long edge. 1920px keeps small print on a receipt
	// legible while shrinking a 12MP phone photo to a fraction of its size.
	MaxDim = 1920
	// Quality is libwebp's lossy 0-100 scale; ~75 is the "good, not
	// pristine" knee -- visually close to the source at roughly a quarter of
	// the JPEG bytes.
	Quality = 75
	// maxDecodePixels rejects decompression bombs before the full decode
	// allocates width*height*4 bytes for them.
	maxDecodePixels = 100_000_000

	ContentType = "image/webp"
	Ext         = ".webp"
)

// encodeSlots bounds concurrent decode+resize+encode jobs: each holds the
// full decoded bitmap (~50MB for a 12MP photo) plus the WASM encoder's
// working memory, so a burst of uploads shouldn't all run at once.
var encodeSlots = make(chan struct{}, 2)

// Result is what a caller persists: the bytes to write and the metadata that
// must describe *those* bytes (not the original upload).
type Result struct {
	Data        []byte
	ContentType string
	Ext         string
	// Optimized is false when the original bytes were kept unchanged.
	Optimized bool
}

// Optimize decodes the photo (honouring EXIF orientation, which a re-encode
// would otherwise drop and leave phone photos sideways), downsizes it to
// MaxDim, and re-encodes it as lossy WebP.
//
// It never makes an image worse: if the result isn't smaller than the
// original, or the format can't be decoded here (e.g. HEIC), the original
// bytes come back unchanged under their declared type.
func Optimize(original []byte, declaredContentType, declaredExt string) Result {
	keep := Result{Data: original, ContentType: declaredContentType, Ext: declaredExt}

	img, resized, err := Decode(original, MaxDim)
	if err != nil {
		log.Printf("imageopt: keeping original (%s, %d bytes): %v", declaredContentType, len(original), err)
		return keep
	}

	encoded, err := EncodeWebP(img, Quality)
	if err != nil {
		log.Printf("imageopt: webp encode failed, keeping original: %v", err)
		return keep
	}

	// Untouched dimensions + no size win means the image was already
	// efficient (e.g. a small WebP); re-encoding only adds generation loss.
	if !resized && len(encoded) >= len(original) {
		return keep
	}

	return Result{Data: encoded, ContentType: ContentType, Ext: Ext, Optimized: true}
}

// Decode returns the decoded, orientation-corrected image, downscaled if its
// long edge exceeds maxDim; resized reports whether that downscale happened.
func Decode(data []byte, maxDim int) (img image.Image, resized bool, err error) {
	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return nil, false, err
	}
	if cfg.Width <= 0 || cfg.Height <= 0 || cfg.Width*cfg.Height > maxDecodePixels {
		return nil, false, fmt.Errorf("image dimensions %dx%d out of range", cfg.Width, cfg.Height)
	}

	img, err = imaging.Decode(bytes.NewReader(data), imaging.AutoOrientation(true))
	if err != nil {
		return nil, false, err
	}

	b := img.Bounds()
	if b.Dx() > maxDim || b.Dy() > maxDim {
		// Fit inside the square; imaging keeps the aspect ratio.
		img = imaging.Fit(img, maxDim, maxDim, imaging.Lanczos)
		resized = true
	}
	return img, resized, nil
}

// EncodeWebP encodes img as lossy WebP, holding one of the bounded encode
// slots for the duration.
func EncodeWebP(img image.Image, quality int) ([]byte, error) {
	encodeSlots <- struct{}{}
	defer func() { <-encodeSlots }()

	var buf bytes.Buffer
	if err := webp.Encode(&buf, img, webp.Options{Quality: quality, Method: 4}); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
