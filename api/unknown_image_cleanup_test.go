package api

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// Deleting a picture also removes the thumbnails cached beside it, and only
// that picture's.
func TestRemoveStoredImageTakesItsThumbnails(t *testing.T) {
	dir := t.TempDir()
	mine, other := filepath.Join(dir, "a.jpg"), filepath.Join(dir, "b.jpg")
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".thumb"), 0o755))
	mineFiles := []string{
		mine, thumbCachePath(mine, 96, 0), thumbCachePath(mine, 480, 0), thumbCachePath(mine, 0, 90),
	}
	for _, f := range append([]string{other, thumbCachePath(other, 96, 0)}, mineFiles...) {
		require.NoError(t, os.WriteFile(f, []byte("x"), 0o644))
	}

	removeStoredImage(mine)

	for _, f := range mineFiles {
		require.NoFileExists(t, f)
	}
	require.FileExists(t, other)
	require.FileExists(t, thumbCachePath(other, 96, 0))
}
