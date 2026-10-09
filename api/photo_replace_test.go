package api

import (
	"bytes"
	"image"
	"image/png"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	mockdb "github.com/mhdna/kashi/db/mock"
	db "github.com/mhdna/kashi/db/sqlc"
	"github.com/stretchr/testify/require"
)

func newReplacePhotoRequest(t *testing.T, url string, imageBytes []byte) *http.Request {
	t.Helper()
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	part, err := mw.CreatePart(map[string][]string{
		"Content-Disposition": {`form-data; name="image"; filename="new.png"`},
		"Content-Type":        {"image/png"},
	})
	require.NoError(t, err)
	_, err = part.Write(imageBytes)
	require.NoError(t, err)
	require.NoError(t, mw.Close())

	req, err := http.NewRequest(http.MethodPut, url, &body)
	require.NoError(t, err)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	return req
}

// The row is pointed at a new file beside the old one, and the old file and
// its cached thumbnails are gone.
func TestReplaceBranchExpenseImage(t *testing.T) {
	dir := t.TempDir()
	oldPath := filepath.Join(dir, "old.webp")
	require.NoError(t, os.WriteFile(oldPath, []byte("old"), 0o644))
	thumb := thumbCachePath(oldPath, 480, 0)
	turned := thumbCachePath(oldPath, 0, 90)
	require.NoError(t, os.MkdirAll(filepath.Dir(thumb), 0o755))
	require.NoError(t, os.WriteFile(thumb, []byte("t"), 0o644))
	require.NoError(t, os.WriteFile(turned, []byte("t"), 0o644))

	var newPNG bytes.Buffer
	require.NoError(t, png.Encode(&newPNG, image.NewRGBA(image.Rect(0, 0, 10, 10))))

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	store := mockdb.NewMockStore(ctrl)
	store.EXPECT().GetBranchExpenseImage(gomock.Any(), int64(7)).
		Return(db.BranchExpenseImage{ID: 7, BranchExpenseID: 3, FilePath: oldPath, Rotation: 90}, nil)
	var newPath string
	store.EXPECT().ReplaceBranchExpenseImageFile(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ any, arg db.ReplaceBranchExpenseImageFileParams) (db.BranchExpenseImage, error) {
			newPath = arg.FilePath
			return db.BranchExpenseImage{ID: 7, BranchExpenseID: 3, FilePath: arg.FilePath, ContentType: arg.ContentType}, nil
		})
	store.EXPECT().GetBranchExpense(gomock.Any(), int64(3)).Return(db.BranchExpense{ID: 3, BranchID: 1}, nil)

	server := newTestServer(t, store)
	recorder := httptest.NewRecorder()
	req := newReplacePhotoRequest(t, "/branch_expense_images/7/replace", newPNG.Bytes())
	addAuthorization(t, req, server.tokenMaker, authorizationTypeBearer, "user", time.Minute)
	server.router.ServeHTTP(recorder, req)

	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	require.Equal(t, dir, filepath.Dir(newPath))
	require.FileExists(t, newPath)
	require.NoFileExists(t, oldPath)
	require.NoFileExists(t, thumb)
	require.NoFileExists(t, turned)
}

func TestReplaceTransferAttachment_NoPhoto(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	store := mockdb.NewMockStore(ctrl)
	store.EXPECT().GetTransfer(gomock.Any(), int64(4)).Return(db.Transfer{ID: 4}, nil)

	server := newTestServer(t, store)
	recorder := httptest.NewRecorder()
	req := newReplacePhotoRequest(t, "/transfers/4/attachment/replace", []byte("x"))
	addAuthorization(t, req, server.tokenMaker, authorizationTypeBearer, "user", time.Minute)
	server.router.ServeHTTP(recorder, req)

	require.Equal(t, http.StatusNotFound, recorder.Code)
}
