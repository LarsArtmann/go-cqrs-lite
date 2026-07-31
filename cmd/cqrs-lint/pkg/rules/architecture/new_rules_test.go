package architecture_test

import (
	"testing"

	"golang.org/x/tools/go/packages"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/architecture"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/ruletest"
)

// --- E001: Layer violation ---

func TestE001_NoCrashOnEmptyContext(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main`,
	})
	findings := ruletest.RunDetector(t, architecture.NewE001Detector(ctx))
	ruletest.AssertRule(t, findings, "E001", 0)
}

func TestE001_DetectsLayerViolation(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main`,
	})
	ctx.Packages = []*packages.Package{
		{
			PkgPath: "github.com/larsartmann/go-cqrs-lite/codec",
			Imports: map[string]*packages.Package{
				"github.com/larsartmann/go-cqrs-lite/decider": {
					PkgPath: "github.com/larsartmann/go-cqrs-lite/decider",
				},
			},
		},
	}
	findings := ruletest.RunDetector(t, architecture.NewE001Detector(ctx))
	ruletest.AssertRule(t, findings, "E001", 1)
}

// --- E002: Circular dependency ---

func TestE002_NoCrashOnEmptyContext(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main`,
	})
	findings := ruletest.RunDetector(t, architecture.NewE002Detector(ctx))
	ruletest.AssertRule(t, findings, "E002", 0)
}

// --- E002: Positive test — circular dependency ---

func TestE002_DetectsCircularDependency(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main`,
	})
	ctx.Packages = []*packages.Package{
		{
			PkgPath: "example.com/app/moduleA",
			Imports: map[string]*packages.Package{
				"example.com/app/moduleB": {PkgPath: "example.com/app/moduleB"},
			},
		},
		{
			PkgPath: "example.com/app/moduleB",
			Imports: map[string]*packages.Package{
				"example.com/app/moduleA": {PkgPath: "example.com/app/moduleA"},
			},
		},
	}
	findings := ruletest.RunDetector(t, architecture.NewE002Detector(ctx))
	ruletest.AssertRule(t, findings, "E002", 1)
}

// --- E003: Positive test — missing module boundary ---

func TestE003_DetectsMixedConcerns(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"domain.go": `package main

import "github.com/larsartmann/go-cqrs-lite/event/v4"

type CreateOrder struct {
	*command.BasicCommand
}

type OrderCreated struct {
	Name string
}

type State struct{ Count int }

func fold(s State, evt event.Event) (State, error) {
	return s, nil
}
`,
	})
	findings := ruletest.RunDetector(t, architecture.NewE003Detector(ctx))
	ruletest.AssertRule(t, findings, "E003", 1)
}

// --- E004: Event not in catalog ---

func TestE004_NoFindingOnEmptyRegistry(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main`,
	})
	findings := ruletest.RunDetector(t, architecture.NewE004Detector(ctx))
	ruletest.AssertRule(t, findings, "E004", 0)
}

// --- E005: Command without handler ---

func TestE005_NoFindingOnEmptyRegistry(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main`,
	})
	findings := ruletest.RunDetector(t, architecture.NewE005Detector(ctx))
	ruletest.AssertRule(t, findings, "E005", 0)
}

// --- E005: Positive test — command without handler ---

func TestE005_DetectsCommandWithoutHandler(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"cmd.go": `package main

type CreateUser struct {
	*BasicCommand
	Name string
}
`,
	})
	findings := ruletest.RunDetector(t, architecture.NewE005Detector(ctx))
	ruletest.AssertRule(t, findings, "E005", 1)
}

// --- E005: Command registered via closure-based RegisterTyped is NOT flagged

func TestE005_NoFindingWhenRegisteredViaClosure(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"cmd.go": `package main

type CreateUser struct {
	*BasicCommand
	Name string
}

func register() {
	command.RegisterTyped(dispatcher, createUserType, func(ctx context.Context, c *CreateUser) error {
		return nil
	})
}
`,
	})
	findings := ruletest.RunDetector(t, architecture.NewE005Detector(ctx))
	ruletest.AssertRule(t, findings, "E005", 0)
}

