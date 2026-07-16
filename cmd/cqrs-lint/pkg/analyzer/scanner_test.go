package analyzer

import (
	"testing"
)

func TestScanCallExpr_EventPayloadCapture(t *testing.T) {
	t.Parallel()

	ctx := BuildContextFromSource(t, map[string]string{
		"events.go": `package main

type UserCreated struct {
	Name string
}

type Config struct {
	Timeout int
}

func emit() {
	_ = event.New("user.created", id, "User", 1, UserCreated{Name: "Alice"})
}
`,
	})

	if !ctx.Registry.EventPayloadTypes["UserCreated"] {
		t.Error("expected UserCreated in EventPayloadTypes")
	}

	if ctx.Registry.EventPayloadTypes["Config"] {
		t.Error("Config should not be in EventPayloadTypes — never used as event.New payload")
	}
}

func TestScanCallExpr_EventTypesEmitted(t *testing.T) {
	t.Parallel()

	ctx := BuildContextFromSource(t, map[string]string{
		"emit.go": `package main

func emit() {
	_ = event.New("user.created", id, "User", 1, payload)
	_ = event.NewEvent("user.deleted", id, "User", 2, payload)
}
`,
	})

	emitted, ok := ctx.Registry.EventTypesEmitted["user.created"]
	if !ok {
		t.Fatal("expected user.created in EventTypesEmitted")
	}

	if emitted.File != "emit.go" {
		t.Errorf("expected file emit.go, got %s", emitted.File)
	}

	if emitted.Line == 0 {
		t.Error("expected non-zero line number for user.created emission")
	}

	emitted2, ok := ctx.Registry.EventTypesEmitted["user.deleted"]
	if !ok {
		t.Fatal("expected user.deleted in EventTypesEmitted")
	}

	if emitted2.File != "emit.go" {
		t.Errorf("expected file emit.go, got %s", emitted2.File)
	}
}

func TestScanCallExpr_CatalogEvent(t *testing.T) {
	t.Parallel()

	ctx := BuildContextFromSource(t, map[string]string{
		"catalog.go": `package main

func register() {
	catalog.Event("user.created", UserCreated{})
	catalog.Event("order.placed", OrderPlaced{})
}
`,
	})

	if !ctx.Registry.EventTypesInCatalog["user.created"] {
		t.Error("expected user.created in EventTypesInCatalog")
	}

	if !ctx.Registry.EventTypesInCatalog["order.placed"] {
		t.Error("expected order.placed in EventTypesInCatalog")
	}

	if !ctx.Registry.IsEventInCatalog("user.created") {
		t.Error("IsEventInCatalog should return true for user.created")
	}

	if ctx.Registry.IsEventInCatalog("unknown.event") {
		t.Error("IsEventInCatalog should return false for unknown.event")
	}
}

func TestScanCallExpr_RegisterTyped(t *testing.T) {
	t.Parallel()

	ctx := BuildContextFromSource(t, map[string]string{
		"register.go": `package main

func setup() {
	dispatcher.RegisterTyped(CreateUserHandler{})
	dispatcher.RegisterTyped(DeleteUserHandler{})
}
`,
	})

	if !ctx.Registry.CommandTypesRegistered["CreateUserHandler"] {
		t.Error("expected CreateUserHandler in CommandTypesRegistered")
	}

	if !ctx.Registry.CommandTypesRegistered["DeleteUserHandler"] {
		t.Error("expected DeleteUserHandler in CommandTypesRegistered")
	}
}

func TestScanCallExpr_NewProjectionRegistration(t *testing.T) {
	t.Parallel()

	ctx := BuildContextFromSource(t, map[string]string{
		"proj.go": `package main

func setupProjection() {
	p := projection.NewProjection("users", handler, []event.Type{"user.created", "user.updated"})
	_ = p
}
`,
	})

	if len(ctx.Registry.Projections) != 1 {
		t.Fatalf("expected 1 projection, got %d", len(ctx.Registry.Projections))
	}

	proj := ctx.Registry.Projections[0]
	if proj.Name != "users" {
		t.Errorf("expected projection name 'users', got %q", proj.Name)
	}

	if len(proj.EventTypes) != 2 {
		t.Fatalf("expected 2 event types, got %d", len(proj.EventTypes))
	}

	if proj.EventTypes[0] != "user.created" || proj.EventTypes[1] != "user.updated" {
		t.Errorf("unexpected event types: %v", proj.EventTypes)
	}
}

func TestScanCallExpr_SubscribeProjection(t *testing.T) {
	t.Parallel()

	ctx := BuildContextFromSource(t, map[string]string{
		"sub.go": `package main

func setup() {
	bus.Subscribe("order.placed", handler)
}
`,
	})

	found := false
	for _, proj := range ctx.Registry.Projections {
		for _, et := range proj.EventTypes {
			if et == "order.placed" {
				found = true
			}
		}
	}

	if !found {
		t.Error("expected order.placed in projection event types from Subscribe call")
	}
}

