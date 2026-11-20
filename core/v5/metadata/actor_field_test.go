package metadata_test

import (
	"bytes"
	"encoding/json/v2"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/core/v5/id"
	"github.com/larsartmann/go-cqrs-lite/core/v5/metadata"
	"github.com/larsartmann/go-cqrs-lite/core/v5/record"
)

func TestTracing_Actor_JSONRoundTrip(t *testing.T) {
	t.Parallel()

	tracing := metadata.Tracing{
		Actor: record.Actor{Kind: record.ActorService, Raw: "api-gateway"},
	}

	data, err := json.Marshal(tracing)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var decoded metadata.Tracing
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if decoded.Actor != tracing.Actor {
		t.Errorf("Actor = %v, want %v across the JSON edge", decoded.Actor, tracing.Actor)
	}
}

func TestTracing_Actor_ZeroStaysOffWire(t *testing.T) {
	t.Parallel()

	data, err := json.Marshal(metadata.Tracing{})
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	if bytes.Contains(data, []byte("actor")) {
		t.Errorf(
			"zero Tracing marshals to %s — the absent actor must stay off the wire (omitzero)",
			data,
		)
	}
}

// TestRecordActor_FieldWins pins THE precedence rule (review P3/R7): the
// structural field beats the stringly twin; the twin beats the UserID
// fallback.
func TestRecordActor_FieldWins(t *testing.T) {
	t.Parallel()

	t.Run("field wins over twin", func(t *testing.T) {
		t.Parallel()

		tracing := metadata.Tracing{
			ActorID: id.NewSystemActor("scheduler"),
			Actor:   record.Actor{Kind: record.ActorUser, Raw: "01JUSER"},
		}

		want := record.Actor{Kind: record.ActorUser, Raw: "01JUSER"}
		if got := metadata.RecordActor(tracing); got != want {
			t.Errorf("RecordActor = %v, want the structural field %v", got, want)
		}
	})

	t.Run("twin parsed at the edge", func(t *testing.T) {
		t.Parallel()

		tracing := metadata.Tracing{ActorID: id.NewSystemActor("scheduler")}

		want := record.Actor{Kind: record.ActorSystem, Raw: "scheduler"}
		if got := metadata.RecordActor(tracing); got != want {
			t.Errorf("RecordActor = %v, want the twin-derived %v", got, want)
		}
	})

	t.Run("legacy UserID is an ActorUser", func(t *testing.T) {
		t.Parallel()

		userID := id.NewUserID()
		tracing := metadata.Tracing{UserID: userID}

		want := record.Actor{Kind: record.ActorUser, Raw: userID.String()}
		if got := metadata.RecordActor(tracing); got != want {
			t.Errorf("RecordActor = %v, want the UserID fallback %v", got, want)
		}
	})
}

func TestTracing_MergeAndIsZero_CarryActor(t *testing.T) {
	t.Parallel()

	actor := record.Actor{Kind: record.ActorBot, Raw: "importer"}

	merged := metadata.Tracing{CorrelationID: id.NewCorrelationID()}.Merge(
		metadata.Tracing{Actor: actor})
	if merged.Actor != actor {
		t.Errorf("Merge dropped the structural actor: %v", merged.Actor)
	}

	if (metadata.Tracing{Actor: actor}).IsZero() {
		t.Error("a Tracing with only a structural actor must not be zero")
	}
}
