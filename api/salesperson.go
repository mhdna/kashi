package api

import (
	"database/sql"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	db "github.com/mhdna/kashi/db/sqlc"
)

// listSalespersons returns every salesperson on a branch's roster,
// including inactive ones -- the admin UI needs to see and reactivate a
// deactivated salesperson, not just the active subset kashi-pos syncs down.
func (server *Server) listSalespersons(ctx *gin.Context) {
	var uri branchIDRequest
	if err := ctx.ShouldBindUri(&uri); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}

	salespersons, err := server.store.ListSalespersonsForBranch(ctx, sql.NullInt64{Int64: uri.ID, Valid: true})
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	ctx.JSON(http.StatusOK, envelope{"salespersons": salespersons})
}

type createSalespersonRequest struct {
	Name string `json:"name" binding:"required"`
	// Optional user number on the branch's fingerprint device.
	AttendanceUserID string `json:"attendance_user_id"`
	// Whether tills offer them at checkout. Nil keeps the old meaning of
	// this endpoint: everyone added here sells.
	IsSalesperson *bool `json:"is_salesperson"`
}

var errAttendanceIDTaken = errors.New("another salesperson of this branch already has that attendance ID")

// attendanceIDTaken reports whether a different salesperson of the branch
// already uses this device number -- the till matches punches by it, so two
// people sharing one would get each other's attendance.
func (server *Server) attendanceIDTaken(ctx *gin.Context, branchID sql.NullInt64, attendanceUserID string, exceptID int64) (bool, error) {
	if attendanceUserID == "" || !branchID.Valid {
		return false, nil
	}
	_, err := server.store.GetSalespersonByAttendanceID(ctx, db.GetSalespersonByAttendanceIDParams{
		BranchID: branchID, AttendanceUserID: attendanceUserID, ID: exceptID,
	})
	if err == sql.ErrNoRows {
		return false, nil
	}
	return err == nil, err
}

func (server *Server) createSalesperson(ctx *gin.Context) {
	var uri branchIDRequest
	if err := ctx.ShouldBindUri(&uri); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}

	var req createSalespersonRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}

	branchID := sql.NullInt64{Int64: uri.ID, Valid: true}
	req.AttendanceUserID = strings.TrimSpace(req.AttendanceUserID)
	if taken, err := server.attendanceIDTaken(ctx, branchID, req.AttendanceUserID, 0); err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	} else if taken {
		server.writeError(ctx, http.StatusConflict, errAttendanceIDTaken)
		return
	}

	salesperson, err := server.store.CreateBranchSalesperson(ctx, db.CreateBranchSalespersonParams{
		Name:          req.Name,
		BranchID:      branchID,
		IsSalesperson: orTrue(req.IsSalesperson),
	})
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	if req.AttendanceUserID != "" {
		salesperson, err = server.store.SetSalespersonAttendanceID(ctx, db.SetSalespersonAttendanceIDParams{
			ID: salesperson.ID, AttendanceUserID: req.AttendanceUserID,
		})
		if err != nil {
			server.writeError(ctx, http.StatusInternalServerError, err)
			return
		}
	}

	server.branchHub.notify(uri.ID, branchWSMessage{Type: "salesperson_updated"})
	ctx.JSON(http.StatusOK, salesperson)
}

// The id and the name are bound apart: binding the URI validates the whole
// struct, and a required name isn't in the URI, so one struct for both
// refused every rename.
type updateSalespersonURI struct {
	ID int64 `uri:"id" binding:"required,min=1"`
}

type updateSalespersonRequest struct {
	ID   int64  `json:"-"`
	Name string `json:"name" binding:"required"`
}

