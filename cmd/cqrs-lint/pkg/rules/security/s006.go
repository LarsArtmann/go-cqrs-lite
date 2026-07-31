package security

import (
	"context"
	"fmt"
	"go/ast"
	"strings"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/lintutil"
)

// S006: Financial data without encryption.
//
// Detects structs containing financially sensitive fields with serialization
// tags (json/cbor/db/gorm) when no encryption module is imported anywhere in
// the project. Uses a three-tier indicator system to balance recall against
// false-positive noise:
//
//   - STRONG (cardNumber, cvv, iban, …) — payment-instrument and banking
//     identifiers. Fire alone at High confidence.
//   - MEDIUM (salary, invoice, payment, …) — financial-domain nouns that
//     imply monetary data. Fire at Medium confidence.
//   - WEAK (amount, price, total, …) — generic monetary lexemes. Require
//     ≥2 distinct WEAK indicators (compound threshold) to fire at Low.
//
// All tiers require serialization evidence — a struct must have at least one
// field tagged json/cbor/db/gorm/sql to qualify. In-memory calculation structs
// (no tags) are never flagged.
//
// The rule is fully suppressed when the encryption module is imported anywhere
// in the module (encryption may be applied at the event-payload boundary, not
// on individual fields).
//
//nolint:ireturn // factory returns public interface
func NewS006Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"S006-financial-data-without-encryption",
		func(_ context.Context) ([]finding.Finding, error) {
			if lintutil.ModuleImportsPath(ctx, "go-cqrs-lite/encryption") {
				return nil, nil
			}

			var matches []financialMatch
			for _, gf := range ctx.GoFiles {
				if gf.IsTest {
					continue
				}

				ast.Inspect(gf.AST, func(n ast.Node) bool {
					ts, ok := n.(*ast.TypeSpec)
					if !ok {
						return true
					}
					st, ok := ts.Type.(*ast.StructType)
					if !ok || st.Fields == nil {
						return true
					}

					bestTier, weakCount := scoreFinancialStruct(ts.Name.Name, st)
					if bestTier == tierNone {
						return true
					}
					if bestTier == tierWeak && weakCount < 2 {
						return true
					}
					if !hasSerializationTags(st) {
						return true
					}

					pos := ctx.Fset.Position(ts.Pos())
					matches = append(matches, financialMatch{
						name:     ts.Name.Name,
						tier:     bestTier,
						filename: pos.Filename,
						line:     pos.Line,
						column:   pos.Column,
					})
					return true
				})
			}

			if len(matches) == 0 {
				return nil, nil
			}

			var findings []finding.Finding
			for _, m := range matches {
				severity := finding.SeverityWarning
				confidence := finding.ConfidenceMedium
				msg := fmt.Sprintf(
					"Financial data in struct %q without encryption — sensitive monetary data is stored in plaintext",
					m.name,
				)
				suggestion := "Import the encryption module and add encryption.EncryptMiddleware to your bus.UsePublish chain"

				switch m.tier {
				case tierStrong:
					severity = finding.SeverityError
					confidence = finding.ConfidenceHigh
					msg = fmt.Sprintf(
						"Financial instrument data in struct %q without encryption — "+
							"payment/banking identifiers are stored in plaintext",
						m.name,
					)
				case tierMedium:
					severity = finding.SeverityWarning
					confidence = finding.ConfidenceMedium
				case tierWeak:
					severity = finding.SeverityInfo
					confidence = finding.ConfidenceLow
					suggestion = "Consider encrypting monetary fields if this data is sensitive"
				case tierNone:
					// Filtered by scoreFinancialStruct; unreachable here.
				}

				if !ctx.FeatureProfile.HasServer {
					severity = finding.SeverityInfo
					if confidence > finding.ConfidenceLow {
						confidence = finding.ConfidenceLow
					}
					suggestion = "This appears to be a local-only project (no HTTP/gRPC server). " +
						"Add encryption if this data may be exposed to networks"
				}

				f, err := finding.NewBuilder(
					"S006", toolName,
					msg,
					severity,
					finding.Pos(finding.FilePath(m.filename), m.line, m.column),
				).
					WithCategory(finding.CategorySecurity).
					WithConfidence(confidence).
					WithSuggestion(suggestion).
					WithSnippet(ctx.SourceLine(m.filename, m.line)).
					Build()
				if err == nil {
					findings = append(findings, f)
				}
			}

			return findings, nil
		},
	)
}

