package architecture_test

import (
	"testing"

	"golang.org/x/tools/go/packages"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/architecture"
)

// --- E001: Layer violation ---

func TestE001_NoCrashOnEmptyContext(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main`,
	})
	findings := runDetector(t, architecture.NewE001Detector(ctx))
	assertRule(t, findings, "E001", 0)
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
	findings := runDetector(t, architecture.NewE001Detector(ctx))
	assertRule(t, findings, "E001", 1)
}

// --- E002: Circular dependency ---

func TestE002_NoCrashOnEmptyContext(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main`,
	})
	findings := runDetector(t, architecture.NewE002Detector(ctx))
	assertRule(t, findings, "E002", 0)
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
	findings := runDetector(t, architecture.NewE002Detector(ctx))
	assertRule(t, findings, "E002", 1)
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
	findings := runDetector(t, architecture.NewE003Detector(ctx))
	assertRule(t, findings, "E003", 1)
}

// --- E004: Event not in catalog ---

func TestE004_NoFindingOnEmptyRegistry(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main`,
	})
	findings := runDetector(t, architecture.NewE004Detector(ctx))
	assertRule(t, findings, "E004", 0)
}

// --- E005: Command without handler ---

func TestE005_NoFindingOnEmptyRegistry(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main`,
	})
	findings := runDetector(t, architecture.NewE005Detector(ctx))
	assertRule(t, findings, "E005", 0)
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
	findings := runDetector(t, architecture.NewE005Detector(ctx))
	assertRule(t, findings, "E005", 1)
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
	findings := runDetector(t, architecture.NewE005Detector(ctx))
	assertRule(t, findings, "E005", 0)
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
	findings := runDetector(t, architecture.NewE005Detector(ctx))
	assertRule(t, findings, "E005", 0)
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
	findings := runDetector(t, architecture.NewE005Detector(ctx))
	assertRule(t, findings, "E005", 0)
}

// --- E006: Event without projection ---

func TestE006_NoFindingOnEmptyRegistry(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main`,
	})
	findings := runDetector(t, architecture.NewE006Detector(ctx))
	assertRule(t, findings, "E006", 0)
}

func TestE006_DetectsEmittedWithoutProjection(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"emit.go": `package main

func emit() {
	_ = event.New("user.created", id, "User", 1, payload)
}
`,
	})
	findings := runDetector(t, architecture.NewE006Detector(ctx))
	assertRule(t, findings, "E006", 1)
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
	findings := runDetector(t, architecture.NewE006Detector(ctx))
	assertRule(t, findings, "E006", 0)
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
	findings := runDetector(t, architecture.NewE007Detector(ctx))
	assertRule(t, findings, "E007", 1)
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
	findings := runDetector(t, architecture.NewE007Detector(ctx))
	assertRule(t, findings, "E007", 0)
}

func TestE007_NoFindingForNonQueryStruct(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"types.go": `package main

type User struct {
	ID string
}
`,
	})
	findings := runDetector(t, architecture.NewE007Detector(ctx))
	assertRule(t, findings, "E007", 0)
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
	findings := runDetector(t, architecture.NewE007Detector(ctx))
	assertRule(t, findings, "E007", 0)
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
	findings := runDetector(t, architecture.NewE007Detector(ctx))
	assertRule(t, findings, "E007", 0)
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
	findings := runDetector(t, architecture.NewE007Detector(ctx))
	assertRule(t, findings, "E007", 1)
}
