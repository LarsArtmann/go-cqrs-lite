package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
)

// writeConfig writes the operator deployment exactly like a real /etc file:
// the developer's Domain() below never sees it.
func writeConfig(t *testing.T, yaml string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "cqrs.yaml")
	if err := os.WriteFile(path, []byte(yaml), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	return path
}

func sqliteConfig(t *testing.T) string {
	t.Helper()

	return writeConfig(t, `engines:
  primary:
    driver: sqlite
    dsn: `+filepath.Join(t.TempDir(), "goal.db")+`
    pragmas: ["journal_mode=WAL", "busy_timeout=5000"]
instances:
  - role: source-of-truth
    engine: primary
  - role: projections
    engine: primary
`)
}

// boot composes the system from an operator config — the same three lines
// main.go runs, and the ONLY place the engine choice can enter.
func boot(t *testing.T, ctx context.Context, configPath string) *system.System {
	t.Helper()

	deployment, err := system.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}

	sys, err := system.New(ctx, Domain(), deployment)
	if err != nil {
		t.Fatalf("system.New: %v", err)
	}

	t.Cleanup(func() { _ = sys.Close() })

	return sys
}

// TestGoal_SqliteEndToEnd runs the full story on a real SQLite engine and
// pins the Doctor story: the read model survived a completed AND a deleted
// task, and the planner explains both placements.
func TestGoal_SqliteEndToEnd(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	sys := boot(t, ctx, sqliteConfig(t))

	view, err := runStory(ctx, sys)
	if err != nil {
		t.Fatalf("runStory: %v", err)
	}

	if view.Status != StatusDone || view.Title != "Ship The Goal demo" || view.Priority != 2 {
		t.Errorf("view: got %+v, want done/Ship The Goal demo/2", view)
	}

	plan := sys.MetaEngine().ExplainPlan()
	for _, want := range []string{tasksCollection, openCollection} {
		if !strings.Contains(plan, want) {
			t.Errorf("ExplainPlan missing query %q:\n%s", want, plan)
		}
	}

	doctor := sys.MetaEngine().Doctor(ctx)
	if !strings.Contains(doctor, "sqlite") {
		t.Errorf("Doctor should name the engine the operator picked:\n%s", doctor)
	}
}

// TestGoal_DeletedTaskStaysDeleted pins the tombstone story: deletion is a
// domain event, the view disappears from the lookup, and the query reports
// not-found instead of resurrecting state.
func TestGoal_DeletedTaskStaysDeleted(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	sys := boot(t, ctx, sqliteConfig(t))

	if err := sys.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}

	goneID := id.NewStreamID()

	if err := createTask(ctx, sys, goneID, "Break the build", 5); err != nil {
		t.Fatalf("createTask: %v", err)
	}

	if err := dispatch(ctx, sys, cmdDeleteTask, goneID); err != nil {
		t.Fatalf("delete: %v", err)
	}

	if err := awaitGone(ctx, sys, goneID.String()); err != nil {
		t.Fatalf("awaitGone: %v", err)
	}
}

// TestGoal_OperatorSwapsDriverByConfig proves the swap story WITHOUT a
// database: the operator flips one config value and the deployment now
// targets postgres, whose driver the app already carries.
func TestGoal_OperatorSwapsDriverByConfig(t *testing.T) {
	t.Parallel()

	path := sqliteConfig(t)

	t.Setenv("CQRS_ENGINES__PRIMARY__DRIVER", "postgres")
	t.Setenv("CQRS_ENGINES__PRIMARY__DSN", "postgres://localhost/goal")

	deployment, err := system.LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}

	got := deployment.Engines["primary"]
	if got.Driver != "postgres" || got.DSN != "postgres://localhost/goal" {
		t.Fatalf("swapped engine: got %+v, want postgres DSN override", got)
	}

	for _, driver := range []string{"sqlite", "postgres", "memory"} {
		if _, err := metaengine.LookupDriver(driver); err != nil {
			t.Errorf("driver %q not registered: %v", driver, err)
		}
	}
}

// TestGoal_UnknownDriverFailsLoud pins the operator guardrail: a typo'd
// driver name fails composition loudly instead of silently defaulting.
func TestGoal_UnknownDriverFailsLoud(t *testing.T) {
	t.Parallel()

	path := writeConfig(t, `engines:
  primary:
    driver: sqllite
    dsn: goal.db
instances:
  - role: source-of-truth
    engine: primary
`)

	deployment, err := system.LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}

	if _, err := system.New(context.Background(), Domain(), deployment); err == nil {
		t.Fatal("system.New with typo'd driver should fail, got nil")
	}
}
