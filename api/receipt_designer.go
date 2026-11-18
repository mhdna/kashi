package api

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	db "github.com/mhdna/kashi/db/sqlc"
	"github.com/mhdna/kashi/receiptrender"
)

// The Receipt Designer draws a branch's real receipt with the design being
// edited, using the till's own renderer (receiptrender, copied from kashi-pos
// by make receipt-renderer), so what it shows is what the till prints.

func (server *Server) getReceiptDesignerCatalog(ctx *gin.Context) {
	server.writeJSON(ctx, http.StatusOK, envelope{"catalog": receiptrender.DesignerCatalog()})
}

type receiptPreviewRequest struct {
	BranchID int64  `form:"branch_id"`
	Theme    string `form:"theme"`
	Layout   string `form:"layout"`
	Mode     string `form:"mode" binding:"omitempty,oneof=sale referral client exchange return item_deal"`
	// texts is the design as receipt_texts JSON.
	Texts string `form:"texts"`
	// shop, when set, is the shop's details being edited (DesignerShop JSON,
	// fields present override the branch's saved ones), so the receipt shows
	// them before they are saved.
	Shop string `form:"shop"`
	// deal, when set, is an invoice discount applied to the sale
	// (DesignerDeal JSON: name, kind, percent, amount), so its line shows.
	Deal string `form:"deal"`
}

// designerShop is the branch's own data the receipt prints, from its
// settings; a branch with none yet (or none chosen) prints the sample shop.
func designerShop(s db.BranchSetting) receiptrender.DesignerShop {
	shop := receiptrender.DesignerShop{
		StoreName: s.MarketName, Branch: s.BranchName, Phone: firstNonEmpty(s.BranchPhone, s.MarketPhone),
		Website: s.Website, ReturnPolicy: s.ReturnPolicy, Font: s.ReceiptFont, BodyFont: s.ReceiptBodyFont,
		TitleSize: s.ReceiptTitleSize, BodySize: s.ReceiptBodySize, ReceiptWidth: s.ReceiptWidth,
	}
	_ = json.Unmarshal(s.SocialPlatforms, &shop.SocialPlatforms)
	_ = json.Unmarshal(s.SocialHandles, &shop.SocialHandles)
	return shop
}

func (server *Server) getReceiptDesignerPreview(ctx *gin.Context) {
	var req receiptPreviewRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	if len(req.Texts) > 64*1024 || len(req.Shop) > 16*1024 {
		server.writeError(ctx, http.StatusBadRequest, errors.New("the design is too long"))
		return
	}
	in := receiptrender.DesignerInput{Theme: req.Theme, Layout: req.Layout, Mode: req.Mode, Texts: req.Texts}
	if req.BranchID > 0 {
		settings, err := server.store.GetBranchSettings(ctx, req.BranchID)
		switch {
		case err == nil:
			in.Shop = designerShop(settings)
		case !errors.Is(err, sql.ErrNoRows):
			server.writeError(ctx, http.StatusInternalServerError, err)
			return
		}
	}
	if strings.TrimSpace(req.Shop) != "" {
		// Over the saved details: only the fields sent change.
		if err := json.Unmarshal([]byte(req.Shop), &in.Shop); err != nil {
			server.writeError(ctx, http.StatusBadRequest, errors.New("the shop details can't be read"))
			return
		}
	}
	if strings.TrimSpace(req.Deal) != "" {
		var deal receiptrender.DesignerDeal
		if err := json.Unmarshal([]byte(req.Deal), &deal); err != nil || len(deal.Name) > 200 {
			server.writeError(ctx, http.StatusBadRequest, errors.New("the deal can't be read"))
			return
		}
		in.Deal = &deal
	}
	if program, err := server.store.GetReferralProgram(ctx); err == nil {
		in.ReferralPercent = int64(program.RewardPercent)
	}
	out, err := receiptrender.RenderDesignerPreview(in)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	server.writeJSON(ctx, http.StatusOK, envelope{
		"image":  "data:image/png;base64," + base64.StdEncoding.EncodeToString(out.PNG),
		"width":  out.Width,
		"height": out.Height,
		"marks":  out.Marks,
		// Each line whose words can change, to type over it on the paper.
		"lines": out.Lines,
		// The design as the till will keep it.
		"texts": receiptrender.NormalizeReceiptTextsJSON(strings.TrimSpace(req.Texts)),
	})
}

// getReceiptDesignerBanner draws one banner (b: its JSON) alone, as the till
// prints it, for the designer's gallery of ready-made banners.
func (server *Server) getReceiptDesignerBanner(ctx *gin.Context) {
	raw := ctx.Query("b")
	if raw == "" || len(raw) > 4096 {
		server.writeError(ctx, http.StatusBadRequest, errors.New("which banner?"))
		return
	}
	png, err := receiptrender.RenderBannerPreview(raw)
	if err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	server.writeJSON(ctx, http.StatusOK, envelope{"image": "data:image/png;base64," + base64.StdEncoding.EncodeToString(png)})
}