func TestScanGenDecl_CommandDetection(t *testing.T) {
	t.Parallel()

	ctx := BuildContextFromSource(t, map[string]string{
		"commands.go": `package main

import "command"

type CreateUser struct {
	*command.BasicCommand
	Name string
}

type NotACommand struct {
	ID int
}
`,
	})

	found := false
	for _, cmd := range ctx.Registry.Commands {
		if cmd.Name == "CreateUser" {
			found = true
			if !cmd.HasBasicCmd {
				t.Error("CreateUser should have HasBasicCmd=true")
			}
		}
	}

	if !found {
		t.Error("expected CreateUser in Registry.Commands")
	}

	for _, cmd := range ctx.Registry.Commands {
		if cmd.Name == "NotACommand" {
			t.Error("NotACommand should not be in Registry.Commands")
		}
	}
}

func TestFilterEventPayloads_RemovesNonPayloads(t *testing.T) {
	t.Parallel()

	ctx := BuildContextFromSource(t, map[string]string{
		"types.go": `package main

type UserCreated struct {
	Name string
}

type Config struct {
	Timeout int
}

type OrderPlaced struct {
	Total int
}

func emit() {
	_ = event.New("user.created", id, "User", 1, UserCreated{})
}
`,
	})

	filterEventPayloads(ctx)

	names := make(map[string]bool)
	for _, evt := range ctx.Registry.Events {
		names[evt.Name] = true
	}

	if !names["UserCreated"] {
		t.Error("UserCreated should remain in Events after filtering")
	}

	if names["Config"] {
		t.Error("Config should be filtered out — not an event payload")
	}

	if names["OrderPlaced"] {
		t.Error("OrderPlaced should be filtered out — not used in event.New")
	}
}

func TestFilterEventPayloads_EmptyPayloadTypes(t *testing.T) {
	t.Parallel()

	ctx := BuildContextFromSource(t, map[string]string{
		"types.go": `package main

type Foo struct {
	Bar int
}
`,
	})

	filterEventPayloads(ctx)

	if len(ctx.Registry.Events) != 0 {
		t.Errorf(
			"expected 0 events after filtering with no payload types, got %d",
			len(ctx.Registry.Events),
		)
	}
}

func TestSourceLine_ValidFile(t *testing.T) {
	t.Parallel()

	ctx := &AnalysisContext{
		Fset:     nil,
		Registry: NewCQRSRegistry(),
	}

	line := ctx.SourceLine("scanner_test.go", 1)
	if line == "" {
		t.Error("expected non-empty source line for scanner_test.go line 1")
	}
}

func TestSourceLine_Caching(t *testing.T) {
	t.Parallel()

	ctx := &AnalysisContext{
		Registry: NewCQRSRegistry(),
	}

	first := ctx.SourceLine("scanner_test.go", 1)
	second := ctx.SourceLine("scanner_test.go", 1)

	if first != second {
		t.Error("cached SourceLine should return same result")
	}

	cached, ok := ctx.lineCache.Load("scanner_test.go")
	if !ok {
		t.Error("expected file to be cached after SourceLine call")
	}

	if cached == nil {
		t.Error("cached value should not be nil")
	}
}

func TestSourceLine_EdgeCases(t *testing.T) {
	t.Parallel()

	ctx := &AnalysisContext{Registry: NewCQRSRegistry()}

	if got := ctx.SourceLine("", 1); got != "" {
		t.Error("empty filename should return empty string")
	}

	if got := ctx.SourceLine("scanner_test.go", 0); got != "" {
		t.Error("line 0 should return empty string")
	}

	if got := ctx.SourceLine("scanner_test.go", -1); got != "" {
		t.Error("negative line should return empty string")
	}

	if got := ctx.SourceLine("nonexistent.go", 1); got != "" {
		t.Error("nonexistent file should return empty string")
	}

	if got := ctx.SourceLine("scanner_test.go", 99999); got != "" {
		t.Error("out-of-range line should return empty string")
	}
}

func TestRegistry_CommandByName(t *testing.T) {
	t.Parallel()

	ctx := BuildContextFromSource(t, map[string]string{
		"cmd.go": `package main

import "command"

type CreateUser struct {
	*command.BasicCommand
}
`,
	})

	cmd := ctx.Registry.CommandByName("CreateUser")
	if cmd == nil {
		t.Fatal("expected to find CreateUser command")
	}

	if cmd.Name != "CreateUser" {
		t.Errorf("expected name CreateUser, got %s", cmd.Name)
	}

	if ctx.Registry.CommandByName("NonExistent") != nil {
		t.Error("expected nil for non-existent command")
	}
}
