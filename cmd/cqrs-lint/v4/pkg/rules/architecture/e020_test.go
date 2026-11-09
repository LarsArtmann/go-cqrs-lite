package architecture_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/rules/architecture"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/ruletest"
)

// --- E020: a test file boots the composition root by hand ---

func TestE020_FiresOnHandrolledBootInTest(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"boot_test.go": `package main

import (
	"context"

	"github.com/larsartmann/go-cqrs-lite/system/v4"
)

func TestBoot(t *testing.T) {
	sys, err := system.New(context.Background(), domain, deploy)
	if err != nil {
		t.Fatal(err)
	}
	_ = sys
}
`,
	})

	findings := ruletest.RunDetector(t, architecture.NewE020Detector(ctx))
	ruletest.AssertRule(t, findings, "E020", 1)
}

// A test file that imports systemscenario (booting to hand the system to
// Adopt, or using System directly) is adopting the harness — silent.
func TestE020_SilentWhenHarnessAdopted(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"adopt_test.go": `package main

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/system/v4"
	"github.com/larsartmann/go-cqrs-lite/systemscenario/v4"
)

func TestAdopted(t *testing.T) {
	ctx := context.Background()
	sys, _ := system.New(ctx, domain, deploy)
	sc := systemscenario.Adopt(t, ctx, sys)
	_ = sc
}
`,
	})

	findings := ruletest.RunDetector(t, architecture.NewE020Detector(ctx))
	ruletest.AssertRule(t, findings, "E020", 0)
}

// Production files boot the composition root by design — never flagged.
func TestE020_SilentInProductionFile(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

import (
	"context"

	"github.com/larsartmann/go-cqrs-lite/system/v4"
)

func run(ctx context.Context) error {
	sys, err := system.New(ctx, domain, deploy)
	if err != nil {
		return err
	}
	return sys.Start(ctx)
}
`,
	})

	findings := ruletest.RunDetector(t, architecture.NewE020Detector(ctx))
	ruletest.AssertRule(t, findings, "E020", 0)
}

// An aliased system import (sys "…/system/v4") must still be detected: the
// local-name fallback alone would miss `sys.New`.
func TestE020_DetectsAliasedImport(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"alias_test.go": `package main

import (
	"context"

	sys "github.com/larsartmann/go-cqrs-lite/system/v4"
)

func TestAliasedBoot(t *testing.T) {
	s, err := sys.New(context.Background(), domain, deploy)
	if err != nil {
		t.Fatal(err)
	}
	_ = s
}
`,
	})

	findings := ruletest.RunDetector(t, architecture.NewE020Detector(ctx))
	ruletest.AssertRule(t, findings, "E020", 1)
}

// A different package's New (systemscenario has no New, but sibling
// packages under go-cqrs-lite/system/ must not trip the exact-path check)
// stays silent.
func TestE020_SilentOnNonSystemNew(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"other_test.go": `package main

type integration struct{}

func (integration) New() error { return nil }

func TestOther(t *testing.T) {
	var in integration
	if err := in.New(); err != nil {
		t.Fatal(err)
	}
}
`,
	})

	findings := ruletest.RunDetector(t, architecture.NewE020Detector(ctx))
	ruletest.AssertRule(t, findings, "E020", 0)
}
