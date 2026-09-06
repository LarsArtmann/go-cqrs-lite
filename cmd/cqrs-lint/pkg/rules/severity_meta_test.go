package rules

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// The severity/confidence contract: what a detector's finding.NewBuilder call
// statically declares must match the catalog entry for the same rule ID.
// The catalog feeds `rules --json`, the README table, and SARIF metadata; the
// builder is what findings actually carry. When they disagree, one of them is
// lying — this meta-test makes that a CI failure instead of a silent
// split-brain (14 such mismatches were found and fixed when this was
// introduced, including security rules that warned instead of erroring).

// builderDecl is one statically-extractable finding.NewBuilder call.
type builderDecl struct {
	rule string
	sev  string // "finding.SeverityWarning" or "IDENT:sev" for parameters
	conf string // "finding.ConfidenceHigh", "IDENT:confidence", or "(absent)"
	pos  string
}

// conditionalSeverityRules escalate severity/confidence at runtime from
// detector inputs; their catalog value documents the TYPICAL case.
var conditionalSeverityRules = map[string]string{
	"B008": "manual-retry severity depends on loop shape",
	"C008": "escalates when the field is a confirmed money name",
	"S002": "escalates for high-sensitivity payload names",
	"S006": "escalates for confirmed financial field names",
}

// helperMediatedRules build findings through parameterized helpers
// (singleFinding/singleInfoFinding and siblings), so no literal rule ID sits
// inside a finding.NewBuilder call for the meta-test to read. Each batch was
// or will be verified in the rule audit waves (TODO_LIST § cqrs-lint).
// helperMediatedRules is built by a function (not init) so gochecknoinits
// stays quiet and the F001-F030 family needs no 30 hand-written entries.
var helperMediatedRules = buildHelperMediated()

func buildHelperMediated() map[string]string {
	m := map[string]string{
		"B023": "built via boilerplate helper",
		"B029": "built via boilerplate helper",
		"B030": "built via boilerplate helper",
		"B031": "built via boilerplate helper",
	}
	for i := 1; i <= 8; i++ {
		m[fmt.Sprintf("E%03d", i+7)] = "built via singleFinding helper"
		m[fmt.Sprintf("T%03d", i)] = "built via testing helper"
	}
	for i := 1; i <= 30; i++ {
		m[fmt.Sprintf("F%03d", i)] = "built via adoption helper"
	}
	return m
}

// severityVariants allowlist documented secondary emission paths for a rule
// whose sub-conditions warrant a different severity than the catalog
// headline. Key: "RULE|severity|confidence".
var severityVariants = map[string]string{
	"A017|info|low": "advisory branch (no snapshot store AND no state cache); the catalog headline (warning/high) covers the WithSnapshotStore-without-strategy misconfiguration",
}

var sevLiterals = map[string]string{
	"finding.SeverityCritical": "critical",
	"finding.SeverityError":    "error",
	"finding.SeverityWarning":  "warning",
	"finding.SeverityInfo":     "info",
}

var confLiterals = map[string]string{
	"finding.ConfidenceNone":   "none",
	"finding.ConfidenceLow":    "low",
	"finding.ConfidenceMedium": "medium",
	"finding.ConfidenceHigh":   "high",
	"finding.ConfidenceFull":   "full",
}

func exprLabel(e ast.Expr) string {
	switch t := e.(type) {
	case *ast.SelectorExpr:
		if id, ok := t.X.(*ast.Ident); ok {
			return id.Name + "." + t.Sel.Name
		}

		return "?.?" + t.Sel.Name
	case *ast.Ident:
		return "IDENT:" + t.Name
	case *ast.BasicLit:
		return t.Value
	}

	return "?expr"
}

