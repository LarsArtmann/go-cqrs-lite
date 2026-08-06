package benchkit

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// durabilityPhase measures on-disk storage footprint.
// Tries the DiskSizer interface first (precise per-backend sizing via
// stack.Bundle.DiskSize); falls back to filesystem walk of Config.DiskPath.
// DiskSize() returns -1 when no disk-size reporter is registered (memory,
// SQLite without WithDiskSize), signaling the fallback path.
func (r *runner) durabilityPhase() {
	if sizer, ok := any(r.bundle).(DiskSizer); ok {
		if size := sizer.DiskSize(); size >= 0 {
			r.result.Disk.DatabaseBytes = size

			return
		}
	}

	if r.config.DiskPath != "" {
		r.result.Disk.DatabaseBytes = measureDirSize(r.config.DiskPath)
	} else {
		r.warn("durability phase: disk size not measured (no DiskSizer and no DiskPath set)")
	}
}

func measureDirSize(path string) int64 {
	var total int64

	_ = filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return nil //nolint:nilerr // skip unreadable entries
		}

		if !info.IsDir() {
			total += info.Size()
		}

		return nil
	})

	return total
}

// recoveryPhase simulates crash recovery: closes the current bundle
// (flushing all writes to disk), reopens it via the factory (reopening at
// the same path for persistent backends), and loads all streams to measure
// replay time. For memory backends, the reopened store is empty, so
// RecoveredEvents will be zero — this is expected and documents that
// memory backends have no crash recovery.
func (r *runner) recoveryPhase(parent context.Context) error {
	// Recovery is a post-benchmark durability check — it must run even if
	// the benchmark context has expired. WithoutCancel inherits values
	// (tracing, etc.) but strips the deadline/cancellation.
	ctx := context.WithoutCancel(parent)

	// Close the current bundle to flush all writes.
	//cqrs-lint:ignore(C015,C023) deliberate close-and-reopen for recovery test
	_ = r.bundle.Close()
	r.bundle = nil // prevent double-close in teardown

	// Reopen via factory — for persistent backends (SQLite, Pebble),
	// the factory reopens at the same path and all events are recovered.
	recovered, err := r.factory()
	if err != nil {
		return fmt.Errorf("recovery factory: %w", err)
	}

	if recovered == nil || recovered.EventSource == nil {
		r.warn("recovery phase: skipped (reopened bundle has no EventSource)")

		if recovered != nil {
			//cqrs-lint:ignore(C015,C023) cleanup before error return
			_ = recovered.Close()
		}

		return nil
	}

	defer func() { _ = recovered.Close() }()

	start := time.Now()
	totalEvents := 0

	for _, ref := range r.refs {
		events, err := recovered.EventSource.Load(ctx, ref)
		if err != nil {
			// Memory backend: streams don't exist after reopen — skip.
			continue
		}

		totalEvents += len(events)
	}

	r.result.RecoveryTime = time.Since(start)
	r.result.RecoveredEvents = totalEvents

	return nil
}
