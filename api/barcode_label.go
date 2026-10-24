package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

// Barcode label design.
//
// A label is drawn from a labelDesign: which elements show (price, name,
// brand, code, barcode, digits), where each sits (y, in design points from the
// bottom of a 288 x 144 label), how big it is, and one font family. A theme is
// a named labelDesign; the branch saves the full design it chose (not a
// reference to the theme), so a printed label never changes because a theme was
// tuned later. The default is the original look, theme "classic".
//
// layoutLabel turns a design and an item into drawing operations; the PDF
// writer and the SVG writer (the live preview) both consume the same list, so
// the preview is the label.

const (
	labelDesignW  = 288.0
	labelDesignH  = 144.0
	labelSideBarW = 14.0 // border strip width on the design canvas

	defaultLabelTheme = "classic"
)

type labelText struct {
	Enabled bool    `json:"enabled"`
	Y       float64 `json:"y"`    // baseline, design points from the bottom
	Size    float64 `json:"size"` // points
	Bold    bool    `json:"bold"`
	Align   string  `json:"align"` // center | left | right; empty follows the design
	Upper   bool    `json:"upper"` // capitals
	Fit     bool    `json:"fit"`   // shrink to the label's width instead of overflowing
}

type labelPrice struct {
	labelText
	Prefix    string  `json:"prefix"`
	Suffix    string  `json:"suffix"`
	Rule      bool    `json:"rule"`       // line under the price
	RuleY     float64 `json:"rule_y"`     // its height
	RuleScale float64 `json:"rule_scale"` // its width as a multiple of the price's
}

type labelName struct {
	labelText
	Variant bool `json:"variant"` // append " / colour / size"
}

type labelBarcode struct {
	Enabled bool    `json:"enabled"`
	Y       float64 `json:"y"`      // bottom of the data bars
	Height  float64 `json:"height"` // of the data bars
	Module  float64 `json:"module"` // width of one module, points
	Guard   float64 `json:"guard"`  // how far the guard bars hang below the data bars
	Align   string  `json:"align"`  // center | left | right; empty follows the design
}

type labelDigits struct {
	Enabled bool    `json:"enabled"`
	Size    float64 `json:"size"`
	Gap     float64 `json:"gap"` // below the lowest bar to the baseline
}

type labelDesign struct {
	Font    string       `json:"font"`   // helvetica | times | courier
	Align   string       `json:"align"`  // default for the text blocks
	Margin  float64      `json:"margin"` // side margin for left and right alignment
	Price   labelPrice   `json:"price"`
	Name    labelName    `json:"name"`
	Brand   labelText    `json:"brand"`
	Code    labelText    `json:"code"`
	Barcode labelBarcode `json:"barcode"`
	Digits  labelDigits  `json:"digits"`
}

// savedLabelDesign is what app_settings.barcode_label_design holds.
type savedLabelDesign struct {
	Theme  string      `json:"theme"` // the theme it started from; informational
	Design labelDesign `json:"design"`
}

type labelTheme struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	Description string      `json:"description"`
	BestFor     string      `json:"best_for"`
	Design      labelDesign `json:"design"`
}

