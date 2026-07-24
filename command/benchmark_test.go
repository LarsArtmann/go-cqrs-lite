package command_test

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

func BenchmarkNew(b *testing.B) {
	b.ReportAllocs()

	streamID := id.NewStreamID()

	b.ResetTimer()

	for b.Loop() {
		_, err := command.New("bench.cmd", streamID)
		if err != nil {
			b.Fatalf("New: %v", err)
		}
	}
}

func BenchmarkMustNew(b *testing.B) {
	b.ReportAllocs()

	streamID := id.NewStreamID()

	b.ResetTimer()

	for b.Loop() {
		_ = newCmd(b, "bench.cmd", streamID)
	}
}

func BenchmarkNew_WithMetadata(b *testing.B) {
	b.ReportAllocs()

	streamID := id.NewStreamID()
	corrID := id.NewCorrelationID()

	b.ResetTimer()

	for b.Loop() {
		_, err := command.New(
			"bench.cmd", streamID,
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

func BenchmarkMetadataConstruction(b *testing.B) {
	b.ReportAllocs()

	for b.Loop() {
		_ = command.Metadata{}
	}
}
