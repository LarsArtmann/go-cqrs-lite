package adttest

import (
	"fmt"
	"testing"
	"time"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/record/v4"
)

type groupedCapEvent struct {
	ID    string
	Owner string
	Seq   int
}

type plainCapEvent struct {
	ID  string
	Seq int
}

type plainCapItem struct {
	ID  string
	Seq int
}

// AssertPlainCapContract pins the plain WithCap contract (keep-newest-N,
// #72) on one engine, with the Go trim path over whatever item rendering the
// engine produces — SQL engines hand back decoded JSON documents
// (map[string]any), not typed structs, and the key extractor must read both.
func AssertPlainCapContract(t *testing.T, eng metaengine.Engine) {
	t.Helper()

	suffix := fmt.Sprintf("_%d", time.Now().UnixNano())

	query := metaengine.Query[struct{}, plainCapItem]("plain_cap_items"+suffix,
		metaengine.OnRecord(plainCapEvent{}, func(_ record.Record, e plainCapEvent) (string, plainCapItem) {
			return e.ID, plainCapItem(e)
		}),
		metaengine.WithCap(3, "Seq"),
	)

	store, err := metaengine.Plan([]metaengine.Engine{eng}, query)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}

	defer func() { _ = store.Close() }()

	ctx := t.Context()

	for i := 1; i <= 6; i++ {
		evt := plainCapEvent{ID: fmt.Sprintf("e%d", i), Seq: i}
		if err := store.Apply(ctx, "plainCapEvent", evt); err != nil {
			t.Fatalf("Apply e%d: %v", i, err)
		}
	}

	name := "plain_cap_items" + suffix

	removed, err := store.EnforceCap(ctx, name)
	if err != nil {
		t.Fatalf("EnforceCap: %v", err)
	}

	if removed != 3 {
		t.Fatalf("EnforceCap removed %d rows, want 3", removed)
	}

	reader := metaengine.NewReader[plainCapItem](store, name)

	rows, err := reader.Scan(ctx, metaengine.WithLimit(0), metaengine.WithSort("Seq", false))
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}

	if len(rows) != 3 || rows[0].Seq != 4 || rows[2].Seq != 6 {
		t.Fatalf("after trim: %+v, want Seq 4,5,6 (newest three survive)", rows)
	}

	again, err := store.EnforceCap(ctx, name)
	if err != nil {
		t.Fatalf("second EnforceCap: %v", err)
	}

	if again != 0 {
		t.Fatalf("second EnforceCap removed %d rows, want 0 (idempotent)", again)
	}
}

type groupedCapItem struct {
	ID    string
	Owner string
	Seq   int
}

// AssertGroupedCapContract pins the per-group cap contract (ADR-0164, #114)
// on one engine: WithCapPerGroup keeps the newest N rows PER distinct group
// value, groups are isolated from each other's floods, the trim is
// idempotent, and EnforceCap's global ceiling still applies above it. The
// engine exercises its own enforcement path (window-function pushdown where
// implemented, the store's Go scan otherwise) — the observable contract is
// identical. Run from every engine suite. Extra query options (e.g.
// WithColumnarLayout) are appended to the declaration so a suite can steer
// the layout onto the path under test.
func AssertGroupedCapContract(
	t *testing.T,
	eng metaengine.Engine,
	opts ...metaengine.QueryOption,
) {
	t.Helper()

	suffix := fmt.Sprintf("_%d", time.Now().UnixNano())

	args := make([]any, 0, 3+len(opts))
	args = append(args,
		metaengine.OnRecord(groupedCapEvent{}, func(_ record.Record, e groupedCapEvent) (string, groupedCapItem) {
			return e.ID, groupedCapItem(e)
		}),
		metaengine.WithCap(8, "Seq"),
		metaengine.WithCapPerGroup("Owner", 2),
	)

	for _, opt := range opts {
		args = append(args, opt)
	}

	query := metaengine.Query[struct{}, groupedCapItem]("grouped_cap_items"+suffix, args...)

	store, err := metaengine.Plan([]metaengine.Engine{eng}, query)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}

	defer func() { _ = store.Close() }()

	ctx := t.Context()

	// Three owners; owner "a" floods (6 rows), "b" and "c" stay within cap.
	seed := []groupedCapEvent{
		{ID: "a1", Owner: "a", Seq: 1},
		{ID: "a2", Owner: "a", Seq: 2},
		{ID: "a3", Owner: "a", Seq: 3},
		{ID: "a4", Owner: "a", Seq: 4},
		{ID: "a5", Owner: "a", Seq: 5},
		{ID: "a6", Owner: "a", Seq: 6},
		{ID: "b1", Owner: "b", Seq: 1},
		{ID: "c1", Owner: "c", Seq: 1},
	}

	for _, evt := range seed {
		if err := store.Apply(ctx, "groupedCapEvent", evt); err != nil {
			t.Fatalf("Apply %s: %v", evt.ID, err)
		}
	}

	name := "grouped_cap_items" + suffix

	removed, err := store.EnforceCap(ctx, name)
	if err != nil {
		t.Fatalf("EnforceCap: %v", err)
	}

	if removed != 4 {
		t.Fatalf("EnforceCap removed %d rows, want 4 (a keeps Seq 5,6; global cap 8 holds)", removed)
	}

	assertGroupedRows(t, store, name, map[string][]int{
		"a": {5, 6},
		"b": {1},
		"c": {1},
	})

	again, err := store.EnforceCap(ctx, name)
	if err != nil {
		t.Fatalf("second EnforceCap: %v", err)
	}

	if again != 0 {
		t.Fatalf("second EnforceCap removed %d rows, want 0 (idempotent)", again)
	}

	// A second flood from owner "b" evicts only b's oldest rows; a, c untouched.
	for i := 2; i <= 5; i++ {
		if err := store.Apply(ctx, "groupedCapEvent", groupedCapEvent{
			ID: fmt.Sprintf("b%d", i), Owner: "b", Seq: i,
		}); err != nil {
			t.Fatalf("Apply b%d: %v", i, err)
		}
	}

	if _, err := store.EnforceCap(ctx, name); err != nil {
		t.Fatalf("EnforceCap after b flood: %v", err)
	}

	assertGroupedRows(t, store, name, map[string][]int{
		"a": {5, 6},
		"b": {4, 5},
		"c": {1},
	})
}

func assertGroupedRows(
	t *testing.T,
	store *metaengine.Store,
	name string,
	want map[string][]int,
) {
	t.Helper()

	reader := metaengine.NewReader[groupedCapItem](store, name)

	rows, err := reader.Scan(
		t.Context(),
		metaengine.WithLimit(0),
		metaengine.WithSort("Seq", false),
	)
	if err != nil {
		t.Fatalf("Scan %s: %v", name, err)
	}

	got := make(map[string][]int)

	for _, row := range rows {
		got[row.Owner] = append(got[row.Owner], row.Seq)
	}

	if len(got) != len(want) {
		t.Fatalf("after trim: %d groups (%v), want %d", len(got), got, len(want))
	}

	for owner, seqs := range want {
		if len(got[owner]) != len(seqs) {
			t.Fatalf("owner %s: rows %v, want %v (cross-group isolation)", owner, got[owner], seqs)
		}

		for i, seq := range seqs {
			if got[owner][i] != seq {
				t.Fatalf("owner %s: rows %v, want %v (newest survive)", owner, got[owner], seqs)
			}
		}
	}
}