// labelThemes is the catalog, the default first.
func labelThemes() []labelTheme {
	base := labelDesign{
		Font: "helvetica", Align: "center", Margin: 14,
		Price:   labelPrice{labelText: labelText{Enabled: true, Y: 118, Size: 24, Bold: true}, Suffix: " $", Rule: true, RuleY: 112, RuleScale: 1.8},
		Name:    labelName{labelText: labelText{Enabled: true, Y: 92, Size: 17}, Variant: true},
		Brand:   labelText{Y: 104, Size: 12},
		Code:    labelText{Enabled: true, Y: 72, Size: 14},
		Barcode: labelBarcode{Enabled: true, Y: 38, Height: 26, Module: 1.7, Guard: 8},
		Digits:  labelDigits{Enabled: true, Size: 14, Gap: 12},
	}

	priceTag := base
	priceTag.Price = labelPrice{labelText: labelText{Enabled: true, Y: 96, Size: 44, Bold: true}, Suffix: " $", RuleY: 90, RuleScale: 1}
	priceTag.Name = labelName{labelText: labelText{Enabled: true, Y: 72, Size: 15, Fit: true}, Variant: true}
	priceTag.Code.Enabled = false
	priceTag.Barcode = labelBarcode{Enabled: true, Y: 28, Height: 24, Module: 1.5, Guard: 6}
	priceTag.Digits = labelDigits{Enabled: true, Size: 11, Gap: 10}

	compact := base
	compact.Align = "left"
	compact.Price = labelPrice{labelText: labelText{Enabled: true, Y: 114, Size: 24, Bold: true}, Suffix: " $", RuleY: 108, RuleScale: 1}
	compact.Name = labelName{labelText: labelText{Enabled: true, Y: 94, Size: 14, Bold: true, Fit: true}, Variant: true}
	compact.Code = labelText{Enabled: true, Y: 78, Size: 11}
	compact.Barcode = labelBarcode{Enabled: true, Y: 40, Height: 28, Module: 1.6, Guard: 6, Align: "left"}
	compact.Digits = labelDigits{Enabled: true, Size: 11, Gap: 10}

	boutique := base
	boutique.Font = "times"
	boutique.Brand = labelText{Enabled: true, Y: 126, Size: 11, Upper: true}
	boutique.Price = labelPrice{labelText: labelText{Enabled: true, Y: 76, Size: 26, Bold: true}, Suffix: " $", Rule: true, RuleY: 68, RuleScale: 1}
	boutique.Name = labelName{labelText: labelText{Enabled: true, Y: 106, Size: 18, Fit: true}, Variant: true}
	boutique.Code.Enabled = false
	boutique.Barcode = labelBarcode{Enabled: true, Y: 36, Height: 22, Module: 1.6, Guard: 6}
	boutique.Digits = labelDigits{Enabled: true, Size: 11, Gap: 10}

	inventory := base
	inventory.Price.Enabled = false
	inventory.Name = labelName{labelText: labelText{Enabled: true, Y: 122, Size: 13, Bold: true, Fit: true}, Variant: true}
	inventory.Code = labelText{Enabled: true, Y: 106, Size: 13}
	inventory.Barcode = labelBarcode{Enabled: true, Y: 42, Height: 52, Module: 2.0, Guard: 0}
	inventory.Digits = labelDigits{Enabled: true, Size: 15, Gap: 18}

	retro := base
	retro.Font = "courier"
	retro.Price = labelPrice{labelText: labelText{Enabled: true, Y: 112, Size: 26, Bold: true}, Prefix: "$", Rule: true, RuleY: 105, RuleScale: 1.3}
	retro.Name = labelName{labelText: labelText{Enabled: true, Y: 88, Size: 13, Upper: true, Fit: true}, Variant: true}
	retro.Code = labelText{Enabled: true, Y: 72, Size: 12}
	retro.Barcode = labelBarcode{Enabled: true, Y: 40, Height: 26, Module: 1.7, Guard: 8}
	retro.Digits = labelDigits{Enabled: true, Size: 12, Gap: 12}

	return []labelTheme{
		{"classic", "Classic", "The original label: price over a rule, name, code, then the barcode.", "Any shop", base},
		{"price-tag", "Price tag", "A very large price with a small name; the barcode sits low.", "Shelves and promotions", priceTag},
		{"compact", "Compact", "Left aligned and dense, for small labels.", "Small labels, many details", compact},
		{"boutique", "Boutique", "Serif type with the brand above the name.", "Clothing, jewellery, gifts", boutique},
		{"inventory", "Inventory", "No price; a tall barcode with the code beside it.", "Stock, warehouse and returns", inventory},
		{"retro", "Retro", "Courier capitals with a dollar-sign price.", "Bookshops, hardware, craft shops", retro},
	}
}

func defaultLabelDesign() labelDesign { return labelThemes()[0].Design }

func findLabelTheme(id string) (labelTheme, bool) {
	for _, theme := range labelThemes() {
		if theme.ID == id {
			return theme, true
		}
	}
	return labelTheme{}, false
}

func clampF(v, lo, hi float64) float64 {
	switch {
	case v != v: // NaN
		return lo
	case v < lo:
		return lo
	case v > hi:
		return hi
	}
	return v
}

func oneOf(v string, allowed ...string) string {
	for _, a := range allowed {
		if v == a {
			return v
		}
	}
	return allowed[0]
}

func (t *labelText) normalize() {
	t.Y = clampF(t.Y, 0, labelDesignH)
	t.Size = clampF(t.Size, 4, 80)
	if t.Align != "" {
		t.Align = oneOf(t.Align, "center", "left", "right")
	}
}

