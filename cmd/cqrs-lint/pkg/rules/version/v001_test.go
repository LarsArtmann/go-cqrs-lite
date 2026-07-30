package version_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/version"
)

func TestV001_DetectsMixedVersions(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"old.go": `package main

import "github.com/larsartmann/go-cqrs-lite/event/v3"
`,
		"new.go": `package main

import "github.com/larsartmann/go-cqrs-lite/event/v4"
`,
	})
	findings := runDetector(t, version.NewV001Detector(ctx))
	assertRule(t, findings, "V001", 1)
}

func TestV001_NoFindingForV4Only(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"a.go": `package main

import "github.com/larsartmann/go-cqrs-lite/event/v4"
`,
		"b.go": `package main

import "github.com/larsartmann/go-cqrs-lite/decider/v4"
`,
	})
	findings := runDetector(t, version.NewV001Detector(ctx))
	assertRule(t, findings, "V001", 0)
}
