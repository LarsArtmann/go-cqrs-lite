package pebble

import (
	"fmt"
	"log/slog"

	"github.com/cockroachdb/pebble"
	"github.com/cockroachdb/pebble/bloom"
)

const bloomFilterBitsPerKey = 10

// DefaultOptions returns recommended pebble.Options for CQRS event store workloads.
// These defaults optimize for the read-heavy, append-only access pattern of
// event sourcing: bloom filters for fast point reads, concurrent compactions
// for write throughput, and an EventListener for operational visibility.
//
// The returned Options are ready to pass to Open(dir, opts, logger).
// Callers may override any field after calling DefaultOptions.
//
//	backend, _ := pebble.Open(dir, pebble.DefaultOptions(), logger)
func DefaultOptions() *pebble.Options {
	opts := &pebble.Options{}

	// Bloom filters dramatically reduce read amplification for point Gets
	// (snapshot/checkpoint lookups). 10 bits per key gives ~1% FPR.
	filterPolicy := bloom.FilterPolicy(bloomFilterBitsPerKey)
	opts.Levels = make([]pebble.LevelOptions, 7)

	for i := range opts.Levels {
		opts.Levels[i].FilterPolicy = filterPolicy
		opts.Levels[i].FilterType = pebble.TableFilter
	}

	// Concurrent compactions improve write throughput on multi-core systems.
	opts.MaxConcurrentCompactions = func() int { return 4 }

	return opts
}

// DefaultOptionsWithLogging returns DefaultOptions with an EventListener that
// logs flush, compaction, and write-stall events at Debug level.
//
//	backend, _ := pebble.Open(dir, pebble.DefaultOptionsWithLogging(logger), logger)
func DefaultOptionsWithLogging(logger *slog.Logger) *pebble.Options {
	opts := DefaultOptions()
	eventListener := pebble.MakeLoggingEventListener(pebbleLogger{logger: logger})
	opts.EventListener = &eventListener

	return opts
}

// pebbleLogger adapts slog to pebble's Logger interface.
type pebbleLogger struct {
	logger *slog.Logger
}

func (l pebbleLogger) Infof(format string, args ...any) {
	l.logger.Info(fmt.Sprintf(format, args...))
}

func (l pebbleLogger) Fatalf(format string, args ...any) {
	l.logger.Error(fmt.Sprintf(format, args...))
}
