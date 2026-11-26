package main

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/benchkit/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/sqlite/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/v4"
)

func TestParseDurability_Valid(t *testing.T) {
	t.Parallel()

	cases := []struct {
		input string
		tier  stack.DurabilityTier
		isSet bool
	}{
		{"strict", stack.DurabilityStrict, true},
		{"STRICT", stack.DurabilityStrict, true},
		{" strict ", stack.DurabilityStrict, true},
		{"normal", stack.DurabilityNormal, true},
		{"relaxed", stack.DurabilityRelaxed, true},
		{"", stack.DurabilityNormal, false},
	}

	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			t.Parallel()

			tier, isSet, err := parseDurability(tc.input)
			if err != nil {
				t.Fatalf("parseDurability(%q) error: %v", tc.input, err)
			}

			if tier != tc.tier {
				t.Errorf("tier = %q, want %q", tier, tc.tier)
			}

			if isSet != tc.isSet {
				t.Errorf("isSet = %v, want %v", isSet, tc.isSet)
			}
		})
	}
}

func TestParseDurability_InvalidReturnsError(t *testing.T) {
	t.Parallel()

	if _, _, err := parseDurability("bogus"); err == nil {
		t.Fatal(`parseDurability("bogus") should return an error`)
	}
}

// TestOpenBackend_ChecksRequirementsBeforeTouchingTheDisk: an unknown name,
// a CGo-only backend in a pure-Go build, and a server backend without --dsn
// are all rejected without creating anything.
func TestOpenBackend_ChecksRequirementsBeforeTouchingTheDisk(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		backend string
		env     openEnv
		want    error
		teaches string
	}{
		{name: "unknown backend", backend: "bogus", want: errBackend, teaches: "sqlite"},
		{
			name: "cgo-only sqlite", backend: "sqlite-cgo", env: openEnv{cgo: false},
			want: errBackendUnavailable, teaches: "CGO_ENABLED=1",
		},
		{
			name: "cgo-only duckdb", backend: "duckdb", env: openEnv{cgo: false},
			want: errBackendUnavailable, teaches: "CGO_ENABLED=1",
		},
		{name: "postgres without dsn", backend: "postgres", want: errDSNRequired, teaches: "postgres://"},
		{name: "mysql without dsn", backend: "maria", want: errDSNRequired, teaches: "tcp("},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			bound, err := openBackend(tc.backend, tc.env)
			if !errors.Is(err, tc.want) {
				t.Fatalf("openBackend(%q) error = %v, want %v", tc.backend, err, tc.want)
			}

			if !strings.Contains(err.Error(), tc.teaches) {
				t.Errorf("error %q should teach the fix (%q)", err, tc.teaches)
			}

			if bound.site != nil {
				t.Errorf("a rejected backend must not create a store, got %q", bound.site.measuredDir)
			}
		})
	}
}

// TestOpenBackend_ResolvesAliasesToCanonicalNames: results are always
// labelled with the canonical backend name, however it was spelled.
func TestOpenBackend_ResolvesAliasesToCanonicalNames(t *testing.T) {
	t.Parallel()

	for alias, want := range map[string]string{"mem": "memory", "sys-mem": "system-memory"} {
		t.Run(alias, func(t *testing.T) {
			t.Parallel()

			bound, err := openBackend(alias, openEnv{})
			if err != nil {
				t.Fatalf("openBackend(%q): %v", alias, err)
			}

			if bound.name != want {
				t.Errorf("name = %q, want %q", bound.name, want)
			}
		})
	}
}

// TestOpenBackend_WarnsAboutIgnoredFlags: a flag the backend does not act on
// is reported instead of being dropped in silence.
func TestOpenBackend_WarnsAboutIgnoredFlags(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		backend string
		env     openEnv
		want    []string
	}{
		{
			name: "memory ignores every location flag", backend: "memory",
			env:  openEnv{dsn: "x", dir: "y", durability: stack.DurabilityStrict, durabilitySet: true},
			want: []string{"--dsn", "--dir", "--durability"},
		},
		{
			name: "mysql ignores durability", backend: "mysql",
			env:  openEnv{dsn: "user:pass@tcp(localhost:3306)/bench", durabilitySet: true},
			want: []string{"--durability"},
		},
		{
			name: "sqlite honors all three", backend: "sqlite",
			env: openEnv{dsn: ":memory:", durabilitySet: true},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			bound, err := openBackend(tc.backend, tc.env)
			if err != nil {
				t.Fatalf("openBackend(%q): %v", tc.backend, err)
			}

			defer func() { _ = bound.close() }()

			if len(bound.warnings) != len(tc.want) {
				t.Fatalf("warnings = %q, want exactly %q", bound.warnings, tc.want)
			}

			for _, flag := range tc.want {
				if !slices.ContainsFunc(bound.warnings, func(warning string) bool {
					return strings.Contains(warning, flag)
				}) {
					t.Errorf("no warning mentions %s: %q", flag, bound.warnings)
				}
			}
		})
	}
}

