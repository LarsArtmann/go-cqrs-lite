package metadata_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/metadata/v4"
	"github.com/larsartmann/go-cqrs-lite/record/v4"
)

func TestRecordActor_KindDiscriminatedWins(t *testing.T) {
	t.Parallel()

	tracing := metadata.Tracing{
		UserID:  id.NewUserID(),
		ActorID: id.NewSystemActor("migration"),
	}

	got := metadata.RecordActor(tracing)
	want := record.Actor{Kind: record.ActorSystem, Raw: "migration"}
	if got != want {
		t.Errorf("RecordActor = %+v, want %+v", got, want)
	}
}

func TestRecordActor_UserIDFallbackIsActorUser(t *testing.T) {
	t.Parallel()

	userID := id.NewUserID()
	got := metadata.RecordActor(metadata.Tracing{UserID: userID})

	want := record.Actor{Kind: record.ActorUser, Raw: userID.String()}
	if got != want {
		t.Errorf("RecordActor = %+v, want %+v (a user ID is a human user)", got, want)
	}
}

func TestRecordActor_AllKindsMap(t *testing.T) {
	t.Parallel()

	cases := []struct {
		actor id.ActorID
		want  record.ActorKind
	}{
		{id.NewUserActor(id.NewUserID()), record.ActorUser},
		{id.NewBotActor("bot-1"), record.ActorBot},
		{id.NewSystemActor("gc"), record.ActorSystem},
		{id.NewServiceActor("svc"), record.ActorService},
	}

	for _, tc := range cases {
		got := metadata.RecordActor(metadata.Tracing{ActorID: tc.actor})
		if got.Kind != tc.want {
			t.Errorf("kind for %v = %v, want %v", tc.actor, got.Kind, tc.want)
		}

		if got.Raw != tc.actor.Raw() {
			t.Errorf("raw for %v = %q, want %q", tc.actor, got.Raw, tc.actor.Raw())
		}
	}
}

func TestRecordActor_ZeroTracingYieldsZeroActor(t *testing.T) {
	t.Parallel()

	if got := metadata.RecordActor(metadata.Tracing{}); !got.IsZero() {
		t.Errorf("RecordActor(zero Tracing) = %+v, want zero Actor", got)
	}
}

func TestRecordActor_MatchesActorString(t *testing.T) {
	t.Parallel()

	tracing := metadata.Tracing{
		UserID:  id.NewUserID(),
		ActorID: id.NewServiceActor("api-gateway"),
	}

	if got, want := metadata.RecordActor(tracing).String(), metadata.ActorString(tracing); got != want {
		t.Errorf("RecordActor.String() = %q, want %q (must match ActorString wire form)", got, want)
	}

	legacy := metadata.Tracing{UserID: id.NewUserID()}
	if got, want := metadata.RecordActor(legacy).String(), metadata.ActorString(legacy); got != want {
		t.Errorf("legacy fallback RecordActor.String() = %q, want %q", got, want)
	}
}
