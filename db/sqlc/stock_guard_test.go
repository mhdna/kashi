package db

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
	"testing"
)

// TestApplyStockMovementIsTheOnlyStockWriter enforces the audit coupling
// applyStockMovement documents: AddInventoryStockQuantity and
// ReceiveInventoryStock -- the two queries that actually change on-hand --
// may only be called from inside applyStockMovement itself. Every other
// caller must go through applyStockMovement so a stock_movements audit row
// is structurally guaranteed to exist alongside the change. This mirrors
// aman-lomr's TestNoRawDatabaseTransactionsOutsideHelper.
func TestApplyStockMovementIsTheOnlyStockWriter(t *testing.T) {
	guarded := []string{"AddInventoryStockQuantity", "ReceiveInventoryStock"}

	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("glob source files: %v", err)
	}

	fset := token.NewFileSet()
	for _, file := range files {
		if filepath.Ext(file) != ".go" {
			continue
		}
		// Skip test files: the guard's intent is that production code paths
		// can't bypass applyStockMovement's audit-trail coupling, not that no
		// test may ever exercise the raw query directly. inventory_test.go's
		// TestAddInventoryStockQuantity/TestReceiveInventoryStockMovingAverage
		// deliberately call these functions directly to unit-test the SQL
		// itself (the upsert and moving-average arithmetic), which is a
		// separate, legitimate concern from what this guard protects.
		if strings.HasSuffix(file, "_test.go") {
			continue
		}

		src, err := parser.ParseFile(fset, file, nil, parser.ParseComments)
		if err != nil {
			t.Fatalf("parse %s: %v", file, err)
		}

		var currentFunc string
		ast.Inspect(src, func(n ast.Node) bool {
			if fn, ok := n.(*ast.FuncDecl); ok {
				currentFunc = fn.Name.Name
				return true
			}
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			for _, name := range guarded {
				if sel.Sel.Name == name && currentFunc != "applyStockMovement" {
					t.Errorf(
						"%s:%d: %s must only be called from applyStockMovement (called from %s), "+
							"otherwise the change bypasses the stock_movements audit trail",
						file, fset.Position(call.Pos()).Line, name, currentFunc,
					)
				}
			}
			return true
		})
	}
}
