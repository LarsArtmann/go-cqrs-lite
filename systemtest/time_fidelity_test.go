package systemtest_test

// GitHub #50: time.Time payload fidelity through system + sqliteengine.
//
// Traced mechanism (2026-10-04; this file pins it):
//
//	The DEFAULT event codec (go-codec CBOR) encodes fractional-seconds
//	time.Time values as float64 unix seconds (cbor.TimeUnixDynamic in
//	go-codec's canonicalEncMode). float64 mantissa at 2026-era epochs
//	(~1.76e9 s, 31 integer bits) leaves ~22 fraction bits ≈ ≤256 ns
//	quantization per round-trip — measured 165 ns single-hop, ≤611 ns in
//	the consumer repro across multiple hops. The sqliteengine journal
//	(encodeStreamValue → encoding/json/v2 → RFC3339 string) is EXACT; the
//	loss enters at the CBOR event hop BEFORE the journal. The JSON event
//	codec is nano-exact end-to-end, and ADR-0056 is amended accordingly
//	(its "(CBOR TimeUnixDynamic preserves nanos)" parenthetical was wrong).
//
// The iroh loopback transport's TimeUnixDynamic opEncMode is a DIFFERENT,
// deliberate choice (LWW sub-second ordering survives float64) and is fine.

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/larsartmann/go-codec"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/record/v4"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
)

// nanoDriftTolerance bounds the CBOR-default drift: measured ≤611 ns
// multi-hop; float64 theory caps a single hop at ~256 ns. Sub-microsecond
// is the contract until the go-codec default flips to TimeRFC3339Nano
// (decode-compatible, 0 ns, 36-byte wire — verified 2026-10-04).
const nanoDriftTolerance = 1000 * time.Nanosecond

type nanoView struct {
	Title string
	At    time.Time
}

type findNano struct {
	ID string
}

// autoPayloadDecoder decodes task.created payloads regardless of event codec,
// so the CBOR-default and JSON legs share one projection decoder.
func autoPayloadDecoder(eventType string, payload []byte) (any, error) {
	if eventType != "task.created" {
		return nil, errors.New("unknown event type: " + eventType)
	}

	var e TaskCreated
	switch codec.AutoDetect(payload) {
	case codec.EncodingCBOR:
		if err := (codec.CBORCodec{}).Decode(payload, &e); err != nil {
			return nil, fmt.Errorf("cbor decode task.created: %w", err)
		}
	default:
		if err := (codec.JSONCodec{}).Decode(payload, &e); err != nil {
			return nil, fmt.Errorf("json decode task.created: %w", err)
		}
	}

	return e, nil
}

