package main

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"testing"
)

// mirrorPairs enumerates the ADR-0152 dual-support mirrors: a v4 module whose
// surface is deliberately forked into the single core/v5 module. Drift between
// the two sides is the actual failure mode of dual-support: v4 keeps tagging
// fixes while v5 evolves, and nothing else notices when the copies diverge
// silently. The registry check below also forces NEW core/v5 packages to be
// registered here the moment they appear in the golden.
var mirrorPairs = []struct {
	v4Module string // golden prefix on the v4 side (module dir)
}{
	{"command"},
	{"dedup"},
	{"dispatcher"},
	{"event"},
	{"id"},
	{"kv"},
	{"metadata"},
	{"query"},
	{"record"},
}

const coreV5Prefix = "core/v5"

// mirrorAllowlistRow classifies ONE deliberate divergence between a v4 module
// and its core/v5 mirror. side=v4 rows are transition lag (or never-mirror
// verdicts) for symbols present only on the v4 side; side=v5 rows document
// deliberate v5 evolution. A trailing "/..." on symbol covers a whole
// subpackage subtree. Every row carries a reason — the file is the audit
// trail ADR-0152 asks for, and the ratchet: rows must be pruned (by hand)
// once the sides reconverge, which the stale check below enforces.
type mirrorAllowlistRow struct {
	module string
	side   string
	symbol string
	reason string
}

// TestMirrorLockstep fails on any UNCLASSIFIED divergence between a v4 module
// and its core/v5 mirror (either direction), and on stale register rows.
//
// Policy (resolves the M4 gate-policy question, plan doc
// docs/planning/2026-10-09_18-16_SUPERB-branching-flow-triage.md):
// hard gate WITH a classified escape hatch. Deliberate divergence is fine —
// but it must be a reviewed row with a reason, never silence. Known
// limitation: the golden tracks kind+name ("func Foo", "method Bar"), not
// signatures; same-name parameter drift is invisible here and needs the
// compile-time cross-import pattern (event.Type lockstep tests) once v5
// stabilizes.
func TestMirrorLockstep(t *testing.T) {
	t.Parallel()

	goldenPath := filepath.Join(repoRoot(t), "docs", "api_surface.txt")

	data, err := os.ReadFile(goldenPath)
	if os.IsNotExist(err) {
		t.Skip("golden file does not exist; run with --update first")
	}
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}

	allowlistPath := filepath.Join(".", "testdata", "mirror_lockstep_allowlist.txt")
	allowData, err := os.ReadFile(allowlistPath)
	if err != nil {
		t.Fatalf("read mirror allowlist (create it next to this test): %v", err)
	}

	rows, parseProblems := parseMirrorAllowlist(string(allowData))
	problems, summary := checkMirrorLockstep(strings.Split(string(data), "\n"), rows)

	for _, p := range parseProblems {
		t.Errorf("allowlist: %s", p)
	}

	for _, s := range summary {
		t.Log(s)
	}

	sorted := slices.Clone(problems)
	sort.Strings(sorted)

	const maxReported = 30

	for i, p := range sorted {
		if i >= maxReported {
			t.Errorf("... and %d more mirror drift problems", len(sorted)-maxReported)

			break
		}
		t.Errorf("%s", p)
	}
}

// TestMirrorLockstepBites proves the detector actually fails on both failure
// classes (unclassified drift, stale register rows) plus registry violations.
// A drift gate that cannot be seen failing is not a gate.
func TestMirrorLockstepBites(t *testing.T) {
	t.Parallel()

	// Parser bites: structural violations are reported and dropped so the
	// checker never sees a row it cannot classify.
	badAllowlist := "" +
		"nosuch v4 func X reason\n" +
		"kv v9 func X reason\n" +
		"id v4 func Dup reason\n" +
		"id v4 func Dup duplicate\n" +
		"id v4 func NoReason\n" +
		"id v4 func OK valid row\n"

	rows, parseProblems := parseMirrorAllowlist(badAllowlist)

	parsed := strings.Join(parseProblems, "\n")

	for _, mustMention := range []string{"nosuch", "v9", "duplicate", "<reason>"} {
		if !strings.Contains(parsed, mustMention) {
			t.Errorf("parser output missing %q; problems were:\n%s", mustMention, parsed)
		}
	}

	if len(rows) != 2 || rows[0].symbol != "func Dup" || rows[1].symbol != "func OK" {
		t.Errorf("invalid rows must be dropped, valid kept; got %+v", rows)
	}

	// Checker bites: both drift directions, stale rows, registry violations.
	golden := []string{
		"id/func Shared",                  // shared symbol: fine
		coreV5Prefix + "/id/func Shared",  //
		"id/func Lagging",                 // v4-side drift, unclassified
		coreV5Prefix + "/id/func Evolved", // v5-side drift, unclassified
		coreV5Prefix + "/newpkg/func New", // package missing from mirrorPairs
		"event/v4/eventtest/func Fixture", // subtree drift, unclassified
	}
	rows = []mirrorAllowlistRow{
		{module: "id", side: "v4", symbol: "func Gone", reason: "covers nothing -> stale"},
	}

	problems, _ := checkMirrorLockstep(golden, rows)

	joined := strings.Join(problems, "\n")
	for _, mustMention := range []string{
		"func Lagging",
		"func Evolved",
		"func Gone",     // stale row
		"newpkg",        // registry violation
		"v4/eventtest/", // subtree drift reported with its relative path
	} {
		if !strings.Contains(joined, mustMention) {
			t.Errorf("detector output missing %q; problems were:\n%s", mustMention, joined)
		}
	}

	// Positive control: a clean mirror with one exact row and one subtree row
	// stays silent.
	cleanGolden := []string{
		"kv/func Shared",
		coreV5Prefix + "/kv/func Shared",
		"kv/func Lagging",
		"event/v4/eventtest/func Fixture",
	}
	cleanRows := []mirrorAllowlistRow{
		{module: "kv", side: "v4", symbol: "func Lagging", reason: "classified lag"},
		{module: "event", side: "v4", symbol: "v4/eventtest/...", reason: "classified subtree"},
	}
	if problems, _ := checkMirrorLockstep(cleanGolden, cleanRows); len(problems) != 0 {
		t.Errorf("clean input produced problems: %v", problems)
	}
}