// TestOpenBackend_ServerRepeatsAreNotIsolated: the tool cannot give every
// repeat its own database on a server, so it says so.
func TestOpenBackend_ServerRepeatsAreNotIsolated(t *testing.T) {
	t.Parallel()

	env := openEnv{dsn: "user:pass@tcp(localhost:3306)/bench", generations: true}

	bound, err := openBackend("mysql", env)
	if err != nil {
		t.Fatalf("openBackend: %v", err)
	}

	if !slices.ContainsFunc(bound.warnings, func(warning string) bool {
		return strings.Contains(warning, "not isolated")
	}) {
		t.Errorf("repeats against a server must warn about missing isolation, got %q", bound.warnings)
	}
}

// runSQLite benchmarks the dev profile against a sqlite binding opened for
// env and returns the binding (kept open for inspection) with its result.
func runSQLite(t *testing.T, env openEnv, config benchkit.Config) (binding, *benchkit.Result) {
	t.Helper()

	bound, err := openBackend("sqlite", env)
	if err != nil {
		t.Fatalf("openBackend(sqlite): %v", err)
	}

	t.Cleanup(func() { _ = bound.close() })

	config.Profile = benchkit.ProfileDev

	result, err := benchkit.Run(t.Context(), config, bound.factory)
	if err != nil {
		t.Fatalf("benchkit.Run: %v", err)
	}

	return bound, result
}

// storedEvents counts the journal rows of the sqlite store in dir.
func storedEvents(t *testing.T, dir string) int {
	t.Helper()

	bundle, err := sqlite.New(filepath.Join(dir, storeFileName))
	if err != nil {
		t.Fatalf("open store in %s: %v", dir, err)
	}

	defer func() { _ = bundle.Close() }()

	events, err := bundle.Journal.ReadAll(t.Context())
	if err != nil {
		t.Fatalf("read journal: %v", err)
	}

	return len(events)
}

// TestSQLiteRepeats_StartFromAnEmptyStore pins the fix for the repeat
// contamination: every repeat, sweep point, and soak iteration opens an
// empty store, so the store a repeated run leaves holds exactly one run's
// rows. Before the fix the factory reopened one path for all of them and
// the store grew by a full run per repeat.
func TestSQLiteRepeats_StartFromAnEmptyStore(t *testing.T) {
	t.Parallel()

	single, _ := runSQLite(t, openEnv{}, benchkit.Config{})
	oneRun := storedEvents(t, single.diskPath)

	repeated, _ := runSQLite(t, openEnv{}, benchkit.Config{Repeat: 3})

	if got := storedEvents(t, repeated.diskPath); got != oneRun {
		t.Errorf("store after 3 repeats holds %d events, want the %d of one run", got, oneRun)
	}
}

// TestSQLiteRepeats_ReportOneRunsDiskFootprint: disk size, overhead, and
// write amplification of a repeat describe that repeat's own store, not the
// sum of every repeat before it.
func TestSQLiteRepeats_ReportOneRunsDiskFootprint(t *testing.T) {
	t.Parallel()

	_, single := runSQLite(t, openEnv{}, benchkit.Config{})

	repeated, err := benchkit.RunRepeated(t.Context(), benchkit.Config{
		Profile: benchkit.ProfileDev, Repeat: 3,
	}, mustOpenSQLite(t).factory)
	if err != nil {
		t.Fatalf("benchkit.RunRepeated: %v", err)
	}

	for i, run := range repeated.Runs {
		if run.Disk.DatabaseBytes > single.Disk.DatabaseBytes*110/100 {
			t.Errorf("repeat %d reports %d disk bytes, a single run reports %d", i+1,
				run.Disk.DatabaseBytes, single.Disk.DatabaseBytes)
		}
	}
}

