package correctness_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/correctness"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/ruletest"
)

func TestC028_DetectsSwallowedDispatch(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"handler.go": `package main

func handle(ctx context.Context) {
	_ = cmds.Dispatch(ctx, cmd)
}
`,
	})
	findings := ruletest.RunDetector(t, correctness.NewC028Detector(ctx))
	ruletest.AssertRule(t, findings, "C028", 1)
}

func TestC028_DetectsSwallowedExecute(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"handler.go": `package main

func handle(ctx context.Context) {
	_ = repo.Execute(ctx, aggID, "User", decide)
}
`,
	})
	findings := ruletest.RunDetector(t, correctness.NewC028Detector(ctx))
	ruletest.AssertRule(t, findings, "C028", 1)
}

func TestC028_DetectsSwallowedRegister(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"setup.go": `package main

func setup() {
	_ = command.RegisterTyped(cmds, "user.create", handler)
}
`,
	})
	findings := ruletest.RunDetector(t, correctness.NewC028Detector(ctx))
	ruletest.AssertRule(t, findings, "C028", 1)
}

func TestC028_NoFindingWhenErrorChecked(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"handler.go": `package main

func handle(ctx context.Context) {
	if err := cmds.Dispatch(ctx, cmd); err != nil {
		return
	}
}
`,
	})
	findings := ruletest.RunDetector(t, correctness.NewC028Detector(ctx))
	ruletest.AssertRule(t, findings, "C028", 0)
}

func TestC028_NoFindingForNonCQRSMethod(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"util.go": `package main

func doSomething() {
	_ = logger.Debug("msg")
}
`,
	})
	findings := ruletest.RunDetector(t, correctness.NewC028Detector(ctx))
	ruletest.AssertRule(t, findings, "C028", 0)
}

func TestC028_NoFindingOnEmptyContext(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main`,
	})
	findings := ruletest.RunDetector(t, correctness.NewC028Detector(ctx))
	ruletest.AssertRule(t, findings, "C028", 0)
}
