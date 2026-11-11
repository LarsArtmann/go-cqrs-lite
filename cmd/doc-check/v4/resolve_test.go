package main

import (
	"os"
	"path/filepath"
	"testing"
)

func writePkg(t *testing.T, dir, clause, decls string) {
	t.Helper()

	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", dir, err)
	}

	src := "package " + clause + "\n\n" + decls + "\n"

	if err := os.WriteFile(filepath.Join(dir, "store.go"), []byte(src), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
}

// newTestRepo builds a repo root with two repo packages that share the
// package name sqlstore but export disjoint symbols.
func newTestRepo(t *testing.T) string {
	t.Helper()

	root := t.TempDir()

	writePkg(t, filepath.Join(root, "alpha", "sqlstore"), "sqlstore",
		"type AlphaStore struct{}\n\nfunc NewAlpha() *AlphaStore { return &AlphaStore{} }")
	writePkg(t, filepath.Join(root, "beta", "sqlstore"), "sqlstore",
		"type BetaStore struct{}")

	return root
}

func TestResolve_BlockScopedImportsDoNotCrossResolve(t *testing.T) {
	t.Parallel()

	root := newTestRepo(t)

	res := newResolver(root)

	alpha := repoImportPrefix + "alpha/sqlstore/v4"

	res.warm(alpha)

	b := block{imports: []string{alpha}}

	if !res.resolve(b, ref{pkg: "sqlstore", symbol: "NewAlpha"}) {
		t.Error("NewAlpha should resolve against the block's own import")
	}

	if res.resolve(b, ref{pkg: "sqlstore", symbol: "BetaStore"}) {
		t.Error("BetaStore must NOT resolve via another same-named package")
	}
}

func TestResolve_AliasFallbackUnionsSameNamedPackages(t *testing.T) {
	t.Parallel()

	res := newResolver(newTestRepo(t))

	b := block{}

	if !res.resolve(b, ref{pkg: "sqlstore", symbol: "BetaStore"}) {
		t.Error("BetaStore should resolve via the repo alias union (no visible import)")
	}

	if res.resolve(b, ref{pkg: "sqlstore", symbol: "NoSuchSymbol"}) {
		t.Error("unknown symbol in a known repo package must be broken")
	}
}

func TestResolve_UnknownAliasIsSkippedAsExternal(t *testing.T) {
	t.Parallel()

	res := newResolver(newTestRepo(t))

	b := block{}

	if !res.resolve(b, ref{pkg: "watermillgochannel", symbol: "Whatever"}) {
		t.Error("aliases matching no repo package are external and must be skipped")
	}
}

func TestParsePackageExports_ReturnsClause(t *testing.T) {
	t.Parallel()

	root := t.TempDir()

	writePkg(t, filepath.Join(root, "weirdname"), "actual", "func Exported() {}")

	exports, clause, warns := parsePackageExports(filepath.Join(root, "weirdname"))
	if clause != "actual" {
		t.Errorf("clause = %q, want %q", clause, "actual")
	}

	if !exports["Exported"] {
		t.Error("Exported missing from exports")
	}

	if len(warns) != 0 {
		t.Errorf("unexpected warnings: %v", warns)
	}
}

