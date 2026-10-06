package metaengine_test

import (
	"context"
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// TestValidateFilterSpecs pins M04/F11+F15: operators outside the defined
// FilterOp constants are rejected with an error naming the operator and the
// column, while every registered constant passes. FilterOp is an open string
// type and the operator is spliced into SQL text — an unvalidated op is an
// injection surface, not a typo.
func TestValidateFilterSpecs(t *testing.T) {
	t.Parallel()

	for _, op := range metaengine.AllFilterOps() {
		if err := metaengine.ValidateFilterSpecs([]metaengine.FilterSpec{
			{Column: "status", Op: op, Value: "x"},
		}); err != nil {
			t.Errorf("registered op %q must validate, got: %v", string(op), err)
		}
	}

	for _, op := range []metaengine.FilterOp{
		"",
		"LIKE",
		"1=1; DROP TABLE meta_map; --",
		"= 1 OR 1=1",
		"≠",
	} {
		err := metaengine.ValidateFilterSpecs([]metaengine.FilterSpec{
			{Column: "status", Op: op, Value: "x"},
		})
		if err == nil {
			t.Errorf("op %q must be rejected", string(op))
			continue
		}

		if !strings.Contains(err.Error(), string(op)) || !strings.Contains(err.Error(), "status") {
			t.Errorf("error must name the op and column, got: %v", err)
		}
	}
}

// TestScanRejectsInvalidFilterOp pins M04/F11: the typed scan entry point
// refuses an invalid operator before any engine (closure or pushdown) renders
// SQL with it.
func TestScanRejectsInvalidFilterOp(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	store, err := metaengine.Plan(
		[]metaengine.Engine{metaengine.NewMemoryEngine()},
		findTaskQuery(),
	)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}

	defer store.Close()

	reader := metaengine.NewReader[FindTaskResult](store, "find_task")

	_, err = reader.Scan(ctx,
		metaengine.WithFilter("status", "1=1; DROP TABLE meta_map; --", "open"))
	if err == nil {
		t.Fatal("Scan must reject an invalid filter operator")
	}

	if !strings.Contains(err.Error(), "invalid filter operator") {
		t.Errorf("error should name the invalid operator, got: %v", err)
	}
}

// TestScanRejectsHostileColumns pins M04/F14 at the scan entry: filter and
// sort column names carry quotes/parens/semicolons into json_extract path
// expressions and planned-table identifiers, so they must pass the shared
// identifier allowlist before any SQL is built.
func TestScanRejectsHostileColumns(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	store, err := metaengine.Plan(
		[]metaengine.Engine{metaengine.NewMemoryEngine()},
		findTaskQuery(),
	)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}

	defer store.Close()

	reader := metaengine.NewReader[FindTaskResult](store, "find_task")

	hostileColumns := []string{
		"x') OR ('1'='1",
		`status" --`,
		"id; DROP TABLE meta_map",
		"id=1 OR 2=2",
		"col`desc",
	}
	for _, column := range hostileColumns {
		if _, err := reader.Scan(
			ctx,
			metaengine.WithFilter(column, metaengine.FilterEq, "x"),
		); err == nil {
			t.Errorf("Scan must reject hostile filter column %q", column)
		}

		if _, err := reader.Scan(ctx, metaengine.WithSort(column, false)); err == nil {
			t.Errorf("Scan must reject hostile sort column %q", column)
		}
	}

	for _, column := range []string{"status", "user.Name", "created-at", "外のキー"} {
		if _, err := reader.Scan(
			ctx,
			metaengine.WithFilter(column, metaengine.FilterEq, "x"),
		); err != nil {
			t.Errorf("Scan must accept benign filter column %q, got: %v", column, err)
		}
	}
}

// TestValidateIdentifier pins M04/F13: the shared allowlist (formerly the
// matview-local validateMatViewString) accepts the identifier vocabulary and
// rejects injection-relevant ASCII.
func TestValidateIdentifier(t *testing.T) {
	t.Parallel()

	for _, s := range []string{"tasks", "user_id", "a.b", "created-at", "with space", "外のキー", "Col42"} {
		if err := metaengine.ValidateIdentifier(s); err != nil {
			t.Errorf("ValidateIdentifier(%q) = %v, want nil", s, err)
		}
	}

	for _, s := range []string{"'", `"`, "a;b", "f(x)", "a=b", "`tick`", "[bracket]", "a\tb"} {
		if err := metaengine.ValidateIdentifier(s); err == nil {
			t.Errorf("ValidateIdentifier(%q) = nil, want error", s)
		}
	}
}
