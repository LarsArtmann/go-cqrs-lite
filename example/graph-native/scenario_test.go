package main

import (
	"context"
	"slices"
	"testing"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/systemscenario/v4"
)

// The BDD suite (ADR-0153): the SAME DomainConfig the binary boots, driven
// through Given/When/Then over a real composition root. Covers the graph
// behaviors the printed demo asserts: multi-hop traversal and edge
// retraction, plus the decider guards.

func TestFollowGraph_TraversalAndRetraction(t *testing.T) {
	ctx := context.Background()
	sc := systemscenario.System(t, ctx, graphDomain(), systemscenario.Memory())

	sc.When(followCmd(t, "alice", "bob")).
		Command(followCmd(t, "bob", "carol")).
		Command(followCmd(t, "carol", "dave")).
		Command(followCmd(t, "alice", "carol")).
		Then("user.followed", "user.followed", "user.followed", "user.followed")

	reach := func(query ReachabilityQuery) func() ([]string, error) {
		return func() ([]string, error) {
			return metaengine.ExecuteTyped[ReachabilityQuery, []string](
				ctx, sc.System().MetaEngine(), query)
		}
	}

	systemscenario.ThenQueryTyped(sc.Phase(),
		reach(ReachabilityQuery{Node: "alice", Depth: 1}),
		func(got []string) error { return wantReach(got, "bob", "carol") })

	systemscenario.ThenQueryTyped(sc.Phase(),
		reach(ReachabilityQuery{Node: "alice", Depth: 2}),
		func(got []string) error { return wantReach(got, "bob", "carol", "dave") })

	// Retraction: carol unfollows dave → the EdgeRemoval fold deletes exactly
	// that edge, and dave leaves every traversal that used it. (Then diffs
	// against the scenario's first-act baseline, so the journal shows all
	// five events.)
	sc.When(unfollowCmd(t, "carol", "dave")).
		Then("user.followed", "user.followed", "user.followed", "user.followed", "user.unfollowed")

	systemscenario.ThenQueryTyped(sc.Phase(),
		reach(ReachabilityQuery{Node: "alice", Depth: 2}),
		func(got []string) error { return wantReach(got, "bob", "carol") })

	systemscenario.ThenQueryTyped(sc.Phase(),
		reach(ReachabilityQuery{Node: "carol", Depth: 1}),
		func(got []string) error { return wantReach(got) })
}

func TestFollowGraph_Guards(t *testing.T) {
	ctx := context.Background()
	sc := systemscenario.System(t, ctx, graphDomain(), systemscenario.Memory())

	sc.When(followCmd(t, "alice", "bob")).Then("user.followed")

	sc.When(followCmd(t, "alice", "alice")).
		ThenErrorFamily(errorfamily.Rejection)

	sc.When(followCmd(t, "alice", "bob")).
		ThenErrorFamily(errorfamily.Rejection)

	sc.When(unfollowCmd(t, "alice", "carol")).
		ThenErrorFamily(errorfamily.Conflict)
}

func followCmd(t *testing.T, follower, followee string) command.Command {
	t.Helper()

	return newFollowCmd(t, cmdFollow, follower, followee)
}

func unfollowCmd(t *testing.T, follower, followee string) command.Command {
	t.Helper()

	return newFollowCmd(t, cmdUnfollow, follower, followee)
}

func newFollowCmd(t *testing.T, cmdType command.Type, follower, followee string) command.Command {
	t.Helper()

	stream, err := userStream(follower)
	if err != nil {
		t.Fatalf("parse stream %q: %v", follower, err)
	}

	basic, err := command.New(cmdType, stream)
	if err != nil {
		t.Fatalf("new command %q: %v", cmdType, err)
	}

	if cmdType == cmdFollow {
		return FollowCmd{BasicCommand: basic, Followee: followee}
	}

	return UnfollowCmd{BasicCommand: basic, Followee: followee}
}

// wantReach compares against the sorted expected reachable set.
func wantReach(got []string, want ...string) error {
	slices.Sort(got)
	slices.Sort(want)

	if slices.Equal(got, want) {
		return nil
	}

	return &reachMismatchError{got: got, want: want}
}

type reachMismatchError struct {
	got  []string
	want []string
}

func (e *reachMismatchError) Error() string {
	return "reachable set mismatch: want " + sprintSet(e.want) + ", got " + sprintSet(e.got)
}

func sprintSet(values []string) string {
	if len(values) == 0 {
		return "(nobody)"
	}

	out := "["
	for i, value := range values {
		if i > 0 {
			out += " "
		}

		out += value
	}

	return out + "]"
}