// scanBuilderDecls walks the rules tree and returns every finding.NewBuilder
// call with a literal rule ID, plus the WithConfidence value reached through
// the builder chain when one is present.
func scanBuilderDecls(t *testing.T) []builderDecl {
	t.Helper()

	var out []builderDecl
	confByPos := map[string]string{}
	fset := token.NewFileSet()

	visit := func(path string) {
		rel, err := filepath.Rel(".", path)
		if err != nil {
			return
		}

		if strings.Contains(rel, "testrules") {
			return
		}

		src, perr := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
		if perr != nil {
			t.Logf("severity meta-test: skipped unparseable %s: %v", rel, perr)
			return
		}

		ast.Inspect(src, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}

			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}

			switch sel.Sel.Name {
			case "NewBuilder":
				lit, ok := call.Args[0].(*ast.BasicLit)
				if !ok {
					return true
				}

				pos := rel + ":" + strconv.Itoa(fset.Position(call.Pos()).Line)
				out = append(out, builderDecl{
					rule: strings.Trim(lit.Value, `"`),
					sev:  exprLabel(call.Args[3]),
					pos:  pos,
				})
			case "WithConfidence":
				x := sel.X

				for range 12 {
					call2, ok := x.(*ast.CallExpr)
					if !ok {
						break
					}

					sel2, ok := call2.Fun.(*ast.SelectorExpr)
					if !ok {
						break
					}

					if sel2.Sel.Name == "NewBuilder" {
						if _, ok := call2.Args[0].(*ast.BasicLit); ok {
							pos2 := rel + ":" + strconv.Itoa(fset.Position(call2.Pos()).Line)
							confByPos[pos2] = exprLabel(call.Args[0])
						}

						break
					}

					x = sel2.X
				}
			}

			return true
		})
	}

	_ = filepath.WalkDir(".", func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			name := d.Name()
			hidden := strings.HasPrefix(name, ".") && name != "." && name != ".."

			if name == "vendor" || name == "testdata" || hidden {
				return filepath.SkipDir
			}

			return nil
		}

		if strings.HasSuffix(path, ".go") && !strings.HasSuffix(path, "_test.go") {
			visit(path)
		}

		return nil
	})

	for i := range out {
		if conf, ok := confByPos[out[i].pos]; ok {
			out[i].conf = conf
		} else {
			out[i].conf = "(absent)"
		}
	}

	sort.Slice(out, func(i, j int) bool { return out[i].pos < out[j].pos })

	return out
}

func TestRuleSeverityMatchesCatalog(t *testing.T) {
	decls := scanBuilderDecls(t)
	if len(decls) < 100 {
		t.Fatalf("only %d builder declarations found; scanner is broken", len(decls))
	}

	var problems []string
	seen := map[string]bool{}

	for _, d := range decls {
		entry, ok := LookupRule(d.rule)
		if !ok {
			problems = append(problems, fmt.Sprintf("%s: %s has no catalog entry", d.pos, d.rule))
			continue
		}

		seen[d.rule] = true

		sev, sevIsLiteral := sevLiterals[d.sev]
		conf, confIsLiteral := confLiterals[d.conf]

		if sevIsLiteral && confIsLiteral {
			if _, variant := severityVariants[d.rule+"|"+sev+"|"+conf]; variant {
				continue
			}
		}

		if sev, isLiteral := sevLiterals[d.sev]; isLiteral && sev != entry.Severity {
			problems = append(problems, fmt.Sprintf(
				"%s: %s builder emits severity %q but catalog says %q",
				d.pos,
				d.rule,
				sev,
				entry.Severity,
			))
		}

		if conf, isLiteral := confLiterals[d.conf]; isLiteral && conf != entry.Confidence {
			problems = append(problems, fmt.Sprintf(
				"%s: %s builder emits confidence %q but catalog says %q",
				d.pos,
				d.rule,
				conf,
				entry.Confidence,
			))
		}

		if strings.HasPrefix(d.sev, "IDENT:") {
			if _, allowed := conditionalSeverityRules[d.rule]; !allowed {
				problems = append(problems, fmt.Sprintf(
					"%s: %s passes a dynamic severity but is not in conditionalSeverityRules",
					d.pos,
					d.rule,
				))
			}
		}
	}

	if len(problems) > 0 {
		sort.Strings(problems)
		t.Fatalf("severity/confidence split-brains between builders and catalog:\n%s",
			strings.Join(problems, "\n"))
	}
}

func TestCatalogRulesDeclareFindings(t *testing.T) {
	decls := scanBuilderDecls(t)

	seen := map[string]bool{}
	for _, d := range decls {
		seen[d.rule] = true
	}

	var unexplained []string

	for _, entry := range AllRules() {
		if seen[entry.ID] {
			continue
		}

		if _, ok := helperMediatedRules[entry.ID]; !ok {
			unexplained = append(unexplained, entry.ID)
		}
	}

	if len(unexplained) > 0 {
		sort.Strings(unexplained)
		t.Fatalf(
			"catalog rules with no finding.NewBuilder site and no helperMediatedRules entry:\n%s",
			strings.Join(unexplained, ", "),
		)
	}
}
