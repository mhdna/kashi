package api

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lib/pq"
	db "github.com/mhdna/kashi/db/sqlc"
	"github.com/mhdna/kashi/token"
	"github.com/mhdna/kashi/util"
)

// Passwords for cloud (admin site) users. This is a separate list from the
// POS operators in branch_users, whose 4-6 digit PINs are pushed down to each
// till (see branch_user.go). Accounts created before this rule existed may
// still have a short PIN as their password; they keep working until changed.
const (
	passwordMinLen = 8
	// bcrypt only hashes the first 72 bytes and errors past that.
	passwordMaxLen = 72
)

var errPasswordFormat = errors.New("password must be 8 to 72 characters")

func validPassword(s string) bool {
	return len(s) >= passwordMinLen && len(s) <= passwordMaxLen
}

// A username is what a user signs in with; `name` is the display name shown
// everywhere else. Usernames are stored lowercase (the column is citext, so
// sign-in ignores case anyway) and kept to characters a phone keyboard won't
// mangle.
const usernameMaxLen = 32

var (
	usernamePattern   = regexp.MustCompile(`^[a-z0-9._-]+$`)
	errUsernameFormat = errors.New("username must be up to 32 lowercase letters, digits, '.', '_' or '-'")
	errNameRequired   = errors.New("display name is required")
	errUsernameTaken  = errors.New("username is already taken")
)

// normalizeUserNames lowercases the username and capitalises each word of the
// display name, and rejects either if it isn't usable.
func normalizeUserNames(username, name string) (string, string, error) {
	username = strings.ToLower(strings.TrimSpace(username))
	if len(username) > usernameMaxLen || !usernamePattern.MatchString(username) {
		return "", "", errUsernameFormat
	}
	name = util.CapitalizeWords(name)
	if name == "" {
		return "", "", errNameRequired
	}
	return username, name, nil
}

type createUserRequest struct {
	Username  string `json:"username" binding:"required"`
	Name      string `json:"name" binding:"required"`
	Password  string `json:"password" binding:"required"`
	Activated bool   `json:"activated"`
	RoleID    int64  `json:"role_id" binding:"required,min=1"`
}

type userResponse struct {
	ID                int64     `json:"id"`
	Username          string    `json:"username"`
	Name              string    `json:"name"`
	Activated         bool      `json:"activated"`
	RoleID            int64     `json:"role_id"`
	PasswordChangedAt time.Time `json:"password_changed_at"`
	CreatedAt         time.Time `json:"created_at"`
}

func newUserResponse(user db.User) userResponse {
	return userResponse{
		ID:                user.ID,
		Username:          user.Username,
		Name:              user.Name,
		Activated:         user.Activated,
		RoleID:            user.RoleID,
		PasswordChangedAt: user.PasswordChangedAt,
		CreatedAt:         user.CreatedAt,
	}
}

// getMe returns the caller's own user and access so the UI can hide what
// their role can't use. It's re-fetched on load rather than trusted from the
// login response, so a role change shows up without logging out.
func (server *Server) getMe(ctx *gin.Context) {
	payload := ctx.MustGet(authoizationPayloadKey).(*token.Payload)
	user, err := server.store.GetUser(ctx, payload.UserID)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	access, _ := currentAccess(ctx)
	server.writeJSON(ctx, http.StatusOK, envelope{"user": newUserResponse(user), "access": access})
}

// callerID is the signed-in user's id. It comes from the token, not a lookup
// by username, so renaming a username doesn't break sessions already open.
// Tokens issued before Payload.UserID existed carry 0 and fall back to the
// username.
func (server *Server) callerID(ctx context.Context, payload *token.Payload) (int64, error) {
	if payload.UserID != 0 {
		return payload.UserID, nil
	}
	user, err := server.store.GetUserByUsername(ctx, payload.Username)
	if err != nil {
		return 0, err
	}
	return user.ID, nil
}

// displayName is the caller's display name for text shown to other people.
// The token only carries the username; fall back to it if the lookup fails.
func (server *Server) displayName(ctx context.Context, payload *token.Payload) string {
	if user, err := server.store.GetUser(ctx, payload.UserID); err == nil {
		return user.Name
	}
	return payload.Username
}

// listOnlineUsers gives the admin UI an initial value for "who is signed in
// right now" on load -- the live figure after that comes from the
// "admin_presence_changed" push over the admin WebSocket (see adminWS in
// admin_ws.go), same poll-then-push pattern as listConnectedBranches.
func (server *Server) listOnlineUsers(ctx *gin.Context) {
	server.writeJSON(ctx, http.StatusOK, envelope{"users": server.adminHub.onlineUsers()})
}

