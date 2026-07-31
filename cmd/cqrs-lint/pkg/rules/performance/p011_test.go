package performance_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/performance"
)

func TestP011_DetectsUnboundedMapInReadModel(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"model.go": `package main

type UserReadModel struct {
	users map[string]*User
}
`,
	})
	findings := ruletest.RunDetector(t, performance.NewP011Detector(ctx))
	ruletest.AssertRule(t, findings, "P011", 1)
}

func TestP011_NoFindingForNonReadModelStruct(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"config.go": `package main

type Config struct {
	options map[string]string
}
`,
	})
	findings := ruletest.RunDetector(t, performance.NewP011Detector(ctx))
	ruletest.AssertRule(t, findings, "P011", 0)
}

func TestP011_NoFindingForStructWithSyncMutex(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"model.go": `package main

import "sync"

type SafeReadModel struct {
	mu    sync.RWMutex
	items map[string]int
}
`,
	})
	findings := ruletest.RunDetector(t, performance.NewP011Detector(ctx))
	ruletest.AssertRule(t, findings, "P011", 0)
}

func TestP011_NoFindingOnEmptyContext(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main`,
	})
	findings := ruletest.RunDetector(t, performance.NewP011Detector(ctx))
	ruletest.AssertRule(t, findings, "P011", 0)
}
