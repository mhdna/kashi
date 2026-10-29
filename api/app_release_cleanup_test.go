package api

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/lib/pq"
	mockdb "github.com/mhdna/kashi/db/mock"
	db "github.com/mhdna/kashi/db/sqlc"
	"github.com/stretchr/testify/require"
)

// writeReleaseFile puts a fake build at <storage>/<dir>/release.
func writeReleaseFile(t *testing.T, server *Server, dir string) string {
	rel := filepath.Join(dir, appReleaseStoredName)
	path := filepath.Join(server.appReleaseStorageDir(), rel)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte("MZ"), 0o644))
	return rel
}

func TestCleanAppReleases_RemovesDeliveredFilesAndKeepsSetting(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	server := newTestServer(t, store)
	server.config.AppReleaseStorageDir = t.TempDir()
	done := writeReleaseFile(t, server, "done")
	kept := writeReleaseFile(t, server, "kept")

	gomock.InOrder(
		store.EXPECT().GetAppSettings(gomock.Any()).Return(db.AppSetting{AppReleaseKeep: 2}, nil),
		store.EXPECT().ListCleanableAppReleases(gomock.Any(), int64(2)).
			Return([]db.ListCleanableAppReleasesRow{{ID: 7, Version: "1.1.0", Flavor: "standard", FilePath: done}}, nil),
		store.EXPECT().MarkAppReleaseFileRemoved(gomock.Any(), int64(7)).Return(nil),
	)
	server.cleanAppReleases(context.Background())

	_, err := os.Stat(filepath.Join(server.appReleaseStorageDir(), "done"))
	require.True(t, os.IsNotExist(err))
	_, err = os.Stat(filepath.Join(server.appReleaseStorageDir(), kept))
	require.NoError(t, err)

	// "Keep all" never even looks for builds to remove.
	store.EXPECT().GetAppSettings(gomock.Any()).Return(db.AppSetting{AppReleaseKeep: appReleaseKeepAll}, nil)
	server.cleanAppReleases(context.Background())
}

func TestSetAppReleaseKeep_ValidatesRange(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	server := newAppUpdateTestServer(t, store)

	for _, bad := range []map[string]any{{}, {"keep": -2}, {"keep": appReleaseKeepMax + 1}} {
		rec := httptest.NewRecorder()
		server.router.ServeHTTP(rec, adminJSONRequest(t, server, http.MethodPut, "/app_updates/keep", bad))
		require.Equal(t, http.StatusBadRequest, rec.Code, bad)
	}

	store.EXPECT().SetAppReleaseKeep(gomock.Any(), int32(0)).Return(db.AppSetting{}, nil)
	rec := httptest.NewRecorder()
	server.router.ServeHTTP(rec, adminJSONRequest(t, server, http.MethodPut, "/app_updates/keep", map[string]any{"keep": 0}))
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
}

func TestCreateAppRelease_ReattachesCleanedUpVersion(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	server := newAppUpdateTestServer(t, store)
	content := []byte("MZ again")
	sum := sha256.Sum256(content)
	sig := base64.StdEncoding.EncodeToString(make([]byte, 64))

	store.EXPECT().CreateAppRelease(gomock.Any(), gomock.Any()).Return(db.AppRelease{}, &pq.Error{Code: "23505"})
	store.EXPECT().ReattachAppReleaseFile(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ any, arg db.ReattachAppReleaseFileParams) (db.AppRelease, error) {
			require.Equal(t, "1.1.0", arg.Version)
			stored, err := os.ReadFile(filepath.Join(server.appReleaseStorageDir(), arg.FilePath))
			require.NoError(t, err)
			require.Equal(t, content, stored)
			return db.AppRelease{ID: 7, Version: arg.Version, Flavor: arg.Flavor}, nil
		})
	rec := httptest.NewRecorder()
	server.router.ServeHTTP(rec, releaseUploadRequest(t, server, content, "1.1.0", "standard", hex.EncodeToString(sum[:]), sig))
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())

	// Still published with its file: the usual "bump the version".
	store.EXPECT().CreateAppRelease(gomock.Any(), gomock.Any()).Return(db.AppRelease{}, &pq.Error{Code: "23505"})
	store.EXPECT().ReattachAppReleaseFile(gomock.Any(), gomock.Any()).Return(db.AppRelease{}, sql.ErrNoRows)
	rec = httptest.NewRecorder()
	server.router.ServeHTTP(rec, releaseUploadRequest(t, server, content, "1.1.0", "standard", hex.EncodeToString(sum[:]), sig))
	require.Equal(t, http.StatusConflict, rec.Code)
}

func TestSetAppUpdateTarget_RefusesCleanedUpBuildForTillsWithoutIt(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	server := newAppUpdateTestServer(t, store)
	release := db.AppRelease{ID: 3, Version: "1.1.0", Flavor: "standard", FileRemovedAt: sql.NullTime{Time: time.Now(), Valid: true}}
	store.EXPECT().GetAppRelease(gomock.Any(), int64(3)).Return(release, nil)
	store.EXPECT().ListBranchRegistersByIDs(gomock.Any(), []int64{1, 2}).
		Return([]db.BranchRegister{{ID: 1, Code: "CN1", AppVersion: "1.1.0"}, {ID: 2, Code: "CN2", AppVersion: "1.0.9"}}, nil)

	rec := httptest.NewRecorder()
	server.router.ServeHTTP(rec, adminJSONRequest(t, server, http.MethodPut, "/app_updates/target", map[string]any{"release_id": 3, "register_ids": []int64{1, 2}}))
	require.Equal(t, http.StatusConflict, rec.Code)
	require.Contains(t, rec.Body.String(), "CN2")
	require.NotContains(t, rec.Body.String(), "CN1")
}

func TestBranchAppUpdateDownload_CleanedUpIsGone(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	stubBranchAuth(store)
	server := newAppUpdateTestServer(t, store)
	store.EXPECT().GetRegisterTargetRelease(gomock.Any(), testRegisterID).Return(db.GetRegisterTargetReleaseRow{
		ID: 3, Version: "1.1.0", FileRemovedAt: sql.NullTime{Time: time.Now(), Valid: true},
	}, nil)

	rec := httptest.NewRecorder()
	server.router.ServeHTTP(rec, newBranchRequest(t, http.MethodGet, "/branch/app_update/download/3", nil))
	require.Equal(t, http.StatusGone, rec.Code)
}