// TestSQLiteWarmup_StaysOutOfTheMeasuredStore pins the README claim that
// warmup runs on a separate store: the measured store holds the same rows
// with and without --warmup. Warmup is 1 because benchkit v4.7.0, the pin
// this module builds against, rejects a second warmup event on a store that
// checks versions (fixed in benchkit on 2026-11-25).
func TestSQLiteWarmup_StaysOutOfTheMeasuredStore(t *testing.T) {
	t.Parallel()

	plain, _ := runSQLite(t, openEnv{}, benchkit.Config{})
	warm, result := runSQLite(t, openEnv{calls: storeCalls{warmup: true}}, benchkit.Config{Warmup: 1})

	if result.WarmupEvents != 1 {
		t.Fatalf("WarmupEvents = %d, want 1", result.WarmupEvents)
	}

	if got, want := storedEvents(t, warm.diskPath), storedEvents(t, plain.diskPath); got != want {
		t.Errorf("measured store holds %d events with warmup and %d without", got, want)
	}
}

// TestSQLiteRecovery_ReadsTheMeasuredStore: the recovery reopen must land on
// the store the run just wrote. This is the contract the store site's
// position-based role table relies on; if benchkit ever reorders its factory
// calls, the recovered event count collapses to zero here first.
func TestSQLiteRecovery_ReadsTheMeasuredStore(t *testing.T) {
	t.Parallel()

	_, result := runSQLite(t, openEnv{calls: storeCalls{recovery: true}}, benchkit.Config{Recovery: true})

	if result.RecoveredEvents < result.TotalEvents {
		t.Errorf("recovered %d events, want at least the %d written", result.RecoveredEvents, result.TotalEvents)
	}
}

// mustOpenSQLite opens a sqlite temp binding for the test and closes it at
// the end.
func mustOpenSQLite(t *testing.T) binding {
	t.Helper()

	bound, err := openBackend("sqlite", openEnv{})
	if err != nil {
		t.Fatalf("openBackend(sqlite): %v", err)
	}

	t.Cleanup(func() { _ = bound.close() })

	return bound
}

// TestOpenBackend_DirKeepsTheStoreUnderItsOwnSubtree: --dir may name a
// directory that does not exist yet; the store lands in the tool's subtree,
// survives close for inspection, and the warmup scratch is removed.
func TestOpenBackend_DirKeepsTheStoreUnderItsOwnSubtree(t *testing.T) {
	t.Parallel()

	root := filepath.Join(t.TempDir(), "missing", "bench-dir")

	bound, err := openBackend("sqlite", openEnv{dir: root, calls: storeCalls{warmup: true}})
	if err != nil {
		t.Fatalf("openBackend with a missing --dir: %v", err)
	}

	storeDir := filepath.Join(root, toolDataDir, "sqlite", storeDirName)
	if bound.diskPath != storeDir {
		t.Fatalf("diskPath = %q, want %q", bound.diskPath, storeDir)
	}

	if _, err := benchkit.Run(t.Context(), benchkit.Config{
		Profile: benchkit.ProfileDev, Warmup: 1,
	}, bound.factory); err != nil {
		t.Fatalf("benchkit.Run: %v", err)
	}

	if err := bound.close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	if _, err := os.Stat(filepath.Join(storeDir, storeFileName)); err != nil {
		t.Errorf("the measured store must survive close: %v", err)
	}

	warmupDir := filepath.Join(root, toolDataDir, "sqlite", warmupDirName)
	if _, err := os.Stat(warmupDir); !os.IsNotExist(err) {
		t.Errorf("the warmup scratch must be removed on close: %v", err)
	}
}

