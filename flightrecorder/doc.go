// Package flightrecorder wraps Go 1.25's runtime/trace.FlightRecorder
// with a clean lifecycle API and composable trigger conditions.
//
// A flight recorder buffers the last few seconds of execution trace in memory.
// When a problem is detected (slow operation, error, panic), the program can
// snapshot exactly the problematic window of time for offline analysis with
// `go tool trace`.
//
// This is the zero-dependency core (stdlib only). For CQRS-aware middleware
// integration (Command/Event/Query dispatch triggers), use
// [github.com/larsartmann/go-cqrs-lite/middleware/v4].
//
// # Quick start
//
//	recorder, _ := flightrecorder.New(
//	    flightrecorder.WithMinAge(10*time.Second),
//	    flightrecorder.WithMaxBytes(1<<20), // 1 MiB
//	    flightrecorder.WithWriter(os.Stdout),
//	)
//	recorder.Start()
//	defer recorder.Stop()
//
//	// Later, when something goes wrong:
//	recorder.Snapshot(context.Background())
//
// # Trigger integration
//
//	// Fire only when an operation exceeds 100ms:
//	trigger := flightrecorder.OnLatency(100*time.Millisecond)
//	recorder.SnapshotIf(ctx, flightrecorder.TriggerContext{
//	    Kind:     "command",
//	    Type:     "user.create",
//	    Duration: 150 * time.Millisecond,
//	}, trigger)
//
// Analyze the captured trace with: go tool trace snapshot.trace
package flightrecorder
