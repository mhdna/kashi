package api

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	db "github.com/mhdna/kashi/db/sqlc"
)

// A discount list's rules (000200): "Lady + Turkey: 30%", "codes starting
// CA: winter 20%, summer 50%". The first rule a variant matches decides its
// discount; the database writes the result out as the list's ordinary rows
// when the save commits, and keeps them up to date as products change.
//
// Rules sit in named groups (000203), each with an on/off switch, and each
// rule has one too; a rule that is off, or in a group that is off, decides
// nothing. The page reads and saves the whole set of groups at once.

type discountRuleOption struct {
	SaleOptionID int64  `json:"sale_option_id"`
	SaleOption   string `json:"sale_option"`
	Discount     int16  `json:"discount"`
}

type discountRule struct {
	ID                 int64                `json:"id"`
	Name               string               `json:"name"`
	CodePrefixes       []string             `json:"code_prefixes"`
	ExceptCodePrefixes []string             `json:"except_code_prefixes"`
	Kinds              []string             `json:"kinds"`
	Categories         []string             `json:"categories"`
	SubCategories      []string             `json:"sub_categories"`
	Origins            []string             `json:"origins"`
	Brands             []string             `json:"brands"`
	Discount           int16                `json:"discount"`
	Options            []discountRuleOption `json:"options"`
	// The deal the rule puts its products on (000201), 0 for none.
	DealID  int64 `json:"deal_id"`
	Enabled bool  `json:"enabled"`
	// How many variants the rule decides now, after the rules above it.
	Variants int64 `json:"variants"`
}

type discountRuleGroup struct {
	ID      int64          `json:"id"`
	Name    string         `json:"name"`
	Enabled bool           `json:"enabled"`
	Rules   []discountRule `json:"rules"`
}

func (server *Server) discountListRuleGroups(ctx *gin.Context, listID int64) ([]discountRuleGroup, error) {
	groups, err := server.store.ListDiscountListRuleGroups(ctx, listID)
	if err != nil {
		return nil, err
	}
	rows, err := server.store.ListDiscountListRules(ctx, listID)
	if err != nil {
		return nil, err
	}
	options, err := server.store.ListDiscountListRuleOptions(ctx, listID)
	if err != nil {
		return nil, err
	}
	byRule := map[int64][]discountRuleOption{}
	for _, o := range options {
		byRule[o.RuleID] = append(byRule[o.RuleID], discountRuleOption{o.SaleOptionID, o.SaleOption, o.Discount})
	}
	out := make([]discountRuleGroup, len(groups))
	at := map[int64]int{}
	for i, g := range groups {
		out[i] = discountRuleGroup{ID: g.ID, Name: g.Name, Enabled: g.IsEnabled, Rules: []discountRule{}}
		at[g.ID] = i
	}
	for _, r := range rows {
		rule := discountRule{
			ID: r.ID, Name: r.Name,
			CodePrefixes: r.CodePrefixes, ExceptCodePrefixes: r.ExceptCodePrefixes,
			Kinds: r.Kinds, Categories: r.Categories, SubCategories: r.SubCategories,
			Origins: r.Origins, Brands: r.Brands,
			Discount: r.Discount, Options: byRule[r.ID], DealID: r.DealID.Int64,
			Enabled: r.IsEnabled, Variants: r.Variants,
		}
		if rule.Options == nil {
			rule.Options = []discountRuleOption{}
		}
		if i, ok := at[r.GroupID]; ok {
			out[i].Rules = append(out[i].Rules, rule)
		}
	}
	return out, nil
}

func (server *Server) listDiscountListRules(ctx *gin.Context) {
	var uri discountListBranchesURI
	if err := ctx.ShouldBindUri(&uri); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	groups, err := server.discountListRuleGroups(ctx, uri.ID)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	server.writeJSON(ctx, http.StatusOK, envelope{"groups": groups})
}

type discountRuleRequest struct {
	Name               string   `json:"name"`
	CodePrefixes       []string `json:"code_prefixes"`
	ExceptCodePrefixes []string `json:"except_code_prefixes"`
	Kinds              []string `json:"kinds"`
	Categories         []string `json:"categories"`
	SubCategories      []string `json:"sub_categories"`
	Origins            []string `json:"origins"`
	Brands             []string `json:"brands"`
	Discount           int16    `json:"discount" binding:"min=0,max=100"`
	DealID             int64    `json:"deal_id" binding:"min=0"`
	// A rule with no condition takes every product; it has to say so, so an
	// empty rule saved by mistake doesn't discount the whole shop.
	AllProducts bool `json:"all_products"`
	Enabled     bool `json:"enabled"`
	Options     []struct {
		SaleOptionID int64 `json:"sale_option_id" binding:"required,min=1"`
		Discount     int16 `json:"discount" binding:"min=0,max=100"`
	} `json:"options" binding:"dive"`
}