// checkMirrorLockstep is the pure core: golden lines + register rows in,
// problems + per-pair summary out. Same-package sets are deduplicated
// defensively (the golden generator compacts, but a regression there must
// not corrupt drift classification).
func checkMirrorLockstep(goldenLines []string, rows []mirrorAllowlistRow) ([]string, []string) {
	v4Sets := make(map[string]map[string]struct{})
	v5Sets := make(map[string]map[string]struct{})
	seen := make(map[string]struct{})

	for _, pair := range mirrorPairs {
		v4Sets[pair.v4Module] = make(map[string]struct{})
		v5Sets[pair.v4Module] = make(map[string]struct{})
	}

	v5Packages := make(map[string]struct{})

	for _, raw := range goldenLines {
		line := strings.TrimRight(raw, " \t\r")
		if line == "" {
			continue
		}

		if rest, ok := strings.CutPrefix(line, coreV5Prefix+"/"); ok {
			pkg, _, found := strings.Cut(rest, "/")
			if found {
				if _, known := v5Sets[pkg]; !known {
					v5Packages[pkg] = struct{}{}
				}
			}

			continue
		}

		for _, pair := range mirrorPairs {
			if rel, ok := strings.CutPrefix(line, pair.v4Module+"/"); ok {
				if _, dup := seen[line]; !dup {
					seen[line] = struct{}{}
					v4Sets[pair.v4Module][rel] = struct{}{}
				}
			}
		}
	}

	var problems, summary []string

	for pkg := range v5Packages {
		problems = append(problems, fmt.Sprintf(
			"core/v5/%s appears in the golden but is not in mirrorPairs — register the pair (or extend the registry policy) so its drift is governed",
			pkg,
		))
	}

	for _, pair := range mirrorPairs {
		name := pair.v4Module
		v5Prefix := coreV5Prefix + "/" + name

		for _, raw := range goldenLines {
			line := strings.TrimRight(raw, " \t\r")
			if rel, ok := strings.CutPrefix(line, v5Prefix+"/"); ok {
				v5Sets[name][rel] = struct{}{}
			}
		}

		if len(v4Sets[name]) == 0 && len(v5Sets[name]) > 0 {
			problems = append(problems, fmt.Sprintf(
				"mirror %s: v4 side has no golden entries but core/v5/%s does — v4 module deleted or renamed? update mirrorPairs for the v5 cut",
				name,
				name,
			))

			continue
		}

		problems = append(problems, classifyMirrorDrift(name, v4Sets[name], v5Sets[name], rows)...)

		lag, evolved := 0, 0

		for rel := range v4Sets[name] {
			if _, shared := v5Sets[name][rel]; !shared {
				lag++
			}
		}

		for rel := range v5Sets[name] {
			if _, shared := v4Sets[name][rel]; !shared {
				evolved++
			}
		}

		summary = append(summary, fmt.Sprintf(
			"mirror %s: shared=%d lag(v4-only)=%d evolved(v5-only)=%d",
			name, len(v4Sets[name])-lag, lag, evolved,
		))
	}

	problems = append(problems, staleMirrorRows(rows, v4Sets, v5Sets)...)

	return problems, summary
}

