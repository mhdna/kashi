package api

import "github.com/mhdna/kashi/receiptrender"

// receiptTextsJSON is the stored form of a receipt_texts value (the receipt's
// design, from the Receipt Designer): cleaned up by the till's own code, so
// what is stored is what the till prints; '' for every default or anything
// unreadable.
func receiptTextsJSON(raw *string) string {
	if raw == nil {
		return ""
	}
	return receiptrender.NormalizeReceiptTextsJSON(*raw)
}
