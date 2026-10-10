// Package main: graph-native reads on the modern system + metaengine stack.
//
// A social follow network: the write side is two guarded commands per user
// stream; the read side folds the SAME events into graph edges and answers
// reachability traversals ("who is within N hops of alice?"). Retraction is a
// domain event (ADR-0114): user.unfollowed folds to metaengine.EdgeRemoval and
// the edge disappears from every traversal.
//
// Pipeline: Command → Decider (guards) → Event Store → Projection Host →
// metaengine Graph ADT → traversal query. Swap engines in buildDeployment
// ONLY — the domain and folds never change.
package main

import (
	"slices"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/decider/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

const (
	evtUserFollowed   = event.Type("user.followed")
	evtUserUnfollowed = event.Type("user.unfollowed")

	cmdFollow   = command.Type("user.follow")
	cmdUnfollow = command.Type("user.unfollow")

	streamType = "User"
)

// FollowedPayload is the fact, self-describing: who followed whom. Node
// names live in the payload (NOT derived from the stream ID — StreamID.String
// is a brand-prefixed display form, not a clean node name).
type FollowedPayload struct {
	Follower string `json:"follower"`
	Followee string `json:"followee"`
}

type UnfollowedPayload struct {
	Follower string `json:"follower"`
	Followee string `json:"followee"`
}

// FollowState is the replayed per-follower state the guards run against.
type FollowState struct {
	Followees []string
}

func (s FollowState) follows(user string) bool {
	return slices.Contains(s.Followees, user)
}

// applyFollowState folds this stream's own events back into state.
func applyFollowState(state FollowState, evt event.Event) (FollowState, error) {
	switch evt.Type() {
	case evtUserFollowed:
		payload, err := event.DecodePayloadAuto[FollowedPayload](evt)
		if err != nil {
			return state, err
		}

		if !state.follows(payload.Followee) {
			state.Followees = append(state.Followees, payload.Followee)
		}
	case evtUserUnfollowed:
		payload, err := event.DecodePayloadAuto[UnfollowedPayload](evt)
		if err != nil {
			return state, err
		}

		state.Followees = slices.DeleteFunc(
			state.Followees,
			func(candidate string) bool { return candidate == payload.Followee },
		)
	}

	return state, nil
}

// follow decides a Followed fact. Guards: no self-follow, no duplicate edge.
func follow(follower id.StreamID, followee string) decider.DecideFunc[FollowState] {
	return func(state FollowState, version event.Version) ([]event.Event, error) {
		if follower.Get() == followee {
			return nil, errorfamily.NewRejection("follow.self", "cannot follow yourself")
		}

		if state.follows(followee) {
			return nil, errorfamily.NewRejection("follow.duplicate", "already following "+followee)
		}

		evt, err := event.New(evtUserFollowed, follower, streamType,
			version.Increment(), FollowedPayload{Follower: follower.Get(), Followee: followee})
		if err != nil {
			return nil, err
		}

		return []event.Event{evt}, nil
	}
}

// unfollow decides an Unfollowed fact. Guard: the edge must exist.
func unfollow(follower id.StreamID, followee string) decider.DecideFunc[FollowState] {
	return func(state FollowState, version event.Version) ([]event.Event, error) {
		if !state.follows(followee) {
			return nil, errorfamily.NewConflict("unfollow.missing", "not following "+followee)
		}

		evt, err := event.New(evtUserUnfollowed, follower, streamType,
			version.Increment(), UnfollowedPayload{Follower: follower.Get(), Followee: followee})
		if err != nil {
			return nil, err
		}

		return []event.Event{evt}, nil
	}
}

// FollowCmd is the write-side API: dispatch with a follower's stream ID.
type FollowCmd struct {
	*command.BasicCommand

	Followee string
}

// UnfollowCmd retracts a follow edge (ADR-0114 deletion-as-event).
type UnfollowCmd struct {
	*command.BasicCommand

	Followee string
}

// userStream derives the follower's stream ID from their name — semantic,
// caller-chosen keys are string-backed StreamIDs (ADR-0111 identity rules).
func userStream(name string) (id.StreamID, error) {
	return id.ParseStreamID(name)
}
