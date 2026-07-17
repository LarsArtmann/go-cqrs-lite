package correctness

import (
	"context"
	"fmt"
	"go/ast"
	"strings"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

// C008: float64 for money.
// Detects float64 fields with monetary names in struct types.
//
// To avoid false positives on generic field names (value, total) used outside
// monetary contexts (e.g. observability counters, lag metrics), a field fires
// only when it carries a STRONG money signal on its own (amount, price, cost,
// balance, fee) OR when a WEAK signal is corroborated by a money-related
// struct/package name. See feedback: docs/feedback/2026-07-16_DiscordSync.
func NewC008Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	// strongMoneyFields are unambiguous enough to flag on field name alone.
	strongMoneyFields := []string{"amount", "price", "cost", "balance", "fee"}
	// weakMoneyFields are generic (value, total) — only flag when the
	// enclosing struct/package name also looks monetary.
	weakMoneyFields := []string{"value", "total", "charge", "payment", "salary"}
	// moneyStructKeywords corroborate weak field names.
	moneyStructKeywords := []string{
		"order", "invoice", "payment", "price", "cost", "balance",
		"account", "billing", "transaction", "money", "cart",
		"purchase", "refund", "tax", "wallet", "subscription",
		"charge", "fee", "salary", "payroll", "fund", "deposit",
	}

	return finding.NamedDetectorFunc(
		"C008-float64-for-money",
		func(_ context.Context) ([]finding.Finding, error) {
			var findings []finding.Finding

			for _, gf := range ctx.GoFiles {
				if gf.IsTest {
					continue
				}

				pkgMoney := packageLooksMonetary(gf.Pkg.PkgPath)

				handled := make(map[*ast.StructType]bool)

				// Pass 1: named structs (struct name available for corroboration).
				ast.Inspect(gf.AST, func(n ast.Node) bool {
					ts, ok := n.(*ast.TypeSpec)
					if !ok {
						return true
					}

					st, ok := ts.Type.(*ast.StructType)
					if !ok || st.Fields == nil {
						return true
					}

					handled[st] = true

					structMoney := pkgMoney || isMoneyStructName(ts.Name.Name, moneyStructKeywords)
					findings = append(findings, scanMoneyFields(
						ctx, st, structMoney, strongMoneyFields, weakMoneyFields,
					)...)

					return true
				})

				// Pass 2: anonymous structs (no name → weak fields need package corroboration).
				ast.Inspect(gf.AST, func(n ast.Node) bool {
					st, ok := n.(*ast.StructType)
					if !ok || st.Fields == nil || handled[st] {
						return true
					}

					findings = append(findings, scanMoneyFields(
						ctx, st, pkgMoney, strongMoneyFields, weakMoneyFields,
					)...)

					return true
				})
			}

			return findings, nil
		},
	)
}

// scanMoneyFields returns findings for float64 fields that match the money
// heuristic. structMoney indicates whether the enclosing context (struct or
// package) already carries a money signal, allowing weak field names to fire.
func scanMoneyFields(
	ctx *analyzer.AnalysisContext,
	st *ast.StructType,
	structMoney bool,
	strongMoneyFields, weakMoneyFields []string,
) []finding.Finding {
	var findings []finding.Finding

	for _, field := range st.Fields.List {
		if !isFloat64(field.Type) {
			continue
		}

		for _, name := range field.Names {
			lowerName := strings.ToLower(name.Name)

			strong := matchesAny(lowerName, strongMoneyFields)
			weak := matchesAny(lowerName, weakMoneyFields)
			if !strong && !weak {
				continue
			}

			// Weak field names (value, total) require a money context.
			if !strong && !structMoney {
				continue
			}

			pos := ctx.Fset.Position(name.Pos())

			f, err := finding.NewBuilder(
				"C008",
				toolName,
				fmt.Sprintf(
					"Field %s is float64 — use decimal or integer cents for money to avoid rounding errors",
					name.Name,
				),
				finding.SeverityWarning,
				finding.Pos(finding.FilePath(pos.Filename), pos.Line, pos.Column),
			).
				WithCategory(finding.CategoryCorrectness).
				WithConfidence(finding.ConfidenceMedium).
				WithSuggestion("Use shopspring/decimal or int64 cents instead of float64 for monetary values").
				WithSnippet(ctx.SourceLine(pos.Filename, pos.Line)).
				Build()
			if err == nil {
				findings = append(findings, f)
			}
		}
	}

	return findings
}

// matchesAny reports whether name contains any of the substrings.
func matchesAny(name string, terms []string) bool {
	for _, term := range terms {
		if strings.Contains(name, term) {
			return true
		}
	}

	return false
}

// isMoneyStructName reports whether a struct name contains a monetary keyword.
func isMoneyStructName(structName string, moneyStructKeywords []string) bool {
	lower := strings.ToLower(structName)

	return matchesAny(lower, moneyStructKeywords)
}

// packageLooksMonetary reports whether the package path suggests a monetary
// domain (e.g. ".../billing", ".../payments").
func packageLooksMonetary(pkgPath string) bool {
	if pkgPath == "" {
		return false
	}

	lower := strings.ToLower(pkgPath)
	for _, seg := range []string{
		"billing", "payment", "invoice", "order", "checkout",
		"price", "wallet", "refund", "subscription", "payroll", "accounting",
	} {
		if strings.Contains(lower, "/"+seg) || strings.HasSuffix(lower, seg) || lower == seg {
			return true
		}
	}

	return false
}
