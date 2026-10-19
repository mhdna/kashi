package api

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Barcode label design endpoints (the model is in barcode_label.go).
//
//	GET  /barcodes/design          the saved design, the themes with previews, the label size
//	PUT  /barcodes/design          save a design
//	POST /barcodes/design/preview  draw a design as SVG, for the editor's live preview

// sampleLabelItems are the items a design is previewed with: a typical one, one
// whose long name tests the layout, and one with no price.
func sampleLabelItems() []labelItem {
	return []labelItem{
		{Name: "Blue Linen Shirt", Code: "SH-1042", Barcode: "5901234123457", Price: "39.00", Color: "Navy", Size: "M", Brand: "Maison", Qty: 1},
		{Name: "Black Slim Fit Stretch Jeans", Code: "JN-220", Barcode: "4006381333931", Price: "125.50", Color: "Black", Size: "32", Brand: "Denim Co", Qty: 1},
		{Name: "Cotton Socks", Code: "SK-007", Barcode: "9780201379624", Brand: "Basics", Qty: 1},
	}
}

// previewLabelSVG draws design on a width x height label with the sample item.
func previewLabelSVG(design labelDesign, width, height float64, sample int, borders bool) string {
	items := sampleLabelItems()
	if sample < 0 || sample >= len(items) {
		sample = 0
	}
	design.normalize()
	if width <= 0 {
		width = labelDesignW
	}
	if height <= 0 {
		height = labelDesignH
	}
	bars, _ := ean13Bars(items[sample].Barcode)
	return labelSVG(layoutLabel(items[sample], bars, design, borders), design.Font, width, height)
}

type barcodeDesignThemeResponse struct {
	labelTheme
	SVG string `json:"svg"`
}

func (server *Server) getBarcodeDesign(ctx *gin.Context) {
	settings, err := server.store.GetAppSettings(ctx)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	width, height := float64(settings.BarcodeLabelWidth), float64(settings.BarcodeLabelHeight)

	themes := labelThemes()
	out := make([]barcodeDesignThemeResponse, 0, len(themes))
	for _, theme := range themes {
		out = append(out, barcodeDesignThemeResponse{theme, previewLabelSVG(theme.Design, width, height, 0, false)})
	}
	server.writeJSON(ctx, http.StatusOK, envelope{
		"saved":         parseSavedLabelDesign(settings.BarcodeLabelDesign),
		"themes":        out,
		"label":         envelope{"width": width, "height": height},
		"sample_count":  len(sampleLabelItems()),
		"default_theme": defaultLabelTheme,
	})
}

type saveBarcodeDesignRequest struct {
	Theme  string      `json:"theme"`
	Design labelDesign `json:"design"`
}

func (server *Server) saveBarcodeDesign(ctx *gin.Context) {
	var req saveBarcodeDesignRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	if _, ok := findLabelTheme(req.Theme); !ok {
		req.Theme = defaultLabelTheme
	}
	req.Design.normalize()

	encoded, err := json.Marshal(savedLabelDesign{Theme: req.Theme, Design: req.Design})
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	settings, err := server.store.SetBarcodeLabelDesign(ctx, encoded)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	server.writeJSON(ctx, http.StatusOK, envelope{"saved": parseSavedLabelDesign(settings.BarcodeLabelDesign)})
}

type previewBarcodeDesignRequest struct {
	Design labelDesign `json:"design"`
	Width  float64     `json:"width"`
	Height float64     `json:"height"`
	Sample int         `json:"sample"`
	// Borders previews the side strips that print when separator pages are on.
	Borders bool `json:"borders"`
}

func (server *Server) previewBarcodeDesign(ctx *gin.Context) {
	var req previewBarcodeDesignRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	if req.Width <= 0 || req.Height <= 0 {
		settings, err := server.store.GetAppSettings(ctx)
		if err != nil {
			server.writeError(ctx, http.StatusInternalServerError, err)
			return
		}
		req.Width, req.Height = float64(settings.BarcodeLabelWidth), float64(settings.BarcodeLabelHeight)
	}
	server.writeJSON(ctx, http.StatusOK, envelope{"svg": previewLabelSVG(req.Design, req.Width, req.Height, req.Sample, req.Borders)})
}
