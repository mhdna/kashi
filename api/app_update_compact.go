package api

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/klauspost/compress/zstd"
	db "github.com/mhdna/kashi/db/sqlc"
)

// Smaller till updates. A ~28 MB build compresses to ~9 MB with zstd, and a
// patch from the build a till already runs is ~1-2 MB.
//
//   - The publish tool uploads the build zstd-compressed (encoding "zstd"),
//     then one patch per build that tills run (POST /app_releases/:id/patches).
//   - A till that understands them asks GET /branch/app_update?accept=zstd,patch
//     &have=<sha256 of its exe>; the answer names the patch from that exact exe,
//     if there is one, and the compressed download.
//   - An older till asks without accept and gets the plain exe, unpacked here
//     once and kept beside the compressed file.
//
// sha256, size_bytes and the signature are always those of the exe, so the
// till checks exactly what was signed, however it arrived. A patch that
// rebuilds the wrong thing fails that check and the till falls back to the
// full download.

const (
	appReleaseEncodingZstd = "zstd"
	// The plain exe, unpacked for older tills, beside the stored file.
	appReleaseRawName = "release.raw"
	// Most a build may unpack to, so a bad upload can't fill the disk.
	maxAppReleaseExeBytes = 512 << 20
	maxAppReleasePatch    = 200 << 20
)

var sha256HexPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)

// zstdExeDigest unpacks a stored zstd release and returns the exe's hex
// SHA-256 and size, without keeping it.
func zstdExeDigest(path string) (string, int64, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", 0, err
	}
	defer f.Close()
	dec, err := zstd.NewReader(f, zstd.WithDecoderMaxWindow(1<<30), zstd.WithDecoderConcurrency(1))
	if err != nil {
		return "", 0, err
	}
	defer dec.Close()
	hash := sha256.New()
	n, err := io.Copy(hash, io.LimitReader(dec, maxAppReleaseExeBytes+1))
	if err != nil {
		return "", 0, fmt.Errorf("the upload isn't valid zstd: %w", err)
	}
	if n > maxAppReleaseExeBytes {
		return "", 0, errors.New("the build unpacks to more than 512MB")
	}
	return hex.EncodeToString(hash.Sum(nil)), n, nil
}

// storedRelease is what rawReleaseFile needs of a release row.
type storedRelease struct {
	FilePath, Encoding, Sha256 string
	SizeBytes                  int64
}

// rawUnpackMu makes one unpack at a time, so two old tills asking at once
// don't both write the same file.
var rawUnpackMu sync.Mutex

// rawReleaseFile is the plain exe of a release, for a till that can't take
// zstd: the stored file itself, or the stored file unpacked once.
func (server *Server) rawReleaseFile(release storedRelease) (string, error) {
	stored := filepath.Join(server.appReleaseStorageDir(), release.FilePath)
	if release.Encoding != appReleaseEncodingZstd {
		return stored, nil
	}
	raw := filepath.Join(filepath.Dir(stored), appReleaseRawName)
	rawUnpackMu.Lock()
	defer rawUnpackMu.Unlock()
	if info, err := os.Stat(raw); err == nil && info.Size() == release.SizeBytes {
		return raw, nil
	}
	in, err := os.Open(stored)
	if err != nil {
		return "", err
	}
	defer in.Close()
	dec, err := zstd.NewReader(in, zstd.WithDecoderMaxWindow(1<<30), zstd.WithDecoderConcurrency(1))
	if err != nil {
		return "", err
	}
	defer dec.Close()
	tmp := raw + ".tmp"
	out, err := os.Create(tmp)
	if err != nil {
		return "", err
	}
	hash := sha256.New()
	if _, err := io.Copy(io.MultiWriter(out, hash), io.LimitReader(dec, maxAppReleaseExeBytes)); err != nil {
		out.Close()
		os.Remove(tmp)
		return "", err
	}
	if err := out.Close(); err != nil {
		os.Remove(tmp)
		return "", err
	}
	if hex.EncodeToString(hash.Sum(nil)) != release.Sha256 {
		os.Remove(tmp)
		return "", errors.New("the stored release doesn't unpack to its exe")
	}
	return raw, os.Rename(tmp, raw)
}

// acceptsEncoding: the till's accept list (comma-separated) names it.
func acceptsEncoding(accept, name string) bool {
	for _, a := range strings.Split(accept, ",") {
		if strings.TrimSpace(a) == name {
			return true
		}
	}
	return false
}

type appUpdatePatchManifest struct {
	ID         int64  `json:"id"`
	BaseSha256 string `json:"base_sha256"`
	SizeBytes  int64  `json:"size_bytes"`
}

