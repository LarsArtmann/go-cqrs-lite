package correctness_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/correctness"
)

func TestC031_DetectsSwallowedErrorInRegisterTypedHandler(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"handler.go": `package main

import "context"

func setup() {
	command.RegisterTyped(disp, "user.create", func(ctx context.Context, cmd *CreateUser) error {
		data, err := validate(cmd)
		if err != nil {
			return nil
		}
		_ = data
		return nil
	})
}
`,
	})
	findings := runDetector(t, correctness.NewC031Detector(ctx))
	assertRule(t, findings, "C031", 1)
}

func TestC031_DetectsBareReturnOnError(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"handler.go": `package main

import "context"

func setup() {
	query.RegisterTyped(qdisp, "user.get", func(ctx context.Context, q *GetUser) error {
		_, err := db.Query(ctx, q.ID)
		if err != nil {
			return
		}
		return nil
	})
}
`,
	})
	findings := runDetector(t, correctness.NewC031Detector(ctx))
	assertRule(t, findings, "C031", 1)
}

func TestC031_NoFindingWhenErrorPropagated(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"handler.go": `package main

import (
	"context"
	"fmt"
)

func setup() {
	command.RegisterTyped(disp, "user.create", func(ctx context.Context, cmd *CreateUser) error {
		data, err := validate(cmd)
		if err != nil {
			return fmt.Errorf("validate: %w", err)
		}
		_ = data
		return nil
	})
}
`,
	})
	findings := runDetector(t, correctness.NewC031Detector(ctx))
	assertRule(t, findings, "C031", 0)
}

func TestC031_NoFindingOutsideRegisterTyped(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"handler.go": `package main

func regularFunc() error {
	err := doSomething()
	if err != nil {
		return nil
	}
	return nil
}
`,
	})
	findings := runDetector(t, correctness.NewC031Detector(ctx))
	assertRule(t, findings, "C031", 0)
}

func TestC031_NoFindingOnEmptyContext(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main`,
	})
	findings := runDetector(t, correctness.NewC031Detector(ctx))
	assertRule(t, findings, "C031", 0)
}
