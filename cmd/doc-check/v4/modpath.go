package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// modPathToken matches a full first-party module-path citation anywhere in a
// doc line (fences, tables, prose): the import prefix plus at least two
// segments. Trailing punctuation (backtick, comma, period, parenthesis)
// cannot match the charset, so citations need no trimming.
var modPathToken = regexp.MustCompile(
	`github\.com/larsartmann/go-cqrs-lite/[a-zA-Z0-9_-]+(?:/[a-zA-Z0-9_-]+)+`,
)

// deletionMarker matches the historical-citation vocabulary on the same line
// as a dead path. Docs deliberately cite deleted modules (registry rows,
// migration notes); those citations must SAY so on the same line.
var deletionMarker = regexp.MustCompile(`(?i)delet|remov|gone|retir|supersed|no longer`)

// checkModulePaths verifies that every full first-party module-path citation
// in the corpus maps to a module that still exists at HEAD. A citation of a
// module with no go.mod anywhere along its path segments is stale guidance —
// it teaches a consumer to import a tag-only relic whose source is gone —
// unless the same line marks it as historical deletion vocabulary.
//
// Fence imports already fail through the resolver's no-exports warm warning;
// this pass covers the rest: prose, tables, and inline code spans.
func checkModulePaths(files []string, repoRoot string) []navIssue {
	var out []navIssue

	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			continue // unreadable files fail loudly in the block scanner
		}

		rel, relErr := filepath.Rel(repoRoot, file)
		if relErr != nil {
			rel = file
		}

		for i, line := range strings.Split(string(data), "\n") {
			for _, path := range modPathToken.FindAllString(line, -1) {
				if moduleExistsAtHEAD(repoRoot, strings.TrimPrefix(path, repoImportPrefix)) {
					continue
				}

				if deletionMarker.MatchString(line) {
					continue
				}

				out = append(out, navIssue{
					File: rel,
					Line: i + 1,
					Msg: fmt.Sprintf(
						"citation of deleted module %s has no go.mod at HEAD and no same-line "+
							"deletion marker (deleted/removed/gone/retired/superseded/no longer) — "+
							"update the guidance to a live module or mark the citation as historical",
						path),
				})
			}
		}
	}

	return out
}

// moduleExistsAtHEAD reports whether rel — a module path, possibly carrying a
// major-version suffix or a sub-package tail — belongs to a live module: some
// prefix of its segments holds a go.mod at HEAD, either directly (pre-reorg
// rel/go.mod layout) or under a major-version directory (rel/v4/go.mod, the
// post-reorg layout). A module deleted at HEAD matches no prefix.
func moduleExistsAtHEAD(repoRoot, rel string) bool {
	segs := strings.Split(rel, "/")

	for i := len(segs); i >= 1; i-- {
		cand := filepath.Join(append([]string{repoRoot}, segs[:i]...)...)

		if _, err := os.Stat(filepath.Join(cand, "go.mod")); err == nil {
			return true
		}

		if matches, _ := filepath.Glob(filepath.Join(cand, "v[0-9]*", "go.mod")); len(matches) > 0 {
			return true
		}
	}

	return false
}
