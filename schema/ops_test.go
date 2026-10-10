package schema

import (
	"reflect"
	"testing"

	"github.com/larsartmann/go-codec"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4/idtest"
)

func newPayloadEvent(
	tb testing.TB,
	eventType event.Type,
	schemaVersion int,
	payload any,
	opts ...event.Option,
) event.Event {
	tb.Helper()

	streamID := idtest.ParseStreamID(tb, "01HK1540X0841Y0A6BSX1VKR95")
	newOpts := append(
		[]event.Option{event.WithSchemaVersion(event.SchemaVersion(schemaVersion))},
		opts...)

	if raw, isBytes := payload.([]byte); isBytes {
		evt, err := event.NewEvent(eventType, streamID, "Bank", event.Version(1), raw, newOpts...)
		if err != nil {
			tb.Fatalf("NewEvent: %v", err)
		}

		return evt
	}

	evt, err := event.New(eventType, streamID, "Bank", event.Version(1), payload, newOpts...)
	if err != nil {
		tb.Fatalf("New: %v", err)
	}

	return evt
}

func decodedFields(tb testing.TB, evt event.Event) map[string]any {
	tb.Helper()

	var fields map[string]any
	if err := (codec.JSONCodec{}).Decode(evt.Payload(), &fields); err != nil {
		tb.Fatalf("decode payload: %v", err)
	}

	return fields
}

func TestChainFieldOps(t *testing.T) {
	t.Parallel()

	transform := schemaTransformMoney(t)

	tests := []struct {
		name        string
		ops         func() []Op
		eventType   event.Type
		version     int
		payload     string
		wantVersion int
		wantFields  map[string]any
	}{
		{
			name: "rename field",
			ops: func() []Op {
				return []Op{RenameField("user.created", 1, "name", "displayName")}
			},
			eventType:   "user.created",
			version:     1,
			payload:     `{"name":"Lars","age":30}`,
			wantVersion: 2,
			wantFields:  map[string]any{"displayName": "Lars", "age": float64(30)},
		},
		{
			name: "rename absent field still advances version",
			ops: func() []Op {
				return []Op{RenameField("user.created", 1, "nickname", "handle")}
			},
			eventType:   "user.created",
			version:     1,
			payload:     `{"name":"Lars"}`,
			wantVersion: 2,
			wantFields:  map[string]any{"name": "Lars"},
		},
		{
			name: "add field with default",
			ops: func() []Op {
				return []Op{AddField("user.created", 1, "country", "US")}
			},
			eventType:   "user.created",
			version:     1,
			payload:     `{"name":"Lars"}`,
			wantVersion: 2,
			wantFields:  map[string]any{"name": "Lars", "country": "US"},
		},
		{
			name: "add field never overwrites",
			ops: func() []Op {
				return []Op{AddField("user.created", 1, "country", "US")}
			},
			eventType:   "user.created",
			version:     1,
			payload:     `{"name":"Lars","country":"DE"}`,
			wantVersion: 2,
			wantFields:  map[string]any{"name": "Lars", "country": "DE"},
		},
		{
			name: "remove field",
			ops: func() []Op {
				return []Op{RemoveField("user.created", 2, "legacyToken")}
			},
			eventType:   "user.created",
			version:     2,
			payload:     `{"name":"Lars","legacyToken":"x"}`,
			wantVersion: 3,
			wantFields:  map[string]any{"name": "Lars"},
		},
		{
			name:        "transform reshapes payload (bank-sync money case)",
			ops:         func() []Op { return []Op{transform} },
			eventType:   "balance.updated",
			version:     1,
			payload:     `{"amountCents":100,"reservedCents":20,"currency":"EUR"}`,
			wantVersion: 2,
			wantFields: map[string]any{
				"amount":   map[string]any{"cents": float64(100), "currency": "EUR"},
				"reserved": map[string]any{"cents": float64(20), "currency": "EUR"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			chain, err := Compile(tt.ops()...)
			if err != nil {
				t.Fatalf("Compile: %v", err)
			}

			evt := newPayloadEvent(t, tt.eventType, tt.version, []byte(tt.payload))
			got, err := chain.upcastAll([]event.Event{evt})
			if err != nil {
				t.Fatalf("upcastAll: %v", err)
			}

			if len(got) != 1 {
				t.Fatalf("got %d events, want 1", len(got))
			}

			if got[0].SchemaVersion() != event.SchemaVersion(tt.wantVersion) {
				t.Errorf("schema version = %d, want %d", got[0].SchemaVersion(), tt.wantVersion)
			}

			if !reflect.DeepEqual(decodedFields(t, got[0]), tt.wantFields) {
				t.Errorf("fields = %v, want %v", decodedFields(t, got[0]), tt.wantFields)
			}
		})
	}
}

func schemaTransformMoney(tb testing.TB) Op {
	tb.Helper()

	return Transform("balance.updated", 1, func(fields map[string]any) (map[string]any, error) {
		currency, ok := fields["currency"]
		if !ok {
			tb.Fatalf("currency missing")
		}

		fields["amount"] = map[string]any{"cents": fields["amountCents"], "currency": currency}
		fields["reserved"] = map[string]any{"cents": fields["reservedCents"], "currency": currency}
		delete(fields, "currency")
		delete(fields, "amountCents")
		delete(fields, "reservedCents")

		return fields, nil
	})
}

func TestChainPreservesIdentity(t *testing.T) {
	t.Parallel()

	chain, err := Compile(RenameField("user.created", 1, "name", "displayName"))
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}

	source := newPayloadEvent(t, "user.created", 1, []byte(`{"name":"Lars"}`),
		event.WithCustom("origin", "test"))

	got, err := chain.upcastAll([]event.Event{source})
	if err != nil {
		t.Fatalf("upcastAll: %v", err)
	}

	upcasted := got[0]
	if upcasted == source {
		t.Fatal("upcasted event must be a NEW instance")
	}

	if upcasted.ID() != source.ID() {
		t.Error("event ID not preserved")
	}

	if !upcasted.OccurredAt().Equal(source.OccurredAt()) {
		t.Error("occurredAt not preserved")
	}

	if !reflect.DeepEqual(upcasted.Metadata(), source.Metadata()) {
		t.Error("metadata not preserved")
	}

	if upcasted.StreamID() != source.StreamID() || upcasted.Version() != source.Version() {
		t.Error("stream identity not preserved")
	}
}

