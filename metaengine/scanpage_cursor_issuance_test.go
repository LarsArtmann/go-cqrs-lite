package metaengine_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/record/v4"
)

// Fixtures for the compound-cursor issuance protocol (T07): ScanPage must
// return the engine-issued SortKeyCursor so tie-heavy datasets paginate
// exactly once, and ParseCursor must normalize the encoded compound shape
// back into a SortKeyCursor on the string round-trip.

type tieItem struct {
	ID       string
	Status   string
	Priority int
}

type tieListInput struct {
	Status string
}

func tieScanQuery() metaengine.QueryDecl[tieListInput, tieItem] {
	return metaengine.Query[tieListInput, tieItem](
		"tie_scan",
		metaengine.OnRecord(
			tieItem{},
			func(_ record.Record, e tieItem) (string, tieItem) { return e.ID, e },
		),
		metaengine.FilterOnField[tieItem]("Status", metaengine.FilterEq),
		metaengine.SortOnField[tieItem]("Priority", true),
	)
}

// setupTieScan seeds n items whose Priority is i%3 — 3-item tie blocks that
// any limit not divisible by 3 straddles across page boundaries.
func setupTieScan(t *testing.T, n int) *metaengine.TypedReader[tieItem] {
	t.Helper()

	store, err := metaengine.Plan([]metaengine.Engine{metaengine.NewMemoryEngine()}, tieScanQuery())
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}

	t.Cleanup(func() { _ = store.Close() })

	ctx := context.Background()

	for i := range n {
		item := tieItem{
			ID:       fmt.Sprintf("item-%03d", i),
			Status:   "open",
			Priority: i % 3,
		}

		if err := store.Apply(ctx, "tieItem", item); err != nil {
			t.Fatalf("seed Apply[%d]: %v", i, err)
		}
	}

	return metaengine.NewReader[tieItem](store, "tie_scan")
}

func tieIDs(items []tieItem) []string {
	ids := make([]string, len(items))
	for i, item := range items {
		ids[i] = item.ID
	}

	return ids
}

// walkPages drives ScanPage to exhaustion, passing each cursor back in the
// given mode — raw (WithCursor(cursor.Value)) or encoded string
// (WithCursorString) — and returns every visited ID plus each page's size.
// The 50-page bound is a safety net: a cursor that re-serves rows must fail
// the exact-once assertions, not hang the suite.
func walkPages(
	t *testing.T,
	reader *metaengine.TypedReader[tieItem],
	limit int,
	asString bool,
) (visited []string, pageSizes []int) {
	t.Helper()

	ctx := context.Background()

	nextRaw := any(nil)
	nextStr := ""

	for range 50 {
		opts := []metaengine.ScanOption{
			metaengine.WithSort("Priority", false),
			metaengine.WithLimit(limit),
		}

		switch {
		case asString && nextStr != "":
			opts = append(opts, metaengine.WithCursorString(nextStr))
		case !asString && nextRaw != nil:
			opts = append(opts, metaengine.WithCursor(nextRaw))
		}

		items, cursor, err := reader.ScanPage(ctx, opts...)
		if err != nil {
			t.Fatalf("ScanPage: %v", err)
		}

		visited = append(visited, tieIDs(items)...)
		pageSizes = append(pageSizes, len(items))

		if cursor == nil {
			return visited, pageSizes
		}

		if asString {
			nextStr, err = cursor.Encode()
			if err != nil {
				t.Fatalf("cursor Encode: %v", err)
			}
		} else {
			nextRaw = cursor.Value
		}
	}

	t.Fatalf("pagination did not terminate within 50 pages (cursor loop?): pages %v, visited %v", pageSizes, visited)

	return nil, nil
}

// TestScanPage_CompoundCursor_TieHeavyExactOnce pins the compound-cursor
// contract: with i%3 priorities and limit 5, page 2 ends INSIDE a Priority-2
// tie block. A value-only cursor would skip the whole block (3 dropped rows);
// the engine-issued (sort, key) cursor must walk all 13 items exactly once.
func TestScanPage_CompoundCursor_TieHeavyExactOnce(t *testing.T) {
	t.Parallel()

	const (
		items = 13
		limit = 5
	)

	want := []string{
		"item-000", "item-003", "item-006", "item-009", "item-012",
		"item-001", "item-004", "item-007", "item-010", "item-002",
		"item-005", "item-008", "item-011",
	}

	for _, tc := range []struct {
		name     string
		asString bool
	}{
		{name: "raw cursor", asString: false},
		{name: "encoded string cursor", asString: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			reader := setupTieScan(t, items)

			visited, pages := walkPages(t, reader, limit, tc.asString)

			if !reflect.DeepEqual(visited, want) {
				t.Fatalf("walk visited %v\nwant           %v", visited, want)
			}

			wantPages := []int{limit, limit, items - 2*limit}
			if !reflect.DeepEqual(pages, wantPages) {
				t.Fatalf("page sizes %v, want %v", pages, wantPages)
			}
		})
	}
}

