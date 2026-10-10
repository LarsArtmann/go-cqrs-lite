package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"slices"
	"time"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
)

// The runtime loop: start the projection host, build a small follow network,
// traverse it, retract an edge, and traverse again. Every printed claim is
// asserted — the program exits non-zero if any expectation fails.

const (
	traversalDeadline = 5 * time.Second
	traversalPoll     = 20 * time.Millisecond
)

var errTraversalConverge = errors.New("traversal did not converge")

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	sqliteDSN := flag.String("sqlite", "", "sqlite DSN for the source-of-truth engine")
	dgraphAddr := flag.String("dgraph", "", "route projections to Dgraph at this gRPC address")
	configPath := flag.String("config", "", "boot the deployment from this cqrs.yaml")

	flag.Parse()

	ctx := context.Background()

	deployment, err := buildDeployment(deploymentOptions{
		sqliteDSN:  *sqliteDSN,
		dgraphAddr: *dgraphAddr,
		configPath: *configPath,
	})
	if err != nil {
		return err
	}

	sys, err := buildSystem(ctx, deployment)
	if err != nil {
		return err
	}
	defer metaengine.DeferClose(sys)

	hostCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	go func() { _ = sys.ProjectionHost().Start(hostCtx) }()

	return runScenario(ctx, sys)
}

// runScenario drives the demo graph:
//
//	alice→bob, bob→carol, carol→dave, alice→carol
//
// depth-1 from alice sees bob+carol; depth-2 adds dave (via carol). After
// carol unfollows dave the retraction fold removes that edge and depth-2
// drops dave — deletion-as-domain-event, visible in the read model.
func runScenario(ctx context.Context, sys *system.System) error {
	seedFollows(ctx, sys)
	demonstrateGuards(ctx, sys)

	reach, err := awaitTraversal(ctx, sys,
		ReachabilityQuery{Node: "alice", Depth: 1}, []string{"bob", "carol"})
	if err != nil {
		return err
	}

	fmt.Printf("alice depth-1 reaches: %v\n", reach)

	reach, err = awaitTraversal(ctx, sys,
		ReachabilityQuery{Node: "alice", Depth: 2}, []string{"bob", "carol", "dave"})
	if err != nil {
		return err
	}

	fmt.Printf("alice depth-2 reaches: %v\n", reach)

	if err := dispatchUnfollow(ctx, sys, "carol", "dave"); err != nil {
		return err
	}

	reach, err = awaitTraversal(ctx, sys,
		ReachabilityQuery{Node: "alice", Depth: 2}, []string{"bob", "carol"})
	if err != nil {
		return err
	}

	fmt.Printf("alice depth-2 after unfollow(carol→dave): %v (dave retracted)\n", reach)

	return nil
}

func seedFollows(ctx context.Context, sys *system.System) {
	for _, pair := range [][2]string{
		{"alice", "bob"}, {"bob", "carol"}, {"carol", "dave"}, {"alice", "carol"},
	} {
		if err := dispatchFollow(ctx, sys, pair[0], pair[1]); err != nil {
			log.Fatalf("seed follow %s→%s: %v", pair[0], pair[1], err)
		}
	}
}

// demonstrateGuards proves the decider rejections: self-follow and duplicate.
func demonstrateGuards(ctx context.Context, sys *system.System) {
	if err := dispatchFollow(ctx, sys, "alice", "alice"); err == nil {
		log.Fatal("guard demo: self-follow unexpectedly accepted")
	} else {
		fmt.Printf("guard: follow(alice, alice) rejected: %v\n", err)
	}

	if err := dispatchFollow(ctx, sys, "alice", "bob"); err == nil {
		log.Fatal("guard demo: duplicate follow unexpectedly accepted")
	} else {
		fmt.Printf("guard: follow(alice, bob) again rejected: %v\n", err)
	}
}

func dispatchFollow(ctx context.Context, sys *system.System, follower, followee string) error {
	stream, err := userStream(follower)
	if err != nil {
		return err
	}

	basic, err := command.New(cmdFollow, stream)
	if err != nil {
		return err
	}

	return sys.CommandDispatcher().
		Dispatch(ctx, FollowCmd{BasicCommand: basic, Followee: followee})
}

func dispatchUnfollow(ctx context.Context, sys *system.System, follower, followee string) error {
	stream, err := userStream(follower)
	if err != nil {
		return err
	}

	basic, err := command.New(cmdUnfollow, stream)
	if err != nil {
		return err
	}

	return sys.CommandDispatcher().
		Dispatch(ctx, UnfollowCmd{BasicCommand: basic, Followee: followee})
}

// awaitTraversal polls until the projection converges on the expected
// reachable set — the host applies journal events asynchronously.
func awaitTraversal(
	ctx context.Context,
	sys *system.System,
	query ReachabilityQuery,
	want []string,
) ([]string, error) {
	slices.Sort(want)

	deadline := time.Now().Add(traversalDeadline)

	var got []string

	for {
		result, err := sys.MetaEngine().ExecuteCtx(ctx, query)
		if err != nil {
			return nil, fmt.Errorf("traverse %s depth %d: %w", query.Node, query.Depth, err)
		}

		neighbors, ok := result.([]any)
		if !ok {
			return nil, fmt.Errorf("%w: %T", errUnexpectedResult, result)
		}

		got = toStrings(neighbors)

		if slices.Equal(got, want) {
			return got, nil
		}

		if time.Now().After(deadline) {
			return nil, fmt.Errorf("%w: %s depth %d: got %v, want %v",
				errTraversalConverge, query.Node, query.Depth, got, want)
		}

		time.Sleep(traversalPoll)
	}
}

var errUnexpectedResult = errors.New("unexpected traversal result type")

func toStrings(values []any) []string {
	out := make([]string, len(values))
	for i, value := range values {
		out[i] = fmt.Sprint(value)
	}

	slices.Sort(out)

	return out
}