func TestChainCBORStaysCBOR(t *testing.T) {
	t.Parallel()

	chain, err := Compile(AddField("user.created", 1, "country", "US"))
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}

	source := newPayloadEvent(t, "user.created", 1,
		map[string]any{"name": "Lars"},
		event.WithCodec(codec.CBORCodec{}))

	if source.Encoding() != codec.EncodingCBOR {
		t.Fatalf("test setup: source encoding = %s", source.Encoding())
	}

	got, err := chain.upcastAll([]event.Event{source})
	if err != nil {
		t.Fatalf("upcastAll: %v", err)
	}

	if got[0].Encoding() != codec.EncodingCBOR {
		t.Errorf("encoding = %s, want cbor", got[0].Encoding())
	}

	var fields map[string]any
	if err := (codec.CBORCodec{}).Decode(got[0].Payload(), &fields); err != nil {
		t.Fatalf("decode cbor: %v", err)
	}

	if fields["country"] != "US" {
		t.Errorf("country = %v, want US", fields["country"])
	}
}

func TestChainRenameTypeChainsIntoNewType(t *testing.T) {
	t.Parallel()

	chain, err := Compile(
		RenameField("user.renamed", 1, "name", "displayName"),
		RenameType("user.name_changed", "user.renamed"),
	)
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}

	evt := newPayloadEvent(t, "user.name_changed", 1, []byte(`{"name":"Lars"}`))

	got, err := chain.upcastAll([]event.Event{evt})
	if err != nil {
		t.Fatalf("upcastAll: %v", err)
	}

	if got[0].Type() != "user.renamed" {
		t.Errorf("type = %s, want user.renamed", got[0].Type())
	}

	if got[0].SchemaVersion() != 2 {
		t.Errorf(
			"schema version = %d, want 2 (rename keeps 1, field op advances)",
			got[0].SchemaVersion(),
		)
	}

	if _, renamed := decodedFields(t, got[0])["displayName"]; !renamed {
		t.Error("field rename after type rename did not apply")
	}
}