func cleanAffix(s string) string {
	s = strings.Map(func(r rune) rune {
		if r < 32 || r > 126 {
			return -1 // base-14 PDF fonts: printable ASCII only
		}
		return r
	}, s)
	if len(s) > 8 {
		s = s[:8]
	}
	return s
}

// normalize clamps every number and enum to something the renderer can draw,
// so a hand-edited or old saved design can never produce a broken label.
func (d *labelDesign) normalize() {
	d.Font = oneOf(d.Font, "helvetica", "times", "courier")
	d.Align = oneOf(d.Align, "center", "left", "right")
	d.Margin = clampF(d.Margin, 0, 60)

	d.Price.normalize()
	d.Price.Prefix, d.Price.Suffix = cleanAffix(d.Price.Prefix), cleanAffix(d.Price.Suffix)
	d.Price.RuleY = clampF(d.Price.RuleY, 0, labelDesignH)
	d.Price.RuleScale = clampF(d.Price.RuleScale, 0.2, 4)
	d.Name.normalize()
	d.Brand.normalize()
	d.Code.normalize()

	d.Barcode.Y = clampF(d.Barcode.Y, 0, labelDesignH)
	d.Barcode.Height = clampF(d.Barcode.Height, 6, 120)
	d.Barcode.Module = clampF(d.Barcode.Module, 0.8, 3)
	d.Barcode.Guard = clampF(d.Barcode.Guard, 0, 24)
	if d.Barcode.Align != "" {
		d.Barcode.Align = oneOf(d.Barcode.Align, "center", "left", "right")
	}
	d.Digits.Size = clampF(d.Digits.Size, 4, 40)
	d.Digits.Gap = clampF(d.Digits.Gap, 0, 40)
}

// parseSavedLabelDesign reads app_settings.barcode_label_design. Empty or
// unreadable (nothing saved yet) is the default theme.
func parseSavedLabelDesign(raw []byte) savedLabelDesign {
	var saved savedLabelDesign
	if len(bytes.TrimSpace(raw)) == 0 || json.Unmarshal(raw, &saved) != nil || saved.Design.Font == "" {
		return savedLabelDesign{Theme: defaultLabelTheme, Design: defaultLabelDesign()}
	}
	if _, ok := findLabelTheme(saved.Theme); !ok {
		saved.Theme = defaultLabelTheme
	}
	saved.Design.normalize()
	return saved
}

// --- text metrics ------------------------------------------------------

// AFM advance widths (1/1000 em) for ASCII 32-126 of the base-14 families the
// PDF names, so text can be centred without embedding a font.
var helveticaBoldWidths = map[rune]float64{
	' ': 278, '!': 333, '"': 474, '#': 556, '$': 556, '%': 889, '&': 722, '\'': 238,
	'(': 333, ')': 333, '*': 389, '+': 584, ',': 278, '-': 333, '.': 278, '/': 278,
	'0': 556, '1': 556, '2': 556, '3': 556, '4': 556, '5': 556, '6': 556, '7': 556, '8': 556, '9': 556,
	':': 333, ';': 333, '<': 584, '=': 584, '>': 584, '?': 611, '@': 975,
	'A': 722, 'B': 722, 'C': 722, 'D': 722, 'E': 667, 'F': 611, 'G': 778, 'H': 722, 'I': 278, 'J': 556,
	'K': 722, 'L': 611, 'M': 833, 'N': 722, 'O': 778, 'P': 667, 'Q': 778, 'R': 722, 'S': 667, 'T': 611,
	'U': 722, 'V': 667, 'W': 944, 'X': 667, 'Y': 667, 'Z': 611,
	'[': 333, '\\': 278, ']': 333, '^': 584, '_': 556, '`': 333,
	'a': 556, 'b': 611, 'c': 556, 'd': 611, 'e': 556, 'f': 333, 'g': 611, 'h': 611, 'i': 278, 'j': 278,
	'k': 556, 'l': 278, 'm': 889, 'n': 611, 'o': 611, 'p': 611, 'q': 611, 'r': 389, 's': 556, 't': 333,
	'u': 611, 'v': 556, 'w': 778, 'x': 556, 'y': 556, 'z': 500,
	'{': 389, '|': 280, '}': 389, '~': 584,
}