// --- E005: Non-CQRS type with Type() method (e.g., pflag.Value) is NOT flagged

func TestE005_NoFindingForPflagValueType(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"flag.go": `package main

type idFlag[T any] struct {
	flagType string
	value    T
}

func (f *idFlag[T]) Type() string { return f.flagType }
func (f *idFlag[T]) String() string { return "" }
func (f *idFlag[T]) Set(s string) error { return nil }
`,
	})
	findings := ruletest.RunDetector(t, architecture.NewE005Detector(ctx))
	ruletest.AssertRule(t, findings, "E005", 0)
}

// E005 must NOT fire when a command is registered via plain dispatcher.Register
// (the string-type-based API) with a command.Type constant and a method value.
// The command struct name is resolved by cross-referencing the const declaration.
// Regression for the browser-history false positives (3 E005 findings on
// ExtractVisitCommand, ClassifyURLCommand, DeleteVisitCommand).

func TestE005_NoFindingWhenRegisteredViaDispatcherRegisterAndTypeConst(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"commands.go": `package main

type ExtractVisitCommand struct {
	*command.BasicCommand
	UserID string
}

type DeleteVisitCommand struct {
	*command.BasicCommand
	VisitID string
}
`,
		"consts.go": `package main

import "github.com/larsartmann/go-cqrs-lite/command/v4"

const CommandExtractHistory command.Type = "ExtractVisitCommand"
const CommandDeleteVisit command.Type = "DeleteVisitCommand"
`,
		"register.go": `package main

func register(cmdDisp *dispatcher) {
	cmdDisp.Register(aggregate.CommandExtractHistory, extractHandler.Handle)
	cmdDisp.Register(aggregate.CommandDeleteVisit, deleteHandler.Handle)
}
`,
	})
	findings := ruletest.RunDetector(t, architecture.NewE005Detector(ctx))
	ruletest.AssertRule(t, findings, "E005", 0)
}

// TestE005_NoFindingWhenHandlerUsesRequireCommandType verifies E005 is
// suppressed when a handler body contains a generic type assertion
// requireCommandType[*MyCommand](cmd). This is the browser-history pattern:
// registration uses dispatcher.Register(typeConst, handler) with an
// event-style const value ("browser_history.extract_history"), so the
// handler→struct link is only recoverable from the generic type argument
// inside the handler body, not from the const value or registration call.
func TestE005_NoFindingWhenHandlerUsesRequireCommandType(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"commands.go": `package main

type ExtractVisitCommand struct {
	URL string
}
`,
		"handlers.go": `package main

func handleVisit(cmd any) error {
	_, err := requireCommandType[*ExtractVisitCommand](cmd, "*ExtractVisitCommand")
	return err
}

func requireCommandType[T any](cmd any, expected string) (T, error) {
	var zero T
	return zero, nil
}
`,
	})
	findings := ruletest.RunDetector(t, architecture.NewE005Detector(ctx))
	ruletest.AssertRule(t, findings, "E005", 0)
}

// TestE005_NoFindingWhenClosureUsesPackageQualifiedType verifies E005 is
// suppressed when a RegisterTyped closure takes a package-qualified pointer
// type: func(ctx, cmd *pkg.MyCmd) error. The handlerTypeFromClosure function
// extracts the trailing identifier ("MyCmd") from SelectorExpr params.
// This is the SwettySwipper pattern.
func TestE005_NoFindingWhenClosureUsesPackageQualifiedType(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"commands.go": `package main

type CreateBattleCmd struct {
	ID string
}
`,
		"register.go": `package main

import "context"

func register(d *dispatcher) {
	_ = d.RegisterTyped("create_battle", func(ctx context.Context, cmd *battle.CreateBattleCmd) error {
		return nil
	})
}

type dispatcher struct{}

func (d *dispatcher) RegisterTyped(t string, h interface{}) error { return nil }

type contextType struct{}

type battle struct{}
`,
	})
	findings := ruletest.RunDetector(t, architecture.NewE005Detector(ctx))
	ruletest.AssertRule(t, findings, "E005", 0)
}

