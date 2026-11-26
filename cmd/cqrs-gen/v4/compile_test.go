package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const fixtureModule = "module genfixture\n\ngo 1.27\n"

// TestGeneratedCodeCompiles is the generator's defining test: parseable text
// proves nothing, so the output is generated into a real module and handed to
// the compiler together with call sites that exercise every signature.
func TestGeneratedCodeCompiles(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Fatalf("go toolchain not on PATH: %v", err)
	}

	root := repoRoot(t)

	dir := t.TempDir()

	writeFixture(t, dir, "go.mod", fixtureModule)
	writeFixture(t, dir, "task/domain.go", taskDomain)
	writeFixture(t, dir, "task/wire.go", taskWire)
	writeFixture(t, dir, "legacy/domain.go", legacyDomain)
	writeFixture(t, dir, "legacy/wire.go", legacyWire)

	taskDir, legacyDir := filepath.Join(dir, "task"), filepath.Join(dir, "legacy")

	cli := buildTestCLI(t)

	if err := runCLI(t, cli, "--catalog", "--handlers", taskDir); err != nil {
		t.Fatalf("generate composition-root package: %v", err)
	}

	if err := runCLI(t, cli, "--target", "dispatcher", legacyDir); err != nil {
		t.Fatalf("generate dispatcher package: %v", err)
	}

	writeFixture(t, dir, "go.work", workspaceFor(t, root, dir))

	build := exec.Command("go", "build", "./...")
	build.Dir = dir
	build.Env = append(os.Environ(), "GOWORK="+filepath.Join(dir, "go.work"), "CGO_ENABLED=0")

	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("generated code does not compile against the workspace modules:\n%s", out)
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()

	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.work")); err == nil {
			return dir
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not locate the repository root (no go.work above the test directory)")
		}

		dir = parent
	}
}

// workspaceFor renders the repository's go.work with every use path
// absolutized, plus dir itself, so the fixture module builds against the
// sibling modules without fetching anything.
func workspaceFor(t *testing.T, root, dir string) string {
	t.Helper()

	raw, err := os.ReadFile(filepath.Join(root, "go.work"))
	if err != nil {
		t.Fatalf("read go.work: %v", err)
	}

	var b strings.Builder

	inUse := false

	for line := range strings.SplitSeq(string(raw), "\n") {
		trimmed := strings.TrimSpace(line)

		switch {
		case trimmed == "use (":
			inUse = true
		case inUse && trimmed == ")":
			b.WriteString("\t" + dir + "\n")

			inUse = false
		case inUse && trimmed != "" && !strings.HasPrefix(trimmed, "//"):
			if !filepath.IsAbs(trimmed) {
				line = strings.Replace(line, trimmed, filepath.Join(root, trimmed), 1)
			}
		}

		b.WriteString(line + "\n")
	}

	return b.String()
}

const taskDomain = `package task

import (
	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/query/v4"
)

const cmdArchiveTask = command.Type("task.archive")

//cqrs:command task.create
type CreateTaskCmd struct {
	*command.BasicCommand
	Title string
}

//cqrs:command cmdArchiveTask
type ArchiveTaskCmd struct {
	*command.BasicCommand
}

//cqrs:query task.get
type GetTask struct {
	*query.BasicQuery
	ID string
}

//cqrs:event task.created
type TaskCreated struct {
	ID string
}

//cqrs:event invoice.received receives
type InvoiceReceived struct {
	Ref string
}
`

const taskWire = `package task

import (
	"context"

	"github.com/larsartmann/go-cqrs-lite/catalog/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
)

type taskState struct{ exists bool }

func wire(sys *system.System, bus event.Subscriber, streamID id.StreamID) error {
	if err := RegisterCreateTaskCmd(sys, "Task", decideCreate); err != nil {
		return err
	}

	if err := RegisterArchiveTaskCmd[taskState](sys, "Task", func(_ context.Context, cmd ArchiveTaskCmd, _ taskState, _ event.Version) ([]event.Event, error) {
		evt, err := NewTaskCreatedEvent(cmd.StreamID(), "Task", 2, TaskCreated{})
		if err != nil {
			return nil, err
		}

		return []event.Event{evt}, nil
	}); err != nil {
		return err
	}

	if err := RegisterGetTask[*TaskCreated](sys, func(_ context.Context, q GetTask) (*TaskCreated, error) {
		if _, err := NewTaskCreatedEvent(streamID, "Task", 1, TaskCreated{ID: q.ID}); err != nil {
			return nil, err
		}

		return &TaskCreated{ID: q.ID}, nil
	}); err != nil {
		return err
	}

	if err := RegisterInvoiceReceivedHandler(bus, func(context.Context, InvoiceReceived) error {
		return nil
	}); err != nil {
		return err
	}

	var _ []catalog.MessageConfig = CatalogMessages(catalog.WithName("x"))

	return nil
}

func decideCreate(_ context.Context, cmd CreateTaskCmd, _ taskState, version event.Version) ([]event.Event, error) {
	evt, err := NewTaskCreatedEvent(cmd.StreamID(), "Task", version.Increment(), TaskCreated{})
	if err != nil {
		return nil, err
	}

	return []event.Event{evt}, nil
}
`

const legacyDomain = `package legacy

import (
	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/query/v4"
)

//cqrs:command legacy.run
type RunCmd struct {
	*command.BasicCommand
}

//cqrs:query legacy.status
type StatusQuery struct {
	*query.BasicQuery
}
`

const legacyWire = `package legacy

import (
	"context"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/query/v4"
)

func wire(cmds *command.Dispatcher, queries *query.Dispatcher) error {
	if err := RegisterRunCmdHandler(cmds, func(context.Context, RunCmd) error { return nil }); err != nil {
		return err
	}

	return RegisterStatusQueryHandler[string](queries, func(context.Context, StatusQuery) (string, error) {
		return "ok", nil
	})
}
`

// TestDogfoodIsFresh keeps the committed generated code of the examples equal
// to what the generator emits today, so their own tests always exercise the
// real output.
func TestDogfoodIsFresh(t *testing.T) {
	cli := buildTestCLI(t)

	path := filepath.Join("..", "..", "..", "example", "goal-shaped-app")

	if err := runCLI(t, cli, "--catalog", "--check", path); err != nil {
		t.Fatalf("example/goal-shaped-app needs `cqrs-gen --catalog .`: %v", err)
	}
}