func (server *Server) createUser(ctx *gin.Context) {
	var req createUserRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}

	username, name, err := normalizeUserNames(req.Username, req.Name)
	if err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}

	if !validPassword(req.Password) {
		server.writeError(ctx, http.StatusBadRequest, errPasswordFormat)
		return
	}

	hashPassword, err := util.HashPassword(req.Password)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	arg := db.CreateUserParams{
		Username: username,
		Name:     name,
		// email is unused in this POS but the column is NOT NULL and unique;
		// fill it with a throwaway value so it never has to be entered.
		Email:        util.RandomEmail(),
		PasswordHash: hashPassword,
		Activated:    req.Activated,
		RoleID:       req.RoleID,
	}

	user, err := server.store.CreateUser(ctx, arg)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok {
			switch pqErr.Code.Name() {
			case "foreign_key_violation":
				server.writeError(ctx, http.StatusBadRequest, errors.New("unknown role"))
				return
			case "unique_violation":
				server.writeError(ctx, http.StatusConflict, errUsernameTaken)
				return
			}
		}
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	rsp := newUserResponse(user)

	ctx.JSON(http.StatusOK, rsp)
}

type getUserRequest struct {
	ID int64 `uri:"id" binding:"required,min=1"`
}

func (server *Server) getUser(ctx *gin.Context) {
	var req getUserRequest
	if err := ctx.ShouldBindUri(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}

	user, err := server.store.GetUser(ctx, req.ID)
	if err != nil {
		if err == sql.ErrNoRows {
			server.writeError(ctx, http.StatusNotFound, err)
			return
		}

		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	ctx.JSON(http.StatusOK, newUserResponse(user))
}

type listUsersRequest struct {
	PageSize int32 `form:"page_size,default=10" binding:"min=5,max=10"`
	PageID   int32 `form:"page_id,default=0" binding:"min=0"`
}

func (server *Server) listUsers(ctx *gin.Context) {
	var req listUsersRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}

	arg := db.ListUsersParams{
		Limit:  req.PageSize,
		Offset: req.PageID,
	}
	users, err := server.store.ListUsers(ctx, arg)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	rsp := make([]userResponse, len(users))
	for i, u := range users {
		rsp[i] = newUserResponse(u)
	}
	ctx.JSON(http.StatusOK, rsp)
}

type updateUserRequest struct {
	ID        int64  `json:"id" binding:"required,min=1"`
	Username  string `json:"username" binding:"required"`
	Name      string `json:"name" binding:"required"`
	Password  string `json:"password"` // optional -- blank keeps the current password
	Activated bool   `json:"activated"`
	RoleID    int64  `json:"role_id" binding:"required,min=1"`
}

func (server *Server) updateUser(ctx *gin.Context) {
	var req updateUserRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}

	username, name, err := normalizeUserNames(req.Username, req.Name)
	if err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}

	current, err := server.store.GetUser(ctx, req.ID)
	if err != nil {
		if err == sql.ErrNoRows {
			server.writeError(ctx, http.StatusNotFound, err)
			return
		}
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	// Changing a user's role must leave someone who can still manage users and
	// roles.
	if req.RoleID != current.RoleID {
		if err := server.ensureAnotherAdmin(ctx, current); err != nil {
			server.writeAdminGuardError(ctx, err)
			return
		}
	}

	// Keep the existing hash unless a new password was supplied.
	hashPassword := current.PasswordHash
	if req.Password != "" {
		if !validPassword(req.Password) {
			server.writeError(ctx, http.StatusBadRequest, errPasswordFormat)
			return
		}
		hashPassword, err = util.HashPassword(req.Password)
		if err != nil {
			server.writeError(ctx, http.StatusInternalServerError, err)
			return
		}
	}

	arg := db.UpdateUserParams{
		ID:           req.ID,
		Username:     username,
		Name:         name,
		Email:        current.Email,
		PasswordHash: hashPassword,
		Activated:    req.Activated,
		RoleID:       req.RoleID,
	}

	err = server.store.UpdateUser(ctx, arg)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code.Name() == "unique_violation" {
			server.writeError(ctx, http.StatusConflict, errUsernameTaken)
			return
		}
		if err == sql.ErrNoRows {
			server.writeError(ctx, http.StatusBadRequest, err)
			return
		}
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	ctx.JSON(http.StatusOK, userResponse{
		ID:                req.ID,
		Username:          username,
		Name:              name,
		Activated:         req.Activated,
		RoleID:            req.RoleID,
		PasswordChangedAt: current.PasswordChangedAt,
		CreatedAt:         current.CreatedAt,
	})
}

var errLastAdmin = errors.New("there must be at least one admin user left")

// ensureAnotherAdmin is called before a change that could remove `user` from
// the admin set (delete, demote, deactivate). It's a no-op for a non-admin.
func (server *Server) ensureAnotherAdmin(ctx *gin.Context, user db.User) error {
	role, err := server.store.GetRole(ctx, user.RoleID)
	if err != nil {
		return err
	}
	if !role.IsAdmin {
		return nil
	}
	others, err := server.store.CountOtherAdmins(ctx, user.ID)
	if err != nil {
		return err
	}
	if others == 0 {
		return errLastAdmin
	}
	return nil
}

func (server *Server) writeAdminGuardError(ctx *gin.Context, err error) {
	if errors.Is(err, errLastAdmin) {
		server.writeError(ctx, http.StatusConflict, err)
		return
	}
	server.writeError(ctx, http.StatusInternalServerError, err)
}