type discountRuleGroupRequest struct {
	Name    string                `json:"name"`
	Enabled bool                  `json:"enabled"`
	Rules   []discountRuleRequest `json:"rules" binding:"dive"`
}

type setDiscountListRulesRequest struct {
	Groups []discountRuleGroupRequest `json:"groups" binding:"dive"`
	// Only a page from before groups sends "rules". Saving it as no groups
	// would delete every rule, so it is refused instead.
	Rules *[]any `json:"rules"`
}

// setDiscountListRules replaces the list's rule groups with the ones sent,
// in order, and answers with them as saved (with what each rule now
// decides).
func (server *Server) setDiscountListRules(ctx *gin.Context) {
	var uri discountListBranchesURI
	if err := ctx.ShouldBindUri(&uri); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	var req setDiscountListRulesRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	if req.Rules != nil {
		server.writeError(ctx, http.StatusConflict, errors.New("this page is out of date: reload it, then save the rules again"))
		return
	}
	if _, err := server.store.GetDiscountList(ctx, uri.ID); err != nil {
		server.writeError(ctx, statusForLookup(err), err)
		return
	}
	groups := make([]db.DiscountListRuleGroupInput, len(req.Groups))
	n := 0 // rules are numbered through the groups in errors, as on the page
	for g, group := range req.Groups {
		groups[g] = db.DiscountListRuleGroupInput{Name: strings.TrimSpace(group.Name), Enabled: group.Enabled}
		for _, r := range group.Rules {
			n++
			in, err := server.discountRuleInput(ctx, n, r)
			if err != nil {
				return
			}
			groups[g].Rules = append(groups[g].Rules, in)
		}
	}
	if err := server.store.ReplaceDiscountListRuleGroupsTx(ctx, uri.ID, groups); err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	saved, err := server.discountListRuleGroups(ctx, uri.ID)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	server.writeJSON(ctx, http.StatusOK, envelope{"groups": saved})
}

// errRuleAnswered says discountRuleInput has already written the error.
var errRuleAnswered = errors.New("rule refused")

// discountRuleInput checks one rule (the n-th on the page) and makes it
// ready to save; a rule that doesn't fit is answered here.
func (server *Server) discountRuleInput(ctx *gin.Context, i int, r discountRuleRequest) (db.DiscountListRuleInput, error) {
	in := db.DiscountListRuleInput{
		Rule: db.CreateDiscountListRuleParams{
			Name:               r.Name,
			CodePrefixes:       cleanEntries(r.CodePrefixes),
			ExceptCodePrefixes: cleanEntries(r.ExceptCodePrefixes),
			Kinds:              cleanEntries(r.Kinds),
			Categories:         cleanEntries(r.Categories),
			SubCategories:      cleanEntries(r.SubCategories),
			Origins:            cleanEntries(r.Origins),
			Brands:             cleanEntries(r.Brands),
			Discount:           r.Discount,
			IsEnabled:          r.Enabled,
		},
		Options: map[int64]int16{},
	}
	p := in.Rule
	if len(p.CodePrefixes)+len(p.Kinds)+len(p.Categories)+len(p.SubCategories)+len(p.Origins)+len(p.Brands) == 0 && !r.AllProducts {
		// A rule with no condition would take every product in the shop.
		server.writeError(ctx, http.StatusBadRequest, fmt.Errorf("rule %d (%q) has no condition", i, r.Name))
		return in, errRuleAnswered
	}
	for _, o := range r.Options {
		if _, err := server.store.GetSaleOption(ctx, o.SaleOptionID); err != nil {
			server.writeError(ctx, statusForLookup(err), err)
			return in, errRuleAnswered
		}
		if _, dup := in.Options[o.SaleOptionID]; dup {
			server.writeError(ctx, http.StatusBadRequest, errors.New("an option is given twice in one rule"))
			return in, errRuleAnswered
		}
		in.Options[o.SaleOptionID] = o.Discount
	}
	if r.DealID != 0 {
		// As for a hand-set row (bindRowDeal): the deal replaces the
		// discount, whatever the option.
		if r.Discount != 0 || len(r.Options) > 0 {
			server.writeError(ctx, http.StatusBadRequest, fmt.Errorf("rule %d (%q): a deal replaces the discount, so it takes no discount or option discounts", i, r.Name))
			return in, errRuleAnswered
		}
		deal, err := server.dealParam(ctx, r.DealID)
		if err != nil {
			server.writeError(ctx, statusForLookup(err), err)
			return in, errRuleAnswered
		}
		in.Rule.DealID = deal
	}
	return in, nil
}