func TestChainExactMatchBeatsTypeOnly(t *testing.T) {
	t.Parallel()

	chain, err := Compile(
		AddField("user.created", 1, "country", "US"),
		RenameType("user.created", "user.registered"),
	)
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}

	v1 := newPayloadEvent(t, "user.created", 1, []byte(`{"name":"Lars"}`))
	v5 := newPayloadEvent(t, "user.created", 5, []byte(`{"name":"Lars"}`))

	got, err := chain.upcastAll([]event.Event{v1, v5})
	if err != nil {
		t.Fatalf("upcastAll: %v", err)
	}

	if got[0].Type() != "user.registered" || got[0].SchemaVersion() != 2 {
		t.Errorf("v1: type=%s v=%d, want user.registered v2", got[0].Type(), got[0].SchemaVersion())
	}

	if got[1].Type() != "user.registered" || got[1].SchemaVersion() != 5 {
		t.Errorf(
			"v5: type=%s v=%d, want user.registered v5 (rename keeps version)",
			got[1].Type(),
			got[1].SchemaVersion(),
		)
	}
}

func TestChainSplitAndDrop(t *testing.T) {
	t.Parallel()

	chain, err := Compile(
		Drop("audit.legacy_ping"),
		Split(
			"checkout.completed",
			1,
			Producing(
				"cart.checked_out",
				func(evt event.Event, fields map[string]any) (any, error) {
					return map[string]any{"cartID": fields["cartID"]}, nil
				},
			),
			Producing(
				"payment.requested",
				func(evt event.Event, fields map[string]any) (any, error) {
					return map[string]any{"totalCents": fields["totalCents"]}, nil
				},
			),
		),
	)
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}

	checkout := newPayloadEvent(
		t,
		"checkout.completed",
		1,
		[]byte(`{"cartID":"c1","totalCents":999}`),
	)
	legacy := newPayloadEvent(t, "audit.legacy_ping", 3, []byte(`{"x":1}`))

	got, err := chain.upcastAll([]event.Event{checkout, legacy})
	if err != nil {
		t.Fatalf("upcastAll: %v", err)
	}

	if len(got) != 2 {
		t.Fatalf("got %d events, want 2 (split expands, drop removes)", len(got))
	}

	if got[0].Type() != "cart.checked_out" || got[1].Type() != "payment.requested" {
		t.Errorf("split types = %s, %s", got[0].Type(), got[1].Type())
	}

	for _, output := range got {
		if output.ID() == checkout.ID() {
			t.Error("split outputs must get fresh IDs")
		}

		if output.StreamID() != checkout.StreamID() || output.Version() != checkout.Version() {
			t.Error("split outputs must inherit stream identity and position")
		}

		if output.SchemaVersion() != 2 {
			t.Errorf("split output version = %d, want 2", output.SchemaVersion())
		}
	}

	if decodedFields(t, got[0])["cartID"] != "c1" {
		t.Error("cart.checked_out payload wrong")
	}

	if decodedFields(t, got[1])["totalCents"] != float64(999) {
		t.Error("payment.requested payload wrong")
	}
}

