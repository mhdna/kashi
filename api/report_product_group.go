package api

import (
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	db "github.com/mhdna/kashi/db/sqlc"
)

// The Product group report: the sales of the products whose code starts
// with one of a few prefixes ("TT", or "TT, TX"), for the same days,
// branches and left-out customers and products as the other reports. One
// query returns the group's invoice lines and productGroupReport adds them
// up by product, day, branch, salesperson, customer, size and colour, so
// every table adds up to the same totals. The products come back shaped
// like /reports' products, so the page draws the Sales & goods parts (best
// sellers, profit, brands, sizes...) for the group as they are.

// productGroupLineCap guards the one query; a group that sells more lines
// than this in the period asks for a narrower one.
const productGroupLineCap = 200000

type productGroupRequest struct {
	reportsRequest
	Codes string `form:"codes" binding:"required"` // comma separated prefixes
}

// parseCodePrefixes reads the prefixes as typed: split on commas (an Arabic
// comma or semicolon too), trimmed, upper-cased, repeats dropped, in the
// order given.
func parseCodePrefixes(raw string) ([]string, error) {
	fields := strings.FieldsFunc(raw, func(r rune) bool { return r == ',' || r == '،' || r == ';' })
	codes := []string{}
	seen := map[string]bool{}
	for _, f := range fields {
		code := strings.ToUpper(strings.TrimSpace(f))
		if code == "" || seen[code] {
			continue
		}
		if len(code) > 40 {
			return nil, fmt.Errorf("code %q is too long", code)
		}
		seen[code] = true
		codes = append(codes, code)
	}
	if len(codes) == 0 {
		return nil, errors.New("type the start of the product codes")
	}
	if len(codes) > 20 {
		return nil, errors.New("pick at most 20 code beginnings")
	}
	return codes, nil
}

// codeLikePatterns turns prefixes into LIKE patterns: % and _ (and the
// escape itself) in a code are matched literally.
func codeLikePatterns(codes []string) []string {
	escape := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	patterns := make([]string, len(codes))
	for i, c := range codes {
		patterns[i] = escape.Replace(c) + "%"
	}
	return patterns
}

// parseReportScope reads the days and branches every report takes.
func parseReportScope(req reportsRequest) (time.Time, time.Time, []int64, error) {
	from, errFrom := time.Parse(time.DateOnly, req.From)
	to, errTo := time.Parse(time.DateOnly, req.To)
	if errFrom != nil || errTo != nil || to.Before(from) {
		return from, to, nil, errors.New("from and to must be YYYY-MM-DD, from first")
	}
	if to.Sub(from) > 400*24*time.Hour {
		return from, to, nil, errors.New("pick at most a year")
	}
	branchIDs, err := parseBranchIDList(req.BranchIDs)
	return from, to, branchIDs, err
}

func (server *Server) getProductGroupReport(ctx *gin.Context) {
	var req productGroupRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, errors.New("from, to and codes are needed"))
		return
	}
	from, to, branchIDs, err := parseReportScope(req.reportsRequest)
	if err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	codes, err := parseCodePrefixes(req.Codes)
	if err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	ex, err := server.reportExclusions(ctx)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	patterns := codeLikePatterns(codes)
	lines := func(from, to time.Time) ([]db.ReportProductGroupLinesRow, error) {
		rows, err := server.store.ReportProductGroupLines(ctx, db.ReportProductGroupLinesParams{
			CodePatterns: patterns, FromDay: from, ToDay: to, BranchIds: branchIDs,
			ExcludeProductIds: ex.products, ExcludeClientIds: ex.clients,
		})
		if err == nil && len(rows) > productGroupLineCap {
			err = errTooManyGroupLines
		}
		return rows, err
	}

	rows, err := lines(from, to)
	if errors.Is(err, errTooManyGroupLines) {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	prevFrom, prevTo := previousPeriod(from, to)
	prevRows, err := lines(prevFrom, prevTo)
	if errors.Is(err, errTooManyGroupLines) {
		// Too much to compare with is no reason to fail the report.
		prevRows, err = nil, nil
	}
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	allRevenue, err := server.store.ReportLinesRevenue(ctx, db.ReportLinesRevenueParams{
		FromDay: from, ToDay: to, BranchIds: branchIDs, ExcludeProductIds: ex.products, ExcludeClientIds: ex.clients,
	})
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	report := productGroupReport(rows)
	productIDs := make([]int64, len(report.Products))
	for i, p := range report.Products {
		productIDs[i] = p.ProductID
	}
	attributes, err := server.store.PeriodReportProductAttributes(ctx, productIDs)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}

	server.writeJSON(ctx, http.StatusOK, envelope{
		"codes":              codes,
		"summary":            report.Summary,
		"all_revenue":        allRevenue,
		"products":           report.Products,
		"product_attributes": orEmpty(attributes),
		"variant_attributes": report.VariantAttributes,
		"daily":              report.Daily,
		"branches":           report.Branches,
		"salespersons":       report.Salespersons,
		"clients":            report.Clients,
		"previous": envelope{
			"from":    prevFrom.Format(time.DateOnly),
			"to":      prevTo.Format(time.DateOnly),
			"summary": productGroupReport(prevRows).Summary,
		},
	})
}