var timesWidths = map[rune]float64{
	' ': 250, '!': 333, '"': 408, '#': 500, '$': 500, '%': 833, '&': 778, '\'': 180,
	'(': 333, ')': 333, '*': 500, '+': 564, ',': 250, '-': 333, '.': 250, '/': 278,
	'0': 500, '1': 500, '2': 500, '3': 500, '4': 500, '5': 500, '6': 500, '7': 500, '8': 500, '9': 500,
	':': 278, ';': 278, '<': 564, '=': 564, '>': 564, '?': 444, '@': 921,
	'A': 722, 'B': 667, 'C': 667, 'D': 722, 'E': 611, 'F': 556, 'G': 722, 'H': 722, 'I': 333, 'J': 389,
	'K': 722, 'L': 611, 'M': 889, 'N': 722, 'O': 722, 'P': 556, 'Q': 722, 'R': 667, 'S': 556, 'T': 611,
	'U': 722, 'V': 722, 'W': 944, 'X': 722, 'Y': 722, 'Z': 611,
	'[': 333, '\\': 278, ']': 333, '^': 469, '_': 500, '`': 333,
	'a': 444, 'b': 500, 'c': 444, 'd': 500, 'e': 444, 'f': 333, 'g': 500, 'h': 500, 'i': 278, 'j': 278,
	'k': 500, 'l': 278, 'm': 778, 'n': 500, 'o': 500, 'p': 500, 'q': 500, 'r': 333, 's': 389, 't': 278,
	'u': 500, 'v': 500, 'w': 722, 'x': 500, 'y': 500, 'z': 444,
	'{': 480, '|': 200, '}': 480, '~': 541,
}

var timesBoldWidths = map[rune]float64{
	' ': 250, '!': 333, '"': 555, '#': 500, '$': 500, '%': 1000, '&': 833, '\'': 278,
	'(': 333, ')': 333, '*': 500, '+': 570, ',': 250, '-': 333, '.': 250, '/': 278,
	'0': 500, '1': 500, '2': 500, '3': 500, '4': 500, '5': 500, '6': 500, '7': 500, '8': 500, '9': 500,
	':': 333, ';': 333, '<': 570, '=': 570, '>': 570, '?': 500, '@': 930,
	'A': 722, 'B': 667, 'C': 722, 'D': 722, 'E': 667, 'F': 611, 'G': 778, 'H': 778, 'I': 389, 'J': 500,
	'K': 778, 'L': 667, 'M': 944, 'N': 722, 'O': 778, 'P': 611, 'Q': 778, 'R': 722, 'S': 556, 'T': 667,
	'U': 722, 'V': 722, 'W': 1000, 'X': 722, 'Y': 722, 'Z': 667,
	'[': 333, '\\': 278, ']': 333, '^': 581, '_': 500, '`': 333,
	'a': 500, 'b': 556, 'c': 444, 'd': 556, 'e': 444, 'f': 333, 'g': 500, 'h': 556, 'i': 278, 'j': 333,
	'k': 556, 'l': 278, 'm': 833, 'n': 556, 'o': 500, 'p': 556, 'q': 556, 'r': 444, 's': 389, 't': 333,
	'u': 556, 'v': 500, 'w': 722, 'x': 500, 'y': 500, 'z': 444,
	'{': 394, '|': 220, '}': 394, '~': 520,
}

// labelTextWidth is the advance width of s in points for the family and
// weight; characters outside ASCII use the family's digit width.
func labelTextWidth(font string, bold bool, s string, size float64) float64 {
	var units float64
	for _, r := range s {
		switch font {
		case "courier":
			units += 600
		default:
			table := helveticaWidths
			fallback := 556.0
			switch {
			case font == "times" && bold:
				table, fallback = timesBoldWidths, 500.0
			case font == "times":
				table, fallback = timesWidths, 500.0
			case bold:
				table = helveticaBoldWidths
			}
			if w, ok := table[r]; ok {
				units += w
			} else {
				units += fallback
			}
		}
	}
	return units * size / 1000
}

// --- layout -------------------------------------------------------------

type labelOpKind int

const (
	labelOpRect   labelOpKind = iota
	labelOpBorder             // a side strip; a rect the PDF writes at lower precision
	labelOpLine
	labelOpText
)

// labelOp is one drawing primitive in design coordinates (origin bottom-left).
type labelOp struct {
	Kind       labelOpKind
	X, Y, W, H float64 // rect
	X2, Y2     float64 // line end; W is its thickness
	Text       string
	Size       float64
	Bold       bool
}

