package consistency_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/consistency"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/ruletest"
)

func TestD017_RawErrorInDomainFile(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"fold.go": `package main

func foldCounter(state CounterState, evt event.Event) (CounterState, error) {
	switch evt.Type() {
	case "counter.incremented":
		return state, nil
	default:
		return state, errors.New("unknown event type")
	}
}
`,
	})

	findings := ruletest.RunDetector(t, consistency.NewD017Detector(ctx))
	ruletest.AssertRule(t, findings, "D017", 1)
}

func TestD017_FmtErrorfInDomainFile(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"fold.go": `package main

func foldCounter(state CounterState, evt event.Event) (CounterState, error) {
	switch evt.Type() {
	case "counter.incremented":
		return state, nil
	default:
		return state, fmt.Errorf("unknown event type: %s", evt.Type())
	}
}
`,
	})

	findings := ruletest.RunDetector(t, consistency.NewD017Detector(ctx))
	ruletest.AssertRule(t, findings, "D017", 1)
}

func TestD017_FmtErrorfWithWrapNoFinding(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"fold.go": `package main

func foldCounter(state CounterState, evt event.Event) (CounterState, error) {
	switch evt.Type() {
	case "counter.incremented":
		return state, nil
	default:
		return state, fmt.Errorf("unknown event: %w", errUnexpected)
	}
}
`,
	})

	findings := ruletest.RunDetector(t, consistency.NewD017Detector(ctx))
	ruletest.AssertRule(t, findings, "D017", 0)
}

func TestD017_PackageLevelSentinelNoFinding(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"fold.go": `package main

var ErrUnknownEvent = errors.New("unknown event type")

func foldCounter(state CounterState, evt event.Event) (CounterState, error) {
	return state, nil
}
`,
	})

	findings := ruletest.RunDetector(t, consistency.NewD017Detector(ctx))
	ruletest.AssertRule(t, findings, "D017", 0)
}

func TestD017_NonDomainFileNoFinding(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"handler.go": `package main

func handleRequest() error {
	return errors.New("request failed")
}
`,
	})

	findings := ruletest.RunDetector(t, consistency.NewD017Detector(ctx))
	ruletest.AssertRule(t, findings, "D017", 0)
}
