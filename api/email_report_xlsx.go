package api

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/xuri/excelize/v2"
)

// The emailed report as a workbook rather than a CSV: an "Invoices" sheet
// with a title block, a coloured centred header, real numbers in proper
// formats, a section and subtotal per register when grouping, and a grand
// total; then a "Summary" sheet with one line per register. Same columns
// and order as the CSV (buildEmailReportAttachment).

const (
	xlsxNavy      = "1F3A5F"
	xlsxBlue      = "2F5597"
	xlsxGroupFill = "DDEBF7"
	xlsxZebra     = "F5F8FC"
	xlsxSubtotal  = "EEF1F5"
	xlsxTotal     = "FFF2CC"
	xlsxGrid      = "D9D9D9"
	xlsxMuted     = "595959"
)

// xlsxStyles hands out excelize style ids, one per distinct look, so a
// thousand rows share a handful of styles.
type xlsxStyles struct {
	f     *excelize.File
	cache map[string]int
}

type xlsxLook struct {
	bold    bool
	italic  bool
	size    float64
	color   string
	fill    string
	align   string
	numFmt  string
	border  bool
	topLine int // 0 none, 1 thin, 6 double
	wrap    bool
}

func (s *xlsxStyles) id(look xlsxLook) (int, error) {
	key := fmt.Sprintf("%+v", look)
	if id, ok := s.cache[key]; ok {
		return id, nil
	}
	style := &excelize.Style{
		Font:      &excelize.Font{Bold: look.bold, Italic: look.italic, Size: look.size, Color: look.color, Family: "Calibri"},
		Alignment: &excelize.Alignment{Horizontal: look.align, Vertical: "center", WrapText: look.wrap},
	}
	if style.Font.Size == 0 {
		style.Font.Size = 11
	}
	if look.fill != "" {
		style.Fill = excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{look.fill}}
	}
	if look.numFmt != "" {
		numFmt := look.numFmt
		style.CustomNumFmt = &numFmt
	}
	if look.border {
		for _, side := range []string{"left", "right", "bottom", "top"} {
			style.Border = append(style.Border, excelize.Border{Type: side, Color: xlsxGrid, Style: 1})
		}
	}
	if look.topLine > 0 {
		style.Border = append(style.Border, excelize.Border{Type: "top", Color: xlsxMuted, Style: look.topLine})
	}
	id, err := s.f.NewStyle(style)
	if err != nil {
		return 0, err
	}
	s.cache[key] = id
	return id, nil
}

type xlsxSheet struct {
	f      *excelize.File
	name   string
	styles *xlsxStyles
	row    int
	widths []int
	err    error
}

// emailReportRates says which LBP rate the report used: "LBP rate 90,000",
// or per branch when they differ ("LBP rate Centro 90,000 · Hara 89,500").
func emailReportRates(rows []emailReportRow) string {
	var branches []string
	rates := map[string]int64{}
	distinct := map[int64]bool{}
	for _, row := range rows {
		if _, seen := rates[row.BranchName]; !seen {
			branches = append(branches, row.BranchName)
		}
		rates[row.BranchName] = emailReportRate(row)
		distinct[emailReportRate(row)] = true
	}
	if len(distinct) <= 1 {
		rate := int64(emailReportFallbackRate)
		for r := range distinct {
			rate = r
		}
		return "LBP rate " + groupThousands(rate)
	}
	sort.Strings(branches)
	parts := make([]string, len(branches))
	for i, branch := range branches {
		parts[i] = branch + " " + groupThousands(rates[branch])
	}
	return "LBP rate " + strings.Join(parts, " · ")
}

