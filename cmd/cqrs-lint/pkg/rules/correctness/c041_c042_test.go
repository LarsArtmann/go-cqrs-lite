package correctness_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/rules/correctness"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/ruletest"
)

func TestC041_SaveIgnoresExpectedVersion(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"store.go": `package main

import (
	"context"
)

type MyStore struct{}

func (s *MyStore) Save(ctx context.Context, streamID string, expectedVersion int, events []int) error {
	// Bug: expectedVersion is never checked
	for _, e := range events {
		_ = e
	}
	return nil
}
`,
	})

	findings := ruletest.RunDetector(t, correctness.NewC041Detector(ctx))
	ruletest.AssertRule(t, findings, "C041", 1)
}

func TestC041_SaveUsesExpectedVersion(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"store.go": `package main

import (
	"context"
	"errors"
)

type MyStore struct{}

func (s *MyStore) Save(ctx context.Context, streamID string, expectedVersion int, events []int) error {
	if expectedVersion != s.currentVersion(streamID) {
		return errors.New("conflict")
	}
	return nil
}
`,
	})

	findings := ruletest.RunDetector(t, correctness.NewC041Detector(ctx))
	ruletest.AssertRule(t, findings, "C041", 0)
}

func TestC042_SaveWithZeroVersion(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

import "context"

func saveEvent(ctx context.Context, store *Store) error {
	return store.Save(ctx, ref, events, 0)
}
`,
	})

	findings := ruletest.RunDetector(t, correctness.NewC042Detector(ctx))
	ruletest.AssertRule(t, findings, "C042", 1)
}

func TestC042_SaveWithVersionConversionZero(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

import (
	"context"

	"github.com/larsartmann/go-cqrs-lite/event"
)

func saveEvent(ctx context.Context, store *Store) error {
	return store.Save(ctx, ref, events, event.Version(0))
}
`,
	})

	findings := ruletest.RunDetector(t, correctness.NewC042Detector(ctx))
	ruletest.AssertRule(t, findings, "C042", 1)
}

func TestC042_SaveWithNonZeroVersion(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

import "context"

func saveEvent(ctx context.Context, store *Store) error {
	return store.Save(ctx, ref, events, version)
}
`,
	})

	findings := ruletest.RunDetector(t, correctness.NewC042Detector(ctx))
	ruletest.AssertRule(t, findings, "C042", 0)
}

func TestC042_SaveWithNonZeroConversion(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

import (
	"context"

	"github.com/larsartmann/go-cqrs-lite/event"
)

func saveEvent(ctx context.Context, store *Store, v int) error {
	return store.Save(ctx, ref, events, event.Version(v))
}
`,
	})

	findings := ruletest.RunDetector(t, correctness.NewC042Detector(ctx))
	ruletest.AssertRule(t, findings, "C042", 0)
}
