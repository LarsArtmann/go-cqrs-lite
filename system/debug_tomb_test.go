package system_test

import (
	"context"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

func TestDebugTombHost(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	sys := startTombstoneSystem(t, ctx)

	streamID := id.NewStreamID()
	dispatchTomb(t, ctx, sys, "evotomb.create", streamID)

	if view := awaitTombPhase(ctx, sys, streamID.String(), true); view == nil {
		t.Fatal("create never appeared")
	}

	t.Logf("after create: host status %+v", sys.ProjectionHost().Status())

	dispatchTomb(t, ctx, sys, "evotomb.delete", streamID)

	time.Sleep(2 * time.Second)

	t.Logf("after delete: host status %+v", sys.ProjectionHost().Status())

	_, gone := awaitTombView(ctx, sys, streamID.String())
	t.Logf("gone=%v", gone)
}
