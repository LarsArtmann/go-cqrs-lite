package main

import (
	"path/filepath"
	"testing"
)

// exampleModules are the four example apps; they double as the canonical
// v5-clean consumers. This meta-test mechanizes the recurring
// "audit examples for v5-removed API" TODO: every example must scan clean
// with the strict V007 detector, and a failed scan fails the test too (a
// false-green is exactly what the manual audit used to catch, late).
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
			dir := filepath.Join("..", "..", "example", name)

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
