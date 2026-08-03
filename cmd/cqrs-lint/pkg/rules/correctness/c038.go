package correctness

import (
	"context"
	"fmt"
	"go/ast"
	"strings"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/lintutil"
)

// C038: Event-type string typo detection.
//
// Cross-references event type strings emitted via event.New/NewEvent against
// the case labels in fold/apply functions. If an emitted type does NOT match
// any fold case but is within edit distance 1-2 of a case that does, the event
// is likely a typo — it will be silently dropped by the fold because the switch
// arm never matches.
//
// This catches one of the most insidious event-sourcing bugs: the decider
// emits "user.creted" (typo) but the fold handles "user.created". The event
// is persisted to the store, replayed on every load, but never applied to
// state — the aggregate silently loses data.
//
//nolint:ireturn // factory returns public interface
func NewC038Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"C038-event-type-typo",
		func(_ context.Context) ([]finding.Finding, error) {
			foldCases := collectFoldCaseStrings(ctx)
			if len(foldCases) == 0 {
				return nil, nil
			}

			emitted := ctx.Registry.EventTypesEmitted
			if len(emitted) == 0 {
				return nil, nil
			}

			caseSet := make(map[string]bool, len(foldCases))
			for _, c := range foldCases {
				caseSet[c] = true
			}

			var findings []finding.Finding

			for eventType, emission := range emitted {
				if caseSet[eventType] {
					continue
				}

				if closest, dist := nearestMatch(eventType, foldCases); closest != "" && dist <= 2 {
					f, err := finding.NewBuilder(
						"C038", toolName,
						fmt.Sprintf(
							"event type %q is emitted but no fold handles it — "+
								"did you mean %q? (edit distance %d) — "+
								"events emitted but not handled are silently dropped during replay",
							eventType, closest, dist,
						),
						finding.SeverityError,
						finding.Pos(finding.FilePath(emission.File), emission.Line, 1),
					).
						WithCategory(finding.CategoryCorrectness).
						WithConfidence(finding.ConfidenceHigh).
						WithFixStrategy(finding.FixStrategySuggest).
						WithSuggestion(fmt.Sprintf("Change the event type string from %q to %q", eventType, closest)).
						WithSnippet(ctx.SourceLine(emission.File, emission.Line)).
						Build()
					lintutil.AppendBuild(&findings, f, err)
				}
			}

			return findings, nil
		},
	)
}

// collectFoldCaseStrings walks all fold functions in the registry and extracts
// the string-literal case labels from their switch statements. A fold function
// is identified by the scanner as a function with signature
// func(state, event) (state, error) — see detectFoldFunc in scanner_folds.go.
func collectFoldCaseStrings(ctx *analyzer.AnalysisContext) []string {
	var cases []string

	for _, fold := range ctx.Registry.Folds {
		for _, gf := range ctx.GoFiles {
			if gf.Path != fold.File {
				continue
			}

			ast.Inspect(gf.AST, func(n ast.Node) bool {
				fn, ok := n.(*ast.FuncDecl)
				if !ok || fn.Name == nil {
					return true
				}

				if fn.Name.Name != fold.FuncName {
					return true
				}

				ast.Inspect(fn.Body, func(nn ast.Node) bool {
					sw, ok := nn.(*ast.SwitchStmt)
					if !ok {
						return true
					}

					for _, stmt := range sw.Body.List {
						cc, ok := stmt.(*ast.CaseClause)
						if !ok || cc.List == nil {
							continue
						}

						for _, expr := range cc.List {
							if s := analyzer.StringLit(expr); s != "" {
								cases = append(cases, s)
							}
						}
					}

					return true
				})

				return true
			})
		}
	}

	return cases
}

// nearestMatch returns the closest string from candidates to target, along
// with its edit distance. Returns "" if candidates is empty.
func nearestMatch(target string, candidates []string) (string, int) {
	best := ""
	bestDist := -1

	for _, c := range candidates {
		d := editDistance(target, c)
		if bestDist == -1 || d < bestDist {
			best = c
			bestDist = d
		}
	}

	return best, bestDist
}

// editDistance computes the Levenshtein distance between two strings.
// Used for typo detection — a distance of 1-2 indicates a likely typo.
func editDistance(a, b string) int {
	a = strings.ToLower(a)
	b = strings.ToLower(b)

	la, lb := len(a), len(b)
	if la == 0 {
		return lb
	}

	if lb == 0 {
		return la
	}

	prev := make([]int, lb+1)
	curr := make([]int, lb+1)

	for j := 0; j <= lb; j++ {
		prev[j] = j
	}

	for i := 1; i <= la; i++ {
		curr[0] = i

		for j := 1; j <= lb; j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}

			curr[j] = min3(curr[j-1]+1, prev[j]+1, prev[j-1]+cost)
		}

		prev, curr = curr, prev
	}

	return prev[lb]
}

func min3(a, b, c int) int {
	if b < a {
		a = b
	}

	if c < a {
		return c
	}

	return a
}