func (server *Server) updateSalesperson(ctx *gin.Context) {
	var uri updateSalespersonURI
	if err := ctx.ShouldBindUri(&uri); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	req := updateSalespersonRequest{ID: uri.ID}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}

	existing, err := server.store.GetSalesperson(ctx, req.ID)
	if err != nil {
		if err == sql.ErrNoRows {
			server.writeError(ctx, http.StatusNotFound, err)
			return
		}
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	salesperson, err := server.store.UpdateBranchSalespersonName(ctx, db.UpdateBranchSalespersonNameParams{
		ID:       req.ID,
		Name:     req.Name,
		BranchID: existing.BranchID,
	})
	if err != nil {
		if err == sql.ErrNoRows {
			server.writeError(ctx, http.StatusNotFound, err)
			return
		}
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	if existing.BranchID.Valid {
		server.branchHub.notify(existing.BranchID.Int64, branchWSMessage{Type: "salesperson_updated"})
	}
	ctx.JSON(http.StatusOK, salesperson)
}

type setSalespersonActiveRequest struct {
	ID       int64 `uri:"id" binding:"required,min=1"`
	IsActive bool  `json:"is_active"`
}

func (server *Server) setSalespersonActive(ctx *gin.Context) {
	var req setSalespersonActiveRequest
	if err := ctx.ShouldBindUri(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}

	salesperson, err := server.store.SetSalespersonActive(ctx, db.SetSalespersonActiveParams{
		ID:       req.ID,
		IsActive: req.IsActive,
	})
	if err != nil {
		if err == sql.ErrNoRows {
			server.writeError(ctx, http.StatusNotFound, err)
			return
		}
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	if salesperson.BranchID.Valid {
		server.branchHub.notify(salesperson.BranchID.Int64, branchWSMessage{Type: "salesperson_updated"})
	}
	ctx.JSON(http.StatusOK, salesperson)
}

type setSalespersonIsSalespersonRequest struct {
	ID            int64 `uri:"id" binding:"required,min=1"`
	IsSalesperson bool  `json:"is_salesperson"`
}

// setSalespersonIsSalesperson marks whether the branch's tills offer this
// person at checkout. Someone who only punches attendance (a guard, a
// cleaner) stays on the roster with it off.
func (server *Server) setSalespersonIsSalesperson(ctx *gin.Context) {
	var req setSalespersonIsSalespersonRequest
	if err := ctx.ShouldBindUri(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	salesperson, err := server.store.SetSalespersonIsSalesperson(ctx, db.SetSalespersonIsSalespersonParams{
		ID: req.ID, IsSalesperson: req.IsSalesperson,
	})
	if err != nil {
		if err == sql.ErrNoRows {
			server.writeError(ctx, http.StatusNotFound, err)
			return
		}
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	if salesperson.BranchID.Valid {
		server.branchHub.notify(salesperson.BranchID.Int64, branchWSMessage{Type: "salesperson_updated"})
	}
	ctx.JSON(http.StatusOK, salesperson)
}

type setSalespersonAttendanceIDURI struct {
	ID int64 `uri:"id" binding:"required,min=1"`
}

type setSalespersonAttendanceIDRequest struct {
	AttendanceUserID string `json:"attendance_user_id"`
}

// setSalespersonAttendanceID sets (or, with "", clears) the salesperson's
// user number on their branch's fingerprint device. From then on kashi
// decides it and the till takes this value on its next salesperson pull.
func (server *Server) setSalespersonAttendanceID(ctx *gin.Context) {
	var uri setSalespersonAttendanceIDURI
	if err := ctx.ShouldBindUri(&uri); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	var req setSalespersonAttendanceIDRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	req.AttendanceUserID = strings.TrimSpace(req.AttendanceUserID)

	existing, err := server.store.GetSalesperson(ctx, uri.ID)
	if err != nil {
		if err == sql.ErrNoRows {
			server.writeError(ctx, http.StatusNotFound, err)
			return
		}
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	if taken, err := server.attendanceIDTaken(ctx, existing.BranchID, req.AttendanceUserID, existing.ID); err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	} else if taken {
		server.writeError(ctx, http.StatusConflict, errAttendanceIDTaken)
		return
	}

	salesperson, err := server.store.SetSalespersonAttendanceID(ctx, db.SetSalespersonAttendanceIDParams{
		ID: uri.ID, AttendanceUserID: req.AttendanceUserID,
	})
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	if salesperson.BranchID.Valid {
		server.branchHub.notify(salesperson.BranchID.Int64, branchWSMessage{Type: "salesperson_updated"})
	}
	ctx.JSON(http.StatusOK, salesperson)
}

type setSalespersonColorURI struct {
	ID int64 `uri:"id" binding:"required,min=1"`
}

// Separate from the URI struct: binding the URI validates every field, so a
// required body field there would fail before the body is read.
type setSalespersonColorRequest struct {
	Color string `json:"color" binding:"required,hexcolor"`
}

// setSalespersonColor changes the avatar color kashi-pos shows for this
// salesperson. A random one is assigned on creation (see migration 000112).
func (server *Server) setSalespersonColor(ctx *gin.Context) {
	var uri setSalespersonColorURI
	if err := ctx.ShouldBindUri(&uri); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	var req setSalespersonColorRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}

	salesperson, err := server.store.SetSalespersonColor(ctx, db.SetSalespersonColorParams{
		ID:    uri.ID,
		Color: strings.ToUpper(req.Color),
	})
	if err != nil {
		if err == sql.ErrNoRows {
			server.writeError(ctx, http.StatusNotFound, err)
			return
		}
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	if salesperson.BranchID.Valid {
		server.branchHub.notify(salesperson.BranchID.Int64, branchWSMessage{Type: "salesperson_updated"})
	}
	ctx.JSON(http.StatusOK, salesperson)
}
