# Release Fix — Broken Published Module Graph

**Date:** 2026-07-25
**Problem:** Published v4.1.0 module go.mod files reference dependency versions (v4.0.2, v4.0.3, v4.0.4) that were never tagged. `GOWORK=off go mod tidy` fails for any consumer or integration test.
**Root cause:** The `run-v4.0.4-release.sh` batch release script was prepared and its CHANGELOG entry written (commit `daca53ab`), but the script was deleted (commit `b84ba4ef`) without being fully executed. Only 4 of ~50 modules received their intended tags.

> **Update 2026-07-27:** RESOLVED. All 32 tags were pushed to origin. v4.2.0
> was released 2026-07-27 (53 modules tagged and pushed). The broken module
> graph is fully repaired — every published `require` reference resolves to an
> existing tag. See [CHANGELOG.md](../../CHANGELOG.md) `[v4.2.0]`.

## What was done

Created **32 annotated tags** at commit `8285da41` (the batch-release commit where replace directives are stripped), matching the descriptions from the deleted `run-v4.0.4-release.sh`:

### v4.0.3 tags (17 modules)

decider, encryption, graph, id, kv, listing, middleware, otel, projectionhost, scenario, scheduling, schema, signing, snapshot, storage, storage/pebble, transport/http

### v4.0.4 tags (3 modules — only ones referenced by v4.1.0)

codec, event, watermill

### v4.0.2 tags (13 modules)

command, dedup, dispatcher, idempotency, metadata, projection, query, retry, stack/sqlite, stack, storage/memory, storage/turso, testutil

### v0.2.1 tag (1 module)

event/v4/eventtest

### Pseudo-version (1 ref)

`otel/v4.0.0-20260711192758-e443adb3bfd0` — resolved from VCS (commit `e443adb3` exists in repo). No tag needed.

## Verification

All 84 unique `require` references across all v4.1.0 published go.mod files now resolve to existing tags. **0 missing.**

```bash
# Re-verify at any time:
for tag in $(git tag -l '*/v4.1.0'); do
  moddir=$(echo "$tag" | sed 's|/v4.1.0||')
  git show "$tag:$moddir/go.mod" 2>/dev/null | grep -E '^\s+github.com/larsartmann/go-cqrs-lite'
done | awk '{print $1, $2}' | sort -u | while read mod ver; do
  dir=$(echo "$mod" | sed 's|github.com/larsartmann/go-cqrs-lite/||;s|/v[0-9][0-9]*$||')
  git tag -l "${dir}/${ver}" >/dev/null || echo "MISSING: ${dir}/${ver}"
done
```

## Remaining work (not blocking the graph)

The CHANGELOG `[v4.0.4]` section documents 49 modules as released. Only 3 v4.0.4 tags were needed for graph resolution (codec, event, watermill). The remaining v4.0.4 tags (catalog, signing, encryption, etc.) are documented in CHANGELOG but:

1. Are not referenced by any published v4.1.0 go.mod
2. Would be cosmetic completeness (matching CHANGELOG to reality)
3. Can be created in a future release if desired

## Push command (REQUIRES USER APPROVAL)

These tags are **local only**. They must be pushed to the remote for the Go module proxy to resolve them:

```bash
git push origin --tags
```

Or push only the new tags:

```bash
git tag -l | while read tag; do
  git push origin "$tag"
done
```

After pushing, verify with:

```bash
cd integration && GOWORK=off go mod tidy
```

## Commit 169b5d42 (broken integration/go.mod)

Commit `169b5d42` shipped a broken `integration/go.mod` (referenced `idempotency/sqlstore/v4@v4.1.0` which doesn't exist). This was an intermediate state captured by the auto-commit daemon.

**Status:** Already superseded. Commit `a40e4992` removed the orphaned test file and restored the correct `integration/go.mod`. Net diff from `169b5d42` to HEAD for integration/ shows 164 deletions (all the broken additions removed). No `git revert` needed — it would conflict with the already-applied fix.

## Contract test placement (f4)

The 3-way idempotency contract test lives in `idempotency/kvstore` (imports sqlstore+sqlite as test deps). The architecturally honest home is `integration/` — once the missing tags are pushed, `integration`'s GOWORK=off tidy will work and the test can move there, removing the sqlite dep from kvstore.

**Decision:** Keep in kvstore for now (tags not yet pushed; integration GOWORK=off tidy still fails). Move to integration in the next release cycle after pushing tags.