// TestE005_NoFindingWhenHandlerUsesMethodValue verifies E005 is suppressed
// when RegisterTyped takes a method value (h.handleX) and the method's
// FuncDecl has a typed parameter (*MyCmd). This is the SEC pattern.
func TestE005_NoFindingWhenHandlerUsesMethodValue(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"commands.go": `package main

type CreateGameCmd struct {
	ID string
}
`,
		"register.go": `package main

import "context"

type GameCommandHandler struct{}

func (h *GameCommandHandler) Register(d *dispatcher) error {
	return d.RegisterTyped("create_game", h.handleCreateGame)
}

func (h *GameCommandHandler) handleCreateGame(ctx context.Context, cmd *CreateGameCmd) error {
	return nil
}

type dispatcher struct{}

func (d *dispatcher) RegisterTyped(t string, h interface{}) error { return nil }
`,
	})
	findings := ruletest.RunDetector(t, architecture.NewE005Detector(ctx))
	ruletest.AssertRule(t, findings, "E005", 0)
}

// TestE005_NoFindingWhenHandlerUsesTypeAssertion verifies E005 is suppressed
// when a handler closure type-asserts the command: cmd.(*MyCmd). This is the
// SwettySwipper RegisterAll pattern — closures take corecmd.Command (interface)
// and type-assert internally.
func TestE005_NoFindingWhenHandlerUsesTypeAssertion(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"commands.go": `package main

type CreateBattleCmd struct {
	ID string
}
`,
		"register.go": `package main

func register(d *dispatcher) {
	d.Register("create_battle", func(cmd interface{}) error {
		c, ok := cmd.(*CreateBattleCmd)
		_ = c
		_ = ok
		return nil
	})
}

type dispatcher struct{}

func (d *dispatcher) Register(t string, h interface{}) {}
`,
	})
	findings := ruletest.RunDetector(t, architecture.NewE005Detector(ctx))
	ruletest.AssertRule(t, findings, "E005", 0)
}

// --- E006: Event without projection ---

