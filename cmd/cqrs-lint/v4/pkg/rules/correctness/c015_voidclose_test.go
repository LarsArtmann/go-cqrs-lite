package correctness_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/rules/correctness"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/ruletest"
)

// TestC015_NoFindingForVoidClose verifies that C015 does not fire on Close()
// calls that return no value (e.g. metaengine.Watcher.Close). There is no
// error to discard — firing there is a false positive by the rule's own
// definition ("whose error return is discarded").
func TestC015_NoFindingForVoidClose(t *testing.T) {
	t.Parallel()

	ctx, cleanup := analyzer.BuildContextWithTypes(t, "1.26", map[string]string{
		"main.go": `package main

type Watcher struct{}

func (w *Watcher) Close() {}

type Store struct{}

func (s *Store) Close() error { return nil }

func shutdown(w *Watcher, s *Store) {
	if w != nil {
		w.Close()
	}

	s.Close()
}
`,
	})
	defer cleanup()

	findings := ruletest.RunDetector(t, correctness.NewC015Detector(ctx))
	ruletest.AssertRule(t, findings, "C015", 1) // only the error-returning Store.Close
}
