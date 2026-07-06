package turso

import (
	"context"
	"errors"
	"testing"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"
	tursoclient "turso.tech/database/tursogo"
)

// fakeSyncEngine is a test double for syncEngine. Each method returns the
// configured result and increments its call counter.
type fakeSyncEngine struct {
	pushErr       error
	pullChanged   bool
	pullErr       error
	checkpointErr error
	statsResult   tursoclient.TursoSyncDbStats
	statsErr      error

	pushCalls       int
	pullCalls       int
	checkpointCalls int
	statsCalls      int
}

func (f *fakeSyncEngine) Push(ctx context.Context) error {
	f.pushCalls++

	return f.pushErr
}

func (f *fakeSyncEngine) Pull(ctx context.Context) (bool, error) {
	f.pullCalls++

	return f.pullChanged, f.pullErr
}

func (f *fakeSyncEngine) Checkpoint(ctx context.Context) error {
	f.checkpointCalls++

	return f.checkpointErr
}

func (f *fakeSyncEngine) Stats(ctx context.Context) (tursoclient.TursoSyncDbStats, error) {
	f.statsCalls++

	return f.statsResult, f.statsErr
}

func TestSyncDB_Push_Success(t *testing.T) {
	t.Parallel()

	engine := &fakeSyncEngine{}
	sdb := newSyncDBWithEngine(nil, engine)

	err := sdb.Push(context.Background())
	if err != nil {
		t.Fatalf("Push() error = %v, want nil", err)
	}
	if engine.pushCalls != 1 {
		t.Errorf("pushCalls = %d, want 1", engine.pushCalls)
	}
}

func TestSyncDB_Push_ErrorWrapping(t *testing.T) {
	t.Parallel()

	sentinel := errors.New("network unreachable")
	sdb := newSyncDBWithEngine(nil, &fakeSyncEngine{pushErr: sentinel})

	err := sdb.Push(context.Background())
	if err == nil {
		t.Fatal("Push() error = nil, want error")
	}
	if !errors.Is(err, sentinel) {
		t.Errorf("Push() error = %v, want wrapping %v", err, sentinel)
	}
	if !isInfraError(t, err) {
		t.Errorf("Push() error = %v, want Infrastructure family", err)
	}
}

func TestSyncDB_Pull_SuccessWithChanges(t *testing.T) {
	t.Parallel()

	engine := &fakeSyncEngine{pullChanged: true}
	sdb := newSyncDBWithEngine(nil, engine)

	changed, err := sdb.Pull(context.Background())
	if err != nil {
		t.Fatalf("Pull() error = %v, want nil", err)
	}
	if !changed {
		t.Error("Pull() changed = false, want true")
	}
	if engine.pullCalls != 1 {
		t.Errorf("pullCalls = %d, want 1", engine.pullCalls)
	}
}

func TestSyncDB_Pull_SuccessNoChanges(t *testing.T) {
	t.Parallel()

	sdb := newSyncDBWithEngine(nil, &fakeSyncEngine{pullChanged: false})

	changed, err := sdb.Pull(context.Background())
	if err != nil {
		t.Fatalf("Pull() error = %v, want nil", err)
	}
	if changed {
		t.Error("Pull() changed = true, want false")
	}
}

func TestSyncDB_Pull_ErrorWrapping(t *testing.T) {
	t.Parallel()

	sentinel := errors.New("connection reset")
	sdb := newSyncDBWithEngine(nil, &fakeSyncEngine{pullErr: sentinel})

	_, err := sdb.Pull(context.Background())
	if err == nil {
		t.Fatal("Pull() error = nil, want error")
	}
	if !errors.Is(err, sentinel) {
		t.Errorf("Pull() error = %v, want wrapping %v", err, sentinel)
	}
	if !isInfraError(t, err) {
		t.Errorf("Pull() error = %v, want Infrastructure family", err)
	}
}

func TestSyncDB_Checkpoint_Success(t *testing.T) {
	t.Parallel()

	engine := &fakeSyncEngine{}
	sdb := newSyncDBWithEngine(nil, engine)

	err := sdb.Checkpoint(context.Background())
	if err != nil {
		t.Fatalf("Checkpoint() error = %v, want nil", err)
	}
	if engine.checkpointCalls != 1 {
		t.Errorf("checkpointCalls = %d, want 1", engine.checkpointCalls)
	}
}

