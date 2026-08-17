package sqlstore

import (
	"testing"

	errorfamily "github.com/larsartmann/go-error-family"
)

type decodeTestPayload struct {
	Action string `json:"action"`
	Amount int    `json:"amount"`
}

// TestDecodeTimerPayload locks the payload-column decoder contract: v1
// envelopes decode with the actor, legacy bare-P rows decode with an empty
// actor, and the dual-key probe (v==1 AND payload key) rejects single-key
// lookalikes so a legacy payload carrying its own "v" field is not misread.
func TestDecodeTimerPayload(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		data       string
		wantVer    int
		wantActor  string
		wantAction string
	}{
		{
			name:       "v1 envelope with actor",
			data:       `{"v":1,"actor":"user:01HK1540X0841Y0A6BSX1VKR99","payload":{"action":"cancel","amount":3}}`,
			wantVer:    1,
			wantActor:  "user:01HK1540X0841Y0A6BSX1VKR99",
			wantAction: "cancel",
		},
		{
			name:       "v1 envelope without actor field",
			data:       `{"v":1,"payload":{"action":"remind","amount":7}}`,
			wantVer:    1,
			wantAction: "remind",
		},
		{
			name:       "legacy bare payload object",
			data:       `{"action":"remind","amount":7}`,
			wantAction: "remind",
		},
		{
			name: "legacy payload carrying only a v field is not misread as envelope",
			// Probe requires BOTH keys: without "payload" this decodes as P.
			data:       `{"v":1}`,
			wantAction: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			env, err := decodeTimerPayload[decodeTestPayload]("probe-timer", []byte(tt.data))
			if err != nil {
				t.Fatalf("decodeTimerPayload(%s): %v", tt.name, err)
			}

			if env.Version != tt.wantVer {
				t.Errorf("Version: got %d, want %d", env.Version, tt.wantVer)
			}

			if env.Actor != tt.wantActor {
				t.Errorf("Actor: got %q, want %q", env.Actor, tt.wantActor)
			}

			if env.Payload.Action != tt.wantAction {
				t.Errorf("Payload.Action: got %q, want %q", env.Payload.Action, tt.wantAction)
			}
		})
	}
}

// TestDecodeTimerPayload_CorruptionClassified proves undecodable rows surface
// as Corruption-family errors naming the offending timer, on both decoder
// paths (envelope and legacy fallback).
func TestDecodeTimerPayload_CorruptionClassified(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		data string
	}{
		{
			name: "envelope payload type mismatch",
			data: `{"v":1,"payload":{"action":42}}`,
		},
		{
			name: "legacy payload type mismatch",
			data: `{"action":42}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := decodeTimerPayload[decodeTestPayload]("corrupt-timer", []byte(tt.data))
			if err == nil {
				t.Fatalf("%s: expected error, got nil", tt.name)
			}

			if got := errorfamily.Classify(err); got != errorfamily.Corruption {
				t.Errorf("%s: family: got %v, want Corruption", tt.name, got)
			}
		})
	}
}
