package main

import (
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
)

// adtHeadingPatterns maps a metaengine ADT identifier (as returned by
// metaengine.AllADTs) to a case-insensitive regex that a recipes.md heading
// must match for the ADT to count as recipe-covered.
//
// The key set is cross-checked against the real AllADTs() body in
// metaengine/enum_validation.go — a new ADT fails this test until it has a
// recipe heading or an explicit waiver below. No module dependency needed:
// the source is parsed.
var adtHeadingPatterns = map[string]*regexp.Regexp{
	"ADTMap":       regexp.MustCompile(`(?i)\bmap\b`),
	"ADTSet":       regexp.MustCompile(`(?i)\bset\b`),
	"ADTCounter":   regexp.MustCompile(`(?i)\bcounter\b`),
	"ADTGraph":     regexp.MustCompile(`(?i)\bgraph\b`),
	"ADTLog":       regexp.MustCompile(`(?i)\blog\b`),
	"ADTStreamLog": regexp.MustCompile(`(?i)stream.?log`),
	"ADTSortedMap": regexp.MustCompile(`(?i)sorted.?map`),
	"ADTMultimap":  regexp.MustCompile(`(?i)multimap`),
	"ADTVector":    regexp.MustCompile(`(?i)\bvector\b`),
	"ADTSearch":    regexp.MustCompile(`(?i)full.?text|search adt`),
	"ADTSpatial":   regexp.MustCompile(`(?i)\bspatial\b|\bgeo\b`),
}

// adtCoverageWaivers lists ADTs with NO recipe heading yet, each with a
// reason and an owner action. Waivers are visible in test output (t.Logf),
// never silent. Empty reason is a hard error — an unexplained gap is a hole,
// not a decision.
var adtCoverageWaivers = map[string]string{
	"ADTSet":       "no recipe yet; FoldSet mirrors the Map fold pattern — add alongside the next Set consumer",
	"ADTLog":       "no recipe yet; log-tail reads documented only in readmodels.md tier notes",
	"ADTStreamLog": "no recipe yet; single prose mention (structural routing note in 2.19)",
	"ADTSortedMap": "no recipe yet; no fleet consumer uses ordered maps",
	"ADTMultimap":  "no recipe yet; no fleet consumer uses multimaps",
}

// TestRecipeADTCoverage is the T18 ratchet: every metaengine ADT is either
// demonstrated by a recipes.md heading or carried in the explicit waiver
// table. It keeps the recipe book from silently drifting away from the ADT
// surface as new ADTs land.
func TestRecipeADTCoverage(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("repo root: %v", err)
	}

	adts := parseAllADTs(t, filepath.Join(root, "metaengine", "enum_validation.go"))
	if len(adts) == 0 {
		t.Fatal("parsed zero ADTs from metaengine/enum_validation.go — parser is broken")
	}

	md, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(recipesRelPath)))
	if err != nil {
		t.Fatalf("read recipes.md: %v", err)
	}

	var headings []string
	for line := range strings.SplitSeq(string(md), "\n") {
		if strings.HasPrefix(line, "### ") || strings.HasPrefix(line, "#### ") {
			headings = append(headings, line)
		}
	}

	for _, adt := range adts {
		pattern, ok := adtHeadingPatterns[adt]
		if !ok {
			t.Errorf(
				"ADT %s exists in metaengine.AllADTs() but has no heading pattern here — add coverage or a waiver",
				adt,
			)

			continue
		}

		if reason, waived := adtCoverageWaivers[adt]; waived {
			if strings.TrimSpace(reason) == "" {
				t.Errorf(
					"ADT %s waiver has an EMPTY reason — unexplained gaps are not allowed",
					adt,
				)
			}
			if matchesHeading(headings, pattern) {
				t.Errorf(
					"ADT %s is waived but a recipes.md heading matches it — drop the waiver",
					adt,
				)
			}
			t.Logf("WAIVED %s: %s", adt, reason)

			continue
		}

		if !matchesHeading(headings, pattern) {
			t.Errorf(
				"ADT %s has NO recipes.md heading matching %q — write a recipe or waive with a reason",
				adt,
				pattern,
			)
		}
	}

	for adt := range adtHeadingPatterns {
		if !strings.Contains(strings.Join(adts, "\n"), adt) {
			t.Errorf(
				"heading pattern registered for %s but that ADT no longer exists in AllADTs() — remove the entry",
				adt,
			)
		}
	}
}

func matchesHeading(headings []string, pattern *regexp.Regexp) bool {
	return slices.ContainsFunc(headings, pattern.MatchString)
}

// parseAllADTs extracts the ADT identifier list from the AllADTs() function
// body so the ratchet tracks the real enum without a module dependency.
func parseAllADTs(t *testing.T, path string) []string {
	t.Helper()

	src, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read enum_validation.go: %v", err)
	}

	body := regexp.MustCompile(`(?s)func AllADTs\(\) \[\]ADT \{(.*?)\n\}`).
		FindStringSubmatch(string(src))
	if body == nil {
		t.Fatalf("AllADTs() not found in %s — did it move? update the parser path", path)
	}

	ident := regexp.MustCompile(`\bADT\w+`)
	found := ident.FindAllString(body[1], -1)
	if len(found) == 0 {
		t.Fatalf("AllADTs() body parsed empty from %s", path)
	}

	return found
}