func TestSyncDB_Checkpoint_ErrorWrapping(t *testing.T) {
	t.Parallel()

	sentinel := errors.New("wal corrupt")
	sdb := newSyncDBWithEngine(nil, &fakeSyncEngine{checkpointErr: sentinel})

	err := sdb.Checkpoint(context.Background())
	if err == nil {
		t.Fatal("Checkpoint() error = nil, want error")
	}
	if !errors.Is(err, sentinel) {
		t.Errorf("Checkpoint() error = %v, want wrapping %v", err, sentinel)
	}
	if !isInfraError(t, err) {
		t.Errorf("Checkpoint() error = %v, want Infrastructure family", err)
	}
}

func TestSyncDB_Stats_Success(t *testing.T) {
	t.Parallel()

	want := tursoclient.TursoSyncDbStats{
		CdcOperations:    42,
		NetworkSentBytes: 1024,
	}
	sdb := newSyncDBWithEngine(nil, &fakeSyncEngine{statsResult: want})

	got, err := sdb.Stats(context.Background())
	if err != nil {
		t.Fatalf("Stats() error = %v, want nil", err)
	}
	if got.CdcOperations != want.CdcOperations {
		t.Errorf("Stats().CdcOperations = %d, want %d", got.CdcOperations, want.CdcOperations)
	}
	if got.NetworkSentBytes != want.NetworkSentBytes {
		t.Errorf(
			"Stats().NetworkSentBytes = %d, want %d",
			got.NetworkSentBytes,
			want.NetworkSentBytes,
		)
	}
}

func TestSyncDB_Stats_ErrorWrapping(t *testing.T) {
	t.Parallel()

	sentinel := errors.New("stats unavailable")
	sdb := newSyncDBWithEngine(nil, &fakeSyncEngine{statsErr: sentinel})

	_, err := sdb.Stats(context.Background())
	if err == nil {
		t.Fatal("Stats() error = nil, want error")
	}
	if !errors.Is(err, sentinel) {
		t.Errorf("Stats() error = %v, want wrapping %v", err, sentinel)
	}
	if !isInfraError(t, err) {
		t.Errorf("Stats() error = %v, want Infrastructure family", err)
	}
}

func TestSyncDB_SyncClient_NilForTestConstructor(t *testing.T) {
	t.Parallel()

	sdb := newSyncDBWithEngine(nil, &fakeSyncEngine{})
	if sdb.SyncClient() != nil {
		t.Error("SyncClient() = non-nil, want nil for test-constructed SyncDB")
	}
}

// isInfraError reports whether err is classified as an Infrastructure error.
func isInfraError(t *testing.T, err error) bool {
	t.Helper()
	var famErr *errorfamily.Error
	if !errors.As(err, &famErr) {
		return false
	}

	return famErr.ErrorFamily() == errorfamily.Infrastructure
}

func TestSyncDB_HealthCheck_Success(t *testing.T) {
	t.Parallel()

	db, err := OpenTemp(t.TempDir())
	if err != nil {
		t.Fatalf("OpenTemp: %v", err)
	}
	defer func() { _ = db.Close() }()

	sdb := newSyncDBWithEngine(db, &fakeSyncEngine{})
	if err := sdb.HealthCheck(context.Background()); err != nil {
		t.Fatalf("HealthCheck() error = %v, want nil", err)
	}
}

func TestSyncDB_HealthCheck_ClosedDB(t *testing.T) {
	t.Parallel()

	db, err := OpenTemp(t.TempDir())
	if err != nil {
		t.Fatalf("OpenTemp: %v", err)
	}
	_ = db.Close()

	sdb := newSyncDBWithEngine(db, &fakeSyncEngine{})
	err = sdb.HealthCheck(context.Background())
	if err == nil {
		t.Fatal("HealthCheck() error = nil, want error on closed DB")
	}
	if !isInfraError(t, err) {
		t.Errorf("HealthCheck() error = %v, want Infrastructure family", err)
	}
}

func TestSyncDB_Close(t *testing.T) {
	t.Parallel()

	db, err := OpenTemp(t.TempDir())
	if err != nil {
		t.Fatalf("OpenTemp: %v", err)
	}

	sdb := newSyncDBWithEngine(db, &fakeSyncEngine{})
	if err := sdb.Close(); err != nil {
		t.Fatalf("Close() error = %v, want nil", err)
	}
}

// withSwappedFactory replaces createSyncDb for the duration of the test.
// The original is restored automatically via t.Cleanup.
// Tests using this MUST NOT call t.Parallel (shared global state).
func withSwappedFactory(
	t *testing.T,
	fn func(context.Context, tursoclient.TursoSyncDbConfig) (syncDbConnection, error),
) {
	t.Helper()

	orig := createSyncDb
	t.Cleanup(func() { createSyncDb = orig })
	createSyncDb = fn
}