// TestOpenBackend_ReplayReadsTheStoreARunLeft: --replay opens the store the
// previous measuring run kept under --dir, in the current layout and in the
// layout older versions wrote straight into --dir.
func TestOpenBackend_ReplayReadsTheStoreARunLeft(t *testing.T) {
	t.Parallel()

	root := t.TempDir()

	measured, err := openBackend("sqlite", openEnv{dir: root})
	if err != nil {
		t.Fatalf("openBackend: %v", err)
	}

	if _, err := benchkit.Run(
		t.Context(),
		benchkit.Config{Profile: benchkit.ProfileDev},
		measured.factory,
	); err != nil {
		t.Fatalf("benchkit.Run: %v", err)
	}

	_ = measured.close()

	replayEnv := openEnv{dir: root, calls: storeCalls{replay: true}}

	replayed, err := openBackend("sqlite", replayEnv)
	if err != nil {
		t.Fatalf("openBackend for --replay: %v", err)
	}

	if replayed.diskPath != measured.diskPath {
		t.Errorf("--replay measures %q, want the kept store %q", replayed.diskPath, measured.diskPath)
	}

	replayResult, err := benchkit.Run(t.Context(), benchkit.Config{
		Profile: benchkit.ProfileDev, ReplayOnly: true,
	}, replayed.factory)
	if err != nil {
		t.Fatalf("replay run: %v", err)
	}

	if replayResult.TotalEvents == 0 {
		t.Error("replay read no events from the kept store")
	}

	_ = replayed.close()

	legacy := t.TempDir()
	legacyStore, err := sqlite.New(filepath.Join(legacy, storeFileName))
	if err != nil {
		t.Fatalf("open legacy-layout store: %v", err)
	}

	_ = legacyStore.Close()

	legacyEnv := openEnv{dir: legacy, calls: storeCalls{replay: true}}

	legacyReplay, err := openBackend("sqlite", legacyEnv)
	if err != nil {
		t.Fatalf("openBackend on a legacy-layout --dir: %v", err)
	}

	if legacyReplay.diskPath != legacy {
		t.Errorf("legacy --replay measures %q, want the root %q", legacyReplay.diskPath, legacy)
	}

	_ = legacyReplay.close()
}

// TestOpenBackend_ReplayNeedsAnExistingStore: without --dir there is nothing
// to replay.
func TestOpenBackend_ReplayNeedsAnExistingStore(t *testing.T) {
	t.Parallel()

	_, err := openBackend("sqlite", openEnv{calls: storeCalls{replay: true}})
	if !errors.Is(err, errFlagCombo) {
		t.Errorf("--replay without --dir must be rejected, got %v", err)
	}

	if _, err := openBackend("sqlite", openEnv{dir: t.TempDir(), calls: storeCalls{replay: true}}); !errors.Is(
		err, errStorePrepare,
	) {
		t.Errorf("--replay on an empty --dir must be rejected, got %v", err)
	}
}

// TestOpenBackend_FileDSN: a path --dsn names a store the operator owns. A
// measuring run refuses a file that already holds data and keeps its own
// file on close; an in-memory DSN owns nothing.
func TestOpenBackend_FileDSN(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "owned.db")
	if err := os.WriteFile(path, []byte("someone's database"), 0o600); err != nil {
		t.Fatalf("plant file: %v", err)
	}

	if _, err := openBackend("sqlite", openEnv{dsn: path}); !errors.Is(err, errStoreExists) {
		t.Errorf("a non-empty --dsn file must be refused, got %v", err)
	}

	fresh := filepath.Join(t.TempDir(), "fresh.db")

	bound, err := openBackend("sqlite", openEnv{dsn: fresh, calls: storeCalls{warmup: true}})
	if err != nil {
		t.Fatalf("openBackend with a fresh --dsn file: %v", err)
	}

	if bound.diskPath != "" {
		t.Errorf("a file DSN has no directory to measure, diskPath = %q", bound.diskPath)
	}

	t.Cleanup(func() { _ = bound.close() })

	if _, err := benchkit.Run(t.Context(), benchkit.Config{
		Profile: benchkit.ProfileDev, Warmup: 1,
	}, bound.factory); err != nil {
		t.Fatalf("benchkit.Run: %v", err)
	}

	if info, err := os.Stat(fresh); err != nil || info.Size() == 0 {
		t.Errorf("the run must leave its store at the --dsn file: %v", err)
	}

	memory, err := openBackend("sqlite", openEnv{dsn: ":memory:"})
	if err != nil {
		t.Fatalf("openBackend(:memory:): %v", err)
	}

	if memory.site != nil {
		t.Errorf("an in-memory DSN must own no store, got %q", memory.site.measuredDir)
	}
}

func TestDSNPaths(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		":memory:":                               ":memory:",
		"/data/bench.db":                         "/data/bench.db",
		"/data/bench.db?_pragma=busy_timeout(1)": "/data/bench.db",
		"file:/data/bench.db?cache=shared":       "/data/bench.db",
		"file:///data/bench.db":                  "/data/bench.db",
	}

	for dsn, want := range cases {
		if got := dsnFilePath(dsn); got != want {
			t.Errorf("dsnFilePath(%q) = %q, want %q", dsn, got, want)
		}
	}
}
