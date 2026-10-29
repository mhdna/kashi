package api

import (
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	db "github.com/mhdna/kashi/db/sqlc"
	"github.com/mhdna/kashi/token"
)

// Remote updates for the Kashi Cloud till app (kashi-pos).
//
// The developer publishes a signed build from the kashi-pos repo (`make
// publish`, which calls createAppRelease). An admin points registers at a
// release on the App Updates page (setAppUpdateTarget), and kashi nudges
// each one over its WebSocket. The till fetches its target
// (branchAppUpdate), downloads it (branchAppUpdateDownload), installs it at
// a safe moment and reports every step (branchAppUpdateStatus). Rolling back
// is the same thing with an older release as the target.
//
// The admin routes need the App Updates section (access.go), a section of its
// own rather than a page under System: putting a program on every till is
// about the most powerful thing kashi can do, so it is granted on purpose,
// never as a side effect of editing settings.

const (
	defaultAppReleaseStorageDir = "./storage/app_releases"
	maxAppReleaseBytes          = 512 << 20
	maxAppUpdateDetailRunes     = 1000
)

var (
	appReleaseVersionPattern = regexp.MustCompile(`^\d+\.\d+\.\d+(\.\d+)?$`)
	appReleaseFlavors        = map[string]bool{"standard": true, "compact": true}
	// The states a till reports -- see updater.go in kashi-pos.
	appUpdateStates = map[string]bool{
		"up_to_date":  true,
		"downloading": true,
		"ready":       true,
		"installing":  true,
		"updated":     true,
		"failed":      true,
		"rolled_back": true,
		"blocked":     true,
	}
)

func (server *Server) appReleaseStorageDir() string {
	if dir := server.config.AppReleaseStorageDir; dir != "" {
		return dir
	}
	return defaultAppReleaseStorageDir
}

// appReleaseResponse is a release without its on-disk path.
type appReleaseResponse struct {
	ID           int64     `json:"id"`
	Version      string    `json:"version"`
	Flavor       string    `json:"flavor"`
	FileName     string    `json:"file_name"`
	SizeBytes    int64     `json:"size_bytes"`
	Sha256       string    `json:"sha256"`
	Signature    string    `json:"signature"`
	Notes        string    `json:"notes"`
	CreatedAt    time.Time `json:"created_at"`
	RunningCount int64     `json:"running_count"`
	TargetCount  int64     `json:"target_count"`
	// The build file was cleaned up (app_release_cleanup.go); tills that
	// don't run it yet can't be sent it until it's published again.
	FileRemoved bool `json:"file_removed"`
}

func newAppReleaseResponse(r db.AppRelease) appReleaseResponse {
	return appReleaseResponse{
		ID: r.ID, Version: r.Version, Flavor: r.Flavor, FileName: r.FileName,
		SizeBytes: r.SizeBytes, Sha256: r.Sha256, Signature: r.Signature,
		Notes: r.Notes, CreatedAt: r.CreatedAt, FileRemoved: r.FileRemovedAt.Valid,
	}
}

func (server *Server) listAppReleases(ctx *gin.Context) {
	rows, err := server.store.ListAppReleases(ctx)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	releases := make([]appReleaseResponse, 0, len(rows))
	for _, r := range rows {
		releases = append(releases, appReleaseResponse{
			ID: r.ID, Version: r.Version, Flavor: r.Flavor, FileName: r.FileName,
			SizeBytes: r.SizeBytes, Sha256: r.Sha256, Signature: r.Signature,
			Notes: r.Notes, CreatedAt: r.CreatedAt,
			RunningCount: r.RunningCount, TargetCount: r.TargetCount,
			FileRemoved: r.FileRemovedAt.Valid,
		})
	}
	settings, err := server.store.GetAppSettings(ctx)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	server.writeJSON(ctx, http.StatusOK, envelope{"releases": releases, "keep": settings.AppReleaseKeep})
}

