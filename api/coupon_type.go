package api

import (
	"database/sql"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	db "github.com/mhdna/kashi/db/sqlc"
)

var errCouponTypeInUse = errors.New("coupon type still has coupons")

type createCouponTypeRequest struct {
	Name                string `json:"name" binding:"required"`
	ValueCents          int64  `json:"value_cents" binding:"required,min=1"`
	CurrencyCode        string `json:"currency_code"`
	ReferenceOnly       bool   `json:"reference_only"`
	CalculateOnCheckout bool   `json:"calculate_on_checkout"`
	CodeOptional        bool   `json:"code_optional"`
	CategoryID          int64  `json:"category_id"`
}

func (server *Server) createCouponType(ctx *gin.Context) {
	var req createCouponTypeRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}

	currencyCode := req.CurrencyCode
	if currencyCode == "" {
		currencyCode = "USD"
	}

	couponType, err := server.store.CreateCouponType(ctx, db.CreateCouponTypeParams{
		Name:                req.Name,
		ValueCents:          req.ValueCents,
		CurrencyCode:        currencyCode,
		ReferenceOnly:       req.ReferenceOnly,
		CalculateOnCheckout: req.CalculateOnCheckout,
		CodeOptional:        req.CodeOptional,
		CategoryID:          couponCategoryID(req.CategoryID),
	})
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	ctx.JSON(http.StatusOK, couponType)
}

type couponTypeIDURI struct {
	ID int64 `uri:"id" binding:"required,min=1"`
}

func (server *Server) getCouponType(ctx *gin.Context) {
	var uri couponTypeIDURI
	if err := ctx.ShouldBindUri(&uri); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	couponType, err := server.store.GetCouponType(ctx, uri.ID)
	if err != nil {
		if err == sql.ErrNoRows {
			server.writeError(ctx, http.StatusNotFound, err)
			return
		}
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	ctx.JSON(http.StatusOK, couponType)
}

func (server *Server) listCouponTypes(ctx *gin.Context) {
	couponTypes, err := server.store.ListCouponTypes(ctx)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"coupon_types": couponTypes})
}

type updateCouponTypeRequest struct {
	Name                string `json:"name" binding:"required"`
	ValueCents          int64  `json:"value_cents" binding:"required,min=1"`
	CurrencyCode        string `json:"currency_code"`
	IsActive            bool   `json:"is_active"`
	ReferenceOnly       bool   `json:"reference_only"`
	CalculateOnCheckout bool   `json:"calculate_on_checkout"`
	CodeOptional        bool   `json:"code_optional"`
	CategoryID          int64  `json:"category_id"`
}

func (server *Server) updateCouponType(ctx *gin.Context) {
	var uri couponTypeIDURI
	if err := ctx.ShouldBindUri(&uri); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	var req updateCouponTypeRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}

	currencyCode := req.CurrencyCode
	if currencyCode == "" {
		currencyCode = "USD"
	}

	couponType, err := server.store.UpdateCouponType(ctx, db.UpdateCouponTypeParams{
		ID:                  uri.ID,
		Name:                req.Name,
		ValueCents:          req.ValueCents,
		CurrencyCode:        currencyCode,
		IsActive:            req.IsActive,
		ReferenceOnly:       req.ReferenceOnly,
		CalculateOnCheckout: req.CalculateOnCheckout,
		CodeOptional:        req.CodeOptional,
		CategoryID:          couponCategoryID(req.CategoryID),
	})
	if err != nil {
		if err == sql.ErrNoRows {
			server.writeError(ctx, http.StatusNotFound, err)
			return
		}
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	ctx.JSON(http.StatusOK, couponType)
}

// deleteCouponType refuses a type still in use, surfaced as a 409 --
// mirrors deleteLoanCategory's foreign-key-violation handling.
func (server *Server) deleteCouponType(ctx *gin.Context) {
	var uri couponTypeIDURI
	if err := ctx.ShouldBindUri(&uri); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	count, err := server.store.CountCouponsByTypeID(ctx, sql.NullInt64{Int64: uri.ID, Valid: true})
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	if count > 0 {
		server.writeError(ctx, http.StatusConflict, errCouponTypeInUse)
		return
	}
	if err := server.store.DeleteCouponType(ctx, uri.ID); err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"message": "coupon type deleted"})
}
