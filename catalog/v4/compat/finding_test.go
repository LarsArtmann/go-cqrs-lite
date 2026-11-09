package compat

import "testing"

// TestKindVocabulary pins the stable identifiers gates and waivers match on:
// every kind must be non-empty, unique, and distinct from every other kind.
func TestKindVocabulary(t *testing.T) {
	t.Parallel()

	all := []Kind{
		KindSchemaAdded,
		KindSchemaRemoved,
		KindFieldAdded,
		KindFieldRemoved,
		KindFieldRenamed,
		KindFieldRequired,
		KindFieldOptional,
		KindTypeChanged,
		KindEnumWidened,
		KindEnumNarrowed,
		KindNullableWidened,
		KindNullableNarrowed,
		KindConstraintAdded,
		KindConstraintRemoved,
		KindDefaultChanged,
		KindDeprecated,
		KindMessageAdded,
		KindMessageRemoved,
		KindVersionNotBumped,
	}

	seen := make(map[Kind]bool, len(all))

	for _, kind := range all {
		if kind == "" {
			t.Fatal("a Kind must not be empty")
		}

		if seen[kind] {
			t.Fatalf("duplicate Kind %q", kind)
		}

		seen[kind] = true
	}
}

// TestOpVocabulary pins the migration-op names the policy prescribes.
func TestOpVocabulary(t *testing.T) {
	t.Parallel()

	all := []string{OpAddField, OpRemoveField, OpRenameField, OpTransform}

	seen := make(map[string]bool, len(all))

	for _, op := range all {
		if op == "" {
			t.Fatal("an Op name must not be empty")
		}

		if seen[op] {
			t.Fatalf("duplicate Op name %q", op)
		}

		seen[op] = true
	}
}

// TestOnlyVersionNotBumpedBlocks pins the gate contract: exactly one kind may
// fail a build, and it is the one that proves the author skipped the bump.
func TestOnlyVersionNotBumpedBlocks(t *testing.T) {
	t.Parallel()

	if KindVersionNotBumped == "" {
		t.Fatal("the blocking kind must be named")
	}

	report := CompatCatalog(
		catalogWithEvents(eventMsg("order_placed", "1.0.0", twoFields)),
		catalogWithEvents(eventMsg("order_placed", "1.0.0", oneField)),
	)

	for _, finding := range report.Blocking() {
		if finding.Kind != KindVersionNotBumped {
			t.Fatalf("unexpected blocking kind %q", finding.Kind)
		}
	}
}
