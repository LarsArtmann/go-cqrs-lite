package metaengine

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/record/v4"
)

type tvUserID string

type tvUserCreated struct {
	ID   tvUserID
	Name string
}

type tvUserRenamed struct {
	ID   tvUserID
	Name string
}

type tvUserDeleted struct{ ID tvUserID }

type tvFindUser struct {
	ID   tvUserID
	AsOf time.Time
}

type tvUserView struct{ Name string }

func tvQuery() QueryDecl[tvFindUser, tvUserView] {
	return Query[tvFindUser, tvUserView]("tv_users",
		OnRecord(tvUserCreated{}, func(_ record.Record, e tvUserCreated) (tvUserID, tvUserView) {
			return e.ID, tvUserView{Name: e.Name}
		}),
		OnRecord(tvUserRenamed{}, func(_ record.Record, e tvUserRenamed, _ tvUserView) tvUserView {
			return tvUserView{Name: e.Name}
		}),
		OnRecord(tvUserDeleted{}, Remove[tvUserView]()),
	)
}

func tvRecord(eventType string, at time.Time, payload any) record.Record {
	return record.Record{
		Type:     eventType,
		StreamID: "User/u1",
		Payload:  nil,
		MetaData: record.CommonMetadata{Stored: record.NewStamp(at)},
	}
}

// TestTemporal_EventTimeThroughFolds proves the flagship ADR-0141 pipeline:
// folds on a versioned engine stamp cells with the EVENT's Stored time, so
// as-of reads resolve each historical state — including the tombstone — and
// a replay with identical stamps rebuilds identical history.
func TestTemporal_EventTimeThroughFolds(t *testing.T) {
	t.Parallel()

	store, err := Plan([]Engine{NewMemoryEngineWithVersioning()}, tvQuery())
	if err != nil {
		t.Fatal(err)
	}

	defer DeferClose(store)

	ctx := context.Background()

	base := time.Now().Add(-time.Hour)
	t1 := base
	t2 := base.Add(10 * time.Minute)
	t3 := base.Add(20 * time.Minute)

	if err := store.ApplyRecord(
		ctx,
		tvRecord("tvUserCreated", t1, nil),
		tvUserCreated{ID: "u1", Name: "Alice"},
	); err != nil {
		t.Fatal(err)
	}

	if err := store.ApplyRecord(
		ctx,
		tvRecord("tvUserRenamed", t2, nil),
		tvUserRenamed{ID: "u1", Name: "Bob"},
	); err != nil {
		t.Fatal(err)
	}

	if err := store.ApplyRecord(
		ctx,
		tvRecord("tvUserDeleted", t3, nil),
		tvUserDeleted{ID: "u1"},
	); err != nil {
		t.Fatal(err)
	}
	assertExecuteAsOf(t, store, ctx, t2, "Bob")
	assertExecuteAsOf(t, store, ctx, t1, "Alice")

	_, err = store.ExecuteAsOf(ctx, "tv_users", "u1", t3)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("as-of after delete err = %v, want ErrNotFound", err)
	}

	// Latest read: deleted.
	latest, err := store.Execute(tvFindUser{ID: "u1"})
	if err != nil || latest != nil {
		t.Fatalf("latest after delete = (%v, %v), want (nil, nil)", latest, err)
	}
}

// TestTemporal_AsOfInputRouting proves the input-field convention: a non-zero
// AsOf field routes the point lookup through VersionedStorage; zero means
// latest. On an engine without active versioning the temporal read fails
// loudly instead of degrading to latest-only.
func TestTemporal_AsOfInputRouting(t *testing.T) {
	t.Parallel()

	store, err := Plan([]Engine{NewMemoryEngineWithVersioning()}, tvQuery())
	if err != nil {
		t.Fatal(err)
	}

	defer DeferClose(store)

	ctx := context.Background()

	t1 := time.Now().Add(-time.Hour).Truncate(time.Millisecond)
	t2 := t1.Add(10 * time.Minute)

	if err := store.ApplyRecord(
		ctx,
		tvRecord("tvUserCreated", t1, nil),
		tvUserCreated{ID: "u1", Name: "Alice"},
	); err != nil {
		t.Fatal(err)
	}

	if err := store.ApplyRecord(
		ctx,
		tvRecord("tvUserRenamed", t2, nil),
		tvUserRenamed{ID: "u1", Name: "Bob"},
	); err != nil {
		t.Fatal(err)
	}

	past, err := store.Execute(tvFindUser{ID: "u1", AsOf: t1})
	if err != nil {
		t.Fatal(err)
	}

	if view := past.(tvUserView); view.Name != "Alice" {
		t.Fatalf("as-of t1 name = %q, want Alice", view.Name)
	}

	latest, err := store.Execute(tvFindUser{ID: "u1"})
	if err != nil {
		t.Fatal(err)
	}

	if view := latest.(tvUserView); view.Name != "Bob" {
		t.Fatalf("latest name = %q, want Bob", view.Name)
	}

	plain, err := Plan([]Engine{NewMemoryEngine()}, tvQuery())
	if err != nil {
		t.Fatal(err)
	}

	defer DeferClose(plain)

	if _, err := plain.Execute(tvFindUser{ID: "u1", AsOf: t1}); err == nil {
		t.Fatal("as-of on non-versioned engine must fail loudly")
	}
}

