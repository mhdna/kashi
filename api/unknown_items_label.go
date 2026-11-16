package api

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"os"
	"strings"
	"time"

	db "github.com/mhdna/kashi/db/sqlc"
)

// What the label reader (label_reader.go) does with a picture: read it as
// soon as the phone sends it, store the reading on the picture, and once
// the sale has landed turn each pictured line into a product -- the one
// its barcode already belongs to, or a new one tagged "From photo".

// readUnknownLabel reads one picture's label in the background and stores
// what it says. Off the upload path: the phone and the till never wait on
// the model, and a failed read just leaves the unit for a person.
func (server *Server) readUnknownLabel(imageID, sessionID int64, path, contentType string) {
	reader := server.labelReader
	if reader == nil {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), labelReadTimeout)
		defer cancel()
		data, err := os.ReadFile(path)
		if err == nil {
			var reading labelReading
			reading, err = reader.Read(ctx, data, contentType)
			if err == nil {
				f := cleanLabelReading(reading)
				price := sql.NullInt64{}
				if f.PriceCents != nil {
					price = sql.NullInt64{Int64: *f.PriceCents, Valid: true}
				}
				err = server.store.SetUnknownItemImageLabel(ctx, db.SetUnknownItemImageLabelParams{
					ID: imageID, LabelStatus: "read", LabelName: f.Name, LabelCode: f.Code,
					LabelBarcode: f.Barcode, LabelPrice: price,
				})
				if err == nil {
					server.autoResolveLabels(ctx, sessionID)
					return
				}
			}
		}
		log.Printf("label reader: image %d: %v", imageID, err)
		_ = server.store.SetUnknownItemImageLabel(ctx, db.SetUnknownItemImageLabelParams{
			ID: imageID, LabelStatus: "failed", LabelError: clip(err.Error(), 300),
		})
	}()
}

// autoResolveLabels turns the read, unresolved lines of one landed sale
// into products. Called when a reading arrives and when the sale lands,
// whichever comes second does the work. Lines it can't settle (no barcode,
// no name) stay in "To resolve" for a person.
func (server *Server) autoResolveLabels(ctx context.Context, sessionID int64) {
	server.labelMu.Lock()
	defer server.labelMu.Unlock()

	rows, err := server.store.ListLabelResolvableSlots(ctx, sessionID)
	if err != nil {
		log.Printf("label reader: session %d: %v", sessionID, err)
		return
	}
	// One picture covers a whole cart line: its units become one product.
	type group struct {
		row     db.ListLabelResolvableSlotsRow
		slotIDs []int64
	}
	var groups []*group
	byImage := map[int64]*group{}
	for _, row := range rows {
		g, ok := byImage[row.ImageID.Int64]
		if !ok {
			g = &group{row: row}
			byImage[row.ImageID.Int64] = g
			groups = append(groups, g)
		}
		g.slotIDs = append(g.slotIDs, row.ID)
	}

	created := false
	for _, g := range groups {
		f := labelFields{Name: g.row.LabelName, Code: g.row.LabelCode, Barcode: g.row.LabelBarcode}
		if g.row.LabelPrice.Valid {
			f.PriceCents = &g.row.LabelPrice.Int64
		}
		plan := planLabelProduct(f, g.row.ScannedBarcode, g.row.UnitPrice)
		if plan.Skip != "" {
			continue
		}
		variantID, isNew, err := server.labelVariant(ctx, plan, g.row.UnitPrice)
		if err != nil {
			log.Printf("label reader: session %d: %v", sessionID, err)
			continue
		}
		if err := server.store.ResolveUnknownItemsTx(ctx, g.slotIDs, variantID); err != nil {
			log.Printf("label reader: session %d: resolve: %v", sessionID, err)
			continue
		}
		for _, id := range g.slotIDs {
			_ = server.store.MarkUnknownItemSlotAutoResolved(ctx, id)
		}
		created = created || isNew
	}
	if created {
		server.branchHub.broadcastAll(branchWSMessage{Type: "products_updated"})
	}
	if len(groups) > 0 {
		server.adminHub.broadcastAll(adminWSMessage{Type: "unknown_item_uploaded"})
	}
}

var errLabelCodeTaken = errors.New("the label's code and its barcode are both taken by other products")

// labelVariant finds the product the barcode already belongs to, or makes
// one: the label's name and code (the barcode as the code when the label's
// is taken or missing), the cashier's price, tagged as made from a photo.
func (server *Server) labelVariant(ctx context.Context, plan labelPlan, unitPrice int64) (int64, bool, error) {
	if variant, err := server.store.GetProductVariantByBarcode(ctx, plan.Barcode); err == nil {
		return variant.ID, false, nil
	} else if !errors.Is(err, sql.ErrNoRows) {
		return 0, false, err
	}
	code := ""
	for _, candidate := range []string{plan.Code, plan.Barcode} {
		if strings.TrimSpace(candidate) == "" {
			continue
		}
		if _, err := server.store.GetProductByCode(ctx, candidate); errors.Is(err, sql.ErrNoRows) {
			code = candidate
			break
		} else if err != nil {
			return 0, false, err
		}
	}
	if code == "" {
		return 0, false, errLabelCodeTaken
	}
	price := unitPrice
	result, err := server.store.CreateProductTx(ctx, db.CreateProductTxParams{
		Code:            code,
		Name:            plan.Name,
		AttributeValues: attributeValues(withDefaultYear(Attributes{}, time.Now())),
		BarcodeText:     plan.Barcode,
		Price:           &price,
		Source:          db.ProductSourceLabelPhoto,
	})
	if err != nil {
		// Another reading made it a moment ago: use theirs.
		if isUniqueViolation(err) {
			if variant, lookupErr := server.store.GetProductVariantByBarcode(ctx, plan.Barcode); lookupErr == nil {
				return variant.ID, false, nil
			}
		}
		return 0, false, err
	}
	return result.Variant.ID, true, nil
}
