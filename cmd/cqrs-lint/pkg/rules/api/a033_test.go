package api_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/api"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/ruletest"
)

func TestA033_StringRoundtrip(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"setup.go": `package main

import "github.com/larsartmann/go-cqrs-lite/id/v4"

func convert(otherID id.Of[UserMarker]) (id.Of[UserMarker], error) {
	return id.Parse[id.Of[UserMarker]](otherID.String())
}
`,
		"markers.go": `package main

type UserMarker struct{}
`,
	})

	findings := ruletest.RunDetector(t, api.NewA033Detector(ctx))
	ruletest.AssertRule(t, findings, "A033", 1)
}

func TestA033_MustParseRoundtrip(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"setup.go": `package main

import "github.com/larsartmann/go-cqrs-lite/id/v4"

func convert(otherID id.Of[UserMarker]) id.Of[UserMarker] {
	return id.MustParse[id.Of[UserMarker]](otherID.String())
}
`,
		"markers.go": `package main

type UserMarker struct{}
`,
	})

	findings := ruletest.RunDetector(t, api.NewA033Detector(ctx))
	ruletest.AssertRule(t, findings, "A033", 1)
}

func TestA033_RawStringArgNoFinding(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"setup.go": `package main

import "github.com/larsartmann/go-cqrs-lite/id/v4"

func fromString(s string) (id.Of[UserMarker], error) {
	return id.Parse[id.Of[UserMarker]](s)
}
`,
		"markers.go": `package main

type UserMarker struct{}
`,
	})

	findings := ruletest.RunDetector(t, api.NewA033Detector(ctx))
	ruletest.AssertRule(t, findings, "A033", 0)
}

func TestA033_NonIDParseNoFinding(t *testing.T) {
	t.Parallel()

	// A different package's Parse with a .String() arg should not fire.
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"setup.go": `package main

type thing struct{}

func (thing) String() string { return "x" }

func f(t thing) string {
	return customParse[thing](t.String())
}

func customParse[T any](s string) string { return s }
`,
	})

	findings := ruletest.RunDetector(t, api.NewA033Detector(ctx))
	ruletest.AssertRule(t, findings, "A033", 0)
}
