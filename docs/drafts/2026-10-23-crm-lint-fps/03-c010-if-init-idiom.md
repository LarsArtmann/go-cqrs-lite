# C010 flags the checked-error idiom `if _, err := decode(...); err != nil`

## Symptom

C010 ("Error from decode/unmarshal call is discarded") fires with **high** confidence on the canonical checked-decode idiom. 11 findings on the CRM, every one of this shape (crm @ 34f26a6, `internal/domain/company/state.go:73`):

```go
if _, err := event.DecodePayloadAuto[ArchivedPayload](evt); err != nil {
	return state, err
}
```

The `_` discards the **payload**, not the error — the error is tested by the enclosing `if` one token later and returned. This is the exact pattern the rule's own suggestion recommends.

## Root cause (source)

`cmd/cqrs-lint/pkg/rules/correctness/swallow_helpers.go:85` (@ ec36eb8d2b) matches any `*ast.AssignStmt` that has an `_` LHS and a Decode-family RHS — and never inspects the statement the assignment lives in:

```go
// skip-validate
for _, lhs := range assign.Lhs {
	id, ok := lhs.(*ast.Ident)
	if !ok || id.Name != "_" {
		continue
	}
	for _, rhs := range assign.Rhs {
		call, ok := rhs.(*ast.CallExpr)
		…
		if cat := swallowedCallCategory(callStr); cat != "" {
			// finding emitted — no check of the enclosing statement
```

When the assign is the `Init` of an `*ast.IfStmt` whose `Cond` references a non-blank LHS (`err != nil`), the error is handled by construction. The same blind spot applies to the closure-callback scan in `c010.go:83`.

## Fix

Before emitting, walk up to the enclosing statement (the inspect closure already has the parent chain available via a stack, or re-scan: given the assign position, find an `*ast.IfStmt` whose `Init == assign`):

- if `assign` is an `IfStmt.Init` and the `IfStmt.Cond` mentions any non-blank ident from `assign.Lhs` → handled, skip;
- optionally also accept a following `if err != nil` on the same var in the immediately-next statement (the two-line form).

## To verify

- `if _, err := event.Decode[T](e); err != nil { return err }` → no finding.
- `p, _ := event.Decode[T](e)` → finding (regression guard, still fires).
- Real consumer: crm @ 34f26a6 drops from 11× C010 to 0; all 11 sites are the if-init form.

💘 Generated with Crush