// classifyMirrorDrift fails on any relative symbol present on only one side
// of a mirror pair unless a register row classifies it.
func classifyMirrorDrift(
	name string,
	v4, v5 map[string]struct{},
	rows []mirrorAllowlistRow,
) []string {
	var problems []string

	for _, rel := range sortedSet(v4) {
		if _, shared := v5[rel]; shared {
			continue
		}

		if !mirrorRowCovers(rows, name, "v4", rel) {
			problems = append(problems, fmt.Sprintf(
				"mirror %s: unclassified v4-side drift %q — mirror it into core/v5/%s or add a register row with a reason to testdata/mirror_lockstep_allowlist.txt",
				name,
				rel,
				name,
			))
		}
	}

	for _, rel := range sortedSet(v5) {
		if _, shared := v4[rel]; shared {
			continue
		}

		if !mirrorRowCovers(rows, name, "v5", rel) {
			problems = append(problems, fmt.Sprintf(
				"mirror %s: unclassified v5-side drift %q — deliberate v5 evolution must be a register row with a reason in testdata/mirror_lockstep_allowlist.txt",
				name,
				rel,
			))
		}
	}

	return problems
}

func mirrorRowCovers(rows []mirrorAllowlistRow, module, side, rel string) bool {
	for _, row := range rows {
		if row.module != module || row.side != side {
			continue
		}

		if subtree, ok := strings.CutSuffix(row.symbol, "/..."); ok {
			if strings.HasPrefix(rel, subtree+"/") || rel == subtree {
				return true
			}

			continue
		}

		if row.symbol == rel {
			return true
		}
	}

	return false
}

// staleMirrorRows fails on register rows that no longer cover any divergence.
// This is the ratchet: when the sides reconverge (v5 catches up, or v4 dies),
// the row must be pruned so the register never rots into noise.
func staleMirrorRows(rows []mirrorAllowlistRow, v4, v5 map[string]map[string]struct{}) []string {
	var problems []string

	for _, row := range rows {
		set := v4[row.module]
		if row.side == "v5" {
			set = v5[row.module]
		}
		if set == nil {
			continue // unknown module already reported by parse validation
		}

		covers := false

		for rel := range set {
			other := v4[row.module]
			if row.side == "v4" {
				other = v5[row.module]
			}

			if _, shared := other[rel]; shared {
				continue
			}

			if mirrorRowCovers([]mirrorAllowlistRow{row}, row.module, row.side, rel) {
				covers = true

				break
			}
		}

		if !covers {
			problems = append(problems, fmt.Sprintf(
				"stale mirror register row %s %s %s — sides reconverged; prune it",
				row.module, row.side, row.symbol,
			))
		}
	}

	return problems
}

func sortedSet(set map[string]struct{}) []string {
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	sort.Strings(out)

	return out
}

// parseMirrorAllowlist parses the register file: one row per line as
// "<module> <side> <symbol> <reason...>", '#' comments and blank lines
// ignored. A symbol is either a subtree path containing '/' (single token,
// e.g. "v4/eventtest/...") or a golden entry "<kind> <Name>" (two tokens,
// e.g. "func MarkInDelivery"). Structural problems (unknown module/side,
// duplicate or reason-less rows) are reported and the offending row is
// DROPPED so the checker never classifies with a row it cannot trust.
func parseMirrorAllowlist(content string) ([]mirrorAllowlistRow, []string) {
	known := make(map[string]struct{}, len(mirrorPairs))
	for _, pair := range mirrorPairs {
		known[pair.v4Module] = struct{}{}
	}

	var rows []mirrorAllowlistRow
	var problems []string
	seen := make(map[string]struct{})

	for lineNo, raw := range strings.Split(content, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		fields := strings.Fields(line)
		where := fmt.Sprintf("line %d", lineNo+1)

		if len(fields) < 4 {
			problems = append(problems, where+": want '<module> <side> <symbol> <reason>'")

			continue
		}

		module, side := fields[0], fields[1]

		symbol, reason := fields[2], fields[3]
		if !strings.Contains(symbol, "/") {
			// Golden entry symbols are "<kind> <Name>" — two tokens.
			if len(fields) < 5 {
				problems = append(problems, where+": want '<module> <side> <kind> <Name> <reason>'")

				continue
			}
			symbol, reason = fields[2]+" "+fields[3], fields[4]
		}

		if _, isKnown := known[module]; !isKnown {
			problems = append(problems, fmt.Sprintf("%s: unknown mirror module %q", where, module))

			continue
		}

		if side != "v4" && side != "v5" {
			problems = append(
				problems,
				fmt.Sprintf("%s: side must be v4 or v5, got %q", where, side),
			)

			continue
		}

		key := module + " " + side + " " + symbol
		if _, dup := seen[key]; dup {
			problems = append(problems, fmt.Sprintf("%s: duplicate register row %q", where, key))

			continue
		}

		seen[key] = struct{}{}
		rows = append(
			rows,
			mirrorAllowlistRow{module: module, side: side, symbol: symbol, reason: reason},
		)
	}

	return rows, problems
}
