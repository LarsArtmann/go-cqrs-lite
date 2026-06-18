package projection_test

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/projection/v2"
)

func TestAlwaysLeader_IsLeader(t *testing.T) {
	t.Parallel()

	le := projection.AlwaysLeader{}

	if !le.IsLeader(context.Background()) {
		t.Error("AlwaysLeader.IsLeader should return true")
	}
}

func TestAlwaysLeader_WaitForLeadership(t *testing.T) {
	t.Parallel()

	le := projection.AlwaysLeader{}

	if err := le.WaitForLeadership(context.Background()); err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}

func TestAlwaysLeader_Resign(t *testing.T) {
	t.Parallel()

	le := projection.AlwaysLeader{}

	if err := le.Resign(context.Background()); err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}

func TestAlwaysLeader_ImplementsLeaderElection(t *testing.T) {
	t.Parallel()

	var _ projection.LeaderElection = projection.AlwaysLeader{}
}

// TestLeaderElection_NeverLeader tests a mock that never holds leadership.
type neverLeader struct{}

func (neverLeader) IsLeader(context.Context) bool               { return false }
func (neverLeader) WaitForLeadership(ctx context.Context) error { return ctx.Err() }
func (neverLeader) Resign(context.Context) error                { return nil }

func TestLeaderElection_InterfaceCompliance(t *testing.T) {
	t.Parallel()

	var _ projection.LeaderElection = neverLeader{}

	nl := neverLeader{}
	if nl.IsLeader(context.Background()) {
		t.Error("neverLeader should not be leader")
	}
}