// TestScanPage_CompoundCursor_ExactEndReturnsNilCursor pins the phantom-page
// kill: 12 items with limit 4 end exactly at a page boundary, so the final
// page is FULL yet the engine's has-more probe reports exhaustion — ScanPage
// must return a nil cursor instead of minting one into an empty next page.
func TestScanPage_CompoundCursor_ExactEndReturnsNilCursor(t *testing.T) {
	t.Parallel()

	const (
		items = 12
		limit = 4
	)

	want := []string{
		"item-000", "item-003", "item-006", "item-009",
		"item-001", "item-004", "item-007", "item-010",
		"item-002", "item-005", "item-008", "item-011",
	}

	reader := setupTieScan(t, items)

	visited, pages := walkPages(t, reader, limit, false)

	if !reflect.DeepEqual(visited, want) {
		t.Fatalf("walk visited %v\nwant           %v", visited, want)
	}

	wantPages := []int{limit, limit, limit}
	if !reflect.DeepEqual(pages, wantPages) {
		t.Fatalf("page sizes %v, want %v (no trailing empty phantom page)", pages, wantPages)
	}
}

// TestParseCursor_CompoundRoundTrip pins the wire format (exactly the two
// keys "Sort" and "Key", Key base64) and that ParseCursor normalizes it back
// into a SortKeyCursor regardless of the JSON number decode flavor.
func TestParseCursor_CompoundRoundTrip(t *testing.T) {
	t.Parallel()

	cursor := &metaengine.Cursor{
		Value: metaengine.SortKeyCursor{Sort: 2, Key: []byte("item-007")},
	}

	encoded, err := cursor.Encode()
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}

	raw, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		t.Fatalf("base64 decode: %v", err)
	}

	var wire map[string]any
	if err := json.Unmarshal(raw, &wire); err != nil {
		t.Fatalf("wire JSON: %v", err)
	}

	if len(wire) != 2 || fmt.Sprintf("%v", wire["Sort"]) != "2" || wire["Key"] != "aXRlbS0wMDc=" {
		t.Fatalf("wire shape %v, want exactly {Sort: 2, Key: aXRlbS0wMDc=}", wire)
	}

	parsed, err := metaengine.ParseCursor(encoded)
	if err != nil {
		t.Fatalf("ParseCursor: %v", err)
	}

	skc, ok := parsed.Value.(metaengine.SortKeyCursor)
	if !ok {
		t.Fatalf("parsed value %T, want SortKeyCursor", parsed.Value)
	}

	if fmt.Sprintf("%v", skc.Sort) != "2" || !bytes.Equal(skc.Key, []byte("item-007")) {
		t.Fatalf("parsed cursor %+v, want Sort=2 Key=item-007", skc)
	}
}

// TestParseCursor_LegacyAndMalformedShapesDegrade pins the guard rails:
// only the exact two-key compound shape normalizes; everything else keeps
// the raw decoded value (the legacy value-cursor contract).
func TestParseCursor_LegacyAndMalformedShapesDegrade(t *testing.T) {
	t.Parallel()

	encodePayload := func(t *testing.T, payload any) string {
		t.Helper()

		rawJSON, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("marshal payload: %v", err)
		}

		return base64.RawURLEncoding.EncodeToString(rawJSON)
	}

	for _, tc := range []struct {
		name    string
		payload any
	}{
		{name: "missing key", payload: map[string]any{"Sort": 2}},
		{name: "missing sort", payload: map[string]any{"Key": "aw=="}},
		{name: "key not a string", payload: map[string]any{"Sort": "x", "Key": 42}},
		{name: "key not base64", payload: map[string]any{"Sort": "x", "Key": "!!!"}},
		{name: "extra keys", payload: map[string]any{"Sort": 1, "Key": "aw==", "Extra": true}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			parsed, err := metaengine.ParseCursor(encodePayload(t, tc.payload))
			if err != nil {
				t.Fatalf("ParseCursor: %v", err)
			}

			if _, ok := parsed.Value.(metaengine.SortKeyCursor); ok {
				t.Fatalf("payload %v normalized to SortKeyCursor; malformed shapes must degrade", tc.payload)
			}

			if _, ok := parsed.Value.(map[string]any); !ok {
				t.Fatalf("parsed value %T, want the raw decoded map", parsed.Value)
			}
		})
	}

	t.Run("string value stays string", func(t *testing.T) {
		t.Parallel()

		encoded, err := (&metaengine.Cursor{Value: "abc"}).Encode()
		if err != nil {
			t.Fatalf("Encode: %v", err)
		}

		parsed, err := metaengine.ParseCursor(encoded)
		if err != nil {
			t.Fatalf("ParseCursor: %v", err)
		}

		if parsed.Value != "abc" {
			t.Fatalf("parsed value %#v, want the plain string \"abc\"", parsed.Value)
		}
	})

	t.Run("empty string is start of stream", func(t *testing.T) {
		t.Parallel()

		parsed, err := metaengine.ParseCursor("")
		if parsed != nil || err != nil {
			t.Fatalf("ParseCursor(\"\") = %v, %v; want nil, nil", parsed, err)
		}
	})
}
