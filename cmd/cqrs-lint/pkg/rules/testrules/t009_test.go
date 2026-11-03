package testrules_test

import (
	"strings"
	"testing"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/ruletest"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/rules/testrules"
)

// t009BootSource is the base positive fixture: a test file hand-boots a real
// system via system.New with no systemscenario anywhere.
const t009BootSource = `package app

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/system/v4"
)

func TestBoot(t *testing.T) {
	sys, err := system.New(t.Context(), domain(), deploy())
	if err != nil {
		t.Fatal(err)
	}
	_ = sys
}
`

// t009BootLine returns the 1-indexed line of the hand-boot call in src,
// derived at runtime so fixture edits above the call do not break the anchor
// assertion.
func t009BootLine(t *testing.T, src string) int {
	t.Helper()

	for i, line := range strings.Split(src, "\n") {
		if strings.Contains(line, "system.New(") {
			return i + 1
		}
	}

	t.Fatal("t009 fixture: system.New call not found")
	return 0
}

func TestT009_HandBootedSystemFires(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"app_test.go": t009BootSource,
	})

	findings := ruletest.RunDetector(t, testrules.NewT009Detector(ctx))
	ruletest.AssertRule(t, findings, "T009", 1)

	if len(findings) != 1 {
		t.Fatalf("expected exactly 1 finding, got %d", len(findings))
	}

	if findings[0].Severity != finding.SeverityInfo {
		t.Errorf("severity = %s, want info (T-family adoption coaching)", findings[0].Severity)
	}

	if want := t009BootLine(t, t009BootSource); findings[0].Position.Line != want {
		t.Errorf("finding anchored at line %d, want %d (the hand-boot call)",
			findings[0].Position.Line, want)
	}
}

func TestT009_SystemscenarioSilences(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"app_test.go": `package app

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/systemscenario/v4"
)

func TestScenario(t *testing.T) {
	sc := systemscenario.System(t, t.Context(), domain(), deploy())
	sc.Given().Command(t, registerUser())
}
`,
	})

	findings := ruletest.RunDetector(t, testrules.NewT009Detector(ctx))
	ruletest.AssertRule(t, findings, "T009", 0)
}

func TestT009_AdoptWrapperSilences(t *testing.T) {
	t.Parallel()

	// The Adopt pattern: a wrapper facade owns system.New itself and adopts
	// the booted system into the harness — the import satisfies the rule
	// even though system.New appears in the same test file.
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"app_test.go": `package app

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/system/v4"
	"github.com/larsartmann/go-cqrs-lite/systemscenario/v4"
)

func TestAdopted(t *testing.T) {
	sys, err := system.New(t.Context(), domain(), deploy())
	if err != nil {
		t.Fatal(err)
	}
	sc := systemscenario.Adopt(sys)
	_ = sc
}
`,
	})

	findings := ruletest.RunDetector(t, testrules.NewT009Detector(ctx))
	ruletest.AssertRule(t, findings, "T009", 0)
}

func TestT009_NoSystemBootInTestsStaysSilent(t *testing.T) {
	t.Parallel()

	// Production boots the system (main.go); tests never do — nothing owed.
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

import (
	"github.com/larsartmann/go-cqrs-lite/system/v4"
)

func run() {
	_, _ = system.New(nil, domain(), deploy())
}
`,
		"app_test.go": `package main

import "testing"

func TestUnit(t *testing.T) {}
`,
	})

	findings := ruletest.RunDetector(t, testrules.NewT009Detector(ctx))
	ruletest.AssertRule(t, findings, "T009", 0)
}

func TestT009_NoTestFilesStaysSilent(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

func main() {}
`,
	})

	findings := ruletest.RunDetector(t, testrules.NewT009Detector(ctx))
	ruletest.AssertRule(t, findings, "T009", 0)
}

func TestT009_UnrelatedNewCallStaysSilent(t *testing.T) {
	t.Parallel()

	// A different package qualifier's New (not system.New) must not fire —
	// the pkgIdent check discriminates.
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"app_test.go": `package app

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/queue/v4"
)

func TestQueue(t *testing.T) {
	q := queue.New(t.Context())
	_ = q
}
`,
	})

	findings := ruletest.RunDetector(t, testrules.NewT009Detector(ctx))
	ruletest.AssertRule(t, findings, "T009", 0)
}
