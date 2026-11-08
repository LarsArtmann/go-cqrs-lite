# ADR-0166: One StreamRef — the id struct is canonical, record aliases it

- Status: Proposed (2026-11-08, brutal-episode7 B3) — ratifies at the v5 cut alongside the
  already-owner-confirmed `record.NewStreamRef` validation ruling (2026-08-22 core-data-model
  execution plan, Appendix B)
- Date: 2026-11-08

## Context

Two public types name a stream and they disagree:

| | `record.StreamRef` | `id.StreamRef` |
| --- | --- | --- |
| Shape | `type StreamRef string` | `struct { Type StreamType; ID StreamID }` |
| Display/wire form | `"StreamType/EntityID"` (slash-joined) | `String()` = display `"Type:ID"`, `StreamKey()` = storage `"type:bare-id"` |
| Validation | none (empty type legal on the wire) | `Validate()` rejects empty type + zero ID; `IsZero()` |
| Construction | raw string cast | `NewStreamRef(type, id)` (validation arrives at v5 per the Appendix B ruling) |

The disagreement is invisible while `record.StreamRef` stays a permissive wire string, but the
v5 plan already commissions validation for `record.NewStreamRef` — at which point two
constructors enforce two rule sets over two textual forms of the same identity, and every
consumer bridge (`system` adapters, `projectionhost`, decider's `*Ref` methods) pays the
conversion tax twice.

The v4 twin exists for module-isolation reasons (record is Tier 0 zero-dep; id owns branded
IDs). Precedent: `event.Type`/`command.Type`/`query.Type` are ALIASES of `record.Type`
(ADR-0111 contract 21a) with lockstep tests — the alias pattern is proven in this codebase.

## Decision (proposed for v5)

1. **`id.StreamRef` (the struct) is the ONE canonical stream identity in `core/v5`.**
2. **`record.StreamRef` becomes a type alias of it** (`type StreamRef = id.StreamRef` in the
   v5 record package, exactly the `event.Type = record.Type` pattern) — one constructor, one
   validation rule set, one wire form.
3. **Validity: empty type is ILLEGAL; zero ID is ILLEGAL** — enforced at construction
   (`NewStreamRef` returns an error; the Appendix B ruling). The permissive raw-string cast
   retires at v5; parse-from-string goes through one `ParseStreamRef` with the same rules.
4. **Wire/storage form stays `StreamKey()` (`type:bare-id`)** per the 2026-10-17 disk-truth
   contract — the display form NEVER reaches storage. The slash-joined textual form of v4
   `record.StreamRef` is deprecated with this ADR and removed at the cut.
5. Adapters keep dual-read for journals written with legacy display-form keys (the
   sticky-key fallback contract) until the deployed-DB census reopens (parked TBD 2026-11-03).

## Consequences

- One vocabulary: `Type/ID` struct everywhere; no more slash-vs-colon ambiguity in docs,
  SQL, or logs. The DOMAIN_LANGUAGE "StreamRef" entry collapses to one row.
- `record` (v5 core) depends on `core/v5/id` — already the direction the core/v5 mirror uses
  for branded IDs; no cycle (id depends only on its Tier-0 base).
- Breaking at v5 (intended): string-cast construction, slash display form, and the
  empty-type loophole all die; the migration guide gets a before/after section
  (rides the F091 guide task).
- The lockstep test pattern (cross-type comparison `event.Type("x") != record.Type("x")`)
  extends to StreamRef aliases.

## Alternatives considered

- **Keep both, bridge forever** — two rule sets and two textual forms of the primary key is
  the split brain this ADR exists to kill; rejected.
- **String-typed canonical with parse/validate** — loses the struct's typed fields
  (`Type StreamType`, `ID StreamID`), which the decider/system surfaces already consume;
  rejected.