// --- Tier system ---

type indicatorTier int

const (
	tierNone indicatorTier = iota
	tierWeak
	tierMedium
	tierStrong
)

func maxTier(a, b indicatorTier) indicatorTier {
	if a > b {
		return a
	}
	return b
}

var strongFinancial = []string{ //nolint:gochecknoglobals // static lookup table
	"cardnumber", "creditcardnumber", "cvv", "cvc",
	"primaryaccountnumber", "iban", "bic", "swift", "routingnumber",
	"sortcode",
}

var mediumFinancial = []string{ //nolint:gochecknoglobals // static lookup table
	"salary", "wage", "payroll", "invoice",
	"payment", "refund", "bankaccount",
	"creditcard", "debitcard", "transaction",
	"billing",
}

var weakFinancial = []string{ //nolint:gochecknoglobals // static lookup table
	"amount", "price", "total", "balance",
	"cost", "fee", "tax", "subtotal",
	"discount", "currency", "monetary",
}

func financialIndicatorLevel(name string) indicatorTier {
	n := strings.ToLower(name)
	for _, s := range strongFinancial {
		if strings.Contains(n, s) {
			return tierStrong
		}
	}
	for _, m := range mediumFinancial {
		if strings.Contains(n, m) {
			return tierMedium
		}
	}
	for _, w := range weakFinancial {
		if strings.Contains(n, w) {
			return tierWeak
		}
	}
	return tierNone
}

// scoreFinancialStruct evaluates a struct's type name and field names,
// returning the highest tier found and the count of weak-only indicators.
func scoreFinancialStruct(
	typeName string,
	st *ast.StructType,
) (bestTier indicatorTier, weakCount int) {
	if t := financialIndicatorLevel(typeName); t != tierNone {
		bestTier = maxTier(bestTier, t)
		if t == tierWeak {
			weakCount++
		}
	}

	for _, field := range st.Fields.List {
		for _, name := range field.Names {
			t := financialIndicatorLevel(name.Name)
			if t == tierNone {
				continue
			}
			bestTier = maxTier(bestTier, t)
			if t == tierWeak {
				weakCount++
			}
		}
	}

	return bestTier, weakCount
}

// --- Serialization gate ---

func hasSerializationTags(st *ast.StructType) bool {
	for _, field := range st.Fields.List {
		if field.Tag == nil {
			continue
		}
		tag := field.Tag.Value
		if strings.Contains(tag, "json:") ||
			strings.Contains(tag, "cbor:") ||
			strings.Contains(tag, "db:") ||
			strings.Contains(tag, "gorm:") ||
			strings.Contains(tag, "sql:") {
			return true
		}
	}
	return false
}

// --- Module-scope encryption check ---

func moduleHasEncryption(ctx *analyzer.AnalysisContext) bool {
	for _, pkg := range ctx.Packages {
		for _, imp := range pkg.Imports {
			if imp == nil {
				continue
			}
			if strings.Contains(imp.PkgPath, "go-cqrs-lite/encryption") {
				return true
			}
		}
	}

	for _, gf := range ctx.GoFiles {
		if gf.IsTest {
			continue
		}
		for _, imp := range gf.AST.Imports {
			if imp.Path == nil {
				continue
			}
			path := strings.Trim(imp.Path.Value, `"`)
			if strings.Contains(path, "go-cqrs-lite/encryption") {
				return true
			}
		}
	}

	return false
}

// --- Finding carrier ---

type financialMatch struct {
	name     string
	tier     indicatorTier
	filename string
	line     int
	column   int
}
