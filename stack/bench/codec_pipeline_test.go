package bench

import (
	"context"
	"fmt"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/codec/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/kv/v4"
	stackmemory "github.com/larsartmann/go-cqrs-lite/stack/memory/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/v4"
)

// codec_pipeline_test.go — measures JSON vs CBOR vs CBOR-Compact in a full
// write+decode cycle. This is the codec decision benchmark with real I/O.

// BenchmarkCodecPipeline_WriteRead measures the full encode → store → decode
// cycle with different codecs. Each iteration: create event with codec, save to
// store, load from store, decode payload.
func BenchmarkCodecPipeline_WriteRead(b *testing.B) {
	codecs := []struct {
		name string
		c    codec.Codec
	}{
		{"json", codec.JSONCodec{}},
		{"cbor", codec.CBORCodec{}},
		{"cbor-compact", codec.CBORCompactCodec{}},
	}

	type payload struct {
		ID      string `json:"id"`
		Name    string `json:"name"`
		Email   string `json:"email"`
		Amount  int64  `json:"amount"`
		Product string `json:"product"`
	}

	for _, c := range codecs {
		b.Run(c.name, func(b *testing.B) {
			bundle, err := stackmemory.New()
			if err != nil {
				b.Fatal(err)
			}
			defer func() { _ = bundle.Close() }()

			store, ok := bundle.EventStore()
			if !ok {
				b.Fatal("no event store")
			}

			rmBackend := kv.NewMemStore()
			rm := kv.NewTypedStore[payload, id.StreamID](rmBackend,
				kv.WithTypedCodec[payload, id.StreamID](c.c),
			)

			ctx := context.Background()

			b.ResetTimer()

			for b.Loop() {
				streamID := id.NewStreamID()
				ref := id.NewStreamRef("Codec", streamID)

				// Write: create event with specified codec.
				evt, err := event.New(
					"codec.test", streamID, "Codec", event.Version(1),
					payload{
						ID: streamID.String(), Name: "Test User",
						Email: "test@test.com", Amount: 5000, Product: "prod-001",
					},
					event.WithCodec(c.c),
				)
				if err != nil {
					b.Fatal(err)
				}

				if err := store.AppendBatch(ctx, ref, []event.Event{evt}); err != nil {
					b.Fatal(err)
				}

				// Read: decode payload + write to read model.
				p, err := event.DecodePayloadAuto[payload](evt)
				if err != nil {
					b.Fatal(err)
				}
				if err := rm.Set(ctx, streamID, &p); err != nil {
					b.Fatal(err)
				}

				// Query: read from read model.
				if _, err := rm.Get(ctx, streamID); err != nil {
					b.Fatal(err)
				}
			}

			b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "cycles/sec")
		})
	}
}

// BenchmarkCodecPipeline_PayloadSizes measures how payload size interacts with
// codec choice. CBOR should win more at larger sizes due to smaller binary format.
func BenchmarkCodecPipeline_PayloadSizes(b *testing.B) {
	codecs := []struct {
		name string
		c    codec.Codec
	}{
		{"json", codec.JSONCodec{}},
		{"cbor", codec.CBORCodec{}},
	}

	sizes := []int{128, 1024, 10240}

	for _, c := range codecs {
		for _, size := range sizes {
			b.Run(fmt.Sprintf("%s/size=%dB", c.name, size), func(b *testing.B) {
				bundle, err := stackmemory.New()
				if err != nil {
					b.Fatal(err)
				}
				defer func() { _ = bundle.Close() }()

				store, ok := bundle.EventStore()
				if !ok {
					b.Fatal("no event store")
				}

				ctx := context.Background()
				data := make([]byte, size)
				for i := range data {
					data[i] = 'x'
				}

				b.ResetTimer()

				for b.Loop() {
					streamID := id.NewStreamID()
					ref := id.NewStreamRef("Codec", streamID)

					evt, err := event.NewEvent(
						"codec.test", streamID, "Codec", event.Version(1),
						data,
					)
					if err != nil {
						b.Fatal(err)
					}
					if err := store.AppendBatch(ctx, ref, []event.Event{evt}); err != nil {
						b.Fatal(err)
					}
				}

				b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "events/sec")
			})
		}
	}
}

// suppress unused import warning.
var _ = stack.New
