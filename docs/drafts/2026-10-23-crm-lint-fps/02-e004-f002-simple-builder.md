# E004/F002: catalog/v4/simple builder registrations are invisible

## Symptom

A consumer that declares its whole event vocabulary through the `catalog/v4/simple` builder gets "Event type X is emitted but not registered in catalog" (E004) and "no catalog.NewBuilder" (F002) for every event. The CRM registers all 25 events through `simple.Event[P](…)` and gets 21× E004 + 1× F002 on a catalog that `TestCatalogIsComplete` pins as complete — /docs serves the rendered artifacts.

Consumer shape (crm @ 34f26a6, `internal/catalog/catalog.go:77`):

```go
simple.Event[P](b, string(eventType), catalog.Sends,
	catalog.WithSummary(summary),
	catalog.Owners(OwnerID),
	catalog.Labels(map[string]string{"aggregate": aggregate}),
)
```

`simple` is our own submodule (`catalog/simple/builder.go:135` @ ec36eb8d2b):

```go
func Event[T any](
	b *Builder,
	id string,
	direction catalog.Direction,
	opts ...catalog.MessageOption,
) *Builder
```

## Root cause (source)

`cmd/cqrs-lint/pkg/analyzer/scanner_calls.go:136` (@ ec36eb8d2b) recognizes only the package-level shape:

```go
case funcName == "Event" && IsQualifierFor(gf, sel, "go-cqrs-lite/catalog"):
	if len(call.Args) > 0 {
		if eventTypeStr := StringLit(call.Args[0]); eventTypeStr != "" { … }
```

Three gaps for the simple-builder shape:

1. The qualifier check passes (`catalog/v4/simple` contains `go-cqrs-lite/catalog`) but **arg 0 is the `*Builder`**, not the event type — the id sits at arg 1.
2. The id arrives as a **`string(…)` conversion of a constant** (`string(eventType)`); this case has no `unwrapStringConv` — that helper exists and is already used by the `Register`/EventMetadata case at `scanner_calls.go:105`.
3. After unwrapping, the constant is often **package-qualified** (`contact.EventCreated` at the call sites into the register helper); `ExprIdentName` only records bare idents into `pendingCatalogEventTypeRefs`, so cross-package const values never resolve.

## Fix

Extend the `Event`/`Command` cases (or add sibling cases qualified for `catalog/simple`):

- accept `(b *Builder, id, …)` — probe for a first arg whose type is the builder and shift the index, or key on `len(call.Args) >= 2` when qualified by the simple submodule;
- run `unwrapStringConv` on the id argument;
- teach the pending-ref post-pass to resolve `*ast.SelectorExpr` constants through go/types (the value is a compile-time constant, so `types.Const` lookup settles it without guessing).

## To verify

- Fixture registering via `simple.Event[T](b, "x.y.z", catalog.Sends)` — E004/F002 stay silent; rename the id string and they fire.
- Real consumer: crm @ 34f26a6 drops from 21× E004 + F002 to 0.

💘 Generated with Crush
