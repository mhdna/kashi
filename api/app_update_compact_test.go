package api

import (
	"bytes"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/klauspost/compress/zstd"
	mockdb "github.com/mhdna/kashi/db/mock"
	db "github.com/mhdna/kashi/db/sqlc"
	"github.com/stretchr/testify/require"
)

func hexSum(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func zstdBytes(t *testing.T, b []byte) []byte {
	t.Helper()
	enc, err := zstd.NewWriter(nil)
	require.NoError(t, err)
	return enc.EncodeAll(b, nil)
}

func compressedUploadRequest(t *testing.T, server *Server, file []byte, exeSum, fileSum string) *http.Request {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	for k, v := range map[string]string{
		"version": "1.2.0", "flavor": "standard", "sha256": exeSum, "file_sha256": fileSum,
		"encoding": "zstd", "signature": base64.StdEncoding.EncodeToString(make([]byte, 64)),
	} {
		require.NoError(t, w.WriteField(k, v))
	}
	part, err := w.CreateFormFile("file", "GB-Cloud.exe")
	require.NoError(t, err)
	_, err = part.Write(file)
	require.NoError(t, err)
	require.NoError(t, w.Close())
	req, err := http.NewRequest(http.MethodPost, "/app_releases", &buf)
	require.NoError(t, err)
	req.Header.Set("Content-Type", w.FormDataContentType())
	addAuthorization(t, req, server.tokenMaker, authorizationTypeBearer, "admin", time.Minute)
	return req
}

// A compressed upload is checked against the exe inside it, and stored with
// both sizes: the exe's for the till's check, the file's for the download.
func TestCreateAppRelease_Zstd(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	server := newAppUpdateTestServer(t, store)
	exe := bytes.Repeat([]byte("MZ fake windows program "), 1000)
	packed := zstdBytes(t, exe)

	// The exe's checksum is what has to match what was signed.
	rec := httptest.NewRecorder()
	server.router.ServeHTTP(rec, compressedUploadRequest(t, server, packed, hexSum([]byte("other")), hexSum(packed)))
	require.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())

	store.EXPECT().CreateAppRelease(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ any, arg db.CreateAppReleaseParams) (db.AppRelease, error) {
			require.Equal(t, "zstd", arg.Encoding)
			require.Equal(t, hexSum(exe), arg.Sha256)
			require.Equal(t, int64(len(exe)), arg.SizeBytes)
			require.Equal(t, hexSum(packed), arg.FileSha256)
			require.Equal(t, int64(len(packed)), arg.FileSizeBytes)
			return db.AppRelease{ID: 1, Version: arg.Version, Flavor: arg.Flavor}, nil
		})
	rec = httptest.NewRecorder()
	server.router.ServeHTTP(rec, compressedUploadRequest(t, server, packed, hexSum(exe), hexSum(packed)))
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
}

// A till that understands zstd and patches is told about both; an older one
// gets the plain exe, unpacked from the stored file.
func TestBranchAppUpdate_ZstdAndPatch(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	stubBranchAuth(store)
	server := newAppUpdateTestServer(t, store)

	exe := bytes.Repeat([]byte("new build "), 500)
	packed := zstdBytes(t, exe)
	rel := filepath.Join("abc", appReleaseStoredName)
	dir := filepath.Join(server.appReleaseStorageDir(), "abc")
	require.NoError(t, os.MkdirAll(dir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(server.appReleaseStorageDir(), rel), packed, 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "patch-1"), []byte("PATCH"), 0o644))
	release := db.GetRegisterTargetReleaseRow{
		ID: 3, Version: "1.2.0", Flavor: "standard", FileName: "GB-Cloud.exe", FilePath: rel,
		SizeBytes: int64(len(exe)), Sha256: hexSum(exe), Encoding: "zstd", FileSizeBytes: int64(len(packed)),
		CreatedAt: time.Now(),
	}
	store.EXPECT().GetRegisterTargetRelease(gomock.Any(), testRegisterID).Return(release, nil).AnyTimes()
	have := hexSum([]byte("old build"))
	store.EXPECT().GetAppReleasePatchForBase(gomock.Any(), db.GetAppReleasePatchForBaseParams{ReleaseID: 3, BaseSha256: have}).
		Return(db.AppReleasePatch{ID: 9, ReleaseID: 3, BaseSha256: have, SizeBytes: 5, FilePath: filepath.Join("abc", "patch-1")}, nil)

	manifest := func(query string) map[string]any {
		t.Helper()
		rec := httptest.NewRecorder()
		server.router.ServeHTTP(rec, newBranchRequest(t, http.MethodGet, "/branch/app_update"+query, nil))
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		var body struct {
			Release map[string]any `json:"release"`
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		return body.Release
	}
	old := manifest("")
	require.NotContains(t, old, "encoding")
	require.NotContains(t, old, "patch")

	got := manifest("?accept=zstd,patch&have=" + have)
	require.Equal(t, "zstd", got["encoding"])
	require.EqualValues(t, len(packed), got["download_bytes"])
	require.EqualValues(t, 9, got["patch"].(map[string]any)["id"])

	download := func(path string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		server.router.ServeHTTP(rec, newBranchRequest(t, http.MethodGet, path, nil))
		return rec
	}
	rec := download("/branch/app_update/download/3")
	require.Contains(t, []int{http.StatusOK, http.StatusPartialContent}, rec.Code)
	require.Equal(t, exe, rec.Body.Bytes(), "an older till gets the plain exe")
	rec = download("/branch/app_update/download/3?encoding=zstd")
	require.Equal(t, packed, rec.Body.Bytes())

	store.EXPECT().GetAppReleasePatch(gomock.Any(), int64(9)).
		Return(db.AppReleasePatch{ID: 9, ReleaseID: 3, FilePath: filepath.Join("abc", "patch-1")}, nil)
	rec = download("/branch/app_update/patch/9")
	require.Equal(t, "PATCH", rec.Body.String())

	// A patch for another release isn't this till's to fetch.
	store.EXPECT().GetAppReleasePatch(gomock.Any(), int64(10)).Return(db.AppReleasePatch{ID: 10, ReleaseID: 4}, nil)
	require.Equal(t, http.StatusNotFound, download("/branch/app_update/patch/10").Code)
	store.EXPECT().GetAppReleasePatch(gomock.Any(), int64(11)).Return(db.AppReleasePatch{}, sql.ErrNoRows)
	require.Equal(t, http.StatusNotFound, download("/branch/app_update/patch/11").Code)
}
