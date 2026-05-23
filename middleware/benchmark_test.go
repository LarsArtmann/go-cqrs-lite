package middleware

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/core/command"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

func benchCommandMiddleware(b *testing.B, mw command.Middleware) {
	handler := mw(func(_ context.Context, _ command.Command) error { return nil })

	cmd := &testCommand{aggregateID: id.NewAggregateID()}
	ctx := context.Background()

	for b.Loop() {
		_ = handler(ctx, cmd)
	}
}

func BenchmarkCommandLogging(b *testing.B) {
	benchCommandMiddleware(b, CommandLogging(&testLogger{}))
}

func BenchmarkCommandRecovery(b *testing.B) {
	benchCommandMiddleware(b, CommandRecovery())
}

func BenchmarkCommandValidation(b *testing.B) {
	benchCommandMiddleware(b, CommandValidation(func(_ command.Command) error { return nil }))
}

func BenchmarkCommandRetry(b *testing.B) {
	benchCommandMiddleware(b, CommandRetry(DefaultRetryConfig()))
}
