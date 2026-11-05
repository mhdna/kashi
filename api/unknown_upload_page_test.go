package api

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// The phone page's refresh rebuilds its cards, so the camera input must not be
// one of the things it builds: an input removed while the camera is open can
// lose the picture, leaving the item on "Needs a picture".
func TestUnknownUploadPageKeepsCameraInputOutOfTheCards(t *testing.T) {
	page := string(unknownUploadPageHTML)
	require.Equal(t, 1, strings.Count(page, `<input type="file" id="camera"`))
	require.NotContains(t, page, "el('input')")
	require.Less(t, strings.Index(page, `<div id="list">`), strings.Index(page, `id="camera"`))
}
