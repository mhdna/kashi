package api

import (
	"database/sql"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/lib/pq"
	db "github.com/mhdna/kashi/db/sqlc"
)

// Roles are admin-only: none of these routes has an entry in routeRules
// (access.go), so permissionMiddleware rejects everyone but an is_admin role.

type roleResponse struct {
	ID          int64           `json:"id"`
	Name        string          `json:"name"`
	IsAdmin     bool            `json:"is_admin"`
	UserCount   int64           `json:"user_count"`
	Permissions rolePermissions `json:"permissions"`
}

func (server *Server) listRoles(ctx *gin.Context) {
	roles, err := server.store.ListRoles(ctx)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	perms, err := server.store.ListRolePermissions(ctx)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	byRole := map[int64][]permissionRow{}
	for _, p := range perms {
		byRole[p.RoleID] = append(byRole[p.RoleID], permissionRow{
			Section: p.Section,
			Page:    p.Page,
			Level:   sectionLevel{View: p.CanView, Edit: p.CanEdit},
		})
	}

	rsp := make([]roleResponse, len(roles))
	for i, r := range roles {
		rsp[i] = roleResponse{
			ID:          r.ID,
			Name:        r.Name,
			IsAdmin:     r.IsAdmin,
			UserCount:   r.UserCount,
			Permissions: resolvePermissions(byRole[r.ID]),
		}
	}
	// The section and page list rides along so the role editor renders
	// exactly what this server knows about.
	server.writeJSON(ctx, http.StatusOK, envelope{"roles": rsp, "sections": accessSections})
}

type saveRoleRequest struct {
	Name        string          `json:"name" binding:"required"`
	Permissions rolePermissions `json:"permissions"`
}

// permissionRows turns the editor's grants into the rows to store, validating
// the keys and normalising edit => view. Every page of every section gets a
// row, "none" included: a section with no page rows reads back as a
// pre-per-page role whose pages inherit the section (see
// resolvePermissions), which is not what an editor that set them all to none
// meant.
func permissionRows(perms rolePermissions) ([]db.RolePermission, error) {
	for section := range perms.Sections {
		if !isAccessSection(section) {
			return nil, errors.New("unknown section: " + section)
		}
	}
	for page := range perms.Pages {
		if _, ok := sectionOfPage(page); !ok {
			return nil, errors.New("unknown page: " + page)
		}
	}

	var rows []db.RolePermission
	for _, s := range accessSections {
		if perms.Sections[s.Key] {
			rows = append(rows, db.RolePermission{Section: s.Key, CanView: true})
		}
		for _, p := range s.Pages {
			level := perms.Pages[p.Key]
			rows = append(rows, db.RolePermission{
				Section: s.Key,
				Page:    p.Key,
				CanView: level.View || level.Edit,
				CanEdit: level.Edit,
			})
		}
	}
	return rows, nil
}

func (server *Server) createRole(ctx *gin.Context) {
	var req saveRoleRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		server.writeError(ctx, http.StatusBadRequest, errors.New("role name is required"))
		return
	}
	perms, err := permissionRows(req.Permissions)
	if err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}

	role, err := server.store.CreateRole(ctx, req.Name)
	if err != nil {
		server.writeRoleError(ctx, err)
		return
	}
	if err := server.store.SaveRolePermissionsTx(ctx, role.ID, perms); err != nil {
		// Don't leave a role with no access behind because the second step failed.
		_ = server.store.DeleteRole(ctx, role.ID)
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	server.writeJSON(ctx, http.StatusOK, envelope{"role": role})
}

type roleURI struct {
	ID int64 `uri:"id" binding:"required,min=1"`
}

func (server *Server) updateRole(ctx *gin.Context) {
	var uri roleURI
	if err := ctx.ShouldBindUri(&uri); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	var req saveRoleRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		server.writeError(ctx, http.StatusBadRequest, errors.New("role name is required"))
		return
	}

	role, err := server.store.GetRole(ctx, uri.ID)
	if err != nil {
		if err == sql.ErrNoRows {
			server.writeError(ctx, http.StatusNotFound, err)
			return
		}
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	if role.IsAdmin {
		server.writeError(ctx, http.StatusConflict, errors.New("the admin role can't be changed"))
		return
	}
	perms, err := permissionRows(req.Permissions)
	if err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}

	if err := server.store.RenameRole(ctx, db.RenameRoleParams{ID: role.ID, Name: req.Name}); err != nil {
		server.writeRoleError(ctx, err)
		return
	}
	if err := server.store.SaveRolePermissionsTx(ctx, role.ID, perms); err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	server.writeJSON(ctx, http.StatusOK, envelope{"message": "role updated"})
}

func (server *Server) deleteRole(ctx *gin.Context) {
	var uri roleURI
	if err := ctx.ShouldBindUri(&uri); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	role, err := server.store.GetRole(ctx, uri.ID)
	if err != nil {
		if err == sql.ErrNoRows {
			server.writeError(ctx, http.StatusNotFound, err)
			return
		}
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	if role.IsAdmin {
		server.writeError(ctx, http.StatusConflict, errors.New("the admin role can't be deleted"))
		return
	}
	if err := server.store.DeleteRole(ctx, role.ID); err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code.Name() == "foreign_key_violation" {
			server.writeError(ctx, http.StatusConflict, errors.New("users are still assigned to this role"))
			return
		}
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	server.writeJSON(ctx, http.StatusOK, envelope{"message": "role deleted"})
}

func (server *Server) writeRoleError(ctx *gin.Context, err error) {
	if pqErr, ok := err.(*pq.Error); ok && pqErr.Code.Name() == "unique_violation" {
		server.writeError(ctx, http.StatusConflict, errors.New("a role with that name already exists"))
		return
	}
	server.writeError(ctx, http.StatusInternalServerError, err)
}
