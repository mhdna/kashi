package api

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	db "github.com/mhdna/kashi/db/sqlc"
)

// A discount list's rules (000200): "Lady + Turkey: 30%", "codes starting
// CA: winter 20%, summer 50%". The first rule a variant matches decides its
// discount; the database writes the result out as the list's ordinary rows
// when the save commits, and keeps them up to date as products change.

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
	// How many variants the rule decides now, after the rules above it.
	Variants int64 `json:"variants"`
}

func (server *Server) discountListRules(ctx *gin.Context, listID int64) ([]discountRule, error) {
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
	rules := make([]discountRule, len(rows))
	for i, r := range rows {
		rules[i] = discountRule{
			ID: r.ID, Name: r.Name,
			CodePrefixes: r.CodePrefixes, ExceptCodePrefixes: r.ExceptCodePrefixes,
			Kinds: r.Kinds, Categories: r.Categories, SubCategories: r.SubCategories,
			Origins: r.Origins, Brands: r.Brands,
			Discount: r.Discount, Options: byRule[r.ID], Variants: r.Variants,
		}
		if rules[i].Options == nil {
			rules[i].Options = []discountRuleOption{}
		}
	}
	return rules, nil
}

func (server *Server) listDiscountListRules(ctx *gin.Context) {
	var uri discountListBranchesURI
	if err := ctx.ShouldBindUri(&uri); err != nil {
		server.writeError(ctx, http.StatusBadRequest, err)
		return
	}
	rules, err := server.discountListRules(ctx, uri.ID)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	server.writeJSON(ctx, http.StatusOK, envelope{"rules": rules})
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
	Options            []struct {
		SaleOptionID int64 `json:"sale_option_id" binding:"required,min=1"`
		Discount     int16 `json:"discount" binding:"min=0,max=100"`
	} `json:"options" binding:"dive"`
}

type setDiscountListRulesRequest struct {
	Rules []discountRuleRequest `json:"rules" binding:"dive"`
}

// setDiscountListRules replaces the list's rules with the ones sent, in
// order, and answers with them as saved (with what each now decides).
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
	if _, err := server.store.GetDiscountList(ctx, uri.ID); err != nil {
		server.writeError(ctx, statusForLookup(err), err)
		return
	}
	inputs := make([]db.DiscountListRuleInput, len(req.Rules))
	for i, r := range req.Rules {
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
			},
			Options: map[int64]int16{},
		}
		p := in.Rule
		if len(p.CodePrefixes)+len(p.Kinds)+len(p.Categories)+len(p.SubCategories)+len(p.Origins)+len(p.Brands) == 0 {
			// A rule with no condition would take every product in the shop.
			server.writeError(ctx, http.StatusBadRequest, fmt.Errorf("rule %d (%q) has no condition", i+1, r.Name))
			return
		}
		for _, o := range r.Options {
			if _, err := server.store.GetSaleOption(ctx, o.SaleOptionID); err != nil {
				server.writeError(ctx, statusForLookup(err), err)
				return
			}
			if _, dup := in.Options[o.SaleOptionID]; dup {
				server.writeError(ctx, http.StatusBadRequest, errors.New("an option is given twice in one rule"))
				return
			}
			in.Options[o.SaleOptionID] = o.Discount
		}
		inputs[i] = in
	}
	if err := server.store.ReplaceDiscountListRulesTx(ctx, uri.ID, inputs); err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	rules, err := server.discountListRules(ctx, uri.ID)
	if err != nil {
		server.writeError(ctx, http.StatusInternalServerError, err)
		return
	}
	server.writeJSON(ctx, http.StatusOK, envelope{"rules": rules})
}
