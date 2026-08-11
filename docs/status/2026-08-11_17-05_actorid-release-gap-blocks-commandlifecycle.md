# id.ActorID Release Gap Blocks commandlifecycle Tag

**Date:** 2026-08-11
**Type:** Release-blocking discovery (no tags created — awaiting dedicated release session)
**TODO items:** "Tag commandlifecycle/v4.0.0" and "Tag benchkit/v4.4.0" (latter verified already tagged)
**Update 2026-08-11:** Still OPEN — no tags created. Tracked in TODO_LIST → Release/Tagging (BLOCKED on user approval + go-codec publish).

## Summary

`commandlifecycle/v4.0.0` cannot be tagged yet. The dependency chain it publishes
depends on `id.ActorID`, which was **never released**:

- `record/v4.1.0` (published) uses `id.ActorID` in `record.go` (6 references)
- `command/v4.4.0` (published) uses `id.ActorID` in 2 files
- `metaengine/v4.8.0` (published) uses `id.ActorID` in 5 files
- …but the newest published `id/v4` tag is **v4.2.0**, which does NOT contain
  `id/actor_id.go`

Result: any `GOWORK=off` (consumer-mode) build that resolves
`record/v4@v4.1.0` (61 go.mod files require it), `command/v4@v4.4.0`, or
`metaengine/v4@v4.8.0` fails with `undefined: id.ActorID`. Verified by direct
build attempt in `commandlifecycle/` and `commandlifecycle/projections/` under
`GOWORK=off`.

## Evidence

| Check | Result |
| --- | --- |
| `git cat-file -e "id/v4.2.0:id/actor_id.go"` | **MISSING** (fatal: path exists on disk, not in tag) |
| `git ls-tree --name-only id/v4.2.0 -- id/` | only `user_id.go` (no actor files) |
| `git merge-base --is-ancestor 7e374b753 id/v4.2.0` | NOT ancestor (actor commit added after tag) |
| `git show record/v4.1.0:record/record.go` | **uses** `id.ActorID` (lines 35-41, 80-81) |
| `git show record/v4.1.0:record/go.mod` | requires `id/v4 v4.2.0` (lacks ActorID) |
| `git show command/v4.4.0` grep ActorID | 2 files |
| `git show metaengine/v4.8.0` grep ActorID | 5 files |
| ActorID commit | `7e374b753` "feat(record): adopt branded ID types and ActorID taxonomy in CommonMetadata" |

## Affected published modules (broken for consumers)

| Module | Published tag | Status |
| --- | --- | --- |
| `id/v4` | v4.2.0 | Missing `ActorID` (never re-tagged) |
| `record/v4` | v4.1.0 | Uses ActorID, requires id/v4 v4.2.0 → broken |
| `command/v4` | v4.4.0 | Uses ActorID, requires id/v4 v4.2.0 → broken |
| `metaengine/v4` | v4.8.0 | Uses ActorID, requires id/v4 v4.2.0 → broken |

Consumers requiring `record/v4 v4.1.0` (61 modules), `command/v4 v4.4.0`,
`metaengine/v4 v4.8.0` inherit the breakage via MVS (id pinned at v4.2.0).

## Proposed fix (for dedicated release session)

1. **Tag `id/v4.3.0`** — publishes `actor_id.go`, `actor_id_json.go`, tests
   (currently in workspace since 7e374b753). Additive; api_surface.txt already
   contains the new exports (`id/func NewActorID`, `id/func ParseActorID`,
   `id/struct ActorID`, constructors NewUserActor/NewBotActor/NewSystemActor/
   NewServiceActor, `id/type ActorKind`).
2. **Re-tag `record/v4.2.0`** with go.mod requiring `id/v4 v4.3.0`.
3. **Re-tag `command/v4.5.0`** with go.mod requiring `id/v4 v4.3.0`.
4. **Re-tag `metaengine/v4.9.0`** with go.mod requiring `id/v4 v4.3.0`.
5. **Tag `commandlifecycle/v4.0.0`** and **`commandlifecycle/projections/v4.0.0`**
   (projections go.mod already pins `commandlifecycle/v4 v4.0.0`; tag in
   dependency order).
6. **Bump downstream go.mod requires** for all modules that use ActorID
   (event, query, metadata, storage/bbolt, storage/pebble, watermill, etc. —
   66 modules require id/v4) to avoid MVS pinning the broken id/v4.2.0.

Always use `./scripts/tag-release.sh <module> vX.Y.Z "desc"` (strips local
replaces, runs tidy, creates annotated tag). Update CHANGELOG.md Unreleased
section + regenerate `docs/api_surface.txt` (`cd cmd/api-stability && GOWORK=off
go run . -update`) before tagging. Run `nix run .#verify` first.

## Related

- benchkit/v4.4.0 tag **already exists** (commit 7d5cd10c7) and contains
  `Truncate`/`TitleCase` — TODO item can be closed.
- This mirrors the DuckDB/PG go.mod drift note in CHANGELOG Unreleased
  (require pinned below actual published version).
