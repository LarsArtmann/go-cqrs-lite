package main

import (
	"math"
	"reflect"
	"testing"
	"time"
)

func TestSplitNumStat(t *testing.T) {
	cases := []struct {
		line    string
		add     int
		del     int
		file    string
		wantOK  bool
	}{
		{"12\t3\tmetaengine/store.go", 12, 3, "metaengine/store.go", true},
		{"0\t5\tcore/event/errors.go", 0, 5, "core/event/errors.go", true},
		{"-\t-\tbinary.png", 0, 0, "", false},
		{"notanumstat", 0, 0, "", false},
	}
	for _, c := range cases {
		add, del, file, ok := splitNumStat(c.line)
		if ok != c.wantOK || (ok && (add != c.add || del != c.del || file != c.file)) {
			t.Errorf("splitNumStat(%q) = (%d,%d,%q,%v), want (%d,%d,%q,%v)",
				c.line, add, del, file, ok, c.add, c.del, c.file, c.wantOK)
		}
	}
}

func TestNormalizeRename(t *testing.T) {
	cases := map[string]string{
		"metaengine/store.go":                     "metaengine/store.go",
		"core/event/errors.go":                    "core/event/errors.go",
		"old.go=>new.go":                          "new.go",
		"pkg/old=>pkg/new":                        "pkg/new",
		"pkg/{old => new}/file.go":                "pkg/new/file.go",
		"pkg/{errors => errors_v2}.go":            "pkg/errors_v2.go",
	}
	for in, want := range cases {
		if got := normalizeRename(in); got != want {
			t.Errorf("normalizeRename(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestParseCommitMarker(t *testing.T) {
	author, date := parseCommitMarker("@@@abc123|2026-08-09T23:45:10+02:00|Lars Artmann")
	if author != "Lars Artmann" {
		t.Errorf("author = %q", author)
	}
	want := time.Date(2026, 8, 9, 23, 45, 10, 0, time.FixedZone("+0200", 2*3600))
	if !date.Equal(want) {
		t.Errorf("date = %v, want %v", date, want)
	}
}

func TestApplyNumStat(t *testing.T) {
	stats := make(map[string]*FileStat)
	date := time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)
	now := time.Date(2026, 8, 9, 0, 0, 0, 0, time.UTC)

	applyNumStat("10\t2\tmetaengine/store.go", stats, "Alice", date, 0, now)
	applyNumStat("5\t1\tmetaengine/store.go", stats, "Bob", date.Add(48*time.Hour), 0, now)
	applyNumStat("3\t0\tcmd/churn/main.go", stats, "Alice", date, 0, now)

	s := stats["metaengine/store.go"]
	if s == nil {
		t.Fatal("missing stat")
	}
	if s.Commits != 2 || s.Added != 15 || s.Deleted != 3 {
		t.Errorf("got commits=%d added=%d del=%d", s.Commits, s.Added, s.Deleted)
	}
	if s.AuthorCount() != 2 {
		t.Errorf("authors = %d, want 2", s.AuthorCount())
	}
	if s.Churn() != 18 {
		t.Errorf("churn = %d, want 18", s.Churn())
	}
}

func TestWeightedRecency(t *testing.T) {
	now := time.Date(2026, 8, 9, 0, 0, 0, 0, time.UTC)
	old := now.AddDate(0, -6, 0) // ~6 months ago

	// No half-life: raw value passes through.
	if got := weighted(100, old, 0, now); got != 100 {
		t.Errorf("disabled weighting = %v, want 100", got)
	}
	// 180-day half-life: ~6 months (183 days) ago ≈ half.
	got := weighted(100, old, 180, now)
	if math.Abs(got-50) > 2 {
		t.Errorf("half-life weighting = %.1f, want ~50", got)
	}
	// Future commit (clock skew) clamps to full weight.
	future := now.Add(24 * time.Hour)
	if got := weighted(100, future, 180, now); got != 100 {
		t.Errorf("future weighting = %v, want 100", got)
	}
}

func TestAggregateByDir(t *testing.T) {
	stats := map[string]*FileStat{
		"metaengine/store.go":          newStat("metaengine/store.go", 5, 10, 2, 200),
		"metaengine/engine.go":         newStat("metaengine/engine.go", 3, 8, 1, 150),
		"metaengine/sub/deep.go":       newStat("metaengine/sub/deep.go", 1, 1, 1, 40),
		"cmd/churn/main.go":            newStat("cmd/churn/main.go", 2, 4, 1, 100),
	}
	rows := aggregate(stats, aggDir)
	byDir := make(map[string]*FileStat, len(rows))
	for _, r := range rows {
		byDir[r.Path] = r
	}
	mg := byDir["metaengine"]
	if mg == nil {
		t.Fatal("missing metaengine dir")
	}
	if mg.Commits != 8 || mg.Churn() != 19 || mg.Complexity != 350 {
		t.Errorf("metaengine dir = commits %d churn %d sloc %d", mg.Commits, mg.Churn(), mg.Complexity)
	}
}

func TestAggregateByFilePassthrough(t *testing.T) {
	stats := map[string]*FileStat{
		"a.go": newStat("a.go", 1, 1, 1, 10),
		"b.go": newStat("b.go", 2, 2, 1, 20),
	}
	rows := aggregate(stats, aggFile)
	if len(rows) != 2 {
		t.Fatalf("got %d rows, want 2", len(rows))
	}
}

func TestKeepFilter(t *testing.T) {
	fo := filterOpts{exts: []string{".go"}, includeTests: false, includeGenerated: false}
	cases := map[string]bool{
		"metaengine/store.go":      true,
		"core/errors_test.go":      false,
		"vendor/x/y.go":            false,
		"cmd/gen.go":               false,
		"cmd/foo.pb.go":            false,
		"readme.md":                false,
		"nested/file.go":           true,
	}
	for path, want := range cases {
		if got := keep(path, fo); got != want {
			t.Errorf("keep(%q) = %v, want %v", path, got, want)
		}
	}
}

func TestSortStatsByHotspot(t *testing.T) {
	rows := []*FileStat{
		newStat("small.go", 1, 10, 1, 5),   // churn 10 * sloc 5 = 50
		newStat("big.go", 1, 100, 1, 1000), // churn 100 * sloc 1000 = 100000
		newStat("mid.go", 1, 50, 1, 20),    // churn 50 * sloc 20 = 1000
	}
	sortStats(rows)
	wantPaths := []string{"big.go", "mid.go", "small.go"}
	got := make([]string, len(rows))
	for i, r := range rows {
		got[i] = r.Path
	}
	if !reflect.DeepEqual(got, wantPaths) {
		t.Errorf("order = %v, want %v", got, wantPaths)
	}
}

func TestModuleKeyAndDirKey(t *testing.T) {
	if got := dirKey("metaengine/pebbleengine/engine.go"); got != "metaengine/pebbleengine" {
		t.Errorf("dirKey = %q", got)
	}
	if got := dirKey("main.go"); got != "." {
		t.Errorf("dirKey = %q", got)
	}
	roots := map[string]bool{"metaengine": true, "cmd/churn": true}
	if got := moduleKey("metaengine/pebbleengine/engine.go", roots); got != "metaengine" {
		t.Errorf("moduleKey = %q, want metaengine", got)
	}
	if got := moduleKey("cmd/churn/main.go", roots); got != "cmd/churn" {
		t.Errorf("moduleKey = %q, want cmd/churn", got)
	}
}

// newStat builds a FileStat with the given metrics for test brevity.
func newStat(path string, commits, churn, authors, sloc int) *FileStat {
	s := &FileStat{Path: path, Commits: commits, Added: churn, Authors: make(map[string]struct{}), Complexity: sloc}
	s.Weight = float64(churn)
	for i := 0; i < authors; i++ {
		s.Authors[string(rune('A'+i))] = struct{}{}
	}
	return s
}
