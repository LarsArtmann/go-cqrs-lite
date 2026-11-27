package main

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
)

var (
	// lintMarkerCitation matches a marker a lint rule cites, such as
	// //cqrs:command.
	lintMarkerCitation = regexp.MustCompile(`//cqrs:([a-z]+)\b`)
	// lintFlagCitation matches the flags a lint rule hands to the generator,
	// such as "cqrs-gen --catalog".
	lintFlagCitation = regexp.MustCompile(`cqrs-gen((?:\s+--?[a-z][a-z-]*)+)`)
	// joinedLiterals bridges a string literal split over a "+" continuation, so
	// a citation split across two literals is read as one.
	joinedLiterals = regexp.MustCompile(`"\s*\+\s*"`)
)

// lintCitations returns the marker kinds and the flags that the rule sources
// of cqrs-lint hand to this generator.
func lintCitations(t *testing.T) (markers, flags []string) {
	t.Helper()

	root := filepath.Join("..", "..", "cqrs-lint", "v4", "pkg")
	if _, err := os.Stat(root); err != nil {
		t.Fatalf("cqrs-lint sources not found at %s: %v", root, err)
	}

	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		switch {
		case err != nil:
			return err
		case entry.IsDir(), !strings.HasSuffix(path, ".go"), strings.HasSuffix(path, "_test.go"):
			return nil
		}

		source, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		text := joinedLiterals.ReplaceAllString(string(source), "")

		for _, citation := range lintMarkerCitation.FindAllStringSubmatch(text, -1) {
			markers = append(markers, citation[1])
		}

		for _, citation := range lintFlagCitation.FindAllStringSubmatch(text, -1) {
			flags = append(flags, strings.Fields(citation[1])...)
		}

		return nil
	})
	if err != nil {
		t.Fatalf("read the cqrs-lint sources: %v", err)
	}

	slices.Sort(markers)
	slices.Sort(flags)

	return slices.Compact(markers), slices.Compact(flags)
}

// TestLintSuggestionsNameRealMarkers: every marker kind a cqrs-lint rule cites
// is one this generator acts on. An unknown kind is ignored silently, so the
// proof is that the generator derives a file from it. B004 and B009 used to
// send readers to codegen that did not exist.
func TestLintSuggestionsNameRealMarkers(t *testing.T) {
	t.Parallel()

	markers, _ := lintCitations(t)
	if len(markers) == 0 {
		t.Fatal("no //cqrs: citation found under cqrs-lint; the scan no longer reads the rule sources")
	}

	for _, marker := range markers {
		t.Run(marker, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()
			writeFixture(t, dir, "cited.go",
				"package cited\n\n//cqrs:"+marker+" cited.sample\ntype Sample struct {\n\tName string\n}\n")

			if err := runCLI(t, buildTestCLI(t), dir); err != nil {
				t.Fatalf("generate from a //cqrs:%s marker: %v", marker, err)
			}

			if _, err := os.Stat(filepath.Join(dir, "cqrs_gen.go")); err != nil {
				t.Errorf("//cqrs:%s is cited by a lint rule, but the generator derives nothing from it", marker)
			}
		})
	}
}

// TestLintSuggestionsNameRealFlags: every flag a cqrs-lint rule hands to
// cqrs-gen is one the generator defines.
func TestLintSuggestionsNameRealFlags(t *testing.T) {
	t.Parallel()

	_, cited := lintCitations(t)

	defined := []string{}
	for _, flag := range buildTestCLI(t).Describe().Flags {
		defined = append(defined, "--"+flag.Name)
	}

	for _, flag := range cited {
		if !slices.Contains(defined, flag) {
			t.Errorf("%s is handed to cqrs-gen by a lint rule; the generator defines %v", flag, defined)
		}
	}
}
