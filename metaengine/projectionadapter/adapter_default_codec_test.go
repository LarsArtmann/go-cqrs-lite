package projectionadapter_test

import (
	"context"
	"testing"

	"github.com/larsartmann/go-codec"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/projectionadapter/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/record/v4"
)

// TestAdapter_DefaultDecode_Codecs covers the zero-config adapter path: no
// PayloadDecoder, no EventDecoder, no TypeDecoder. The adapter must deliver the
// payload in whatever codec the event carries — JSON and CBOR alike — to typed
// folds. A CBOR event (event.New's default) must never become a decode error.
func TestAdapter_DefaultDecode_Codecs(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name  string
		codec codec.Codec
		enc   codec.Encoding
	}{
		{name: "json", codec: codec.JSONCodec{}, enc: codec.EncodingJSON},
		{name: "cbor", codec: codec.CBORCodec{}, enc: codec.EncodingCBOR},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			counterQuery := metaengine.Query[FindCounter, CounterState](
				"counter-default-decode",
				metaengine.OnRecord(
					CounterIncremented{},
					func(_ record.Record, e CounterIncremented) (CounterID, CounterState) {
						return e.ID, CounterState{Value: e.Amount}
					},
				),
			)

			store, err := metaengine.Plan(
				[]metaengine.Engine{metaengine.NewMemoryEngine()},
				counterQuery,
			)
			if err != nil {
				t.Fatalf("metaengine.Plan: %v", err)
			}
			defer store.Close()

			payload, err := tc.codec.Encode(CounterIncremented{ID: "counter-1", Amount: 5})
			if err != nil {
				t.Fatalf("%s encode: %v", tc.name, err)
			}

			evt, err := event.NewEvent(
				event.Type("CounterIncremented"),
				id.NewStreamID(),
				"Counter",
				1,
				payload,
				event.WithEncoding(tc.enc),
			)
			if err != nil {
				t.Fatalf("event.NewEvent: %v", err)
			}

			adapter := projectionadapter.New("default-decode", store, nil)

			if err := adapter.Handle(context.Background(), evt); err != nil {
				t.Fatalf("Handle(%s) = %v, want nil", tc.name, err)
			}

			result, err := metaengine.ExecuteTyped[FindCounter, CounterState](
				context.Background(), store, FindCounter{ID: "counter-1"},
			)
			if err != nil {
				t.Fatalf("ExecuteTyped: %v", err)
			}

			if result.Value != 5 {
				t.Fatalf("counter value = %d, want 5", result.Value)
			}
		})
	}
}
