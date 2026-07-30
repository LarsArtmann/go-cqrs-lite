package correctness_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/consistency"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/correctness"
)

// C025/D006 overlap: In a CQRS-importing file, fmt.Errorf without %w
// should be reported by C025 (warning) and NOT by D006 (info).
// D006 defers fmt.Errorf in CQRS files to C025 to avoid double-reporting.
func TestC025_D006_Overlap_NoDoubleReport(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"handler.go": `package main

import (
	"errors"
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
)

var ErrBadEvent = errors.New("bad event")

func handle(evt event.Event) error {
	if evt == nil {
		return fmt.Errorf("event is nil")
	}
	return nil
}
`,
	})

	c025Findings := runDetector(t, correctness.NewC025Detector(ctx))
	d006Findings := runDetector(t, consistency.NewD006Detector(ctx))

	// C025 fires on fmt.Errorf without %w in the CQRS file.
	assertRule(t, c025Findings, "C025", 1)

	// D006 must NOT report fmt.Errorf in CQRS files (deferred to C025).
	d006ErrorfCount := 0

	for _, f := range d006Findings {
		if string(f.Rule) == "D006" {
			d006ErrorfCount++
		}
	}

	if d006ErrorfCount != 0 {
		t.Errorf(
			"D006 should not report fmt.Errorf in CQRS files (deferred to C025), got %d D006 findings",
			d006ErrorfCount,
		)
		for _, f := range d006Findings {
			t.Logf("  D006: %s", f.Message)
		}
	}
}

// C025/D006 overlap: D006 still reports errors.New in CQRS files
// (only fmt.Errorf is deferred to C025).
func TestC025_D006_Overlap_ErrorsNewStillReportedByD006(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"handler.go": `package main

import (
	"errors"
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
)

func handle(evt event.Event) error {
	if evt == nil {
		return errors.New("event is nil")
	}
	return fmt.Errorf("wrap: %w", evt)
}
`,
	})

	d006Findings := runDetector(t, consistency.NewD006Detector(ctx))

	// D006 reports errors.New even in CQRS files (only fmt.Errorf is deferred).
	d006Count := 0
	for _, f := range d006Findings {
		if string(f.Rule) == "D006" {
			d006Count++
		}
	}

	if d006Count != 1 {
		t.Errorf("D006 should report errors.New in CQRS files, got %d findings", d006Count)
	}
}

// C025/D006 overlap: In a non-CQRS file, D006 reports fmt.Errorf (C025 skips).
func TestC025_D006_Overlap_NonCQRSFile_D006Reports(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"util.go": `package main

import "fmt"

func validate(id string) error {
	if id == "" {
		return fmt.Errorf("empty id")
	}
	return nil
}
`,
	})

	c025Findings := runDetector(t, correctness.NewC025Detector(ctx))
	d006Findings := runDetector(t, consistency.NewD006Detector(ctx))

	// C025 skips non-CQRS files.
	assertRule(t, c025Findings, "C025", 0)

	// D006 reports fmt.Errorf in non-CQRS files.
	d006Count := 0
	for _, f := range d006Findings {
		if string(f.Rule) == "D006" {
			d006Count++
		}
	}

	if d006Count != 1 {
		t.Errorf("D006 should report fmt.Errorf in non-CQRS files, got %d findings", d006Count)
	}
}
