package api

import (
	"crypto/rand"
	"database/sql"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	db "github.com/mhdna/kashi/db/sqlc"
)

// couponListCodeAlphabet excludes visually ambiguous characters (0/O, 1/I/L)
// -- these codes are meant to be read off a printed voucher or typed at
// checkout, unlike the hex upload tokens in shift_upload.go which are
// never hand-typed.
const couponListCodeAlphabet = "ABCDEFGHJKMNPQRSTUVWXYZ23456789"

const (
	couponListCodeLength = 8
	couponListMaxCount   = 5000
)

// generateCouponListCode mints one random, human-typeable code. Collisions
// are handled by the caller retrying on the table's unique-code-per-row
// PK violation, same as branch invoice client_ref retries elsewhere.
func generateCouponListCode() (string, error) {
	raw := make([]byte, couponListCodeLength)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("generate coupon code: %w", err)
	}
	var b strings.Builder
	for _, c := range raw {
		b.WriteByte(couponListCodeAlphabet[int(c)%len(couponListCodeAlphabet)])
	}
	return b.String(), nil
}

type createCouponListRequest struct {
	Name         string `json:"name" binding:"required"`
	CouponTypeID int64  `json:"coupon_type_id" binding:"required"`
	Count        int    `json:"count" binding:"required,min=1"`
	ValidUntil   string `json:"valid_until"`
}

// createCouponList mints Count distinct single-use coupons under one
// campaign in one request. Each code gets a handful of collision retries
// (astronomically unlikely at this alphabet/length, but cheap to guard);
// a code that still can't be minted after that is skipped and reported,
// not a hard failure for the whole batch -- an admin generating 2000
// codes shouldn't lose all of them over one unlucky draw.
func (server *Server) createCouponList(ctx *gin.Context) {
	var req createCouponListRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	if req.Count > couponListMaxCount {
		server.writeError(ctx, http.StatusBadRequest, fmt.Errorf("count must be %d or fewer", couponListMaxCount))
		return
	}

	couponType, err := server.store.GetCouponType(ctx, req.CouponTypeID)
	if err != nil {
		if err == sql.ErrNoRows {
			server.writeError(ctx, http.StatusBadRequest, err)
			return
		}
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	var validUntil sql.NullTime
	if req.ValidUntil != "" {
		parsed, err := time.Parse(time.RFC3339, req.ValidUntil)
		if err != nil {
			server.writeError(ctx, http.StatusBadRequest, err)
			return
		}
		validUntil = sql.NullTime{Time: parsed, Valid: true}
	}

	list, err := server.store.CreateCouponList(ctx, db.CreateCouponListParams{
		Name:         req.Name,
		CouponTypeID: couponType.ID,
	})
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	codes := make([]string, 0, req.Count)
	for i := 0; i < req.Count; i++ {
		var created bool
		for attempt := 0; attempt < 5; attempt++ {
			code, err := generateCouponListCode()
			if err != nil {
				server.writeError(ctx, http.StatusInternalServerError, err)
				return
			}
			if _, err := server.store.CreateListCoupon(ctx, db.CreateListCouponParams{
				Code:       code,
				TypeID:     sql.NullInt64{Int64: couponType.ID, Valid: true},
				ListID:     sql.NullInt64{Int64: list.ID, Valid: true},
				ValueCents: couponType.ValueCents,
				ValidUntil: validUntil,
			}); err != nil {
				if isUniqueViolation(err) {
					continue
				}
				server.writeError(ctx, http.StatusInternalServerError, err)
				return
			}
			codes = append(codes, code)
			created = true
			break
		}
		_ = created
	}

	ctx.JSON(http.StatusOK, gin.H{"list": list, "codes": codes, "requested": req.Count, "generated": len(codes)})
}

func (server *Server) listCouponLists(ctx *gin.Context) {
	lists, err := server.store.ListCouponLists(ctx)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"coupon_lists": lists})
}

type couponListIDURI struct {
	ID int64 `uri:"id" binding:"required,min=1"`
}

// deactivateCouponList closes the campaign and, in the same request,
// deactivates every code it minted that's still 'active' -- a code
// already redeemed keeps its history, one that's redeemed on a
// currently-disconnected till still lands (RedeemCoupon only ever moves
// 'active' -> 'redeemed', never the reverse) but won't be usable again.
func (server *Server) deactivateCouponList(ctx *gin.Context) {
	var uri couponListIDURI
	if err := ctx.ShouldBindUri(&uri); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	if err := server.store.DeactivateCouponList(ctx, uri.ID); err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	rows, err := server.store.DeactivateCouponsForList(ctx, sql.NullInt64{Int64: uri.ID, Valid: true})
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"message": "coupon list deactivated", "coupons_deactivated": rows})
}
