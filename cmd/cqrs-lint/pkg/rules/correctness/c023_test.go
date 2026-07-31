package correctness_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/correctness"
)

func TestC023_DetectsIgnoredStopError(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"service.go": `package main

func shutdown(host *Host) {
	_ = host.Stop()
}
`,
	})
	findings := ruletest.RunDetector(t, correctness.NewC023Detector(ctx))
	ruletest.AssertRule(t, findings, "C023", 1)
}

func TestC023_NoFindingInDefer(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"service.go": `package main

func shutdown(host *Host) {
	defer func() { _ = host.Stop() }()
}
`,
	})
	findings := ruletest.RunDetector(t, correctness.NewC023Detector(ctx))
	ruletest.AssertRule(t, findings, "C023", 0)
}

func TestC023_NoFindingWhenErrorChecked(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"service.go": `package main

func shutdown(host *Host) error {
	return host.Stop()
}
`,
	})
	findings := ruletest.RunDetector(t, correctness.NewC023Detector(ctx))
	ruletest.AssertRule(t, findings, "C023", 0)
}
