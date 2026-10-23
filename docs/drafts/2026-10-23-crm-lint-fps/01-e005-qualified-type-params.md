# E005 still misses cross-package selectors in system.RegisterCommand (follow-up to #64)

## Symptom

Every command on a consumer that registers through `system.RegisterCommand` with a **cross-package selector type parameter** still gets "Command type %q has no registered handler" (E005). #64 fixed the same-package shape (`system.RegisterCommand[CreateTaskCmd, …]`); the CRM registers all 25 commands with qualified params and gets 25 findings on a fully-registered app.

Consumer shape (crm @ 34f26a6, `internal/app/domain.go:196`):

```go
// skip-validate
mustRegister(system.RegisterCommand[deal.LoseCmd, deal.State](sys, deal.CmdLose,
	func(ctx context.Context, cmd deal.LoseCmd) system.Op[deal.State] { … }))
```

Any app whose composition root lives in a different package than its domain (`internal/app` + `internal/domain/deal`) uses exactly this shape — that is the ADR-0123 layout.

## Root cause (source)

Both extraction paths in the registration scanner only accept bare `*ast.Ident`:

`cmd/cqrs-lint/pkg/analyzer/scanner_generic.go:53` (go-cqrs-lite @ ec36eb8d2b):

```go
func typeNameFromGenericArg(expr ast.Expr) string {
	if star, ok := expr.(*ast.StarExpr); ok {
		expr = star.X
	}
	if id, ok := expr.(*ast.Ident); ok {   // deal.LoseCmd is a SelectorExpr → ""
		return id.Name
	}
	return ""
}
```

`cmd/cqrs-lint/pkg/analyzer/scanner_calls.go:196` (`handlerTypeFromClosure`) — the fallback path. Its own doc says it handles "package-qualified types (*pkg.MyCommand → "MyCommand")", but the switch has cases only for `*ast.StarExpr` and `*ast.Ident`; the closure param `cmd deal.LoseCmd` (a `*ast.SelectorExpr`, non-pointer) falls through and returns `""`.

So `commandTypeFromSystemRegisterCommand` (`scanner_calls.go:220`) returns "" on both legs and the registration is never recorded in `CommandTypesRegistered`.

## Fix

- `typeNameFromGenericArg`: add a `*ast.SelectorExpr` case returning `lastIdentSegment(sel)` (or `sel.Sel.Name`).
- `handlerTypeFromClosure`: add a `*ast.SelectorExpr` case (skip `context.Context` — already a SelectorExpr — by qualifier check or by continuing on that specific name).

`lastIdentSegment` already exists and is used for the StarExpr leg, so both are one-liners.

## To verify

- `TestE005_NoFindingWhenRegisteredViaSystem` (#64's test) extended with a cross-package qualifier: registration in package `a` for a command defined in package `b`.
- Real consumer: crm @ 34f26a6 goes from 25 E005 to 0 via `cqrs-lint .` (verified today: all 25 commands register through that exact call shape).

💘 Generated with Crush