func TestChainDecodeErrorPolicies(t *testing.T) {
	t.Parallel()

	scalarPayload := func() event.Event {
		return newPayloadEvent(t, "user.created", 1, []byte(`42`))
	}

	t.Run("fail is the default", func(t *testing.T) {
		t.Parallel()

		chain, err := Compile(RenameField("user.created", 1, "name", "displayName"))
		if err != nil {
			t.Fatalf("Compile: %v", err)
		}

		if _, err := chain.upcastAll([]event.Event{scalarPayload()}); err == nil {
			t.Fatal("want decode error, got nil")
		}
	})

	t.Run("passthrough returns original", func(t *testing.T) {
		t.Parallel()

		chain, err := Compile(RenameField("user.created", 1, "name", "displayName",
			WithDecodePolicy(PassthroughOnDecodeError)))
		if err != nil {
			t.Fatalf("Compile: %v", err)
		}

		source := scalarPayload()
		got, err := chain.upcastAll([]event.Event{source})
		if err != nil {
			t.Fatalf("upcastAll: %v", err)
		}

		if got[0].SchemaVersion() != 1 || string(got[0].Payload()) != "42" {
			t.Errorf(
				"passthrough changed the event: v%d %s",
				got[0].SchemaVersion(),
				got[0].Payload(),
			)
		}
	})

	t.Run("drop removes the event", func(t *testing.T) {
		t.Parallel()

		chain, err := Compile(RenameField("user.created", 1, "name", "displayName",
			WithDecodePolicy(DropOnDecodeError)))
		if err != nil {
			t.Fatalf("Compile: %v", err)
		}

		got, err := chain.upcastAll([]event.Event{scalarPayload()})
		if err != nil {
			t.Fatalf("upcastAll: %v", err)
		}

		if len(got) != 0 {
			t.Errorf("got %d events, want 0", len(got))
		}
	})
}

func TestChainOrderIndependence(t *testing.T) {
	t.Parallel()

	buildOps := func(reverse bool) []Op {
		ops := []Op{
			RenameType("user.name_changed", "user.renamed"),
			RenameField("user.renamed", 1, "name", "displayName"),
			AddField("user.renamed", 2, "country", "US"),
		}

		if reverse {
			ops = []Op{
				AddField("user.renamed", 2, "country", "US"),
				RenameField("user.renamed", 1, "name", "displayName"),
				RenameType("user.name_changed", "user.renamed"),
			}
		}

		return ops
	}

	results := make([]event.Event, 2)

	for i, reverse := range []bool{false, true} {
		chain, err := Compile(buildOps(reverse)...)
		if err != nil {
			t.Fatalf("Compile(reverse=%v): %v", reverse, err)
		}

		evt := newPayloadEvent(t, "user.name_changed", 1, []byte(`{"name":"Lars"}`))
		got, err := chain.upcastAll([]event.Event{evt})
		if err != nil {
			t.Fatalf("upcastAll(reverse=%v): %v", reverse, err)
		}

		results[i] = got[0]
	}

	if results[0].Type() != results[1].Type() ||
		results[0].SchemaVersion() != results[1].SchemaVersion() ||
		!reflect.DeepEqual(decodedFields(t, results[0]), decodedFields(t, results[1])) {
		t.Errorf("declaration order changed the result:\n%+v\n%+v", results[0], results[1])
	}

	if results[0].SchemaVersion() != 3 {
		t.Errorf("schema version = %d, want 3 (rename + two field ops)", results[0].SchemaVersion())
	}
}

