package api

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	db "github.com/mhdna/kashi/db/sqlc"
)

// Price and discount lists sync to a branch as a whole list at a time, items
// included, rather than as a stream of item-level deltas.
//
// That shape is deliberate. kashi-pos has no tombstone mechanism -- an
// incremental feed can say "this row changed" but has no way to say "this row
// is gone", and the till's other pulled entities either carry an is_active
// flag or are reconciled by the periodic full-mirror check. An item removed
// from a list is neither: there is no row left to carry a flag. Sending the
// full item set lets the till apply a list as a replace-in-transaction, so a
// removal is expressed simply by the item's absence.
//
// The cost is that a large list is re-sent whenever any one of its prices
// changes. That is acceptable because the list's updated_at only moves when
// something in it actually changed (the triggers in 000094), so a steady
// state re-sends nothing at all.

type syncPriceListItem struct {
	ProductVariantUID uuid.UUID `json:"product_variant_uid"`
	Price             int64     `json:"price"`
}

type syncPriceList struct {
	UID       uuid.UUID           `json:"uid"`
	Name      string              `json:"name"`
	IsActive  bool                `json:"is_active"`
	IsDefault bool                `json:"is_default"`
	IsSpecial bool                `json:"is_special"`
	ValidFrom time.Time           `json:"valid_from"`
	ValidTo   time.Time           `json:"valid_to"`
	UpdatedAt time.Time           `json:"updated_at"`
	Items     []syncPriceListItem `json:"items"`
}

type syncDiscountListItem struct {
	ProductVariantUID uuid.UUID `json:"product_variant_uid"`
	Discount          int16     `json:"discount"`
	// The deal the variant is on (000201), one of the list's Deals; a row
	// with a deal has discount 0, so a till that predates deals sells it at
	// its list price -- the deal replaces the discount there too.
	DealUID *uuid.UUID `json:"deal_uid,omitempty"`
}

// A deal's terms, sent with every list whose rows name it. See 000201 for
// what each kind means.
type syncDeal struct {
	UID         uuid.UUID `json:"uid"`
	Name        string    `json:"name"`
	Kind        string    `json:"kind"`
	BuyQty      int32     `json:"buy_qty"`
	GetQty      int32     `json:"get_qty"`
	Percent     int16     `json:"percent"`
	BundlePrice int64     `json:"bundle_price"`
	// What units outside a set get (000205); 0 for none.
	FallbackPercent int16 `json:"fallback_percent"`
}

// A discount that applies only when the cashier sells the variant as the
// named sale option (000199). The option travels by name: a till has no copy
// of kashi's sale_options ids.
type syncDiscountListOptionItem struct {
	ProductVariantUID uuid.UUID `json:"product_variant_uid"`
	SaleOption        string    `json:"sale_option"`
	Discount          int16     `json:"discount"`
}

type syncDiscountList struct {
	UID       uuid.UUID              `json:"uid"`
	Name      string                 `json:"name"`
	IsActive  bool                   `json:"is_active"`
	IsDefault bool                   `json:"is_default"`
	IsSpecial bool                   `json:"is_special"`
	ValidFrom time.Time              `json:"valid_from"`
	ValidTo   time.Time              `json:"valid_to"`
	UpdatedAt time.Time              `json:"updated_at"`
	Items     []syncDiscountListItem `json:"items"`
	// A separate field rather than more Items so a till that predates sale
	// options ignores them instead of reading them as plain discounts.
	OptionItems []syncDiscountListOptionItem `json:"option_items"`
	Deals       []syncDeal                   `json:"deals"`
}

// An inactive list is sent with an empty Items set: the till is being told to
// stop using it, and shipping its contents anyway would only be work for both
// sides. The till hard-deletes on !is_active.
func (server *Server) priceListsForBranchSync(ctx *gin.Context, branchID int64, since time.Time) ([]syncPriceList, error) {
	lists, err := server.store.ListPriceListsForBranchSync(ctx, db.ListPriceListsForBranchSyncParams{
		BranchID:  branchID,
		UpdatedAt: since,
	})
	if err != nil {
		return nil, err
	}

	out := make([]syncPriceList, 0, len(lists))
	for _, l := range lists {
		entry := syncPriceList{
			UID:       l.Uid,
			Name:      l.Name,
			IsActive:  l.IsActive,
			IsDefault: l.IsDefault,
			IsSpecial: l.IsSpecial,
			ValidFrom: l.ValidFrom,
			ValidTo:   l.ValidTo,
			UpdatedAt: l.UpdatedAt,
			Items:     []syncPriceListItem{},
		}
		if l.IsActive {
			rows, err := server.store.ListPriceListItemsForSync(ctx, l.ID)
			if err != nil {
				return nil, err
			}
			for _, r := range rows {
				entry.Items = append(entry.Items, syncPriceListItem{
					ProductVariantUID: r.ProductVariantUid,
					Price:             r.Price,
				})
			}
		}
		out = append(out, entry)
	}
	return out, nil
}

func (server *Server) discountListsForBranchSync(ctx *gin.Context, branchID int64, since time.Time) ([]syncDiscountList, error) {
	lists, err := server.store.ListDiscountListsForBranchSync(ctx, db.ListDiscountListsForBranchSyncParams{
		BranchID:  branchID,
		UpdatedAt: since,
	})
	if err != nil {
		return nil, err
	}

	out := make([]syncDiscountList, 0, len(lists))
	for _, l := range lists {
		entry := syncDiscountList{
			UID:         l.Uid,
			Name:        l.Name,
			IsActive:    l.IsActive,
			IsDefault:   l.IsDefault,
			IsSpecial:   l.IsSpecial,
			ValidFrom:   l.ValidFrom,
			ValidTo:     l.ValidTo,
			UpdatedAt:   l.UpdatedAt,
			Items:       []syncDiscountListItem{},
			OptionItems: []syncDiscountListOptionItem{},
			Deals:       []syncDeal{},
		}
		if l.IsActive {
			rows, err := server.store.ListDiscountListItemsForSync(ctx, l.ID)
			if err != nil {
				return nil, err
			}
			for _, r := range rows {
				item := syncDiscountListItem{
					ProductVariantUID: r.ProductVariantUid,
					Discount:          r.Discount,
				}
				if r.DealUid.Valid {
					item.DealUID = &r.DealUid.UUID
				}
				entry.Items = append(entry.Items, item)
			}
			deals, err := server.store.ListDealsForDiscountListSync(ctx, l.ID)
			if err != nil {
				return nil, err
			}
			for _, d := range deals {
				entry.Deals = append(entry.Deals, syncDeal{
					UID: d.Uid, Name: d.Name, Kind: d.Kind, BuyQty: d.BuyQty,
					GetQty: d.GetQty, Percent: d.Percent, BundlePrice: d.BundlePrice,
					FallbackPercent: d.FallbackPercent,
				})
			}
			optionRows, err := server.store.ListDiscountListOptionItemsForSync(ctx, l.ID)
			if err != nil {
				return nil, err
			}
			for _, r := range optionRows {
				entry.OptionItems = append(entry.OptionItems, syncDiscountListOptionItem{
					ProductVariantUID: r.ProductVariantUid,
					SaleOption:        r.SaleOption,
					Discount:          r.Discount,
				})
			}
		}
		out = append(out, entry)
	}
	return out, nil
}