func alignedX(align string, left, right, width float64) float64 {
	switch align {
	case "left":
		return left
	case "right":
		return right - width
	}
	return (labelDesignW - width) / 2
}

// layoutLabel draws one label for it. borders adds the side strips used when
// separator pages are on.
func layoutLabel(it labelItem, bars string, d labelDesign, borders bool) []labelOp {
	var ops []labelOp
	left, right := d.Margin, labelDesignW-d.Margin
	if borders {
		ops = append(ops,
			labelOp{Kind: labelOpBorder, X: 0, Y: 0, W: labelSideBarW, H: labelDesignH},
			labelOp{Kind: labelOpBorder, X: labelDesignW - labelSideBarW, Y: 0, W: labelSideBarW, H: labelDesignH})
		left = maxF(left, labelSideBarW+2)
		right = minF(right, labelDesignW-labelSideBarW-2)
	}
	alignOf := func(own string) string {
		if own != "" {
			return own
		}
		return d.Align
	}
	// text places one line, shrinking it to the label's width when asked.
	text := func(t labelText, s string) {
		if s == "" {
			return
		}
		if t.Upper {
			s = strings.ToUpper(s)
		}
		size := t.Size
		if w := labelTextWidth(d.Font, t.Bold, s, size); t.Fit && w > right-left && w > 0 {
			size = size * (right - left) / w
		}
		w := labelTextWidth(d.Font, t.Bold, s, size)
		ops = append(ops, labelOp{Kind: labelOpText, X: alignedX(alignOf(t.Align), left, right, w), Y: t.Y, Text: s, Size: size, Bold: t.Bold})
	}

	if d.Price.Enabled && it.Price != "" {
		priceText := d.Price.Prefix + it.Price + d.Price.Suffix
		size := d.Price.Size
		w := labelTextWidth(d.Font, d.Price.Bold, priceText, size)
		if d.Price.Fit && w > right-left && w > 0 {
			size, w = size*(right-left)/w, right-left
		}
		x := alignedX(alignOf(d.Price.Align), left, right, w)
		ops = append(ops, labelOp{Kind: labelOpText, X: x, Y: d.Price.Y, Text: priceText, Size: size, Bold: d.Price.Bold})
		if d.Price.Rule {
			lineW := w * d.Price.RuleScale
			if limit := labelDesignW - 2*labelSideBarW - 8; lineW > limit {
				lineW = limit
			}
			lineX := alignedX(alignOf(d.Price.Align), left, right, lineW)
			ops = append(ops, labelOp{Kind: labelOpLine, X: lineX, Y: d.Price.RuleY, X2: lineX + lineW, Y2: d.Price.RuleY, W: 1})
		}
	}
	if d.Brand.Enabled {
		text(d.Brand, it.Brand)
	}
	if d.Name.Enabled {
		name := it.Name
		if d.Name.Variant {
			if it.Color != "" {
				name += " / " + it.Color
			}
			if it.Size != "" {
				name += " / " + it.Size
			}
		}
		text(d.Name.labelText, name)
	}
	if d.Code.Enabled {
		text(d.Code, it.Code)
	}

	if d.Barcode.Enabled && bars != "" {
		module := d.Barcode.Module
		barW := module * 0.82
		barcodeW := float64(len(bars)) * module
		barcodeX := alignedX(alignOf(d.Barcode.Align), left, right, barcodeW)
		barcodeY, barcodeH := d.Barcode.Y, d.Barcode.Height
		guardY, guardH := barcodeY-d.Barcode.Guard, barcodeH+d.Barcode.Guard
		const startGuardLen, digitBlockLen, centerGuardLen = 3, 42, 5
		leftDigitsEnd := startGuardLen + digitBlockLen
		centerGuardEnd := leftDigitsEnd + centerGuardLen
		rightDigitsEnd := centerGuardEnd + digitBlockLen
		isGuard := func(i int) bool {
			return i < startGuardLen || (i >= leftDigitsEnd && i < centerGuardEnd) || i >= rightDigitsEnd
		}
		x := barcodeX
		for j, bit := range bars {
			if bit == '1' {
				if isGuard(j) {
					ops = append(ops, labelOp{Kind: labelOpRect, X: x + (module-barW)/2, Y: guardY, W: barW, H: guardH})
				} else {
					ops = append(ops, labelOp{Kind: labelOpRect, X: x + (module-barW)/2, Y: barcodeY, W: barW, H: barcodeH})
				}
			}
			x += module
		}
		if d.Digits.Enabled {
			digitsW := labelTextWidth(d.Font, false, it.Barcode, d.Digits.Size)
			ops = append(ops, labelOp{Kind: labelOpText, X: barcodeX + (barcodeW-digitsW)/2, Y: guardY - d.Digits.Gap, Text: it.Barcode, Size: d.Digits.Size})
		}
	}
	return ops
}