func groupThousands(n int64) string {
	digits := strconv.FormatInt(n, 10)
	var b strings.Builder
	for i, d := range digits {
		if i > 0 && (len(digits)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(d)
	}
	return b.String()
}

func invoiceCount(n int) string {
	if n == 1 {
		return "1 invoice"
	}
	return groupThousands(int64(n)) + " invoices"
}

func cellName(col, row int) string {
	name, _ := excelize.CoordinatesToCellName(col, row)
	return name
}

// set writes one cell with a look, keeping the first error.
func (s *xlsxSheet) set(col int, value any, look xlsxLook) {
	if s.err != nil {
		return
	}
	cell := cellName(col, s.row)
	if s.err = s.f.SetCellValue(s.name, cell, value); s.err != nil {
		return
	}
	style, err := s.styles.id(look)
	if err != nil {
		s.err = err
		return
	}
	s.err = s.f.SetCellStyle(s.name, cell, cell, style)
	if text, ok := value.(string); ok && col <= len(s.widths) && !look.wrap {
		if n := utf8.RuneCountInString(text); n > s.widths[col-1] {
			s.widths[col-1] = n
		}
	}
}

// banner writes text across every column on its own merged row.
func (s *xlsxSheet) banner(text string, columns int, look xlsxLook, height float64) {
	if s.err != nil {
		return
	}
	look.wrap = true
	for col := 1; col <= columns; col++ {
		value := any("")
		if col == 1 {
			value = text
		}
		s.set(col, value, look)
	}
	if s.err == nil && columns > 1 {
		s.err = s.f.MergeCell(s.name, cellName(1, s.row), cellName(columns, s.row))
	}
	if s.err == nil && height > 0 {
		s.err = s.f.SetRowHeight(s.name, s.row, height)
	}
	s.row++
}

func (s *xlsxSheet) finish(minWidth, maxWidth int) {
	for i, width := range s.widths {
		width += 3
		if width < minWidth {
			width = minWidth
		}
		if width > maxWidth {
			width = maxWidth
		}
		if s.err == nil {
			col, _ := excelize.ColumnNumberToName(i + 1)
			s.err = s.f.SetColWidth(s.name, col, col, float64(width))
		}
	}
}

func buildEmailReportXLSX(report emailReport, groups []emailReportGroup, columns []emailReportColumn) ([]byte, error) {
	f := excelize.NewFile()
	defer f.Close()
	styles := &xlsxStyles{f: f, cache: map[string]int{}}

	business := report.settings.EmailBusinessName
	if business == "" {
		business = "Kashi"
	}
	loc, err := time.LoadLocation("Asia/Beirut")
	if err != nil {
		loc = time.UTC
	}
	generated := time.Now().In(loc).Format("02/01/2006 15:04")

	sheet := &xlsxSheet{f: f, name: "Invoices", styles: styles, row: 1, widths: make([]int, len(columns))}
	if err := f.SetSheetName("Sheet1", sheet.name); err != nil {
		return nil, err
	}
	n := len(columns)

	sheet.banner(business+" Sales Report", n, xlsxLook{bold: true, size: 16, color: "FFFFFF", fill: xlsxNavy, align: "center"}, 32)
	sheet.banner(report.label, n, xlsxLook{bold: true, size: 12, color: xlsxNavy, align: "center"}, 20)
	sheet.banner(fmt.Sprintf("%s · %s · generated %s", invoiceCount(len(report.rows)), emailReportRates(report.rows), generated), n,
		xlsxLook{italic: true, size: 10, color: xlsxMuted, align: "center"}, 16)
	if report.note != "" {
		sheet.banner(report.note, n, xlsxLook{size: 10, color: xlsxMuted, align: "center"}, 0)
	}
	sheet.row++ // breathing room

	headerRow := sheet.row
	for i, column := range columns {
		sheet.set(i+1, column.Header, xlsxLook{bold: true, color: "FFFFFF", fill: xlsxBlue, align: "center", border: true, wrap: true})
		if w := utf8.RuneCountInString(column.Header); w > sheet.widths[i] {
			sheet.widths[i] = w
		}
	}
	if sheet.err == nil {
		sheet.err = f.SetRowHeight(sheet.name, headerRow, 24)
	}
	sheet.row++

	// The label for a totals row goes in the first column that isn't a total.
	labelCol := 0
	for i, column := range columns {
		if !column.total {
			labelCol = i + 1
			break
		}
	}
	totalsRow := func(label string, sums []float64, fill string, topLine int) {
		for i, column := range columns {
			look := xlsxLook{bold: true, fill: fill, align: "center", border: true, topLine: topLine, numFmt: column.numFmt}
			switch {
			case column.total:
				sheet.set(i+1, sums[i], look)
			case i+1 == labelCol:
				sheet.set(i+1, label, look)
			default:
				sheet.set(i+1, "", look)
			}
		}
		sheet.row++
	}
	addTo := func(sums []float64, row emailReportRow) {
		for i, column := range columns {
			if column.total {
				sums[i] += column.number(row)
			}
		}
	}

	grouped := len(groups) > 1 || (len(groups) == 1 && groups[0].title != "")
	grand := make([]float64, n)
	for _, group := range groups {
		sums := make([]float64, n)
		if grouped {
			sheet.banner(fmt.Sprintf("%s  ·  %s", group.title, invoiceCount(len(group.rows))), n,
				xlsxLook{bold: true, color: xlsxNavy, fill: xlsxGroupFill, align: "center"}, 20)
		}
		for r, row := range group.rows {
			fill := ""
			if r%2 == 1 {
				fill = xlsxZebra
			}
			for i, column := range columns {
				look := xlsxLook{fill: fill, align: "center", border: true}
				if column.number != nil {
					look.numFmt = column.numFmt
					sheet.set(i+1, column.number(row), look)
					if w := utf8.RuneCountInString(column.value(row)) + 2; w > sheet.widths[i] {
						sheet.widths[i] = w
					}
					continue
				}
				sheet.set(i+1, column.value(row), look)
			}
			addTo(sums, row)
			addTo(grand, row)
			sheet.row++
		}
		if grouped {
			totalsRow("Subtotal", sums, xlsxSubtotal, 1)
			sheet.row++
		}
	}
	totalsRow("Total · "+invoiceCount(len(report.rows)), grand, xlsxTotal, 6)
	sheet.finish(10, 40)

	if sheet.err == nil {
		sheet.err = f.SetPanes(sheet.name, &excelize.Panes{
			Freeze: true, YSplit: headerRow, TopLeftCell: cellName(1, headerRow+1), ActivePane: "bottomLeft",
		})
	}
	// A filter only makes sense on one plain block of rows.
	if sheet.err == nil && !grouped && len(report.rows) > 0 {
		sheet.err = f.AutoFilter(sheet.name, cellName(1, headerRow)+":"+cellName(n, headerRow+len(report.rows)), nil)
	}
	if sheet.err == nil {
		sheet.err = emailReportPrintSetup(f, sheet.name, n > 6)
	}
	if sheet.err != nil {
		return nil, sheet.err
	}

	if err := buildEmailReportSummarySheet(f, styles, report, business); err != nil {
		return nil, err
	}

	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// buildEmailReportSummarySheet is one line per register: invoices and net
// totals, then a grand total.
func buildEmailReportSummarySheet(f *excelize.File, styles *xlsxStyles, report emailReport, business string) error {
	if _, err := f.NewSheet("Summary"); err != nil {
		return err
	}
	sheet := &xlsxSheet{f: f, name: "Summary", styles: styles, row: 1, widths: make([]int, 6)}
	sheet.banner(business+" · Summary", 6, xlsxLook{bold: true, size: 14, color: "FFFFFF", fill: xlsxNavy, align: "center"}, 28)
	sheet.banner(report.label, 6, xlsxLook{bold: true, color: xlsxNavy, align: "center"}, 18)
	sheet.row++

	headers := []string{"Branch", "Register", "Invoices", "Net Amount (USD)", "LBP Rate", "Net Amount (LBP)"}
	for i, header := range headers {
		sheet.set(i+1, header, xlsxLook{bold: true, color: "FFFFFF", fill: xlsxBlue, align: "center", border: true})
	}
	sheet.row++

	type line struct {
		branch, register string
		count            int
		usd, lbp         float64
		rate             int64
	}
	var lines []*line
	index := map[string]*line{}
	for _, group := range groupEmailReportRows(report.rows, true, false) {
		for _, row := range group.rows {
			l := index[group.title]
			if l == nil {
				l = &line{branch: row.BranchName, register: row.RegisterCode, rate: emailReportRate(row)}
				index[group.title] = l
				lines = append(lines, l)
			}
			l.count++
			l.usd += float64(row.GrandTotal) / 100
			l.lbp += float64(tillLBPTotal(row))
		}
	}
	var count int
	var usd, lbp float64
	for i, l := range lines {
		fill := ""
		if i%2 == 1 {
			fill = xlsxZebra
		}
		cell := xlsxLook{fill: fill, align: "center", border: true}
		sheet.set(1, l.branch, cell)
		sheet.set(2, l.register, cell)
		sheet.set(3, l.count, xlsxLook{fill: fill, align: "center", border: true, numFmt: "#,##0"})
		sheet.set(4, l.usd, xlsxLook{fill: fill, align: "center", border: true, numFmt: "#,##0.00"})
		sheet.set(5, l.rate, xlsxLook{fill: fill, align: "center", border: true, numFmt: "#,##0"})
		sheet.set(6, l.lbp, xlsxLook{fill: fill, align: "center", border: true, numFmt: "#,##0"})
		sheet.row++
		count, usd, lbp = count+l.count, usd+l.usd, lbp+l.lbp
	}
	total := xlsxLook{bold: true, fill: xlsxTotal, align: "center", border: true, topLine: 6}
	sheet.set(1, "Total", total)
	sheet.set(2, "", total)
	total.numFmt = "#,##0"
	sheet.set(3, count, total)
	total.numFmt = "#,##0.00"
	sheet.set(4, usd, total)
	total.numFmt = ""
	sheet.set(5, "", total)
	total.numFmt = "#,##0"
	sheet.set(6, lbp, total)
	for i, w := range []int{14, 10, 10, 18, 12, 18} {
		if w > sheet.widths[i] {
			sheet.widths[i] = w
		}
	}
	sheet.finish(10, 32)
	if sheet.err != nil {
		return sheet.err
	}
	return emailReportPrintSetup(f, sheet.name, false)
}

// emailReportPrintSetup prints a sheet one page wide.
func emailReportPrintSetup(f *excelize.File, sheet string, landscape bool) error {
	orientation := "portrait"
	if landscape {
		orientation = "landscape"
	}
	one, fit := 1, true
	if err := f.SetPageLayout(sheet, &excelize.PageLayoutOptions{Orientation: &orientation, FitToWidth: &one}); err != nil {
		return err
	}
	return f.SetSheetProps(sheet, &excelize.SheetPropsOptions{FitToPage: &fit})
}
