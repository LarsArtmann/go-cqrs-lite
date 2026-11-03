package adoption_test

import (
	"strings"
	"testing"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/rules/adoption"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/ruletest"
)

// f032Source is the base positive fixture: a consumer importing the event
// module (the classic core usage) with no catalog anywhere.
const f032Source = `package main

import (
	"github.com/larsartmann/go-cqrs-lite/event/v4"
)

func emit() {
	_ = event.New("user.registered", nil)
}
`

// f032ImportLine returns the 1-indexed line of the event-module import in
// src, derived at runtime so fixture edits above the import do not break
// the anchor assertion.
func f032ImportLine(t *testing.T, src string) int {
	t.Helper()

	for i, line := range strings.Split(src, "\n") {
		if strings.Contains(line, "go-cqrs-lite/event/v4") {
			return i + 1
		}
	}

	t.Fatal("f032 fixture: event-module import not found")
	return 0
}

func TestF032_LibraryWithoutCatalogFires(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{"main.go": f032Source})

	findings := ruletest.RunDetector(t, adoption.NewF032Detector(ctx))
	ruletest.AssertRule(t, findings, "F032", 1)

	if len(findings) != 1 {
		t.Fatalf("expected exactly 1 finding, got %d", len(findings))
	}

	if findings[0].Severity != finding.SeverityError {
		t.Errorf("severity = %s, want error (the rule must block CI, not just report)",
			findings[0].Severity)
	}

	if findings[0].Confidence != finding.ConfidenceHigh {
		t.Errorf("confidence = %s, want high (import detection is deterministic)",
			findings[0].Confidence)
	}

	if want := f032ImportLine(t, f032Source); findings[0].Position.Line != want {
		t.Errorf("finding anchored at line %d, want %d (the library import)",
			findings[0].Position.Line, want)
	}
}

func TestF032_CatalogImportSilences(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

import (
	"github.com/larsartmann/go-cqrs-lite/catalog/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
)

func emit() {
	_ = event.New("user.registered", nil)
	_ = catalog.NewBuilder()
}
`,
	})

	findings := ruletest.RunDetector(t, adoption.NewF032Detector(ctx))
	ruletest.AssertRule(t, findings, "F032", 0)
}

func TestF032_CatalogSubpackageSilences(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

import (
	"github.com/larsartmann/go-cqrs-lite/catalog/v4/simple"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
)

func emit() {
	_ = event.New("user.registered", nil)
	_, _ = simple.New()
}
`,
	})

	findings := ruletest.RunDetector(t, adoption.NewF032Detector(ctx))
	ruletest.AssertRule(t, findings, "F032", 0)
}

func TestF032_SystemcatalogSilences(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

import (
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/systemcatalog/v4"
)

func emit() {
	_ = event.New("user.registered", nil)
	_ = systemcatalog.DeriveInto
}
`,
	})

	findings := ruletest.RunDetector(t, adoption.NewF032Detector(ctx))
	ruletest.AssertRule(t, findings, "F032", 0)
}

func TestF032_CatalogOnlyStaysSilent(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

import (
	"github.com/larsartmann/go-cqrs-lite/catalog/v4"
)

func docs() {
	_ = catalog.NewBuilder()
}
`,
	})

	findings := ruletest.RunDetector(t, adoption.NewF032Detector(ctx))
	ruletest.AssertRule(t, findings, "F032", 0)
}

func TestF032_NoLibraryUsageStaysSilent(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

import "fmt"

func main() {
	fmt.Println("not a cqrs project")
}
`,
	})

	findings := ruletest.RunDetector(t, adoption.NewF032Detector(ctx))
	ruletest.AssertRule(t, findings, "F032", 0)
}

func TestF032_TestOnlyUsageStaysSilent(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

func main() {}
`,
		"main_test.go": `package main

import (
	"github.com/larsartmann/go-cqrs-lite/event/v4/eventtest"
)

func TestThing(t *testing.T) {
	_ = eventtest.New()
}
`,
	})

	findings := ruletest.RunDetector(t, adoption.NewF032Detector(ctx))
	ruletest.AssertRule(t, findings, "F032", 0)
}
