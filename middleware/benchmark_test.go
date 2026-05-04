package middleware

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/core/command"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

func BenchmarkCommandLogging(b *testing.B) {
	logger := &testLogger{}
	mw := CommandLogging(logger)
	handler := mw(func(_ context.Context, _ command.Command) error { return nil })

	cmd := &testCommand{aggregateID: id.NewAggregateID()}
	ctx := context.Background()

	for b.Loop() {
		_ = handler(ctx, cmd)
	}
}

func BenchmarkCommandRecovery(b *testing.B) {
	mw := CommandRecovery()
	handler := mw(func(_ context.Context, _ command.Command) error { return nil })

	cmd := &testCommand{aggregateID: id.NewAggregateID()}
	ctx := context.Background()

	for b.Loop() {
		_ = handler(ctx, cmd)
	}
}

func BenchmarkCommandValidation(b *testing.B) {
	mw := CommandValidation(func(_ command.Command) error { return nil })
	handler := mw(func(_ context.Context, _ command.Command) error { return nil })

	cmd := &testCommand{aggregateID: id.NewAggregateID()}
	ctx := context.Background()

	for b.Loop() {
		_ = handler(ctx, cmd)
	}
}

func BenchmarkCommandRetry(b *testing.B) {
	mw := CommandRetry(DefaultRetryConfig())
	handler := mw(func(_ context.Context, _ command.Command) error { return nil })

	cmd := &testCommand{aggregateID: id.NewAggregateID()}
	ctx := context.Background()

	for b.Loop() {
		_ = handler(ctx, cmd)
	}
}
