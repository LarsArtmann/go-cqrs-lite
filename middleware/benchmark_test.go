package middleware

import (
	"context"
	"log/slog"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/command/v2"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
)

func benchCommandMiddleware(b *testing.B, mw command.Middleware) {
	handler := mw(NoopCommandHandler())

	cmd := &testCommand{aggregateID: id.NewAggregateID()}
	ctx := context.Background()

	for b.Loop() {
		_ = handler(ctx, cmd)
	}
}

func BenchmarkCommandLogging(b *testing.B) {
	logger := slog.New(slog.DiscardHandler)
	benchCommandMiddleware(b, CommandLogging(logger))
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