func minF(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func maxF(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

// --- PDF ----------------------------------------------------------------

// labelFontRes names the PDF font resource for a design: the Helvetica pair
// the document always has, or the label pair when the design uses another
// family.
func labelFontRes(font string, bold bool) string {
	switch {
	case font == "helvetica" && bold:
		return "/FB"
	case font == "helvetica":
		return "/F1"
	case bold:
		return "/FLB"
	}
	return "/FL"
}

func labelPDFContent(ops []labelOp, font string, scale, yOffset float64) string {
	var c bytes.Buffer
	fmt.Fprintf(&c, "q\n%.4f 0 0 %.4f 0 %.2f cm\n0 0 0 rg\n0 0 0 RG\n", scale, scale, yOffset)
	for _, op := range ops {
		switch op.Kind {
		case labelOpBorder:
			fmt.Fprintf(&c, "%.2f %.2f %.2f %.2f re f\n", op.X, op.Y, op.W, op.H)
		case labelOpRect:
			fmt.Fprintf(&c, "%.3f %.2f %.3f %.2f re f\n", op.X, op.Y, op.W, op.H)
		case labelOpLine:
			fmt.Fprintf(&c, "%.2f w %.2f %.2f m %.2f %.2f l S\n", op.W, op.X, op.Y, op.X2, op.Y2)
		case labelOpText:
			fmt.Fprintf(&c, "BT %s %.2f Tf %.2f %.2f Td (%s) Tj ET\n", labelFontRes(font, op.Bold), op.Size, op.X, op.Y, pdfEscape(op.Text))
		}
	}
	c.WriteString("Q\n")
	return c.String()
}

// --- SVG (the live preview) ---------------------------------------------

func svgEscape(s string) string {
	return strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;").Replace(s)
}

func svgFontFamily(font string) string {
	switch font {
	case "times":
		return `'Times New Roman', Times, serif`
	case "courier":
		return `'Courier New', Courier, monospace`
	}
	return `Helvetica, Arial, sans-serif`
}

// labelSVG draws the label as the PDF will: the same design scaled to the
// label's page size, centred vertically.
func labelSVG(ops []labelOp, font string, width, height float64) string {
	scale := width / labelDesignW
	if hs := height / labelDesignH; hs < scale {
		scale = hs
	}
	yOffset := (height - labelDesignH*scale) / 2
	flip := func(y float64) float64 { return labelDesignH - y }

	var b strings.Builder
	fmt.Fprintf(&b, `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 %.2f %.2f" width="%.2f" height="%.2f" font-family="%s">`,
		width, height, width, height, svgFontFamily(font))
	fmt.Fprintf(&b, `<rect width="%.2f" height="%.2f" fill="#fff"/>`, width, height)
	fmt.Fprintf(&b, `<g transform="translate(0 %.3f) scale(%.5f)" fill="#000" stroke="none">`, yOffset, scale)
	for _, op := range ops {
		switch op.Kind {
		case labelOpRect, labelOpBorder:
			fmt.Fprintf(&b, `<rect x="%.3f" y="%.3f" width="%.3f" height="%.3f"/>`, op.X, flip(op.Y+op.H), op.W, op.H)
		case labelOpLine:
			fmt.Fprintf(&b, `<line x1="%.3f" y1="%.3f" x2="%.3f" y2="%.3f" stroke="#000" stroke-width="%.2f"/>`, op.X, flip(op.Y), op.X2, flip(op.Y2), op.W)
		case labelOpText:
			weight := "400"
			if op.Bold {
				weight = "700"
			}
			fmt.Fprintf(&b, `<text x="%.3f" y="%.3f" font-size="%.2f" font-weight="%s">%s</text>`, op.X, flip(op.Y), op.Size, weight, svgEscape(op.Text))
		}
	}
	b.WriteString(`</g></svg>`)
	return b.String()
}