func TestImportDir_KeepsVersionSuffixInDir(t *testing.T) {
	t.Parallel()

	cases := []struct{ in, want string }{
		{repoImportPrefix + "event/v4", "event/v4"},
		{repoImportPrefix + "scheduling/sqlstore/v4", "scheduling/sqlstore/v4"},
		{repoImportPrefix + "core/v5/event", "core/v5/event"},
		{repoImportPrefix + "event/v4/eventtest", "event/v4/eventtest"},
		{repoImportPrefix + "v3", "."},
	}

	for _, tc := range cases {
		if got := importDir(tc.in); got != tc.want {
			t.Errorf("importDir(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// TestFullDir_ResolvesLiveTreeLayout is the live-tree regression pin for the
// 2026-11-10 module-dir move (0e4fbb3d2): modules live at event/v4/, and the
// resolver must find their exports there. The pre-move importDir trimmed the
// /v4 suffix and pointed at versionless directories that no longer hold Go
// files — every doc reference then failed as "not found" while the
// string-pure importDir test kept passing.
func TestFullDir_ResolvesLiveTreeLayout(t *testing.T) {
	t.Parallel()

	repo := findRepoRootFromPath(".")
	if repo == "" {
		t.Fatal("could not locate the repo root from the test's working directory")
	}

	res := newResolver(repo)

	for _, imp := range []string{
		repoImportPrefix + "event/v4",
		repoImportPrefix + "system/v4",
		repoImportPrefix + "metaengine/sqliteengine/v4",
	} {
		exports, clause, warns := parsePackageExports(res.fullDir(imp))
		if len(exports) == 0 {
			t.Errorf("%s: no exports resolved under %s (warnings: %v)", imp, res.fullDir(imp), warns)
		}

		if got := filepath.Base(res.fullDir(imp)); got == "v4" && clause == "" {
			t.Errorf("%s: resolved to a version directory with no package", imp)
		}
	}

	if res.fullDir(repoImportPrefix+"event/v4") == "" {
		t.Fatal("fullDir returned empty")
	}

	exports, _, _ := parsePackageExports(res.fullDir(repoImportPrefix + "event/v4"))
	if !exports["New"] {
		t.Error("event.New not found under the live event/v4 tree — the resolver points at the wrong directory")
	}
}

// newMirrorRepo builds a repo root with a v4 package and its core/v5
// copy-forward twin sharing one package name (the ADR-0152 shape).
func newMirrorRepo(t *testing.T) string {
	t.Helper()

	root := t.TempDir()

	writePkg(t, filepath.Join(root, "kv"), "kv", "type Cache struct{}")
	writePkg(t, filepath.Join(root, "core", "v5", "kv"), "kv", "type Cache struct{}")

	return root
}

func TestMirrorAliasDirs(t *testing.T) {
	t.Parallel()

	cases := []struct {
		dirs []string
		want bool
	}{
		{[]string{"kv", "core/v5/kv"}, true},
		{[]string{"kv/v4", "core/v5/kv"}, true},
		{[]string{"core/v5/query", "query/v4"}, true},
		{[]string{"kv", "core/v5/query"}, false},
		{[]string{"alpha/sqlstore", "beta/sqlstore"}, false},
		{[]string{"core/v5/kv", "core/v5/query"}, false},
		{[]string{"kv"}, false},
		{[]string{"kv", "core/v5/kv", "other/kv"}, false},
	}

	for _, tc := range cases {
		if got := mirrorAliasDirs(tc.dirs); got != tc.want {
			t.Errorf("mirrorAliasDirs(%v) = %v, want %v", tc.dirs, got, tc.want)
		}
	}
}

func TestVerifyBlocks_MirrorAmbiguityIsKnownNotUnknown(t *testing.T) {
	t.Parallel()

	res := newResolver(newMirrorRepo(t))

	blocks := []block{{
		file: "doc.md", line: 1,
		refs: []ref{{pkg: "kv", symbol: "Cache", file: "doc.md", line: 3}},
	}}

	_, _, _, ambiguities, unknownAmbiguities := verifyBlocks( //nolint:dogsled // fixture asserts the last two returns only
		blocks,
		nil,
		res,
		false,
	)

	if len(ambiguities) != 1 {
		t.Fatalf("ambiguities = %d, want 1 (mirror pair still reported)", len(ambiguities))
	}

	if len(unknownAmbiguities) != 0 {
		t.Fatalf("unknownAmbiguities = %d, want 0 for a v4↔core/v5 mirror: %v",
			len(unknownAmbiguities), unknownAmbiguities)
	}
}

func TestVerifyBlocks_NonMirrorAmbiguityIsUnknown(t *testing.T) {
	t.Parallel()

	res := newResolver(newTestRepo(t))

	blocks := []block{{
		file: "doc.md", line: 1,
		refs: []ref{{pkg: "sqlstore", symbol: "BetaStore", file: "doc.md", line: 3}},
	}}

	_, _, _, ambiguities, unknownAmbiguities := verifyBlocks( //nolint:dogsled // fixture asserts the last two returns only
		blocks,
		nil,
		res,
		false,
	)

	if len(ambiguities) != 1 || len(unknownAmbiguities) != 1 {
		t.Fatalf("ambiguities = %d, unknownAmbiguities = %d; want 1/1 for a non-mirror pair "+
			"(the fatal class)", len(ambiguities), len(unknownAmbiguities))
	}
}
