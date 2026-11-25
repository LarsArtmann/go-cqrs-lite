package system_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
)

const findCapRows = 150

// newRuntimeSet boots the memory-backed runtime projection with `rows`
// created events, more than one default scan page.
func newRuntimeSet(t *testing.T, rows int) (context.Context, *system.System) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	t.Cleanup(cancel)

	sys, err := system.New(ctx, system.DomainConfig{
		Projections: []system.ProjectionDeclaration{
			system.QuerySet[RuntimeView]("rt_set").
				On("rt.created", RuntimeCreated{}).
				Done(),
		},
	}, system.DeploymentConfig{
		Engines: map[string]system.EngineConfig{
			"primary": {Driver: "memory"},
		},
		Instances: []system.InstanceConfig{
			{Role: system.RoleSourceOfTruth, Engine: "primary"},
			{Role: system.RoleProjections, Engine: "primary"},
		},
	})
	if err != nil {
		t.Fatalf("system.New: %v", err)
	}

	t.Cleanup(func() { _ = sys.Close() })

	for i := range rows {
		mustApply(t, sys.MetaEngine(), "rt.created", RuntimeCreated{
			ID:     fmt.Sprintf("%03d", i),
			Title:  "row",
			Status: "open",
		})
	}

	return ctx, sys
}

// The default Find scan stops at the engine's default bound; pin it so a
// change of the bound is deliberate (issue #135).
func TestSystem_Find_DefaultScanCapsAtHundredRows(t *testing.T) {
	t.Parallel()

	ctx, sys := newRuntimeSet(t, findCapRows)

	got, err := system.Find[RuntimeView](ctx, sys, "rt_set")
	if err != nil {
		t.Fatalf("Find: %v", err)
	}

	if len(got) != 100 {
		t.Fatalf("default Find returned %d rows, want the default bound of 100", len(got))
	}
}

func TestSystem_Find_UnlimitedReturnsEveryRow(t *testing.T) {
	t.Parallel()

	ctx, sys := newRuntimeSet(t, findCapRows)

	got, err := system.Find[RuntimeView](ctx, sys, "rt_set", system.Unlimited())
	if err != nil {
		t.Fatalf("Find Unlimited: %v", err)
	}

	if len(got) != findCapRows {
		t.Fatalf("Find Unlimited returned %d rows, want %d", len(got), findCapRows)
	}
}

func TestSystem_Find_ExactFailsWhenBoundedScanTruncates(t *testing.T) {
	t.Parallel()

	ctx, sys := newRuntimeSet(t, findCapRows)

	_, err := system.Find[RuntimeView](ctx, sys, "rt_set", system.Limit(10), system.Exact())
	if !errors.Is(err, metaengine.ErrScanTruncated) {
		t.Fatalf(
			"Find Limit(10) Exact on %d rows = %v, want an error matching metaengine.ErrScanTruncated",
			findCapRows,
			err,
		)
	}

	exactFit, err := system.Find[RuntimeView](ctx, sys, "rt_set", system.Limit(findCapRows), system.Exact())
	if err != nil {
		t.Fatalf("Find exact fit: %v", err)
	}

	if len(exactFit) != findCapRows {
		t.Fatalf("exact fit returned %d rows, want %d", len(exactFit), findCapRows)
	}

	all, err := system.Find[RuntimeView](ctx, sys, "rt_set", system.Unlimited(), system.Exact())
	if err != nil {
		t.Fatalf("Find Unlimited Exact: %v", err)
	}

	if len(all) != findCapRows {
		t.Fatalf("Find Unlimited Exact returned %d rows, want %d", len(all), findCapRows)
	}
}

func TestSystem_Find_WithoutExactReturnsTruncatedRows(t *testing.T) {
	t.Parallel()

	ctx, sys := newRuntimeSet(t, findCapRows)

	got, err := system.Find[RuntimeView](ctx, sys, "rt_set", system.Limit(10))
	if err != nil {
		t.Fatalf("Find Limit(10): %v", err)
	}

	if len(got) != 10 {
		t.Fatalf("Find Limit(10) returned %d rows, want 10", len(got))
	}
}
