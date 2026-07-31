package correctness_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/correctness"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/ruletest"
)

func TestC035_UnprotectedMapInReadModel(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"readmodel.go": `package main

type UserReadModel struct {
	Users map[string]*User
}
`,
	})
	findings := ruletest.RunDetector(t, correctness.NewC035Detector(ctx))
	ruletest.AssertRule(t, findings, "C035", 1)
}

func TestC035_ProtectedWithMutex(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"readmodel.go": `package main

import "sync"

type UserReadModel struct {
	mu    sync.RWMutex
	Users map[string]*User
}
`,
	})
	findings := ruletest.RunDetector(t, correctness.NewC035Detector(ctx))
	ruletest.AssertRule(t, findings, "C035", 0)
}

func TestC035_NotReadModelStruct(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

type Config struct {
	Values map[string]string
}
`,
	})
	findings := ruletest.RunDetector(t, correctness.NewC035Detector(ctx))
	ruletest.AssertRule(t, findings, "C035", 0)
}

func TestC035_NoMapField(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"readmodel.go": `package main

type UserReadModel struct {
	Count int
	Name  string
}
`,
	})
	findings := ruletest.RunDetector(t, correctness.NewC035Detector(ctx))
	ruletest.AssertRule(t, findings, "C035", 0)
}
