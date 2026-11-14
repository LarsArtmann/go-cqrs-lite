package main

import (
	"context"
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/suppression"
)

// pruneVerifySources is a two-verdict fixture:
//   - svc.go: `errors.New` without family — D006 fires at line 7 (the
//     return). No directive covers it, so the finding is LIVE.
//   - stale.go: a whole-line D003 directive where no logging library is
//     imported — genuinely stale, safe to prune.
const pruneVerifySources = `package main

import "errors"

func boom() error {
	return errors.New("boom")
}
`

const pruneStaleSource = `package other

//cqrs-lint:ignore(D003)
func nothing() {}
`

// TestVerifyPruneRemovedNothingLive_HappyPathStale: a genuinely stale
// directive (rule never fires) verifies clean — pruning it changed nothing
// the pipeline can see.
func TestVerifyPruneRemovedNothingLive_HappyPathStale(t *testing.T) {
	t.Parallel()

	ctx, cleanup := analyzer.BuildContextFromTempFiles(t, map[string]string{
		"svc.go":   pruneVerifySources,
		"stale.go": pruneStaleSource,
	})
	defer cleanup()
	cfg := &AppConfig{}

	entries, err := computeSuppressionAudit(context.Background(), cfg, ctx)
	if err != nil {
		t.Fatalf("computeSuppressionAudit: %v", err)
	}

	var stale *suppression.SuppressionAuditEntry

	for i := range entries {
		if entries[i].Rule == "D003" && entries[i].Status == suppression.AuditStale {
			stale = &entries[i]
		}
	}

	if stale == nil {
		t.Fatalf("fixture must yield a stale D003 entry, got %+v", entries)
	}

	if err := verifyPruneRemovedNothingLive(context.Background(), cfg, ctx,
		[]suppression.SuppressionAuditEntry{*stale}); err != nil {
		t.Fatalf("stale directive must verify clean, got: %v", err)
	}
}

// TestVerifyPruneRemovedNothingLive_WrongStaleFiresLoudly is the e2e guard
// the #126 fix owes (M37/F231): a stale verdict that was WRONG — the removed
// directive's rule still fires at its position — must fail LOUDLY, naming the
// directive and demanding a restore, never silently leave a live finding
// unsuppressed. The wrong verdict is injected as a fabricated removed-entry
// over the live D006 finding's position (any future matcher drift between the
// stale detector and the suppression filter would produce exactly this shape).
func TestVerifyPruneRemovedNothingLive_WrongStaleFiresLoudly(t *testing.T) {
	t.Parallel()

	ctx, cleanup := analyzer.BuildContextFromTempFiles(t, map[string]string{
		"svc.go": pruneVerifySources,
	})
	defer cleanup()
	cfg := &AppConfig{}

	var svcPath string

	for _, gf := range ctx.GoFiles {
		if strings.HasSuffix(gf.Path, "svc.go") {
			svcPath = gf.Path
		}
	}

	if svcPath == "" {
		t.Fatal("fixture must contain svc.go")
	}

	// D006 fires on the `return errors.New("boom")` line (line 7).
	wrongStale := suppression.SuppressionAuditEntry{
		File:   svcPath,
		Line:   7,
		Rule:   "D006",
		Status: suppression.AuditStale,
	}

	err := verifyPruneRemovedNothingLive(context.Background(), cfg, ctx,
		[]suppression.SuppressionAuditEntry{wrongStale})
	if err == nil {
		t.Fatal("wrong stale verdict must fail loudly, got nil")
	}

	for _, want := range []string{"RESTORE", "D006", "svc.go"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("loud failure must mention %q, got: %v", want, err)
		}
	}
}

// TestBuildSuppressionAuditJSON_RealPruneVerifies pins the wiring the JSON
// path owed: a real (non-dry-run) prune through buildSuppressionAuditJSON
// must run the same post-prune verification as the text path. The happy path
// (only genuinely stale entries) proves the guard is wired WITHOUT failing;
// the wrong-stale firing case is covered by the verify test above.
func TestBuildSuppressionAuditJSON_RealPruneVerifies(t *testing.T) {
	t.Parallel()

	ctx, cleanup := analyzer.BuildContextFromTempFiles(t, map[string]string{
		"stale.go": pruneStaleSource,
	})
	defer cleanup()
	cfg := &AppConfig{}

	audit, err := buildSuppressionAuditJSON(context.Background(), cfg, ctx,
		true /* fix */, false /* dryRun */)
	if err != nil {
		t.Fatalf("real prune over genuinely stale directive: %v", err)
	}

	if audit.Fix == nil || len(audit.Fix.Removed) != 1 {
		t.Fatalf("expected exactly one removed stale directive, got %+v", audit.Fix)
	}

	if audit.Stale != 0 && audit.Fix == nil {
		t.Fatal("invariant: a fix result implies stale entries were processed")
	}
}
