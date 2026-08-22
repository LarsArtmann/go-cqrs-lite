package record_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/record/v4"
)

func TestActorKind_String(t *testing.T) {
	t.Parallel()

	cases := []struct {
		kind record.ActorKind
		want string
	}{
		{record.ActorUnknown, "unknown"},
		{record.ActorUser, "user"},
		{record.ActorBot, "bot"},
		{record.ActorSystem, "system"},
		{record.ActorService, "service"},
		{record.ActorKind(99), "unknown"},
	}

	for _, tc := range cases {
		if got := tc.kind.String(); got != tc.want {
			t.Errorf("ActorKind(%d).String() = %q, want %q", tc.kind, got, tc.want)
		}
	}
}

func TestActor_ZeroValueIsUnknown(t *testing.T) {
	t.Parallel()

	var a record.Actor
	if !a.IsZero() {
		t.Error("zero Actor must IsZero()")
	}

	if a.Kind != record.ActorUnknown || a.Raw != "" {
		t.Errorf("zero Actor = %+v, want {ActorUnknown \"\"}", a)
	}

	if a.String() != "" {
		t.Errorf("zero Actor String() = %q, want empty", a.String())
	}
}

func TestActor_StringWireFormMatchesActorID(t *testing.T) {
	t.Parallel()

	cases := []struct {
		actor record.Actor
		want  string
	}{
		{record.Actor{Kind: record.ActorUser, Raw: "01JUSER"}, "user:01JUSER"},
		{record.Actor{Kind: record.ActorBot, Raw: "indexer"}, "bot:indexer"},
		{record.Actor{Kind: record.ActorSystem, Raw: "scheduler"}, "system:scheduler"},
		{record.Actor{Kind: record.ActorService, Raw: "api-gateway"}, "service:api-gateway"},
	}

	for _, tc := range cases {
		if got := tc.actor.String(); got != tc.want {
			t.Errorf("String() = %q, want %q (id.ActorID wire form)", got, tc.want)
		}
	}
}
