package command_test

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/command/v2"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
)

func mustNewCmd(commandType command.Type, aggregateID id.AggregateID, opts ...command.Option) *command.BasicCommand {
	cmd, err := command.New(commandType, aggregateID, opts...)
	if err != nil {
		panic(err)
	}
	return cmd
}

func BenchmarkNew(b *testing.B) {
	b.ReportAllocs()

	aggID := id.NewAggregateID()

	b.ResetTimer()

	for b.Loop() {
		_, err := command.New("bench.cmd", aggID)
		if err != nil {
			b.Fatalf("New: %v", err)
		}
	}
}

func BenchmarkMustNew(b *testing.B) {
	b.ReportAllocs()

	aggID := id.NewAggregateID()

	b.ResetTimer()

	for b.Loop() {
		_ = mustNewCmd("bench.cmd", aggID)
	}
}

func BenchmarkNew_WithMetadata(b *testing.B) {
	b.ReportAllocs()

	aggID := id.NewAggregateID()
	corrID := id.NewCorrelationID()

	b.ResetTimer()

	for b.Loop() {
		_, err := command.New(
			"bench.cmd", aggID,
			command.WithCorrelationID(corrID),
		)
		if err != nil {
			b.Fatalf("New: %v", err)
		}
	}
}

func BenchmarkDispatcher_Register(b *testing.B) {
	b.ReportAllocs()

	for b.Loop() {
		d := command.NewDispatcher()
		err := d.Register("bench.cmd", noopCommandHandler())
		if err != nil {
			b.Fatalf("Register: %v", err)
		}

		_ = d.Close()
	}
}

func BenchmarkDispatcher_RegisterTyped(b *testing.B) {
	b.ReportAllocs()

	type benchCmd struct {
		*command.BasicCommand
	}

	for b.Loop() {
		d := command.NewDispatcher()
		err := command.RegisterTyped(
			d, "bench.cmd",
			func(_ context.Context, _ *benchCmd) error { return nil },
		)
		if err != nil {
			b.Fatalf("RegisterTyped: %v", err)
		}

		_ = d.Close()
	}
}

func BenchmarkNewMetadata(b *testing.B) {
	b.ReportAllocs()

	for b.Loop() {
		_ = command.NewMetadata()
	}
}
