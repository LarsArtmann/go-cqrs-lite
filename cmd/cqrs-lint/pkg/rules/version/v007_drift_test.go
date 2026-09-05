package version

import (
	"fmt"
	"sort"
	"strings"
	"testing"
)

// The drift contract: every v5-removal deprecation marker in the repo must be
// represented in the V007 tables (or an explicit allowlist), and every table
// entry must correspond to a live marker. This kills the recurring failure
// class where a new `Deprecated: … v5` marker ships without a V007 entry, or
// a table entry outlives its symbol. Coverage contract details are documented
// on NewV007Detector.

// minExpectedV5Markers guards against the scanner silently breaking and
// returning a tiny inventory (which would make both contract tests vacuous).
const minExpectedV5Markers = 90

func TestV007_TablesCoverAllV5DeprecationMarkers(t *testing.T) {
	markers := scanV5DriftMarkers(t)
	if len(markers) < minExpectedV5Markers {
		t.Fatalf("scanner found only %d v5 markers (expected ≥%d); scanner is broken", len(markers), minExpectedV5Markers)
	}

	var uncovered []string
	for _, m := range markers {
		switch m.kind {
		case "method":
			if _, hit := matchModule(m.frag); hit {
				continue
			}
			if _, hit := matchSymbol(m.frag, m.recv); hit {
				continue // receiver type itself is removed; method dies with it
			}
			if _, ok := v5DriftMethodAllowlist[v5MethodKey{m.frag, m.recv, m.symbol}]; ok {
				continue
			}
			uncovered = append(uncovered, fmt.Sprintf("%s (%s.%s on %s)", m.pos, m.frag, m.symbol, m.recv))
		case "package":
			if _, hit := matchModule(m.frag); hit {
				continue
			}
			if _, ok := v5DriftAllowlist[v5MarkerKey{m.frag, "(package)"}]; ok {
				continue
			}
			uncovered = append(uncovered, fmt.Sprintf("%s (package %s)", m.pos, m.frag))
		default:
			if _, hit := matchModule(m.frag); hit {
				continue
			}
			if _, hit := matchSymbol(m.frag, m.symbol); hit {
				continue
			}
			if _, ok := v5DriftAllowlist[v5MarkerKey{m.frag, m.symbol}]; ok {
				continue
			}
			uncovered = append(uncovered, fmt.Sprintf("%s (%s.%s)", m.pos, m.frag, m.symbol))
		}
	}

	if len(uncovered) > 0 {
		sort.Strings(uncovered)
		t.Fatalf("V007 tables do not cover %d v5 deprecation marker(s).\n"+
			"Add (fragment, symbol) to deprecatedV5Symbols / deprecatedV5Modules in v007_tables.go,\n"+
			"or an explicit allowlist entry with a reason in v007_drift_scan_test.go:\n%s",
			len(uncovered), strings.Join(uncovered, "\n"))
	}
}

func TestV007_TableEntriesHaveLiveMarkers(t *testing.T) {
	markers := scanV5DriftMarkers(t)

	live := make(map[v5MarkerKey]bool, len(markers))
	for _, m := range markers {
		if m.kind == "func" || m.kind == "type" || m.kind == "var" || m.kind == "const" {
			live[v5MarkerKey{m.frag, m.symbol}] = true
		}
	}

	var stale []string
	for _, s := range deprecatedV5Symbols {
		key := v5MarkerKey{s.fragment, s.symbol}
		if live[key] {
			continue
		}
		if _, ok := v5StaleEntryAllowlist[key]; ok {
			continue
		}
		stale = append(stale, fmt.Sprintf("%s.%s (no live v5 marker: symbol deleted, renamed, or unmarked)", s.fragment, s.symbol))
	}

	if len(stale) > 0 {
		sort.Strings(stale)
		t.Fatalf("V007 table holds %d stale entr(y/ies).\n"+
			"Remove them from deprecatedV5Symbols in v007_tables.go, or add a\n"+
			"v5StaleEntryAllowlist reason in v007_drift_scan_test.go:\n%s",
			len(stale), strings.Join(stale, "\n"))
	}
}

func TestV007_FragmentSpaceMatchesDetector(t *testing.T) {
	// The meta-test computes fragments from repo-relative directories; the
	// detector computes them from consumer import paths. Both sides must
	// normalize to the same space, including mid-path /vN segments.
	cases := []struct {
		importPath string
		wantFrag   string
	}{
		{"github.com/larsartmann/go-cqrs-lite/stack/v4", "stack"},
		{"github.com/larsartmann/go-cqrs-lite/stack/sqlite/v4", "stack/sqlite"},
		{"github.com/larsartmann/go-cqrs-lite/storage/v4", "storage"},
		{"github.com/larsartmann/go-cqrs-lite/storage/v4/relational", "storage/relational"},
		{"github.com/larsartmann/go-cqrs-lite/storage/v4/view", "storage/view"},
		{"github.com/larsartmann/go-cqrs-lite/storage/v4/sql", "storage/sql"},
		{"github.com/larsartmann/go-cqrs-lite/event/v4", "event"},
	}
	for _, tc := range cases {
		got, ok := cqrsModuleOf(tc.importPath)
		if !ok || got != tc.wantFrag {
			t.Errorf("cqrsModuleOf(%q) = %q, %v; want %q", tc.importPath, got, ok, tc.wantFrag)
		}
	}
}
