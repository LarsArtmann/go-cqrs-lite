# ADR-0162: Metaengine v5 splits into seven packages instead of one 829-symbol package

| Field      | Value                                                                                            |
| ---------- | ------------------------------------------------------------------------------------------------ |
| Status     | Proposed                                                                                         |
| Date       | 2026-10-26                                                                                       |
| Deciders   | Lars Artmann                                                                                     |
| Related    | ADR-0152 (v5 module topology), ADR-0111 (Record), ADR-0126 (wrapper/decorator contracts), ADR-0142 (engine substrate) |
| Supersedes | nothing — this is the first decision about the internal shape of `metaengine`                    |

## Context

`metaengine`'s root package carries the whole cost-based planner, the `Store` facade, the fold and
materialized-view machinery, the typed read side and ADTs, the diagnostics, the capability contracts,
and the in-process memory driver in **one namespace**: 167 production files, 27,716 LOC, and 829
exported symbols in the api-stability golden (13.1% of the library's entire API).

Two facts make this a decision rather than a preference:

1. **No existing gate could see it.** `check-file-size.sh` bounds files (the largest,
   `metaengine/fold.go`, holds 58 symbols — small by any per-file measure), and the 350-line rule is
   satisfied. Package-level concentration was invisible until `scripts/check-package-surface.sh`
   landed on 2026-10-26, which baselines `metaengine` as the largest of eight historical offenders.
2. **The v5 fork copies code.** `core/v5` is built by copy-forward (ADR-0152). If the fork copies the
   `metaengine` package verbatim, the god package becomes permanent: every later split would then be
   a breaking change to a second consumer surface instead of one planned move.

The v4 line is frozen for structural work, so the split can only happen at v5. That makes the
*shape* of the split a decision to record now, while the census is fresh.

## Decision

`metaengine` becomes **seven packages in one `metaengine/v5` module**:

| Package | Files | Exported decls | Production LOC |
| ------- | ----: | -------------: | -------------: |
| `metaengine/engine` | 28 | 248 | 5,247 |
| `metaengine/plan` | 29 | 142 | 4,117 |
| `metaengine/exec` | 20 | 80 | 3,076 |
| `metaengine/collection` | 22 | 132 | 4,176 |
| `metaengine/adt` | 47 | 195 | 7,472 |
| `metaengine/diagnostic` | 13 | 72 | 2,246 |
| `metaengine/memoryengine` | 8 | 66 | 1,382 |

with the dependency rule

```text
engine <- plan <- exec <- collection <- adt
engine <- memoryengine
diagnostic on top, read-only
```

and no compatibility facade: no umbrella `metaengine` package re-exporting the seven.

The full file-by-file move table, symbol ceilings, migration sequence and risk list live in
[`docs/planning/2026-10-26_metaengine-v5-seven-package-split.md`](../planning/2026-10-26_metaengine-v5-seven-package-split.md).

## Consequences

**Positive**

- The largest package drops from 829 to 248 exported declarations (70% reduction), inside the ≤ 300
  ceiling the architecture review proposed. Six of the seven land under both package-surface
  ceilings and therefore need no baseline row at all.
- Shared contract types get a named home (`engine`) instead of being duplicated or parked in a
  "misc" package, which is the failure mode of the naive five-way split.
- `memoryengine` becomes symmetric with the eleven existing driver modules, so "a driver may live
  inside the planner" stops being a special case.
- The dependency rule is greppable (`rg 'go-cqrs-lite/metaengine/v5/'`) and mechanically enforced by
  `scripts/check-cycles.sh` plus the package-surface ratchet.

**Negative / costs**

- 167 files and 180 test files move; eight compile-gated steps, not one commit.
- Every consumer of `metaengine` needs the codemod mapping before the v5 tag ships. The split must
  therefore land **after** (or with) the `cqrs-upgrade --to-v5` mapping table, which today has zero
  `core/v5` awareness.
- Seven packages inside one module still version together (112 release trains already exist); this
  ADR deliberately does not mint seven modules.

**Neutral**

- v4 keeps its current shape and its pins; nothing here removes or deprecates a v4 symbol.

## Alternatives considered

| Alternative | Why rejected |
| ----------- | ------------ |
| Keep one package, rely on the new package-surface ratchet to freeze it | Freezing 829 symbols at v5 means the fork carries the concentration forever; the ratchet would then protect the wrong shape. |
| Five-way split (engine, plan, exec, collection, diagnostic) as the review sketched | Leaves ~100 shared types/errors with no home: either a sixth package appears anyway or they get duplicated. |
| Seven separate modules | Seven more release trains, seven more go.mod files, and no consumer asked for independent versioning; one module preserves the `core/v5` precedent. |
| Umbrella facade for one release | Re-creates the 935-symbol namespace, fails the new gate by construction, and creates a migration path that never ends. |

## Implementation status

NOT IMPLEMENTED. Design and census only, on basis commit `b2592201b`; execution is blocked on the v5
cut decision and on the codemod mapping (plan tasks T22 / 19-19 T13).
