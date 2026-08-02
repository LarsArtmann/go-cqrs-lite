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

// strongMoneyFields are unambiguous enough to flag on field name alone.
//
//nolint:gochecknoglobals // read-only keyword list
var strongMoneyFields = []string{"amount", "price", "cost", "balance", "fee"}

// weakMoneyFields are generic (value, total) — only flag when the enclosing
// struct/package name also looks monetary.
//
//nolint:gochecknoglobals // read-only keyword list
var weakMoneyFields = []string{"value", "total", "charge", "payment", "salary", "rate"}

// moneyKeywords is the unified set of monetary terms used for struct-name,
// package-path, and embedded-type corroboration. Previously duplicated across
// moneyKeywords (local var) and packageLooksMonetary (hardcoded list).
//
//nolint:gochecknoglobals // read-only keyword list
var moneyKeywords = []string{
	"order", "invoice", "payment", "price", "cost", "balance",
	"account", "billing", "transaction", "money", "cart",
	"purchase", "refund", "tax", "wallet", "subscription",
	"charge", "fee", "salary", "payroll", "fund", "deposit",
}

//nolint:ireturn // factory returns public interface
func NewC008Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"C008-float64-for-money",
		func(_ context.Context) ([]finding.Finding, error) {
			var findings []finding.Finding

			// Project-level monetary signal: if NO package path or struct name
			// anywhere in the project looks monetary, then strong field names
			// (amount, balance) are probably not about money either — downgrade
			// those findings to Info/Low rather than polluting a non-monetary
			// codebase with Warning/Medium noise. Covers item f-8 in the
			// DiscordSync feedback triage.
			projectMonetary := projectHasMonetarySignal(ctx, moneyKeywords)

			for _, gf := range ctx.GoFiles {
				if gf.IsTest {
					continue
				}

				pkgMoney := packageLooksMonetary(gf.Pkg.PkgPath) || projectMonetary

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

					structMoney := pkgMoney ||
						isMoneyStructName(ts.Name.Name, moneyKeywords) ||
						hasMoneyEmbed(st, moneyKeywords)
					findings = append(findings, scanMoneyFields(
						ctx, st, structMoney, projectMonetary,
						strongMoneyFields, weakMoneyFields,
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
						ctx, st, pkgMoney, projectMonetary,
						strongMoneyFields, weakMoneyFields,
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
// projectMonetary, when false, downgrades strong-field findings to Info/Low —
// a non-monetary project's "amount"/"balance" is probably not about money.
func scanMoneyFields(
	ctx *analyzer.AnalysisContext,
	st *ast.StructType,
	structMoney bool,
	projectMonetary bool,
	strongMoneyFields, weakMoneyFields []string,
) []finding.Finding {
	var findings []finding.Finding

	severity := finding.SeverityWarning
	confidence := finding.ConfidenceMedium
	if !projectMonetary {
		// No package or struct anywhere in the project looks monetary. Strong
		// field names (amount, price) stay reportable — they're suspicious — but
		// read as "maybe money" rather than "almost certainly money".
		severity = finding.SeverityInfo
		confidence = finding.ConfidenceLow
	}

	for _, field := range st.Fields.List {
		if !isFloat64(field.Type) {
			continue
		}

		for _, name := range field.Names {
			lowerName := strings.ToLower(name.Name)

			// Config opt-out: fields listed in c008-ignore-fields are
			// intentionally float64 (cost estimates, metrics, etc.).
			if matchesAny(lowerName, ctx.RulesConfig.IgnoreFloatFields) {
				continue
			}

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
					"Field %s is float64/float32 — use decimal or integer cents for money to avoid rounding errors",
					name.Name,
				),
				severity,
				finding.Pos(finding.FilePath(pos.Filename), pos.Line, pos.Column),
			).
				WithCategory(finding.CategoryCorrectness).
				WithConfidence(confidence).
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

// projectHasMonetarySignal reports whether any package path or named struct in
// the project carries a money keyword. This is the project-level "is this
// plausibly a payments/billing app?" signal that C008 consults to downgrade
// findings on non-monetary codebases.
//
// Struct names live at the top level (GenDecl → TypeSpec), so we iterate
// declarations directly instead of a full ast.Inspect tree walk. This skips all
// function bodies and expressions, cutting the cost of the project pre-scan on
// large consumer repos (round-2 self-critique §d-2).
func projectHasMonetarySignal(ctx *analyzer.AnalysisContext, moneyKeywords []string) bool {
	for _, gf := range ctx.GoFiles {
		if gf.IsTest {
			continue
		}

		if packageLooksMonetary(gf.Pkg.PkgPath) {
			return true
		}

		for _, decl := range gf.AST.Decls {
			gd, ok := decl.(*ast.GenDecl)
			if !ok {
				continue
			}

			for _, spec := range gd.Specs {
				ts, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}

				if isMoneyStructName(ts.Name.Name, moneyKeywords) {
					return true
				}
			}
		}
	}

	return false
}

// hasMoneyEmbed reports whether the struct embeds an anonymous field whose type
// name contains a monetary keyword (e.g. `type Order struct { MoneyMixin; ... }`).
// This catches money fields inherited via embedding without requiring cross-file
// type resolution — a conservative heuristic.
func hasMoneyEmbed(st *ast.StructType, moneyKeywords []string) bool {
	if st.Fields == nil {
		return false
	}

	for _, field := range st.Fields.List {
		if len(field.Names) > 0 {
			continue // embedded fields have no names
		}

		var name string
		switch t := field.Type.(type) {
		case *ast.Ident:
			name = t.Name
		case *ast.StarExpr:
			if id, ok := t.X.(*ast.Ident); ok {
				name = id.Name
			}
		case *ast.SelectorExpr:
			name = t.Sel.Name
		}

		if name != "" && isMoneyStructName(name, moneyKeywords) {
			return true
		}
	}

	return false
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
func isMoneyStructName(structName string, moneyKeywords []string) bool {
	lower := strings.ToLower(structName)

	return matchesAny(lower, moneyKeywords)
}

// packageLooksMonetary reports whether the package path suggests a monetary
// domain (e.g. ".../billing", ".../payments"). Uses the shared moneyKeywords
// list.
func packageLooksMonetary(pkgPath string) bool {
	if pkgPath == "" {
		return false
	}

	lower := strings.ToLower(pkgPath)
	for _, seg := range moneyKeywords {
		if strings.Contains(lower, "/"+seg) || strings.HasSuffix(lower, seg) || lower == seg {
			return true
		}
	}

	return false
}
