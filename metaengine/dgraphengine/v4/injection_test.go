package dgraphengine

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestNoDQLInjectionPatterns is a regression test that scans all non-test .go
// files in the dgraphengine package and asserts:
//
//  1. No dqlString function exists (the deleted hand-rolled escaper).
//  2. No fmt.Sprintf call interpolates the "cqrs." predicate prefix (which
//     would indicate query construction from untrusted input instead of
//     parameterized QueryWithVars).
//
// This prevents re-introduction of the DQL injection vulnerability fixed in
// the 2026-08-08 security fix (see docs/status/2026-08-08_21-33_metaengine-v2-gap-closure-dql-injection-fix.md).
func TestNoDQLInjectionPatterns(t *testing.T) {
	t.Parallel()

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd: %v", err)
	}

	files, err := filepath.Glob(filepath.Join(cwd, "*.go"))
	if err != nil {
		t.Fatalf("filepath.Glob: %v", err)
	}

	for _, file := range files {
		// Skip test files — they may reference dqlString in comments about
		// the fix, or use fmt.Sprintf for test data construction.
		if strings.HasSuffix(file, "_test.go") {
			continue
		}

		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("os.ReadFile(%s): %v", file, err)
		}

		content := string(data)

		// Assert no dqlString function definition or call.
		if strings.Contains(content, "dqlString") {
			t.Errorf("%s: contains 'dqlString' — the hand-rolled DQL escaper "+
				"was deleted as part of the injection fix. Use QueryWithVars "+
				"with $variable placeholders instead.", filepath.Base(file))
		}

		// Assert no fmt.Sprintf with "cqrs." prefix interpolation.
		// This catches patterns like:
		//   fmt.Sprintf("...cqrs.%s...", userInput)
		// which would allow DQL injection via predicate name manipulation.
		lines := strings.Split(content, "\n")

		for i, line := range lines {
			trimmed := strings.TrimSpace(line)

			// Skip comments.
			if strings.HasPrefix(trimmed, "//") {
				continue
			}

			// Check for fmt.Sprintf containing "cqrs." — this is only safe
			// when building static queries with no user input interpolation.
			// The pattern "fmt.Sprintf" + "cqrs." on the same line is
			// suspicious and warrants manual review.
			if strings.Contains(trimmed, "fmt.Sprintf") &&
				strings.Contains(trimmed, "cqrs.") &&
				strings.Contains(trimmed, "%") {
				// Allow sanitizePredicate / graphEdgePredicate callers —
				// these produce safe predicate names from sanitized input.
				if strings.Contains(trimmed, "sanitizePredicate") ||
					strings.Contains(trimmed, "graphEdgePredicate") ||
					strings.Contains(trimmed, "firstClause") ||
					strings.Contains(trimmed, "pred :=") {
					continue
				}

				t.Errorf("%s:%d: fmt.Sprintf with 'cqrs.' and '%%' on same "+
					"line — potential DQL injection. Use QueryWithVars with "+
					"$variable placeholders.\n  Line: %s",
					filepath.Base(file), i+1, trimmed)
			}
		}
	}
}
