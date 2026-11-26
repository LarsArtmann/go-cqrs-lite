package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/larsartmann/go-cqrs-lite/benchkit/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/v4"
)

const (
	// storeFileName is the database file name inside a tool-owned store
	// directory (sqlite, sqlite-cgo, bbolt, turso).
	storeFileName = "bench.db"
	// toolDataDir is the directory cqrs-bench owns under a --dir root. Every
	// reset stays inside it, so a --dir that holds other data is never
	// touched beyond this one subtree.
	toolDataDir = ".cqrs-bench"
	// storeDirName and warmupDirName are the two store directories inside a
	// site root.
	storeDirName  = "store"
	warmupDirName = "warmup"
	// storeDirPerm is the mode of every directory the tool creates.
	storeDirPerm = 0o755
)

// errStorePrepare reports that a benchmark store could not be made ready.
var errStorePrepare = errors.New("prepare benchmark store")

// errStoreExists reports a measuring run aimed at a store that already
// holds data.
var errStoreExists = errors.New("store already holds data")

// storeCalls records which extra factory calls benchkit makes per run, so a
// store site can tell those calls apart (see [storeSite.role]).
type storeCalls struct {
	// warmup is benchkit Config.Warmup > 0: the runner opens a second,
	// throwaway bundle while the measured one stays open.
	warmup bool
	// recovery is benchkit Config.Recovery: after all phases the runner
	// closes the measured bundle and calls the factory once more to reopen
	// it at the same location.
	recovery bool
	// replay is benchkit Config.ReplayOnly: every run reads a store that
	// already holds data, so nothing is ever reset.
	replay bool
}

// perRun is the number of factory calls benchkit makes for one run.
func (c storeCalls) perRun() int {
	perRun := 1

	if c.warmup {
		perRun++
	}

	if c.recovery {
		perRun++
	}

	return perRun
}

// storeRole is why benchkit asked for a store.
type storeRole int

const (
	// storeMeasured starts a run on an empty store: the first call of every
	// run, repeat, sweep point, and soak iteration.
	storeMeasured storeRole = iota
	// storeReopened reads back the store a run just finished writing: the
	// recovery reopen, or any call of a replay-only run.
	storeReopened
	// storeWarmup is the throwaway sibling the warmup phase writes to while
	// the measured store stays open.
	storeWarmup
)

// storeSite hands out store locations across benchkit's factory calls.
//
// benchkit calls a Factory more than once per run, and each call wants a
// different store: a fresh one for the measured run, a separate one for the
// throwaway warmup bundle, and the just-closed one again for the recovery
// reopen. A factory that returns the same location every time makes later
// repeats, sweep points, and soak iterations inherit earlier ones' rows
// (two sqlite repeats left 3500 event rows where one run writes 1750) and
// lets warmup events land in the measured store. The site classifies each
// call by its position in the run, following benchkit's documented call
// order (setup, warmup, phases, recovery), and asks location for the
// matching store.
//
// A site is used from one goroutine: benchkit opens stores sequentially.
type storeSite struct {
	calls storeCalls
	taken int
	// location returns the store for a role, emptying it first when the
	// role starts a fresh generation.
	location func(storeRole) (string, error)
	// cleanup removes what the site owns but a later run never reads.
	cleanup func() error
	// measuredDir is the directory whose size is the run's disk footprint;
	// empty when the store is not a directory the tool measures.
	measuredDir string
	// warmupDir is the directory of the throwaway warmup sibling.
	warmupDir string
}

// claim returns the store location for the next factory call.
func (s *storeSite) claim() (string, error) {
	role := s.role(s.taken)
	s.taken++

	location, err := s.location(role)
	if err != nil {
		return "", fmt.Errorf("%w: %w", errStorePrepare, err)
	}

	return location, nil
}

// role maps the nth factory call (zero-based within the command) to its
// store role. Benchkit makes exactly perRun calls per run, so the call's
// position inside its run identifies it.
func (s *storeSite) role(nth int) storeRole {
	position := nth % s.calls.perRun()
	reopens := s.calls.recovery && position == s.calls.perRun()-1

	if position == 0 && s.calls.replay || reopens {
		return storeReopened
	}

	if position == 0 {
		return storeMeasured
	}

	return storeWarmup
}

// factory wraps open into the benchkit Factory for this site.
func (s *storeSite) factory(open func(location string) (*stack.Bundle, error)) benchkit.Factory {
	return func() (*stack.Bundle, error) {
		location, err := s.claim()
		if err != nil {
			return nil, err
		}

		return open(location)
	}
}

// close removes the scratch the site created. The measured store of a --dir
// site is kept for inspection and --replay; a temp site loses everything.
func (s *storeSite) close() error {
	if s.cleanup == nil {
		return nil
	}

	if err := s.cleanup(); err != nil {
		return fmt.Errorf("clean up benchmark store: %w", err)
	}

	return nil
}

// newTempSite keeps both stores in a private temp directory that is removed
// on close.
func newTempSite(calls storeCalls) (*storeSite, error) {
	root, err := os.MkdirTemp("", "cqrs-bench-*")
	if err != nil {
		return nil, fmt.Errorf("%w: create temp dir: %w", errStorePrepare, err)
	}

	return dirSite(root, calls, func() error { return removeDir(root) }), nil
}