func TestE006_NoFindingOnEmptyRegistry(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main`,
	})
	findings := ruletest.RunDetector(t, architecture.NewE006Detector(ctx))
	ruletest.AssertRule(t, findings, "E006", 0)
}

func TestE006_DetectsEmittedWithoutProjection(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"emit.go": `package main

func emit() {
	_ = event.New("user.created", id, "User", 1, payload)
}
`,
	})
	findings := ruletest.RunDetector(t, architecture.NewE006Detector(ctx))
	ruletest.AssertRule(t, findings, "E006", 1)
}

// E006 must NOT flag a SQL row struct whose name coincidentally matches an event
// naming pattern (e.g. "Candidate") but is never emitted via event.New(). The
// registry only tracks types actually emitted, so pure data structs are safe.
// Regression for the DiscordSync false positive on GCSMigrationCandidate.
func TestE006_NoFindingForSQLRowStructNamedCandidate(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"db.go": `package main

// GCSMigrationCandidate is a SQL row struct, NOT an emitted event type.
type GCSMigrationCandidate struct {
	ID     string
	Status string
}

func GetCandidates() []GCSMigrationCandidate {
	return nil
}
`,
	})
	findings := ruletest.RunDetector(t, architecture.NewE006Detector(ctx))
	ruletest.AssertRule(t, findings, "E006", 0)
}

// --- E007: Query without handler ---

func TestE007_DetectsUnregisteredQuery(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"queries.go": `package main

type GetUserQuery struct {
	ID string
}
`,
	})
	findings := ruletest.RunDetector(t, architecture.NewE007Detector(ctx))
	ruletest.AssertRule(t, findings, "E007", 1)
}

// E007 must NOT fire on "*Request" types. These are HTTP/gRPC request DTOs,
// not CQRS queries. The heuristic only matches "*Query" suffix.

func TestE007_NoFindingForRequestTypes(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"api.go": `package main

type LoginRequest struct {
	Email string
}

type RegisterRequest struct {
	Email string
}
`,
	})
	findings := ruletest.RunDetector(t, architecture.NewE007Detector(ctx))
	ruletest.AssertRule(t, findings, "E007", 0)
}

func TestE007_NoFindingForNonQueryStruct(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"types.go": `package main

type User struct {
	ID string
}
`,
	})
	findings := ruletest.RunDetector(t, architecture.NewE007Detector(ctx))
	ruletest.AssertRule(t, findings, "E007", 0)
}

// --- E007: Query registered via closure-based RegisterTyped is NOT flagged

func TestE007_NoFindingWhenRegisteredViaClosure(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"queries.go": `package main

type GetUserQuery struct {
	ID string
}

func register() {
	query.RegisterTyped(dispatcher, getUserType, func(ctx context.Context, q *GetUserQuery) (*User, error) {
		return nil, nil
	})
}
`,
	})
	findings := ruletest.RunDetector(t, architecture.NewE007Detector(ctx))
	ruletest.AssertRule(t, findings, "E007", 0)
}

// E007 must NOT fire when a query is registered via RegisterTyped with a
// type constant (SelectorExpr) and a method value (SelectorExpr). The handler
// type cannot be extracted from the call args directly, but the type constant
// resolves to the query struct name via a const declaration cross-reference.
// Regression for the browser-history false positives (6 E007 findings).

func TestE007_NoFindingWhenRegisteredViaTypeConstAndMethodValue(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"queries.go": `package main

type GetVisitQuery struct {
	ID string
}

type ListVisitsQuery struct {
	Limit int
}
`,
		"consts.go": `package main

import "github.com/larsartmann/go-cqrs-lite/query/v4"

const GetVisitQueryType query.Type = "GetVisitQuery"
const ListVisitsQueryType query.Type = "ListVisitsQuery"
`,
		"register.go": `package main

func register(disp *queryDispatcher) {
	query.RegisterTyped(disp, projection.GetVisitQueryType, projection.NewGetVisitHandler(rm).Handle)
	query.RegisterTyped(disp, projection.ListVisitsQueryType, projection.NewListVisitsHandler(rm).Handle)
}
`,
	})
	findings := ruletest.RunDetector(t, architecture.NewE007Detector(ctx))
	ruletest.AssertRule(t, findings, "E007", 0)
}

// E007 must fire when a type constant is NOT registered: the const declaration
// exists but no Register/RegisterTyped call references it. This guards against
// the suppression becoming too broad.

func TestE007_FiresWhenTypeConstExistsButIsNeverRegistered(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"queries.go": `package main

type GetUserQuery struct {
	ID string
}
`,
		"consts.go": `package main

import "github.com/larsartmann/go-cqrs-lite/query/v4"

const GetUserQueryType query.Type = "GetUserQuery"
`,
	})
	findings := ruletest.RunDetector(t, architecture.NewE007Detector(ctx))
	ruletest.AssertRule(t, findings, "E007", 1)
}

// TestE007_NoFindingWhenHandlerUsesRequireQueryType verifies E007 is suppressed
// when a handler body contains a generic type assertion
// requireQueryType[*MyQuery](q). Mirrors the browser-history pattern where
// query registration uses query.RegisterTyped with a type constant whose value
// is an event-style string, and the handler→struct link is only visible in the
// handler body via the generic type argument.
func TestE007_NoFindingWhenHandlerUsesRequireQueryType(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"queries.go": `package main

type GetVisitQuery struct {
	VisitID string
}
`,
		"handlers.go": `package main

func handleVisit(q any) error {
	_, err := requireQueryType[*GetVisitQuery](q, "*GetVisitQuery")
	return err
}

func requireQueryType[T any](q any, expected string) (T, error) {
	var zero T
	return zero, nil
}
`,
	})
	findings := ruletest.RunDetector(t, architecture.NewE007Detector(ctx))
	ruletest.AssertRule(t, findings, "E007", 0)
}