var errTooManyGroupLines = fmt.Errorf("these codes sold more than %d lines in the period: pick a shorter period or longer codes", productGroupLineCap)

// ---- adding the lines up -----------------------------------------------------

// The money and units every table of the report carries. Units and money
// are net: a returned line counts against them. Returned is the units that
// came back, negative.
type groupFigures struct {
	Quantity      int64 `json:"quantity"`
	Revenue       int64 `json:"revenue"`
	Cost          int64 `json:"cost"`
	CostedRevenue int64 `json:"costed_revenue"`
	Returned      int64 `json:"returned"`
}

func (f *groupFigures) add(r db.ReportProductGroupLinesRow) {
	f.Quantity += r.Quantity
	f.Revenue += r.Revenue
	f.Cost += r.Cost
	f.CostedRevenue += r.CostedRevenue
	f.Returned += r.Returned
}

type groupSummary struct {
	groupFigures
	SalesCount      int64 `json:"sales_count"`   // sales with a line of the group
	ReturnsCount    int64 `json:"returns_count"` // returns and exchanges with one
	Products        int64 `json:"products"`
	Clients         int64 `json:"clients"`
	NamedSalesCount int64 `json:"named_sales_count"`
	SellingDays     int64 `json:"selling_days"`
}

// Shaped like PeriodReportProductsRow, so the Sales & goods parts draw it.
type groupProduct struct {
	ProductID   int64  `json:"product_id"`
	ProductCode string `json:"product_code"`
	ProductName string `json:"product_name"`
	Category    string `json:"category"`
	groupFigures
	Invoices int64 `json:"invoices"`
}

type groupDay struct {
	Day string `json:"day"`
	groupFigures
	SalesCount int64 `json:"sales_count"`
}

type groupBranch struct {
	BranchID   int64  `json:"branch_id"`
	BranchName string `json:"branch_name"`
	groupFigures
	SalesCount int64 `json:"sales_count"`
}

type groupSalesperson struct {
	SalespersonID   int64  `json:"salesperson_id"`
	SalespersonName string `json:"salesperson_name"`
	BranchID        int64  `json:"branch_id"`
	BranchName      string `json:"branch_name"`
	groupFigures
	SalesCount int64 `json:"sales_count"`
}

type groupClient struct {
	ClientID int64  `json:"client_id"`
	Name     string `json:"name"`
	Phone    string `json:"phone"`
	groupFigures
	Visits  int64  `json:"visits"`
	LastDay string `json:"last_day"`
}

// Shaped like PeriodReportVariantAttributesRow.
type groupAttribute struct {
	Kind     string `json:"kind"` // size, color
	Name     string `json:"name"`
	Quantity int64  `json:"quantity"`
	Revenue  int64  `json:"revenue"`
}

type productGroupResult struct {
	Summary           groupSummary
	Products          []groupProduct
	Daily             []groupDay
	Branches          []groupBranch
	Salespersons      []groupSalesperson
	Clients           []groupClient
	VariantAttributes []groupAttribute
}

// invoiceSet counts distinct invoices, sales apart from returns.
type invoiceSet map[int64]bool

func (s invoiceSet) sales() int64 {
	n := int64(0)
	for _, isSale := range s {
		if isSale {
			n++
		}
	}
	return n
}

