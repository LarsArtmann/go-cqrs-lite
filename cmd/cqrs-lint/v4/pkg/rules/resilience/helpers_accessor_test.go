package resilience

import (
	"go/ast"
	"path/filepath"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
)

// TestAccessorReturnsCQRSBus_DiscriminatesOnResultType pins the typed
// tightening of the Bus() accessor signal directly: over the committed
// busfixture module (real packages.Load type info), an accessor whose
// RESULT type is the cqrs event bus resolves as the journal tail, while a
// same-named accessor returning a downstream transport type is refused —
// the discrimination the name-only match could not make.
func TestAccessorReturnsCQRSBus_DiscriminatesOnResultType(t *testing.T) {
	// Not parallel: t.Setenv below.
	t.Setenv("GOWORK", "off")

	fixture, err := filepath.Abs("../../../testdata/busfixture")
	if err != nil {
		t.Fatalf("abs fixture path: %v", err)
	}

	ctx, err := analyzer.BuildContext(fixture)
	if err != nil {
		t.Fatalf("BuildContext(busfixture): %v", err)
	}
	if len(ctx.LoadErrors) > 0 {
		t.Fatalf("bus fixture loaded with errors: %v", ctx.LoadErrors)
	}

	engineTail := false
	rabbitRefused := false

	for _, gf := range ctx.GoFiles {
		if gf.IsTest {
			continue
		}

		ast.Inspect(gf.AST, func(n ast.Node) bool {
			assign, ok := n.(*ast.AssignStmt)
			if !ok {
				return true
			}

			for _, rhs := range assign.Rhs {
				call, ok := rhs.(*ast.CallExpr)
				if !ok {
					continue
				}

				sel, ok := call.Fun.(*ast.SelectorExpr)
				if !ok || sel.Sel.Name != "Bus" {
					continue
				}

				recv, ok := sel.X.(*ast.Ident)
				if !ok {
					continue
				}

				switch recv.Name {
				case "engine":
					engineTail = accessorReturnsCQRSBus(gf.Pkg, call)
				case "rabbit":
					rabbitRefused = !accessorReturnsCQRSBus(gf.Pkg, call)
				}
			}

			return true
		})
	}

	if !engineTail {
		t.Error("engineShell.Bus() (result event.Bus) must resolve as the cqrs journal tail")
	}
	if !rabbitRefused {
		t.Error(
			"rabbitConn.Bus() (result rabbitConn) must be refused by the typed result-type gate",
		)
	}
}