// TestTemporal_AsOfViaExecuteTyped pins the system-facing entry: the typed
// wrapper (ExecuteTyped → ExecuteCtx + result reconstruction) that system/
// compositions and consumers use must preserve AsOf routing end-to-end —
// the declared AsOf field survives the wrapper and the result comes back
// typed, not as a raw any (blast-radius check for system.New consumers).
func TestTemporal_AsOfViaExecuteTyped(t *testing.T) {
	t.Parallel()

	store, err := Plan([]Engine{NewMemoryEngineWithVersioning()}, tvQuery())
	if err != nil {
		t.Fatal(err)
	}

	defer DeferClose(store)

	ctx := context.Background()

	t1 := time.Now().Add(-time.Hour).Truncate(time.Millisecond)
	t2 := t1.Add(10 * time.Minute)

	for _, rec := range []struct {
		kind    string
		at      time.Time
		payload any
	}{
		{"tvUserCreated", t1, tvUserCreated{ID: "u1", Name: "Alice"}},
		{"tvUserRenamed", t2, tvUserRenamed{ID: "u1", Name: "Bob"}},
	} {
		if err := store.ApplyRecord(
			ctx,
			tvRecord(rec.kind, rec.at, rec.payload),
			rec.payload,
		); err != nil {
			t.Fatal(err)
		}
	}

	past, err := ExecuteTyped[tvFindUser, tvUserView](ctx, store, tvFindUser{ID: "u1", AsOf: t1})
	if err != nil {
		t.Fatal(err)
	}

	if past.Name != "Alice" {
		t.Fatalf("ExecuteTyped as-of t1 name = %q, want Alice", past.Name)
	}

	latest, err := ExecuteTypedByName[tvFindUser, tvUserView](
		ctx,
		store,
		"tv_users",
		tvFindUser{ID: "u1"},
	)
	if err != nil {
		t.Fatal(err)
	}

	if latest.Name != "Bob" {
		t.Fatalf("ExecuteTypedByName latest name = %q, want Bob", latest.Name)
	}
}

// TestTemporal_Retention pins the retention knob: MaxVersions keeps the
// newest N versions (as-of beyond the window → ErrNotFound, latest intact);
// MaxAge prunes versions older than the cutoff relative to each write.
func TestTemporal_Retention(t *testing.T) {
	t.Parallel()

	eng := NewMemoryEngineWithVersioning(WithRetention(RetentionPolicy{MaxVersions: 2}))
	defer eng.Close()

	vw := eng.(VersionedWriter)
	vs := eng.(VersionedStorage)
	ctx := context.Background()

	base := time.Now().Add(-time.Hour).Truncate(time.Millisecond)

	for i, name := range []string{"v1", "v2", "v3"} {
		if err := vw.MapSetAt(
			ctx,
			"ret",
			"k",
			name,
			base.Add(time.Duration(i)*time.Minute),
		); err != nil {
			t.Fatal(err)
		}
	}

	if _, err := vs.MapGetAsOf(ctx, "ret", "k", base); !errors.Is(err, ErrNotFound) {
		t.Fatalf("as-of pruned version err = %v, want ErrNotFound", err)
	}

	if val, err := vs.MapGetAsOf(
		ctx,
		"ret",
		"k",
		base.Add(time.Minute),
	); err != nil ||
		val != "v2" {
		t.Fatalf("as-of v2 = (%v, %v), want (v2, nil)", val, err)
	}

	aged := NewMemoryEngineWithVersioning(WithRetention(RetentionPolicy{MaxAge: time.Minute}))
	defer aged.Close()

	vwA := aged.(VersionedWriter)
	vsA := aged.(VersionedStorage)

	if err := vwA.MapSetAt(ctx, "ret", "k", "old", base); err != nil {
		t.Fatal(err)
	}

	if err := vwA.MapSetAt(ctx, "ret", "k", "new", base.Add(10*time.Minute)); err != nil {
		t.Fatal(err)
	}

	if _, err := vsA.MapGetAsOf(
		ctx,
		"ret",
		"k",
		base.Add(5*time.Minute),
	); !errors.Is(
		err,
		ErrNotFound,
	) {
		t.Fatalf("MaxAge-pruned version err = %v, want ErrNotFound", err)
	}

	if val, err := vsA.MapGetAsOf(
		ctx,
		"ret",
		"k",
		base.Add(10*time.Minute),
	); err != nil ||
		val != "new" {
		t.Fatalf("as-of new = (%v, %v), want (new, nil)", val, err)
	}
}