type deleteUserRequest struct {
	ID int64 `uri:"id" binding:"required,min=1"`
}

func (server *Server) deleteUser(ctx *gin.Context) {
	var req deleteUserRequest
	if err := ctx.ShouldBindUri(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}

	target, err := server.store.GetUser(ctx, req.ID)
	if err != nil {
		if err == sql.ErrNoRows {
			server.writeError(ctx, http.StatusNotFound, err)
			return
		}
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	if err := server.ensureAnotherAdmin(ctx, target); err != nil {
		server.writeAdminGuardError(ctx, err)
		return
	}

	err = server.store.DeleteUser(ctx, req.ID)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"message": "user deleted"})
}

type loginUserRequest struct {
	Username string `json:"username"`
	// Name is what the UI sent before usernames existed; still accepted so a
	// tab loaded before the deploy can sign in.
	Name     string `json:"name"`
	Password string `json:"password" binding:"required"`
}

type loginUserResponse struct {
	SessionID             string       `json:"session_id"`
	AccessToken           string       `json:"access_token"`
	AccessTokenExpiresAt  time.Time    `json:"access_token_expires_at"`
	RefreshToken          string       `json:"refresh_token"`
	RefreshTokenExpiresAt time.Time    `json:"refresh_token_expires_at"`
	User                  userResponse `json:"user"`
	Access                userAccess   `json:"access"`
}

func (server *Server) loginUser(ctx *gin.Context) {
	var req loginUserRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	username := strings.TrimSpace(req.Username)
	if username == "" {
		username = strings.TrimSpace(req.Name)
	}
	if username == "" {
		server.writeError(ctx, http.StatusBadRequest, errors.New("username is required"))
		return
	}
	user, err := server.store.GetUserByUsername(ctx, username)
	if err != nil {
		// Same status as a wrong password, so the response doesn't reveal
		// which usernames exist.
		if err == sql.ErrNoRows {
			server.writeError(ctx, http.StatusUnauthorized, err)
			return
		}
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	err = util.CheckPassword(req.Password, user.PasswordHash)
	if err != nil {
		server.writeError(ctx, http.StatusUnauthorized, err)
		return
	}
	accessToken, accessPayload, err := server.tokenMaker.CreateToken(
		user.ID,
		user.Username,
		server.config.AccessTokenDuration,
	)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	refreshToken, refreshPayload, err := server.tokenMaker.CreateToken(
		user.ID,
		user.Username,
		server.config.RefreshTokenDuration,
	)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	session, err := server.store.CreateSession(ctx, db.CreateSessionParams{
		ID:           refreshPayload.ID,
		Username:     user.Username,
		RefreshToken: refreshToken,
		UserAgent:    ctx.Request.UserAgent(),
		ClientIp:     ctx.ClientIP(),
		IsBlocked:    false,
		ExpiresAt:    refreshPayload.ExpiredAt,
	})
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	access, err := server.loadUserAccess(ctx, user.ID)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	rsp := loginUserResponse{
		SessionID:             session.ID.String(),
		AccessToken:           accessToken,
		AccessTokenExpiresAt:  accessPayload.ExpiredAt,
		RefreshToken:          refreshToken,
		RefreshTokenExpiresAt: refreshPayload.ExpiredAt,
		User:                  newUserResponse(user),
		Access:                access,
	}

	ctx.JSON(http.StatusOK, rsp)
}

type renewAccessTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type renewAccessTokenResponse struct {
	AccessToken          string    `json:"access_token"`
	AccessTokenExpiresAt time.Time `json:"access_token_expires_at"`
}

func (server *Server) renewAccessToken(ctx *gin.Context) {
	var req renewAccessTokenRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}

	refreshPayload, err := server.tokenMaker.VerifyToken(req.RefreshToken)
	if err != nil {
		server.writeError(ctx, http.StatusUnauthorized, err)
		return
	}

	session, err := server.store.GetSession(ctx, refreshPayload.ID)
	if err != nil {
		if err == sql.ErrNoRows {
			server.writeError(ctx, http.StatusNotFound, err)
			return
		}
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	if session.IsBlocked {
		server.writeError(ctx, http.StatusUnauthorized, token.ErrInvalidToken)
		return
	}

	if session.Username != refreshPayload.Username {
		server.writeError(ctx, http.StatusUnauthorized, token.ErrInvalidToken)
		return
	}

	if session.RefreshToken != req.RefreshToken {
		server.writeError(ctx, http.StatusUnauthorized, token.ErrInvalidToken)
		return
	}

	userID, err := server.callerID(ctx, refreshPayload)
	if err != nil {
		if err == sql.ErrNoRows {
			server.writeError(ctx, http.StatusUnauthorized, token.ErrInvalidToken)
			return
		}
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	accessToken, accessPayload, err := server.tokenMaker.CreateToken(
		userID,
		refreshPayload.Username,
		server.config.AccessTokenDuration,
	)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	rsp := renewAccessTokenResponse{
		AccessToken:          accessToken,
		AccessTokenExpiresAt: accessPayload.ExpiredAt,
	}

	ctx.JSON(http.StatusOK, rsp)
}
