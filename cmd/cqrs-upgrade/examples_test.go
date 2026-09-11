package main

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// exampleModules are the four example apps; they double as the canonical
// v5-clean consumers. This meta-test mechanizes the recurring
// "audit examples for v5-removed API" TODO: every example must scan clean
// with the strict V007 detector, and a failed scan fails the test too (a
// false-green is exactly what the manual audit used to catch, late).
//
// The scan runs against a THROWAWAY CONSUMER COPY of each example: the
// in-repo module path (github.com/larsartmann/go-cqrs-lite/example/...) is
// prefix-classified as "library self-lint" by the analyzer, and V007
// silently skips self-lint targets — scanning the originals in place always
// reports clean, violation or not (the 2026-09-11 probe that proved this
// produced a false green). Rewriting the module line to a consumer path
// (the documented cqrs-lint consumer-probe pattern) makes the detector
// actually run.
func TestExamples_AreV5Clean(t *testing.T) {
	// NOT t.Parallel: shells out to go list per module; keep the load
	// pressure sequential with the rest of the suite.
	examples := []string{
		"getting-started",
		"metaengine-quickstart",
		"readme-quickstart",
		"taskmanager",
	}

	for _, name := range examples {
		t.Run(name, func(t *testing.T) {
			src := filepath.Join("..", "..", "example", name)
			dir := consumerCopy(t, src, name)

			findings, err := deprecationFindings(dir)
			if err != nil {
				t.Fatalf("deprecation scan failed (v5-readiness unproven): %v", err)
			}

			if len(findings) > 0 {
				for _, f := range findings {
					t.Errorf("%s %s [%s]", f.Position, f.Message, f.Rule)
				}

				t.Errorf(
					"%s uses %d API(s) removed at go-cqrs-lite v5; migrate the "+
						"example so it stays a valid v5 consumer demo",
					name, len(findings),
				)
			}
		})
	}
}

// consumerCopy copies the Go sources + module files of the example at src
// into a temp dir and rewrites the module line to a consumer path so the
// analyzer classifies it as a consumer project, not the library itself.
// Binaries, databases, and other build artifacts are not needed for a scan
// and are skipped.
func consumerCopy(t *testing.T, src, name string) string {
	t.Helper()

	dst := filepath.Join(t.TempDir(), name)

	err := filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			rel, relErr := filepath.Rel(src, path)
			if relErr != nil {
				return relErr
			}

			return os.MkdirAll(filepath.Join(dst, rel), 0o755)
		}

		base := d.Name()
		if !strings.HasSuffix(base, ".go") &&
			base != "go.mod" && base != "go.sum" {
			return nil
		}

		rel, relErr := filepath.Rel(src, path)
		if relErr != nil {
			return relErr
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		content := string(data)
		if base == "go.mod" {
			content = strings.Replace(content,
				"module github.com/larsartmann/go-cqrs-lite/example/"+name,
				"module example.com/"+name, 1)
		}

		return os.WriteFile(filepath.Join(dst, rel), []byte(content), 0o600)
	})
	if err != nil {
		t.Fatalf("copy example %s: %v", name, err)
	}

	return dst
}
