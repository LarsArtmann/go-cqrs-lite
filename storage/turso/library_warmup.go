package turso

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// art-dupl:accept dep-isolated twin of metaengine/tursoengine/library_warmup.go: no legal shared home (tier rules forbid importing this Tier-4 module from the engine), same pattern as the register.go driver twins
// tursoLibraryWarmupTimeout bounds how long this process waits for a
// concurrent process to finish the one-time library extraction before
// proceeding unlocked (stale-lock fallback — same behavior as without this
// package).
const tursoLibraryWarmupTimeout = 30 * time.Second

var (
	warmTursoLibraryOnce  sync.Once //nolint:gochecknoglobals // idiomatic lazy init; pair of the errXxx below
	errTursoLibraryWarmup error
)

// warmTursoLibrary loads the native turso library exactly once per process,
// serialized across processes by a lockfile.
//
// turso-go-platform-libs v0.8.2 writes the embedded libturso_sync_sdk_kit.so
// into the shared user cache (os.UserCacheDir()/turso-go/<hash8>/) with
// os.Create + io.Copy — no locking, no atomic rename. Two processes
// cold-starting concurrently (`go test ./...` package binaries, or two
// services on one host) interleave their writes; the loser then reads a
// corrupt cache file and tursogo.InitLibrary PANICS with "cached library
// file hash sum mismatch". Holding the lockfile across a throwaway
// connection ping guarantees the cache entry is fully written and
// hash-verified before any other process reads it, and converts the
// driver's panic into a returned error.
//
// The ping uses a ":memory:" DSN on purpose: driver.Open runs InitLibrary
// (the extraction) for ANY DSN, and ":memory:" cannot fail for
// DSN-reasons — so a broken first caller DSN can never poison the warmup
// result.
func warmTursoLibrary() error {
	warmTursoLibraryOnce.Do(func() {
		db, err := sql.Open("turso", ":memory:")
		if err != nil {
			errTursoLibraryWarmup = fmt.Errorf("open warmup connection: %w", err)
			return
		}
		defer func() { _ = db.Close() }()

		errTursoLibraryWarmup = withTursoLibraryLock(db.Ping)
	})

	return errTursoLibraryWarmup
}

// withTursoLibraryLock runs ping while holding a best-effort exclusive
// lockfile next to the shared library cache. Every failure mode of the
// lock itself (unwritable cache dir, unexpected error, stale lock after a
// crashed process) degrades to an unlocked ping — never worse than before.
func withTursoLibraryLock(ping func() error) error {
	lockPath := tursoLibraryLockPath()

	if err := os.MkdirAll(filepath.Dir(lockPath), 0o755); err != nil {
		return ping()
	}

	lock, locked := acquireTursoLockFile(lockPath)
	if locked {
		defer func() {
			_ = lock.Close()
			_ = os.Remove(lockPath)
		}()
	}

	return ping()
}

// acquireTursoLockFile blocks until it exclusively creates lockPath, the
// warmup timeout elapses (stale lock), or creation fails outright. It
// reports whether the lock is held.
func acquireTursoLockFile(lockPath string) (*os.File, bool) {
	deadline := time.Now().Add(tursoLibraryWarmupTimeout)
	for {
		lock, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
		if err == nil {
			return lock, true
		}
		if !errors.Is(err, os.ErrExist) || time.Now().After(deadline) {
			return nil, false
		}
		time.Sleep(25 * time.Millisecond)
	}
}

// tursoLibraryLockPath resolves the lockfile next to the cache directory
// the upstream loader itself uses (TURSO_GO_CACHE_DIR override, then
// os.UserCacheDir(), then os.TempDir() — mirroring its precedence).
func tursoLibraryLockPath() string {
	root := os.Getenv("TURSO_GO_CACHE_DIR")
	if root == "" {
		if dir, err := os.UserCacheDir(); err == nil {
			root = dir
		} else {
			root = os.TempDir()
		}
	}

	return filepath.Join(root, "turso-go", "cqrs-first-open.lock")
}
