# mesh-demo — two bounded contexts, bilateral contracts, one hub

The multi-domain proof for go-cqrs-lite's data-mesh story: **ordering** and
**billing** are separate bounded contexts (own state, own commands, own
events, own team, own catalog) wired together ONLY through event contracts.

```
orders ─── order.placed ────▶ billing   (billing folds it, issues invoice)
orders ◀── invoice.issued ── billing   (orders folds it, completes order)
```

Integration is REPLICATION, never a synchronous cross-domain call and never
query federation — see
[ADR-0146](../../docs/adr/0146-no-federated-query-engine.md).

## Layout

| File          | What it shows                                                                                  |
| ------------- | ---------------------------------------------------------------------------------------------- |
| `orders.go`   | Orders context: commands, events, state, deciders; consumes `invoice.issued`                   |
| `billing.go`  | Billing context: consumes `order.placed`, issues `invoice.issued` (idempotently)               |
| `flow.go`     | The load→fold→decide→save loop, bare — what `decider.Repository` automates against a store      |
| `catalog.go`  | Per-context catalog declarations: bilateral `Sends`/`Receives`, data products, teams, owners   |
| `mesh_test.go` | Domain tests: folds, rejection, idempotency, the full cross-domain round trip                  |
| `merge_test.go` | Hub dry-run: coeffects dangling-free per source, bilateral copies carry both sides, manifests union cleanly |

## Run it

```bash
go run . demo
# order:  placed=true total=9900 invoice="inv-order-42" completed=true
# billing: order=order-42 amount=9900 invoice="inv-order-42"
```

## Ship each context's export to the federation hub

Each context exports its OWN EventCatalog tree — the hub union-merges them
(`catalog.home.lan`; see [catalog/README.md "Feeding the federation
hub"](../../catalog/README.md)):

```bash
go run . export -domain orders  -out work/out/orders  -skip-bootstrap
go run . export -domain billing -out work/out/billing -skip-bootstrap
```

For governance linting (`@eventcatalog/linter`), export the plain-refs
variant instead — the two ref formats exist because core and linter resolve
different ID shapes (see [catalog/README.md "Governance
exports"](../../catalog/README.md)):

```bash
go run . export -domain orders -out work/lint/orders -plain -skip-bootstrap
```

Each export carries `catalog.index.json` — the machine-readable manifest the
hub diffs for change detection.

### Bilateral contract rule (why both copies are safe to union)

`order.placed` is declared TWICE: orders sends it with an explicit
`Consumers: [billing-svc]`, billing receives it with an explicit
`Producers: [orders-svc]`. Either copy tells the WHOLE relationship, so the
hub's union-merge cannot lose a side — and `ValidateCoeffects()` passes in
EACH source catalog even though each sees only half the mesh (the explicit
external producer is the "imported, not dangling" marker).

### Onboarding to the real hub

Add both export commands to the hub's `sources.json` (one entry per source
repo — here both live in one module, so the two `-domain` invocations are
the two entries' `export` commands) and register directions faithfully; the
`merge_test.go` assertions are the same checks the hub's CI applies on the
union.

> **Note:** the `replace` block in `go.mod` points at the workspace sibling
> catalog while the exporter options demonstrated here await their first
> release tag; it is deleted by the release sweep once the pin is bumped.