// TestSystem_TimePayloadFidelity_SQLite drives task.created events carrying a
// deterministic 123456789 ns timestamp through the full system + sqliteengine
// path (dispatch → event store → projection → metaengine read) and asserts:
//
//	json event codec → view.At nano-EXACT (the journal is innocent)
//	cbor default     → |drift| < nanoDriftTolerance, measured drift logged
func TestSystem_TimePayloadFidelity_SQLite(t *testing.T) {
	t.Parallel()

	orig := time.Date(2026, 10, 4, 2, 55, 0, 123456789, time.UTC)

	for _, leg := range []struct {
		name      string
		title     string
		jsonCodec bool
		exact     bool
	}{
		{"json codec exact end to end", "nano-json", true, true},
		{"cbor default bounded sub-microsecond", "nano-cbor", false, false},
	} {
		t.Run(leg.name, func(t *testing.T) {
			t.Parallel()

			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()

			nanoQuery := metaengine.Query[findNano, nanoView](
				"nano_views_"+leg.title,
				metaengine.OnRecordTyped(
					"task.created",
					TaskCreated{},
					func(_ record.Record, e TaskCreated) (string, nanoView) {
						return e.Title, nanoView{Title: e.Title, At: e.At}
					},
				),
			)

			var codecOpt []event.Option
			if leg.jsonCodec {
				codecOpt = append(codecOpt, event.WithCodec(codec.JSONCodec{}))
			}

			domain := system.DomainConfig{
				Commands: func(sys *system.System) {
					system.RegisterDecider(sys, "Task", TaskDecider)

					system.RegisterCommand[*command.BasicCommand, TaskState](sys, "task.create",
						func(ctx context.Context, cmd *command.BasicCommand) system.Op[TaskState] {
							return system.Execute(ctx, cmd.StreamID(), "Task",
								func(state TaskState, ver event.Version) ([]event.Event, error) {
									if state.Exists {
										return nil, errors.New("task already exists")
									}

									return []event.Event{mustEvent(event.New("task.created",
										cmd.StreamID(), "Task", ver+1,
										TaskCreated{Title: leg.title, At: orig},
										codecOpt...))}, nil
								})
						})
				},
				Projections:       []system.ProjectionDeclaration{system.RawQuery(nanoQuery)},
				ProjectionDecoder: autoPayloadDecoder,
			}

			sys, err := system.New(ctx, domain, sqliteDeployment(t))
			if err != nil {
				t.Fatalf("system.New: %v", err)
			}
			defer sys.Close()

			streamID := id.NewStreamID()
			if err := sys.CommandDispatcher().
				Dispatch(ctx, newCmd("task.create", streamID)); err != nil {
				t.Fatalf("dispatch create: %v", err)
			}

			// Event-store hop: load back and decode through the event codec.
			loaded, err := sys.EventStore().Load(ctx, id.NewStreamRef("Task", streamID))
			if err != nil {
				t.Fatalf("load events: %v", err)
			}
			if len(loaded) != 1 {
				t.Fatalf("expected 1 event, got %d", len(loaded))
			}

			decoded, err := event.DecodePayloadAuto[TaskCreated](loaded[0])
			if err != nil {
				t.Fatalf("decode payload: %v", err)
			}
			hopDrift := decoded.At.Sub(orig)
			t.Logf("event-store hop drift: %d ns", hopDrift)

			if err := sys.Start(ctx); err != nil {
				t.Fatalf("system.Start: %v", err)
			}

			deadline := time.Now().Add(10 * time.Second)
			for time.Now().Before(deadline) {
				for _, s := range sys.ProjectionHost().Status() {
					if s.Processed >= 1 && s.Errors == 0 {
						result, err := sys.MetaEngine().Execute(findNano{ID: leg.title})
						if err != nil {
							t.Fatalf("store.Execute: %v", err)
						}

						view, ok := result.(nanoView)
						if !ok {
							t.Fatalf("expected nanoView, got %T", result)
						}

						drift := view.At.Sub(orig)
						t.Logf("end-to-end view drift: %d ns (want %s)",
							drift, driftWord(leg.exact))

						if leg.exact && drift != 0 {
							t.Fatalf("JSON codec must be nano-exact, drifted %d ns", drift)
						}
						if !leg.exact && (drift < -nanoDriftTolerance || drift > nanoDriftTolerance) {
							t.Fatalf("CBOR default drift %d ns exceeds %v bound", drift, nanoDriftTolerance)
						}

						return
					}
				}

				select {
				case <-ctx.Done():
					t.Fatal("timeout waiting for projection")
				case <-time.After(50 * time.Millisecond):
				}
			}

			for _, s := range sys.ProjectionHost().Status() {
				if s.Errors > 0 {
					t.Fatalf("projection %q has %d errors", s.Name, s.Errors)
				}
			}

			t.Fatal("projection did not process event within timeout")
		})
	}
}

// TestTimeCodecFidelity pins the codec-level mechanism without the system:
// the go-codec CBOR default quantizes fractional seconds (float64 unix),
// JSON is exact.
func TestTimeCodecFidelity(t *testing.T) {
	t.Parallel()

	orig := time.Date(2026, 10, 4, 2, 55, 0, 123456789, time.UTC)

	cborPayload, err := (codec.CBORCodec{}).Encode(TaskCreated{Title: "t", At: orig})
	if err != nil {
		t.Fatalf("cbor encode: %v", err)
	}

	var cborBack TaskCreated
	if err := (codec.CBORCodec{}).Decode(cborPayload, &cborBack); err != nil {
		t.Fatalf("cbor decode: %v", err)
	}
	cborDrift := cborBack.At.Sub(orig)
	t.Logf("cbor single-hop drift: %d ns (float64 unix quantization)", cborDrift)
	if cborDrift == 0 {
		t.Log("NOTE: cbor drift measured 0 — go-codec default may have flipped to a nano-exact time mode")
	}
	if cborDrift < -nanoDriftTolerance || cborDrift > nanoDriftTolerance {
		t.Fatalf("cbor drift %d ns exceeds %v bound", cborDrift, nanoDriftTolerance)
	}

	jsonPayload, err := (codec.JSONCodec{}).Encode(TaskCreated{Title: "t", At: orig})
	if err != nil {
		t.Fatalf("json encode: %v", err)
	}

	var jsonBack TaskCreated
	if err := (codec.JSONCodec{}).Decode(jsonPayload, &jsonBack); err != nil {
		t.Fatalf("json decode: %v", err)
	}
	if d := jsonBack.At.Sub(orig); d != 0 {
		t.Fatalf("json codec must be nano-exact, drifted %d ns", d)
	}
}

func driftWord(exact bool) string {
	if exact {
		return "exact"
	}

	return "sub-microsecond"
}