func TestCompileRejects(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		ops  []Op
	}{
		{
			name: "duplicate exact op",
			ops:  []Op{RenameField("u.created", 1, "a", "b"), AddField("u.created", 1, "c", 1)},
		},
		{
			name: "duplicate type-only op",
			ops:  []Op{Drop("u.created"), RenameType("u.created", "u.registered")},
		},
		{
			name: "duplicate rename source",
			ops:  []Op{RenameType("a", "b"), RenameType("a", "c")},
		},
		{
			name: "duplicate rename target",
			ops:  []Op{RenameType("a", "c"), RenameType("b", "c")},
		},
		{
			name: "self rename",
			ops:  []Op{RenameType("a", "a")},
		},
		{
			name: "rename cycle",
			ops:  []Op{RenameType("a", "b"), RenameType("b", "a")},
		},
		{
			name: "rename chain cycle",
			ops:  []Op{RenameType("a", "b"), RenameType("b", "c"), RenameType("c", "a")},
		},
		{
			name: "empty type",
			ops:  []Op{RenameField("", 1, "a", "b")},
		},
		{
			name: "zero version",
			ops:  []Op{AddField("u.created", 0, "a", 1)},
		},
		{
			name: "empty field",
			ops:  []Op{RemoveField("u.created", 1, "")},
		},
		{
			name: "nil transform",
			ops:  []Op{Transform("u.created", 1, nil)},
		},
		{
			name: "split without outputs",
			ops:  []Op{Split("u.created", 1)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if _, err := Compile(tt.ops...); err == nil {
				t.Fatal("Compile accepted invalid ops")
			}
		})
	}
}

func TestUpcastersBridge(t *testing.T) {
	t.Parallel()

	t.Run("single ops convert and run", func(t *testing.T) {
		t.Parallel()

		chain, err := Compile(
			RenameField("user.created", 1, "name", "displayName"),
			Transform("balance.updated", 1, func(fields map[string]any) (map[string]any, error) {
				fields["derived"] = true

				return fields, nil
			}),
		)
		if err != nil {
			t.Fatalf("Compile: %v", err)
		}

		upcasters, err := chain.Upcasters()
		if err != nil {
			t.Fatalf("Upcasters: %v", err)
		}

		if len(upcasters) != 2 {
			t.Fatalf("got %d upcasters, want 2", len(upcasters))
		}

		transform := UpcastSourceTransform(upcasters...)
		evt := newPayloadEvent(t, "user.created", 1, []byte(`{"name":"Lars"}`))

		got, err := transform([]event.Event{evt})
		if err != nil {
			t.Fatalf("transform: %v", err)
		}

		if got[0].SchemaVersion() != 2 {
			t.Errorf("schema version = %d, want 2", got[0].SchemaVersion())
		}

		if _, ok := decodedFields(t, got[0])["displayName"]; !ok {
			t.Error("rename did not apply through the Upcaster bridge")
		}
	})

	t.Run("batch ops are rejected", func(t *testing.T) {
		t.Parallel()

		for name, ops := range map[string][]Op{
			"split":  {Split("a", 1, Producing("b", func(event.Event, map[string]any) (any, error) { return nil, nil }))},
			"drop":   {Drop("a")},
			"rename": {RenameType("a", "b")},
		} {
			t.Run(name, func(t *testing.T) {
				t.Parallel()

				chain, err := Compile(ops...)
				if err != nil {
					t.Fatalf("Compile: %v", err)
				}

				if _, err := chain.Upcasters(); err == nil {
					t.Fatal("Upcasters accepted a batch-only op")
				}
			})
		}
	})

	t.Run("drop policy is rejected", func(t *testing.T) {
		t.Parallel()

		chain, err := Compile(RenameField("a", 1, "x", "y",
			WithDecodePolicy(DropOnDecodeError)))
		if err != nil {
			t.Fatalf("Compile: %v", err)
		}

		if _, err := chain.Upcasters(); err == nil {
			t.Fatal("Upcasters accepted DropOnDecodeError")
		}
	})
}

func TestChainEmptyIsIdentity(t *testing.T) {
	t.Parallel()

	chain, err := Compile()
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}

	evt := newPayloadEvent(t, "user.created", 1, []byte(`{"name":"Lars"}`))

	got, err := chain.upcastAll([]event.Event{evt})
	if err != nil {
		t.Fatalf("upcastAll: %v", err)
	}

	if len(got) != 1 || got[0] != evt {
		t.Fatal("empty chain must be the identity transform")
	}
}
