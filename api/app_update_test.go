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
	mockdb "github.com/mhdna/kashi/db/mock"
	db "github.com/mhdna/kashi/db/sqlc"
	"github.com/mhdna/kashi/util"
	"github.com/stretchr/testify/require"
)

func newAppUpdateTestServer(t *testing.T, store *mockdb.MockStore) *Server {
	server := newTestServer(t, store)
	server.config.AppReleaseStorageDir = t.TempDir()
	return server
}

func TestBranchAppUpdate_ReturnsTargetOrNull(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	stubBranchAuth(store)
	server := newAppUpdateTestServer(t, store)

	store.EXPECT().GetRegisterTargetRelease(gomock.Any(), testRegisterID).Return(db.AppRelease{}, sql.ErrNoRows)
	rec := httptest.NewRecorder()
	server.router.ServeHTTP(rec, newBranchRequest(t, http.MethodGet, "/branch/app_update", nil))
	require.Equal(t, http.StatusOK, rec.Code)
	require.JSONEq(t, `{"release":null}`, rec.Body.String())

	store.EXPECT().GetRegisterTargetRelease(gomock.Any(), testRegisterID).
		Return(db.AppRelease{ID: 3, Version: "1.0.9", Flavor: "standard", FileName: "Kashi-Cloud.exe", FilePath: "secret/path", Sha256: "ab", Signature: "sig"}, nil)
	rec = httptest.NewRecorder()
	server.router.ServeHTTP(rec, newBranchRequest(t, http.MethodGet, "/branch/app_update", nil))
	require.Equal(t, http.StatusOK, rec.Code)
	var body struct {
		Release map[string]any `json:"release"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, "1.0.9", body.Release["version"])
	require.NotContains(t, body.Release, "file_path")
}

func TestBranchAppUpdateDownload_OnlyTheTargetAndResumable(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	stubBranchAuth(store)
	server := newAppUpdateTestServer(t, store)

	content := []byte("0123456789abcdef")
	rel := filepath.Join("standard", "1.0.9", "x", "Kashi-Cloud.exe")
	require.NoError(t, os.MkdirAll(filepath.Dir(filepath.Join(server.appReleaseStorageDir(), rel)), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(server.appReleaseStorageDir(), rel), content, 0o644))
	release := db.AppRelease{ID: 3, Version: "1.0.9", FileName: "Kashi-Cloud.exe", FilePath: rel, CreatedAt: time.Now()}
	store.EXPECT().GetRegisterTargetRelease(gomock.Any(), testRegisterID).Return(release, nil).AnyTimes()

	rec := httptest.NewRecorder()
	server.router.ServeHTTP(rec, newBranchRequest(t, http.MethodGet, "/branch/app_update/download/4", nil))
	require.Equal(t, http.StatusNotFound, rec.Code)

	req := newBranchRequest(t, http.MethodGet, "/branch/app_update/download/3", nil)
	req.Header.Set("Range", "bytes=10-")
	rec = httptest.NewRecorder()
	server.router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusPartialContent, rec.Code)
	require.Equal(t, "abcdef", rec.Body.String())
}

func TestBranchAppUpdateStatus_RecordsEventOnlyOnChange(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	hash, err := hashForTest()
	require.NoError(t, err)
	register := db.BranchRegister{ID: testRegisterID, BranchID: testBranchID, Code: testRegisterCode, ApiKeyHash: hash, IsActive: true,
		AppVersion: "1.0.8", UpdateState: "downloading"}
	store.EXPECT().GetBranchRegisterByCode(gomock.Any(), testRegisterCode).Return(register, nil).AnyTimes()
	store.EXPECT().GetBranch(gomock.Any(), testBranchID).Return(db.Branch{ID: testBranchID, IsActive: true}, nil).AnyTimes()
	store.EXPECT().TouchBranchRegister(gomock.Any(), testRegisterID).Return(nil).AnyTimes()
	store.EXPECT().GetBranchRegister(gomock.Any(), testRegisterID).Return(register, nil).AnyTimes()
	store.EXPECT().SetRegisterUpdateStatus(gomock.Any(), gomock.Any()).Return(nil).Times(2)
	server := newAppUpdateTestServer(t, store)

	// Same state, new progress: no history row.
	rec := httptest.NewRecorder()
	server.router.ServeHTTP(rec, newBranchRequest(t, http.MethodPost, "/branch/app_update/status",
		map[string]any{"app_version": "1.0.8", "app_flavor": "standard", "state": "downloading", "release_id": 3, "progress": 40}))
	require.Equal(t, http.StatusOK, rec.Code)

	store.EXPECT().CreateAppUpdateEvent(gomock.Any(), db.CreateAppUpdateEventParams{
		RegisterID: testRegisterID, ReleaseID: sql.NullInt64{Int64: 3, Valid: true}, AppVersion: "1.0.8", State: "ready",
	}).Return(nil)
	rec = httptest.NewRecorder()
	server.router.ServeHTTP(rec, newBranchRequest(t, http.MethodPost, "/branch/app_update/status",
		map[string]any{"app_version": "1.0.8", "app_flavor": "standard", "state": "ready", "release_id": 3}))
	require.Equal(t, http.StatusOK, rec.Code)

	rec = httptest.NewRecorder()
	server.router.ServeHTTP(rec, newBranchRequest(t, http.MethodPost, "/branch/app_update/status",
		map[string]any{"state": "bogus"}))
	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestSetAppUpdateTarget_RejectsOtherFlavorAndNotifies(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	server := newAppUpdateTestServer(t, store)
	release := db.AppRelease{ID: 3, Version: "1.0.9", Flavor: "standard"}
	store.EXPECT().GetAppRelease(gomock.Any(), int64(3)).Return(release, nil).Times(2)

	store.EXPECT().ListBranchRegistersByIDs(gomock.Any(), []int64{1, 2}).
		Return([]db.BranchRegister{{ID: 1, Code: "CN1", AppFlavor: "standard"}, {ID: 2, Code: "CN2", AppFlavor: "compact"}}, nil)
	rec := httptest.NewRecorder()
	server.router.ServeHTTP(rec, adminJSONRequest(t, server, http.MethodPut, "/app_updates/target", map[string]any{"release_id": 3, "register_ids": []int64{1, 2}}))
	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Contains(t, rec.Body.String(), "CN2")

	store.EXPECT().ListBranchRegistersByIDs(gomock.Any(), []int64{1}).
		Return([]db.BranchRegister{{ID: 1, Code: "CN1", AppFlavor: "standard", AppVersion: "1.0.8"}}, nil)
	store.EXPECT().SetRegistersTargetRelease(gomock.Any(), db.SetRegistersTargetReleaseParams{
		ReleaseID: sql.NullInt64{Int64: 3, Valid: true}, RegisterIds: []int64{1},
	}).Return(nil)
	store.EXPECT().CreateAppUpdateEvent(gomock.Any(), gomock.Any()).Return(nil)
	rec = httptest.NewRecorder()
	server.router.ServeHTTP(rec, adminJSONRequest(t, server, http.MethodPut, "/app_updates/target", map[string]any{"release_id": 3, "register_ids": []int64{1}}))
	require.Equal(t, http.StatusOK, rec.Code)
}

func TestCreateAppRelease_ChecksChecksumAndStores(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	server := newAppUpdateTestServer(t, store)
	content := []byte("MZ fake windows program")
	sum := sha256.Sum256(content)
	sig := base64.StdEncoding.EncodeToString(make([]byte, 64))

	rec := httptest.NewRecorder()
	server.router.ServeHTTP(rec, releaseUploadRequest(t, server, content, "1.0.9", "standard", "00", sig))
	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Contains(t, rec.Body.String(), "checksum")

	store.EXPECT().GetUserByUsername(gomock.Any(), "admin").Return(db.User{ID: 5}, nil)
	store.EXPECT().CreateAppRelease(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ any, arg db.CreateAppReleaseParams) (db.AppRelease, error) {
			stored, err := os.ReadFile(filepath.Join(server.appReleaseStorageDir(), arg.FilePath))
			require.NoError(t, err)
			require.Equal(t, content, stored)
			require.Equal(t, hex.EncodeToString(sum[:]), arg.Sha256)
			return db.AppRelease{ID: 1, Version: arg.Version, Flavor: arg.Flavor}, nil
		})
	rec = httptest.NewRecorder()
	server.router.ServeHTTP(rec, releaseUploadRequest(t, server, content, "1.0.9", "standard", hex.EncodeToString(sum[:]), sig))
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())

	rec = httptest.NewRecorder()
	server.router.ServeHTTP(rec, releaseUploadRequest(t, server, content, "latest", "standard", "", sig))
	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestAppReleaseFileName(t *testing.T) {
	require.Equal(t, "Kashi-Cloud.exe", appReleaseFileName(`C:\build\Kashi-Cloud.exe`))
	require.Equal(t, "Kashi Cloud Compact.exe", appReleaseFileName("Kashi Cloud Compact.exe"))
	require.Equal(t, "Kashi-Cloud.exe", appReleaseFileName(".."))
	require.Equal(t, "a_b.exe", appReleaseFileName("a$b.exe"))
}

func hashForTest() (string, error) {
	return util.HashPassword(testBranchKey)
}

func adminJSONRequest(t *testing.T, server *Server, method, path string, body any) *http.Request {
	t.Helper()
	raw, err := json.Marshal(body)
	require.NoError(t, err)
	req, err := http.NewRequest(method, path, bytes.NewReader(raw))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	addAuthorization(t, req, server.tokenMaker, authorizationTypeBearer, "admin", time.Minute)
	return req
}

func releaseUploadRequest(t *testing.T, server *Server, content []byte, version, flavor, sum, sig string) *http.Request {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	for k, v := range map[string]string{"version": version, "flavor": flavor, "sha256": sum, "signature": sig} {
		require.NoError(t, w.WriteField(k, v))
	}
	part, err := w.CreateFormFile("file", "Kashi-Cloud.exe")
	require.NoError(t, err)
	_, err = part.Write(content)
	require.NoError(t, err)
	require.NoError(t, w.Close())
	req, err := http.NewRequest(http.MethodPost, "/app_releases", &buf)
	require.NoError(t, err)
	req.Header.Set("Content-Type", w.FormDataContentType())
	addAuthorization(t, req, server.tokenMaker, authorizationTypeBearer, "admin", time.Minute)
	return req
}
