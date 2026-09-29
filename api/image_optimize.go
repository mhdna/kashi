package api

import (
	"io"
	"mime/multipart"

	"github.com/mhdna/kashi/util/imageopt"
)

// storedImage is what an upload handler persists: the bytes to write and
// the metadata that must describe *those* bytes (not the original upload).
type storedImage = imageopt.Result

// optimizeUploadedImage is the single gate every image upload goes through:
// see imageopt.Optimize for what it does and when it keeps the original.
func optimizeUploadedImage(fileHeader *multipart.FileHeader, declaredContentType, declaredExt string) (storedImage, error) {
	src, err := fileHeader.Open()
	if err != nil {
		return storedImage{}, err
	}
	defer src.Close()

	original, err := io.ReadAll(src)
	if err != nil {
		return storedImage{}, err
	}
	return imageopt.Optimize(original, declaredContentType, declaredExt), nil
}
