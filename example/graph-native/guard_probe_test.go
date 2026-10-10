package main

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

func TestGuardProbe(t *testing.T) {
	follower, _ := id.ParseStreamID("alice")
	decide := follow(follower, "alice")
	_, err := decide(FollowState{}, event.Version(0))
	t.Logf("self-follow err: %v", err)
	if err == nil {
		t.Fatal("self-follow not rejected by decide func")
	}
}