// createAppRelease stores an uploaded build (multipart: file, version,
// flavor, sha256, signature, notes). kashi can't check the signature -- the
// release key's public half lives in kashi-pos, and the publish tool checks
// against it before uploading -- but it does refuse a file whose checksum
// doesn't match what the uploader computed, so a cut-off upload never
// reaches a till.
func (server *Server) createAppRelease(ctx *gin.Context) {
	version := strings.TrimSpace(ctx.PostForm("version"))
	flavor := strings.TrimSpace(ctx.PostForm("flavor"))
	wantSum := strings.ToLower(strings.TrimSpace(ctx.PostForm("sha256")))
	signature := strings.TrimSpace(ctx.PostForm("signature"))
	notes := strings.TrimSpace(ctx.PostForm("notes"))

	if !appReleaseVersionPattern.MatchString(version) {
		server.writeError(ctx, http.StatusBadRequest, fmt.Errorf("version must look like 1.2.3, got %q", version))
		return
	}
	if !appReleaseFlavors[flavor] {
		server.writeError(ctx, http.StatusBadRequest, fmt.Errorf("flavor must be standard or compact, got %q", flavor))
		return
	}
	if raw, err := base64.StdEncoding.DecodeString(signature); err != nil || len(raw) != 64 {
		server.writeError(ctx, http.StatusBadRequest, errors.New("signature must be a base64 ed25519 signature"))
		return
	}
	fileHeader, err := ctx.FormFile("file")
	if err != nil {
		server.writeError(ctx, http.StatusBadRequest, errors.New("file is required"))
		return
	}
	if fileHeader.Size <= 0 || fileHeader.Size > maxAppReleaseBytes {
		server.writeError(ctx, http.StatusBadRequest, fmt.Errorf("the file must be between 1 byte and %dMB", maxAppReleaseBytes>>20))
		return
	}

	// A directory of its own per upload, so a rejected duplicate never
	// overwrites the file of the release it duplicates. Nothing from the
	// request goes into the path: the till's download is named after
	// FileName, not the file on disk.
	relDir := uuid.NewString()
	relPath := filepath.Join(relDir, appReleaseStoredName)
	dir := filepath.Join(server.appReleaseStorageDir(), relDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	fileName := appReleaseFileName(fileHeader.Filename)
	sum, err := saveUploadedFile(fileHeader, filepath.Join(server.appReleaseStorageDir(), relPath))
	if err != nil {
		os.RemoveAll(dir)
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	if wantSum != "" && sum != wantSum {
		os.RemoveAll(dir)
		server.writeError(ctx, http.StatusBadRequest, errors.New("the uploaded file's checksum doesn't match; the upload was probably cut off, try again"))
		return
	}

	userID, err := server.callerID(ctx, ctx.MustGet(authoizationPayloadKey).(*token.Payload))
	if err != nil {
		os.RemoveAll(dir)
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	release, err := server.store.CreateAppRelease(ctx, db.CreateAppReleaseParams{
		Version:    version,
		Flavor:     flavor,
		FileName:   fileName,
		FilePath:   relPath,
		SizeBytes:  fileHeader.Size,
		Sha256:     sum,
		Signature:  signature,
		Notes:      notes,
		UploadedBy: sql.NullInt64{Int64: userID, Valid: true},
	})
	if err != nil && isUniqueViolation(err) {
		// Already published -- but if its file was cleaned up, this upload
		// brings it back.
		reattached, rerr := server.store.ReattachAppReleaseFile(ctx, db.ReattachAppReleaseFileParams{
			Version: version, Flavor: flavor, FileName: fileName, FilePath: relPath,
			SizeBytes: fileHeader.Size, Sha256: sum, Signature: signature, Notes: notes,
		})
		if rerr == nil {
			release, err = reattached, nil
		}
	}
	if err != nil {
		os.RemoveAll(dir)
		if isUniqueViolation(err) {
			server.writeError(ctx, http.StatusConflict, fmt.Errorf("%s %s is already published; bump the version", flavor, version))
			return
		}
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	server.adminHub.broadcastAll(adminWSMessage{Type: "app_update_changed"})
	server.writeJSON(ctx, http.StatusCreated, envelope{"release": newAppReleaseResponse(release)})
}

// appReleaseStoredName is what every release file is called on disk, in a
// directory of its own.
const appReleaseStoredName = "release"

// appReleaseFileName keeps the uploaded name (it's what a till's download
// is called) but only its base name, with anything unusual replaced.
func appReleaseFileName(name string) string {
	name = filepath.Base(strings.ReplaceAll(name, `\`, "/"))
	clean := strings.Map(func(r rune) rune {
		if r == '.' || r == '-' || r == '_' || r == ' ' ||
			(r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			return r
		}
		return '_'
	}, name)
	if strings.Trim(clean, ". ") == "" {
		return "Kashi-Cloud.exe"
	}
	return clean
}

// saveUploadedFile copies the upload to dest and returns its hex SHA-256.
func saveUploadedFile(fileHeader *multipart.FileHeader, dest string) (string, error) {
	src, err := fileHeader.Open()
	if err != nil {
		return "", err
	}
	defer src.Close()
	out, err := os.OpenFile(dest, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		return "", err
	}
	hash := sha256.New()
	if _, err := io.Copy(io.MultiWriter(out, hash), src); err != nil {
		out.Close()
		return "", err
	}
	if err := out.Close(); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

type appReleaseIDRequest struct {
	ID int64 `uri:"id" binding:"required,min=1"`
}

type updateAppReleaseRequest struct {
	Notes string `json:"notes"`
}

func (server *Server) updateAppRelease(ctx *gin.Context) {
	var idReq appReleaseIDRequest
	if err := ctx.ShouldBindUri(&idReq); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	var req updateAppReleaseRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	release, err := server.store.UpdateAppReleaseNotes(ctx, db.UpdateAppReleaseNotesParams{
		ID:    idReq.ID,
		Notes: strings.TrimSpace(req.Notes),
	})
	if err != nil {
		if err == sql.ErrNoRows {
			server.writeError(ctx, http.StatusNotFound, err)
			return
		}
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	server.adminHub.broadcastAll(adminWSMessage{Type: "app_update_changed"})
	server.writeJSON(ctx, http.StatusOK, envelope{"release": newAppReleaseResponse(release)})
}

// deleteAppRelease removes a release no register is set to. Tills keep
// whatever they already installed; only their next update can't use it.
func (server *Server) deleteAppRelease(ctx *gin.Context) {
	var idReq appReleaseIDRequest
	if err := ctx.ShouldBindUri(&idReq); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	release, err := server.store.GetAppRelease(ctx, idReq.ID)
	if err != nil {
		if err == sql.ErrNoRows {
			server.writeError(ctx, http.StatusNotFound, err)
			return
		}
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	targeted, err := server.store.CountRegistersTargetingRelease(ctx, sql.NullInt64{Int64: release.ID, Valid: true})
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	if targeted > 0 {
		server.writeError(ctx, http.StatusConflict, fmt.Errorf("%d register(s) are still set to %s; point them at another release first", targeted, release.Version))
		return
	}
	if err := server.store.DeleteAppRelease(ctx, release.ID); err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	// The row is gone either way; a file left behind is only disk space.
	os.RemoveAll(filepath.Dir(filepath.Join(server.appReleaseStorageDir(), release.FilePath)))

	server.adminHub.broadcastAll(adminWSMessage{Type: "app_update_changed"})
	server.writeJSON(ctx, http.StatusOK, envelope{"success": true})
}

type appUpdateRegisterResponse struct {
	db.ListRegisterUpdateStatusesRow
	Connected bool `json:"connected"`
}

// listAppUpdateRegisters is every register with what it runs, what it's
// set to run, and how its update is going.
func (server *Server) listAppUpdateRegisters(ctx *gin.Context) {
	rows, err := server.store.ListRegisterUpdateStatuses(ctx)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	connected := make(map[int64]bool)
	for _, id := range server.branchHub.connectedRegisterIDs() {
		connected[id] = true
	}
	registers := make([]appUpdateRegisterResponse, 0, len(rows))
	for _, r := range rows {
		registers = append(registers, appUpdateRegisterResponse{ListRegisterUpdateStatusesRow: r, Connected: connected[r.ID]})
	}
	server.writeJSON(ctx, http.StatusOK, envelope{"registers": registers})
}

type setAppUpdateTargetRequest struct {
	// ReleaseID nil clears the target: the register keeps what it runs.
	ReleaseID   *int64  `json:"release_id"`
	RegisterIDs []int64 `json:"register_ids" binding:"required,min=1,dive,min=1"`
	// Force: install as soon as it's downloaded, even with a shift open.
	Force bool `json:"force"`
}

// setAppUpdateTarget tells registers which release to run -- newer to
// update, older to roll back -- and nudges each connected one to fetch it
// now. A register that has reported its flavor only ever gets a release of
// that flavor.
func (server *Server) setAppUpdateTarget(ctx *gin.Context) {
	var req setAppUpdateTargetRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}

	var release db.AppRelease
	target := sql.NullInt64{}
	if req.ReleaseID != nil {
		var err error
		release, err = server.store.GetAppRelease(ctx, *req.ReleaseID)
		if err != nil {
			if err == sql.ErrNoRows {
				server.writeError(ctx, http.StatusNotFound, errors.New("release not found"))
				return
			}
			server.writeError(ctx, http.StatusInternalServerError, err)
			return
		}
		target = sql.NullInt64{Int64: release.ID, Valid: true}
	}

	registers, err := server.store.ListBranchRegistersByIDs(ctx, req.RegisterIDs)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	if len(registers) != len(uniqueInt64s(req.RegisterIDs)) {
		server.writeError(ctx, http.StatusBadRequest, errors.New("unknown register in register_ids"))
		return
	}
	if target.Valid {
		var mismatched []string
		for _, r := range registers {
			if r.AppFlavor != "" && r.AppFlavor != release.Flavor {
				mismatched = append(mismatched, r.Code)
			}
		}
		if len(mismatched) > 0 {
			server.writeError(ctx, http.StatusBadRequest, fmt.Errorf("%s run a different build than %s (%s)", strings.Join(mismatched, ", "), release.Version, release.Flavor))
			return
		}
		if release.FileRemovedAt.Valid {
			var missing []string
			for _, r := range registers {
				if r.AppVersion != release.Version {
					missing = append(missing, r.Code)
				}
			}
			if len(missing) > 0 {
				server.writeError(ctx, http.StatusConflict, fmt.Errorf("%s's file was cleaned up after its tills installed it, so %s can't get it; publish %s again (make publish) first", release.Version, strings.Join(missing, ", "), release.Version))
				return
			}
		}
	}

	ids := make([]int64, 0, len(registers))
	for _, r := range registers {
		ids = append(ids, r.ID)
	}
	if err := server.store.SetRegistersTargetRelease(ctx, db.SetRegistersTargetReleaseParams{ReleaseID: target, RegisterIds: ids, Force: req.Force && target.Valid}); err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	if target.Valid {
		if err := server.store.MarkAppReleaseTargeted(ctx, release.ID); err != nil {
			server.writeError(ctx, http.StatusInternalServerError, err)
			return
		}
	}

	byName := server.displayName(ctx, ctx.MustGet(authoizationPayloadKey).(*token.Payload))
	state, detail := "targeted", fmt.Sprintf("Set to %s by %s", release.Version, byName)
	if req.Force && target.Valid {
		detail = fmt.Sprintf("Forced to %s by %s", release.Version, byName)
	}
	if !target.Valid {
		state, detail = "target_cleared", fmt.Sprintf("Target cleared by %s", byName)
	}
	for _, r := range registers {
		// History only: the target itself is already saved.
		_ = server.store.CreateAppUpdateEvent(ctx, db.CreateAppUpdateEventParams{
			RegisterID: r.ID,
			ReleaseID:  target,
			AppVersion: r.AppVersion,
			State:      state,
			Detail:     detail,
		})
		server.branchHub.notifyRegister(r.ID, branchWSMessage{Type: "app_update"})
	}

	server.cleanAppReleasesSoon()
	server.adminHub.broadcastAll(adminWSMessage{Type: "app_update_changed"})
	server.writeJSON(ctx, http.StatusOK, envelope{"updated": len(ids)})
}

func uniqueInt64s(values []int64) []int64 {
	seen := make(map[int64]bool, len(values))
	out := make([]int64, 0, len(values))
	for _, v := range values {
		if !seen[v] {
			seen[v] = true
			out = append(out, v)
		}
	}
	return out
}

type listAppUpdateEventsRequest struct {
	Limit int32 `form:"limit,default=100" binding:"min=1,max=500"`
}

func (server *Server) listAppUpdateEvents(ctx *gin.Context) {
	var req listAppUpdateEventsRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	events, err := server.store.ListAppUpdateEvents(ctx, req.Limit)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	server.writeJSON(ctx, http.StatusOK, envelope{"events": events})
}

// branchAppUpdateManifest is what a till needs to fetch and verify its
// target release.
type branchAppUpdateManifest struct {
	ID        int64  `json:"id"`
	Version   string `json:"version"`
	Flavor    string `json:"flavor"`
	FileName  string `json:"file_name"`
	SizeBytes int64  `json:"size_bytes"`
	Sha256    string `json:"sha256"`
	Signature string `json:"signature"`
	Notes     string `json:"notes"`
	// Install right away, even mid-shift (App Updates' "force").
	Force bool `json:"force"`
}

// branchAppUpdate answers a till's "what should I run?" -- its target
// release, or null when it has none.
func (server *Server) branchAppUpdate(ctx *gin.Context) {
	registerID := ctx.MustGet(registerIDKey).(int64)
	release, err := server.store.GetRegisterTargetRelease(ctx, registerID)
	if err != nil {
		if err == sql.ErrNoRows {
			server.writeJSON(ctx, http.StatusOK, envelope{"release": nil})
			return
		}
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	server.writeJSON(ctx, http.StatusOK, envelope{"release": branchAppUpdateManifest{
		ID: release.ID, Version: release.Version, Flavor: release.Flavor,
		FileName: release.FileName, SizeBytes: release.SizeBytes,
		Sha256: release.Sha256, Signature: release.Signature, Notes: release.Notes,
		Force: release.TargetForce,
	}})
}

// branchAppUpdateDownload streams a till its target release. It honors
// Range requests, so a download cut off on a weak shop connection resumes
// where it stopped instead of starting over.
func (server *Server) branchAppUpdateDownload(ctx *gin.Context) {
	var idReq appReleaseIDRequest
	if err := ctx.ShouldBindUri(&idReq); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	registerID := ctx.MustGet(registerIDKey).(int64)
	release, err := server.store.GetRegisterTargetRelease(ctx, registerID)
	if err != nil && err != sql.ErrNoRows {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	if err == sql.ErrNoRows || release.ID != idReq.ID {
		server.writeError(ctx, http.StatusNotFound, errors.New("this release isn't this register's update"))
		return
	}
	if release.FileRemovedAt.Valid {
		server.writeError(ctx, http.StatusGone, fmt.Errorf("%s's file was cleaned up; it has to be published again", release.Version))
		return
	}
	file, err := os.Open(filepath.Join(server.appReleaseStorageDir(), release.FilePath))
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, fmt.Errorf("release file: %w", err))
		return
	}
	defer file.Close()
	ctx.Header("Content-Type", "application/octet-stream")
	ctx.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%q", release.FileName))
	// A request without a Range is served as "bytes=0-", i.e. 206 with the
	// whole file. kashi-pos builds up to 1.0.9.1 open their .part file
	// append-only and, on a plain 200, truncate it first -- which Windows
	// refuses for an append-only handle ("Access is denied"), so their first
	// download always failed. On a 206 they just write from byte 0. Builds
	// after that handle a 200 too; this keeps the older ones updatable.
	if ctx.Request.Header.Get("Range") == "" {
		ctx.Request.Header.Set("Range", "bytes=0-")
	}
	http.ServeContent(ctx.Writer, ctx.Request, release.FileName, release.CreatedAt, file)
}

type branchAppUpdateStatusRequest struct {
	AppVersion string `json:"app_version" binding:"max=64"`
	AppFlavor  string `json:"app_flavor" binding:"max=32"`
	State      string `json:"state" binding:"required"`
	ReleaseID  int64  `json:"release_id"`
	Detail     string `json:"detail"`
	Progress   int32  `json:"progress" binding:"min=0,max=100"`
}

// branchAppUpdateStatus records what a till runs and how its update is
// going. Each change of state or version also lands in the history; a
// progress tick within the same state doesn't.
func (server *Server) branchAppUpdateStatus(ctx *gin.Context) {
	var req branchAppUpdateStatusRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	if !appUpdateStates[req.State] {
		server.writeError(ctx, http.StatusBadRequest, fmt.Errorf("unknown update state %q", req.State))
		return
	}
	detail := truncateRunes(strings.TrimSpace(req.Detail), maxAppUpdateDetailRunes)

	registerID := ctx.MustGet(registerIDKey).(int64)
	previous, err := server.store.GetBranchRegister(ctx, registerID)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	if err := server.store.SetRegisterUpdateStatus(ctx, db.SetRegisterUpdateStatusParams{
		ID:             registerID,
		AppVersion:     req.AppVersion,
		AppFlavor:      req.AppFlavor,
		UpdateState:    req.State,
		UpdateDetail:   detail,
		UpdateProgress: req.Progress,
	}); err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	if req.State != previous.UpdateState || req.AppVersion != previous.AppVersion {
		release := sql.NullInt64{Int64: req.ReleaseID, Valid: req.ReleaseID > 0}
		// A release deleted since the till started on it would fail the
		// foreign key; the history row is still worth keeping without it.
		if err := server.store.CreateAppUpdateEvent(ctx, db.CreateAppUpdateEventParams{
			RegisterID: registerID, ReleaseID: release, AppVersion: req.AppVersion, State: req.State, Detail: detail,
		}); err != nil && release.Valid {
			_ = server.store.CreateAppUpdateEvent(ctx, db.CreateAppUpdateEventParams{
				RegisterID: registerID, AppVersion: req.AppVersion, State: req.State, Detail: detail,
			})
		}
	}

	if req.AppVersion != previous.AppVersion || req.State == "updated" || req.State == "up_to_date" {
		server.cleanAppReleasesSoon()
	}
	server.adminHub.broadcastAll(adminWSMessage{Type: "app_update_changed", BranchID: previous.BranchID})
	server.writeJSON(ctx, http.StatusOK, envelope{"success": true})
}

func truncateRunes(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max])
}
