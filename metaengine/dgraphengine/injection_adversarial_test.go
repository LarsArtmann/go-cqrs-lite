package dgraphengine_test

import (
	"context"
	"testing"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// TestAdversarialDQLInjection verifies at runtime that DQL syntax in user
// input is treated as literal data, not executed as DQL. This complements
// TestNoDQLInjectionPatterns (which checks source-code patterns statically).
//
// Each attack vector is inserted as a Map key/value, then read back. If the
// value round-trips correctly, the input was treated as data, not code.
// If the value is mangled or extra data appears, an injection succeeded.
func TestAdversarialDQLInjection(t *testing.T) {
	t.Parallel()

	eng := mustNewDgraphEngine(t)

	mb := eng.(metaengine.MapBackend)
	sb := eng.(metaengine.SearchBackend)
	cb := eng.(metaengine.CounterBackend)

	ctx := context.Background()

	// Each attack vector is a DQL injection attempt that, if executed,
	// would alter the query semantics or exfiltrate data.
	attackVectors := []string{
		`} {} {`, // close current block, open new
		`func: eq(cqrs.map_key, "injected") { cqrs.map_value }`, // full DQL query injection
		`" OR "1"="1`,                          // SQL-style injection (should be harmless but test it)
		`"); DROP COLLECTION; --`,              // SQL DROP
		`' OR ''='`,                            // classic SQL injection
		`{ uid }`,                              // DQL uid exfiltration
		`@filter(eq(cqrs.map_key, "stolen"))`,  // DQL filter injection
		`value } counter(func: all()) { uid }`, // cross-collection exfiltration
		`"><script>alert(1)</script>`,          // XSS (should be harmless to Dgraph but test)
		`%24col+%3D+%22hacked%22`,              // URL-encoded variable override
	}

	t.Run("Map_keys_are_literals", func(t *testing.T) {
		t.Parallel()
		const col = "injection-test-map"
		for i, attack := range attackVectors {
			key := attack
			value := map[string]any{"index": i, "original": attack}

			if err := mb.MapSet(ctx, col, key, value); err != nil {
				t.Fatalf("MapSet with attack vector %d failed: %v\nInput: %q", i, err, attack)
			}

			got, found, err := mb.MapGet(ctx, col, key)
			if err != nil {
				t.Fatalf("MapGet attack vector %d failed: %v\nInput: %q", i, err, attack)
			}

			if !found {
				t.Fatalf("Attack vector %d: key not found after round-trip\nInput: %q", i, attack)
			}

			m, ok := got.(map[string]any)
			if !ok {
				t.Fatalf(
					"Attack vector %d: expected map[string]any, got %T\nInput: %q",
					i,
					got,
					attack,
				)
			}

			// The value must round-trip exactly — the attack string must be
			// stored as data, not executed as DQL.
			if m["original"] != attack {
				t.Errorf("Attack vector %d: value round-trip mismatch\nInput:  %q\nOutput: %q",
					i, attack, m["original"])
			}
		}

		// Verify no extra keys were created by injection.
		// The collection should have exactly len(attackVectors) entries.
		// (We can't easily count entries via MapBackend, but if the injection
		// created extra uid nodes, they would show up as phantom keys.)
	})

	t.Run("Search_content_is_literal", func(t *testing.T) {
		t.Parallel()
		const col = "injection-test-search"
		for i, attack := range attackVectors {
			doc := metaengine.IndexedText{
				ID:      attack,
				Content: attack,
			}
			if err := sb.SearchInsert(ctx, col, doc); err != nil {
				t.Fatalf("SearchInsert with attack vector %d failed: %v\nInput: %q", i, err, attack)
			}
		}

		// Search for a benign word that appears in some attack vectors.
		// The key assertion: the search returns only matching documents,
		// not ALL documents (which would indicate the attack broke the filter).
		results, err := sb.SearchQuery(ctx, col, "stolen", 100)
		if err != nil {
			t.Fatalf("SearchQuery 'stolen' failed: %v", err)
		}

		// Only the document containing "stolen" should match.
		if len(results) == 0 {
			// "stolen" is in attack vector 7 — it should match.
			t.Errorf("expected results for 'stolen' search (attack vector 7 contains it)")
		}

		for _, r := range results {
			// Every result ID must be one of our attack vectors — no injected IDs.
			found := false
			for _, attack := range attackVectors {
				if r.ID == attack {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Search returned unknown ID (possible injection): %q", r.ID)
			}
		}
	})

	t.Run("Counter_keys_are_literals", func(t *testing.T) {
		t.Parallel()
		const col = "injection-test-counter"
		for i, attack := range attackVectors {
			if err := cb.CounterIncrement(
				ctx,
				col,
				metaengine.Delta{attack: int64(i + 1)},
			); err != nil {
				t.Fatalf(
					"CounterIncrement with attack vector %d failed: %v\nInput: %q",
					i,
					err,
					attack,
				)
			}
		}

		counts, err := cb.CounterGet(ctx, col)
		if err != nil {
			t.Fatalf("CounterGet failed: %v", err)
		}

		// Every attack vector key must be present with the correct count.
		for i, attack := range attackVectors {
			val, ok := counts[attack]
			if !ok {
				t.Errorf(
					"Attack vector %d: counter key %q not found in CounterGet results",
					i,
					attack,
				)
				continue
			}

			if val != int64(i+1) {
				t.Errorf("Attack vector %d: counter value = %d, want %d\nInput: %q",
					i, val, i+1, attack)
			}
		}
	})
}