// productGroupReport adds the group's lines up into every table. Each table
// covers every line once, so its figures add up to the summary's.
func productGroupReport(rows []db.ReportProductGroupLinesRow) productGroupResult {
	var out productGroupResult
	invoices := invoiceSet{}
	days := map[string]bool{}
	clientsSeen := map[int64]bool{}
	namedSales := map[int64]bool{}

	products := map[int64]*groupProduct{}
	productInvoices := map[int64]invoiceSet{}
	daily := map[string]*groupDay{}
	dayInvoices := map[string]invoiceSet{}
	branches := map[int64]*groupBranch{}
	branchInvoices := map[int64]invoiceSet{}
	type spKey struct {
		id     int64
		name   string
		branch int64
	}
	salespersons := map[spKey]*groupSalesperson{}
	spInvoices := map[spKey]invoiceSet{}
	clients := map[int64]*groupClient{}
	clientInvoices := map[int64]invoiceSet{}
	type attrKey struct{ kind, name string }
	attributes := map[attrKey]*groupAttribute{}

	note := func(sets map[int64]invoiceSet, key int64, r db.ReportProductGroupLinesRow) {
		if sets[key] == nil {
			sets[key] = invoiceSet{}
		}
		sets[key][r.InvoiceID] = r.Kind == "sales"
	}

	for _, r := range rows {
		isSale := r.Kind == "sales"
		day := r.Day.Format(time.DateOnly)
		out.Summary.add(r)
		invoices[r.InvoiceID] = isSale
		if isSale {
			days[day] = true
		}

		p := products[r.ProductID]
		if p == nil {
			p = &groupProduct{ProductID: r.ProductID, ProductCode: r.ProductCode, ProductName: r.ProductName, Category: r.Category}
			products[r.ProductID] = p
		}
		p.add(r)
		note(productInvoices, r.ProductID, r)

		d := daily[day]
		if d == nil {
			d = &groupDay{Day: day}
			daily[day] = d
			dayInvoices[day] = invoiceSet{}
		}
		d.add(r)
		dayInvoices[day][r.InvoiceID] = isSale

		b := branches[r.BranchID]
		if b == nil {
			b = &groupBranch{BranchID: r.BranchID, BranchName: r.BranchName}
			branches[r.BranchID] = b
		}
		b.add(r)
		note(branchInvoices, r.BranchID, r)

		k := spKey{r.SalespersonID, r.SalespersonName, r.BranchID}
		sp := salespersons[k]
		if sp == nil {
			sp = &groupSalesperson{SalespersonID: r.SalespersonID, SalespersonName: r.SalespersonName, BranchID: r.BranchID, BranchName: r.BranchName}
			salespersons[k] = sp
			spInvoices[k] = invoiceSet{}
		}
		sp.add(r)
		spInvoices[k][r.InvoiceID] = isSale

		if r.ClientID > 0 {
			clientsSeen[r.ClientID] = true
			if isSale {
				namedSales[r.InvoiceID] = true
			}
			c := clients[r.ClientID]
			if c == nil {
				c = &groupClient{ClientID: r.ClientID, Name: r.ClientName, Phone: r.ClientPhone}
				clients[r.ClientID] = c
			}
			c.add(r)
			if day > c.LastDay {
				c.LastDay = day
			}
			note(clientInvoices, r.ClientID, r)
		}

		for _, a := range []attrKey{{"size", r.Size}, {"color", r.Color}} {
			g := attributes[a]
			if g == nil {
				g = &groupAttribute{Kind: a.kind, Name: a.name}
				attributes[a] = g
			}
			g.Quantity += r.Quantity
			g.Revenue += r.Revenue
		}
	}

	out.Summary.SalesCount = invoices.sales()
	out.Summary.ReturnsCount = int64(len(invoices)) - out.Summary.SalesCount
	out.Summary.Products = int64(len(products))
	out.Summary.Clients = int64(len(clientsSeen))
	out.Summary.NamedSalesCount = int64(len(namedSales))
	out.Summary.SellingDays = int64(len(days))

	out.Products = []groupProduct{}
	for id, p := range products {
		p.Invoices = int64(len(productInvoices[id]))
		out.Products = append(out.Products, *p)
	}
	sort.Slice(out.Products, func(i, j int) bool {
		a, b := out.Products[i], out.Products[j]
		if a.Revenue != b.Revenue {
			return a.Revenue > b.Revenue
		}
		return a.ProductCode < b.ProductCode
	})

	out.Daily = []groupDay{}
	for day, d := range daily {
		d.SalesCount = dayInvoices[day].sales()
		out.Daily = append(out.Daily, *d)
	}
	sort.Slice(out.Daily, func(i, j int) bool { return out.Daily[i].Day < out.Daily[j].Day })

	out.Branches = []groupBranch{}
	for id, b := range branches {
		b.SalesCount = branchInvoices[id].sales()
		out.Branches = append(out.Branches, *b)
	}
	sort.Slice(out.Branches, func(i, j int) bool {
		a, b := out.Branches[i], out.Branches[j]
		if a.Revenue != b.Revenue {
			return a.Revenue > b.Revenue
		}
		return a.BranchID < b.BranchID
	})

	out.Salespersons = []groupSalesperson{}
	for k, sp := range salespersons {
		sp.SalesCount = spInvoices[k].sales()
		out.Salespersons = append(out.Salespersons, *sp)
	}
	sort.Slice(out.Salespersons, func(i, j int) bool {
		a, b := out.Salespersons[i], out.Salespersons[j]
		if a.Revenue != b.Revenue {
			return a.Revenue > b.Revenue
		}
		if a.SalespersonName != b.SalespersonName {
			return a.SalespersonName < b.SalespersonName
		}
		return a.BranchID < b.BranchID
	})

	out.Clients = []groupClient{}
	for id, c := range clients {
		c.Visits = clientInvoices[id].sales()
		out.Clients = append(out.Clients, *c)
	}
	sort.Slice(out.Clients, func(i, j int) bool {
		a, b := out.Clients[i], out.Clients[j]
		if a.Revenue != b.Revenue {
			return a.Revenue > b.Revenue
		}
		return a.ClientID < b.ClientID
	})

	out.VariantAttributes = []groupAttribute{}
	for _, g := range attributes {
		out.VariantAttributes = append(out.VariantAttributes, *g)
	}
	sort.Slice(out.VariantAttributes, func(i, j int) bool {
		a, b := out.VariantAttributes[i], out.VariantAttributes[j]
		if a.Kind != b.Kind {
			return a.Kind < b.Kind
		}
		if a.Revenue != b.Revenue {
			return a.Revenue > b.Revenue
		}
		return a.Name < b.Name
	})
	return out
}