// newDirSite keeps a backend's stores under a --dir root, creating the
// root when it does not exist yet. The measured store survives the run at
// <root>/.cqrs-bench/<backend>/store; only the warmup sibling is removed.
func newDirSite(root, backend string, calls storeCalls) (*storeSite, error) {
	if err := os.MkdirAll(root, storeDirPerm); err != nil {
		return nil, fmt.Errorf("%w: create --dir %s: %w", errStorePrepare, root, err)
	}

	home := dirHome(root, backend)

	return dirSite(home, calls, func() error {
		return removeDir(filepath.Join(home, warmupDirName))
	}), nil
}

// dirHome is the directory the tool owns for backend under a --dir root.
func dirHome(root, backend string) string {
	return filepath.Join(root, toolDataDir, backend)
}

// dirSite builds a site whose measured and warmup stores are the store and
// warmup subdirectories of home.
func dirSite(home string, calls storeCalls, cleanup func() error) *storeSite {
	return &storeSite{
		calls:       calls,
		location:    dirLocations(filepath.Join(home, storeDirName), filepath.Join(home, warmupDirName)),
		cleanup:     cleanup,
		measuredDir: filepath.Join(home, storeDirName),
		warmupDir:   filepath.Join(home, warmupDirName),
	}
}

// newReplaySite reads the store a previous run left under a --dir root. The
// current layout wins; legacy reports whether root itself is a store that an
// older cqrs-bench wrote straight into --dir.
func newReplaySite(root, backend string, legacy func(root string) bool) (*storeSite, error) {
	storeDir := dirHome(root, backend)
	storeDir = filepath.Join(storeDir, storeDirName)

	_, err := os.Stat(storeDir)

	switch {
	case err == nil:
	case legacy(root):
		storeDir = root
	default:
		return nil, fmt.Errorf(
			"%w: no %s store found under %s — run a measuring command with --dir first",
			errStorePrepare, backend, root,
		)
	}

	return &storeSite{
		calls:       storeCalls{replay: true},
		location:    func(storeRole) (string, error) { return storeDir, nil },
		measuredDir: storeDir,
	}, nil
}

// newFileSite owns a store kept in one user-named file (a path --dsn). The
// caller has checked that a measuring run may write it (see
// [requireEmptyFile]). Every location the site hands out can be opened as
// is: a measured generation replaces the file, the recovery reopen and
// replay read it, and warmup gets a throwaway file in a private temp
// directory.
func newFileSite(path, warmupName string, calls storeCalls) (*storeSite, error) {
	scratch, err := newTempSite(storeCalls{})
	if err != nil {
		return nil, err
	}

	return &storeSite{
		calls: calls,
		location: func(role storeRole) (string, error) {
			switch role {
			case storeWarmup:
				if err := resetDir(scratch.warmupDir); err != nil {
					return "", err
				}

				return filepath.Join(scratch.warmupDir, warmupName), nil
			case storeMeasured:
				if err := removeStoreFile(path); err != nil {
					return "", err
				}
			}

			return path, nil
		},
		cleanup: scratch.close,
	}, nil
}

// dirLocations serves directory-backed roles: the measured store is emptied
// at the start of every generation, the recovery reopen reads it as is, and
// every warmup call gets an empty sibling.
func dirLocations(storeDir, warmupDir string) func(storeRole) (string, error) {
	return func(role storeRole) (string, error) {
		dir := storeDir
		if role == storeWarmup {
			dir = warmupDir
		}

		if role == storeReopened {
			return dir, nil
		}

		if err := resetDir(dir); err != nil {
			return "", err
		}

		return dir, nil
	}
}

// requireEmptyFile rejects a measuring run onto a store file that already
// holds data: the run would mix its rows into someone else's, and the tool
// must never reset a file it did not create.
func requireEmptyFile(path string) error {
	info, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}

	if err != nil {
		return fmt.Errorf("inspect %s: %w", path, err)
	}

	if info.Size() > 0 {
		return fmt.Errorf("%w: %s (pass --replay to read it, or name a new file)", errStoreExists, path)
	}

	return nil
}

// resetDir makes dir an empty directory, removing whatever the tool or an
// earlier run left there.
func resetDir(dir string) error {
	if err := removeDir(dir); err != nil {
		return err
	}

	if err := os.MkdirAll(dir, storeDirPerm); err != nil {
		return fmt.Errorf("create %s: %w", dir, err)
	}

	return nil
}

// removeDir removes dir when it exists; a missing dir is not an error.
func removeDir(dir string) error {
	if err := os.RemoveAll(dir); err != nil {
		return fmt.Errorf("remove %s: %w", dir, err)
	}

	return nil
}

// removeStoreFile removes a single-file store and its SQLite-style
// sidecars. Only a file the current command created is ever passed here.
func removeStoreFile(path string) error {
	for _, candidate := range []string{path, path + "-wal", path + "-shm"} {
		err := os.Remove(candidate)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("remove %s: %w", candidate, err)
		}
	}

	return nil
}