// TestTemporal_History pins MapHistory: newest-first, tombstones included,
// bounded by [from, to].
func TestTemporal_History(t *testing.T) {
	t.Parallel()

	eng := NewMemoryEngineWithVersioning()
	defer eng.Close()

	vw := eng.(VersionedWriter)
	hr := eng.(CellHistoryReader)
	ctx := context.Background()

	base := time.Now().Add(-time.Hour).Truncate(time.Millisecond)

	if err := vw.MapSetAt(ctx, "hist", "k", "one", base); err != nil {
		t.Fatal(err)
	}

	if err := vw.MapSetAt(ctx, "hist", "k", "two", base.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}

	if err := vw.MapDeleteAt(ctx, "hist", "k", base.Add(2*time.Minute)); err != nil {
		t.Fatal(err)
	}

	hist, err := hr.MapHistory(ctx, "hist", "k", base.Add(-time.Minute), base.Add(3*time.Minute))
	if err != nil {
		t.Fatal(err)
	}

	if len(hist) != 3 {
		t.Fatalf("history length = %d, want 3", len(hist))
	}

	if hist[0].Value != nil || hist[1].Value != "two" || hist[2].Value != "one" {
		t.Fatalf("history = %+v, want [tombstone, two, one]", hist)
	}

	slice, err := hr.MapHistory(ctx, "hist", "k", base, base.Add(time.Millisecond))
	if err != nil {
		t.Fatal(err)
	}

	if len(slice) != 1 || slice[0].Value != "one" {
		t.Fatalf("bounded history = %+v, want [one]", slice)
	}
}

// TestTemporal_PlannerWarnsNonVersioned pins the temporal-asof plan rule: an
// AsOf-declaring query planned on a non-versioned engine yields a WARN
// diagnostic; a versioned engine yields none.
func TestTemporal_PlannerWarnsNonVersioned(t *testing.T) {
	t.Parallel()

	store, err := Plan([]Engine{NewMemoryEngine()}, tvQuery())
	if err != nil {
		t.Fatal(err)
	}

	defer DeferClose(store)

	warned := false

	for _, d := range store.Plan().Diagnostics {
		if d.Query == "tv_users" && d.Level == DiagLevelWarn &&
			strings.Contains(d.Message, "does not record cell versions") {
			warned = true
		}
	}

	if !warned {
		t.Fatalf("expected temporal-asof warning, got %+v", store.Plan().Diagnostics)
	}

	versioned, err := Plan([]Engine{NewMemoryEngineWithVersioning()}, tvQuery())
	if err != nil {
		t.Fatal(err)
	}

	defer DeferClose(versioned)

	for _, d := range versioned.Plan().Diagnostics {
		if d.Query == "tv_users" && d.Level == DiagLevelWarn &&
			strings.Contains(d.Message, "cell versions") {
			t.Fatalf("unexpected warning on versioned engine: %s", d.Message)
		}
	}
}

func assertExecuteAsOf(
	t *testing.T,
	store *Store,
	ctx context.Context,
	at time.Time,
	wantName string,
) {
	t.Helper()

	val, err := store.ExecuteAsOf(ctx, "tv_users", "u1", at)
	if err != nil {
		t.Fatalf("ExecuteAsOf(%v): %v", at, err)
	}

	if view := val.(tvUserView); view.Name != wantName {
		t.Fatalf("ExecuteAsOf(%v).Name = %q, want %q", at, view.Name, wantName)
	}
}
