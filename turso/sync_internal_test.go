package turso

import (
	"context"
	"errors"
	"testing"

	tursoclient "turso.tech/database/tursogo"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
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
	var famErr *event.Error
	if !errors.As(err, &famErr) {
		return false
	}

	return famErr.ErrorFamily() == event.Infrastructure
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