func TestOpenSyncWithConfig_FactorySuccess(t *testing.T) {
	// NOT parallel — swaps package-level createSyncDb.

	testDB, err := OpenTemp(t.TempDir())
	if err != nil {
		t.Fatalf("OpenTemp: %v", err)
	}
	t.Cleanup(func() { _ = testDB.Close() })

	fakeEngine := &fakeSyncEngine{pullChanged: true}
	withSwappedFactory(
		t,
		func(_ context.Context, _ tursoclient.TursoSyncDbConfig) (syncDbConnection, error) {
			return syncDbConnection{db: testDB, engine: fakeEngine}, nil
		},
	)

	sdb, err := OpenSyncWithConfig(
		context.Background(),
		DbPath("/tmp/test.db"),
		RemoteURL("libsql://fake.turso.io"),
		AuthToken("token"),
		WithSyncClientName("test-client"),
	)
	if err != nil {
		t.Fatalf("OpenSyncWithConfig: %v", err)
	}

	if sdb.DB != testDB {
		t.Error("SyncDB.DB should be the factory-provided database")
	}

	if sdb.SyncClient() != nil {
		t.Error("SyncClient() should be nil when factory provides no sync client")
	}

	// Verify the fake engine is wired — Pull should work without network.
	changed, err := sdb.Pull(context.Background())
	if err != nil {
		t.Fatalf("Pull via factory engine: %v", err)
	}

	if !changed {
		t.Error("Pull changed = false, want true from fake engine")
	}

	if fakeEngine.pullCalls != 1 {
		t.Errorf("pullCalls = %d, want 1", fakeEngine.pullCalls)
	}
}

func TestOpenSyncWithConfig_FactoryError(t *testing.T) {
	// NOT parallel — swaps package-level createSyncDb.

	sentinel := errors.New("network unreachable")
	withSwappedFactory(
		t,
		func(_ context.Context, _ tursoclient.TursoSyncDbConfig) (syncDbConnection, error) {
			return syncDbConnection{}, sentinel
		},
	)

	_, err := OpenSyncWithConfig(
		context.Background(),
		DbPath("/tmp/test.db"),
		RemoteURL("libsql://fake.turso.io"),
		AuthToken("token"),
	)

	if err == nil {
		t.Fatal("OpenSyncWithConfig error = nil, want error")
	}

	if !errors.Is(err, sentinel) {
		t.Errorf("error should wrap sentinel: got %v", err)
	}

	if !isInfraError(t, err) {
		t.Errorf("error should be Infrastructure family: got %v", err)
	}
}

func TestOpenSyncWithConfig_OptionsAppliedToConfig(t *testing.T) {
	// NOT parallel — swaps package-level createSyncDb.

	var capturedCfg tursoclient.TursoSyncDbConfig
	withSwappedFactory(
		t,
		func(_ context.Context, cfg tursoclient.TursoSyncDbConfig) (syncDbConnection, error) {
			capturedCfg = cfg

			return syncDbConnection{}, errors.New("stop")
		},
	)

	_, _ = OpenSyncWithConfig(
		context.Background(),
		DbPath("/tmp/real.db"),
		RemoteURL("libsql://real.turso.io"),
		AuthToken("secret-token"),
		WithSyncClientName("my-app"),
		WithSyncNamespace("my-ns"),
		WithSyncBusyTimeout(5*time.Second),
	)

	if capturedCfg.Path != "/tmp/real.db" {
		t.Errorf("cfg.Path = %q, want /tmp/real.db", capturedCfg.Path)
	}

	if capturedCfg.RemoteUrl != "libsql://real.turso.io" {
		t.Errorf("cfg.RemoteUrl = %q, want libsql://real.turso.io", capturedCfg.RemoteUrl)
	}

	if capturedCfg.AuthToken != "secret-token" {
		t.Errorf("cfg.AuthToken = %q, want secret-token", capturedCfg.AuthToken)
	}

	if capturedCfg.ClientName != "my-app" {
		t.Errorf("cfg.ClientName = %q, want my-app", capturedCfg.ClientName)
	}

	if capturedCfg.Namespace != "my-ns" {
		t.Errorf("cfg.Namespace = %q, want my-ns", capturedCfg.Namespace)
	}

	if capturedCfg.BusyTimeout != 5000 {
		t.Errorf("cfg.BusyTimeout = %d, want 5000", capturedCfg.BusyTimeout)
	}
}