// createAppReleasePatch stores a patch from an earlier build to this
// release (multipart: file, base_sha256, base_version, file_sha256).
func (server *Server) createAppReleasePatch(ctx *gin.Context) {
	var idReq appReleaseIDRequest
	if err := ctx.ShouldBindUri(&idReq); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	base := strings.ToLower(strings.TrimSpace(ctx.PostForm("base_sha256")))
	baseVersion := strings.TrimSpace(ctx.PostForm("base_version"))
	wantSum := strings.ToLower(strings.TrimSpace(ctx.PostForm("file_sha256")))
	if !sha256HexPattern.MatchString(base) || !sha256HexPattern.MatchString(wantSum) {
		server.writeError(ctx, http.StatusBadRequest, errors.New("base_sha256 and file_sha256 must be hex SHA-256s"))
		return
	}
	release, err := server.store.GetAppRelease(ctx, idReq.ID)
	if err != nil {
		server.writeError(ctx, statusForLookup(err), err)
		return
	}
	if release.FileRemovedAt.Valid {
		server.writeError(ctx, http.StatusGone, errors.New("this release's file was cleaned up; publish it again"))
		return
	}
	if base == release.Sha256 {
		server.writeError(ctx, http.StatusBadRequest, errors.New("a patch from the release to itself"))
		return
	}
	fileHeader, err := ctx.FormFile("file")
	if err != nil || fileHeader.Size <= 0 || fileHeader.Size > maxAppReleasePatch {
		server.writeError(ctx, http.StatusBadRequest, errors.New("the patch file is missing or too big"))
		return
	}
	dir := filepath.Dir(filepath.Join(server.appReleaseStorageDir(), release.FilePath))
	relPath := filepath.Join(filepath.Dir(release.FilePath), "patch-"+base[:16])
	dest := filepath.Join(server.appReleaseStorageDir(), relPath)
	tmp := filepath.Join(dir, "patch-"+base[:16]+".upload")
	os.Remove(tmp)
	sum, err := saveUploadedFile(fileHeader, tmp)
	if err != nil {
		os.Remove(tmp)
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	if sum != wantSum {
		os.Remove(tmp)
		server.writeError(ctx, http.StatusBadRequest, errors.New("the patch's checksum doesn't match; the upload was probably cut off, try again"))
		return
	}
	if err := os.Rename(tmp, dest); err != nil {
		os.Remove(tmp)
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	patch, err := server.store.UpsertAppReleasePatch(ctx, db.UpsertAppReleasePatchParams{
		ReleaseID: release.ID, BaseSha256: base, BaseVersion: baseVersion,
		FilePath: relPath, SizeBytes: fileHeader.Size, FileSha256: sum,
	})
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	server.adminHub.broadcastAll(adminWSMessage{Type: "app_update_changed"})
	server.writeJSON(ctx, http.StatusCreated, envelope{"patch": patch})
}

// getAppReleaseFile hands the publish tool a release's stored file as it is
// (compressed or not, see the X-Encoding header), so it can make patches
// from a build it no longer has.
func (server *Server) getAppReleaseFile(ctx *gin.Context) {
	var idReq appReleaseIDRequest
	if err := ctx.ShouldBindUri(&idReq); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	release, err := server.store.GetAppRelease(ctx, idReq.ID)
	if err != nil {
		server.writeError(ctx, statusForLookup(err), err)
		return
	}
	if release.FileRemovedAt.Valid {
		server.writeError(ctx, http.StatusGone, errors.New("this release's file was cleaned up"))
		return
	}
	file, err := os.Open(filepath.Join(server.appReleaseStorageDir(), release.FilePath))
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	defer file.Close()
	ctx.Header("Content-Type", "application/octet-stream")
	ctx.Header("X-Encoding", release.Encoding)
	http.ServeContent(ctx.Writer, ctx.Request, release.FileName, release.CreatedAt, file)
}

// branchAppUpdatePatchDownload streams a till a patch to its target release,
// resumable like the full download.
func (server *Server) branchAppUpdatePatchDownload(ctx *gin.Context) {
	var idReq appReleaseIDRequest
	if err := ctx.ShouldBindUri(&idReq); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	patch, err := server.store.GetAppReleasePatch(ctx, idReq.ID)
	if err != nil {
		server.writeError(ctx, statusForLookup(err), err)
		return
	}
	registerID := ctx.MustGet(registerIDKey).(int64)
	release, err := server.store.GetRegisterTargetRelease(ctx, registerID)
	if err != nil && err != sql.ErrNoRows {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	if err == sql.ErrNoRows || release.ID != patch.ReleaseID {
		server.writeError(ctx, http.StatusNotFound, errors.New("this patch isn't for this register's update"))
		return
	}
	if release.FileRemovedAt.Valid {
		server.writeError(ctx, http.StatusGone, errors.New("this release's files were cleaned up"))
		return
	}
	file, err := os.Open(filepath.Join(server.appReleaseStorageDir(), patch.FilePath))
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, fmt.Errorf("patch file: %w", err))
		return
	}
	defer file.Close()
	ctx.Header("Content-Type", "application/octet-stream")
	http.ServeContent(ctx.Writer, ctx.Request, "update.patch", patch.CreatedAt, file)
}
