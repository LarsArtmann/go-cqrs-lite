package correctness_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/correctness"
)

func TestC033_DetectsBareReturnFromCQRSMethod(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"handler.go": `package main

import "context"

func handle(ctx context.Context, store Store) error {
	if err := store.Save(ctx, evt); err != nil {
		return err
	}
	return nil
}
`,
	})
	findings := runDetector(t, correctness.NewC033Detector(ctx))
	assertRule(t, findings, "C033", 1)
}

func TestC033_DetectsBareReturnFromExecute(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"handler.go": `package main

import "context"

func handle(ctx context.Context, repo Repo) error {
	if err := repo.Execute(ctx, id, "User", decide); err != nil {
		return err
	}
	return nil
}
`,
	})
	findings := runDetector(t, correctness.NewC033Detector(ctx))
	assertRule(t, findings, "C033", 1)
}

func TestC033_NoFindingWhenErrorWrapped(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"handler.go": `package main

import (
	"context"
	"fmt"
)

func handle(ctx context.Context, store Store) error {
	if err := store.Save(ctx, evt); err != nil {
		return fmt.Errorf("save event: %w", err)
	}
	return nil
}
`,
	})
	findings := runDetector(t, correctness.NewC033Detector(ctx))
	assertRule(t, findings, "C033", 0)
}

func TestC033_NoFindingForNonCQRSMethod(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"handler.go": `package main

func handle() error {
	if err := validate(); err != nil {
		return err
	}
	return nil
}
`,
	})
	findings := runDetector(t, correctness.NewC033Detector(ctx))
	assertRule(t, findings, "C033", 0)
}

func TestC033_NoFindingOnEmptyContext(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main`,
	})
	findings := runDetector(t, correctness.NewC033Detector(ctx))
	assertRule(t, findings, "C033", 0)
}
