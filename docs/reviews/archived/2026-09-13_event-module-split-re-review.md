> **RESOLVED-BY-ROUTING — docs-health 9th pass (2026-09-20):** DECISION RECORD — stands (re-checked by this pass, 2026-09-20: no ports-without-core consumer population has emerged; the falsifiers below remain unmet). Archived as a completed decision record; the live statement lives in ROADMAP §Non-Goals.

# Re-review: Do NOT split the `event/` module

> **Date:** 2026-09-13
> **Kind:** Point-in-time decision re-review (no code changes)
> **Repo state:** `master` @ 0711ef9e7; every number re-measured this session
> **Decision under review:** "Do NOT split event/" — originally 2026-06-29, recorded in
> [`docs/v4-WISHLIST.md`](../../v4-WISHLIST.md) decision log, the archived v4 prep plan
> (2026-07-10), the archived post-v4 plan (2026-07-12), and [`ROADMAP.md`](../../../ROADMAP.md) §Non-Goals.
> **Method:** go-modularize skill — direction-neutrality tests (decomposition depth,
> composability payoff), co-change analysis, importer-population analysis.
> **Companion reviews:** [command-side depth](2026-09-13_command-side-depth-review.md) ·
> [event/command duplication hypothesis](2026-09-13_event-command-duplication-hypothesis-review.md)

---

## Verdict

**The decision holds — and the evidence is now stronger than the original rationale.**

The original rationale ("27 importers, cohesion is real") was thin: importer count alone
never justifies a boundary, and a large module with real cohesion can still be wrong-depth.
This re-review replaces it with a structural argument.

## Fresh measurements (2026-09-13)

| Measurement                              | 2026-06-29 (decision) | 2026-09-13 (re-review) |
| ---------------------------------------- | --------------------- | ---------------------- |
| Modules directly requiring `event/v4`    | 27                    | **41**                 |
| Dirs importing the package (incl. tests) | —                     | **66**                 |
| Exported symbols in `event/`             | —                     | 232                    |
| Non-test LOC                             | —                     | ~3,700                 |

Cohesion by import volume went **up**, not down, since the decision.

## The structural argument (what "cohesion" precisely means here)

### 1. Zero composability payoff on every candidate seam

The module contains four concern clusters (core event type, persistence ports,
bus ports, time/tombstone values). Splitting requires a consumer population that
imports one side without the other. Measured (production, non-test):

- `event.Event` used by **40 of 41** importing dirs
- Ports: `event.Store` 15, `event.Journal` 12, `event.Checkpoint` 11, `event.Bus`/`Publisher` 10–11
- **Importers using ports (Store/Journal/Bus/Checkpoint/Streaming) WITHOUT `event.Event`: zero.**

The ports are typed over the core type (`EventSink.Save(ctx, ref, []Event, expectedVersion)`),
so a hypothetical `eventports/` module would be imported by nobody without `eventcore/`
too. Per the direction-neutrality framework this is the "seam earns nothing" failure mode
— every consumer imports both sides together.

The remaining clusters are even weaker candidates:

| Cluster                                  | External users                   | Split outcome                            |
| ---------------------------------------- | -------------------------------- | ---------------------------------------- |
| Time types (`Date`/`Instant`/`WallTime`) | 2 files, both in `cmd/cqrs-lint` | 2-importer module; worse than status quo |
| `TombstoneMark`                          | 1 file (`watermill/protocol.go`) | 1-importer module; no payoff             |

### 2. Co-change does not indicate a missing boundary

Since 2026-06-29: 39 commits touch core files (`event.go`, `event_new.go`, `types.go`,
`builder.go`, `options.go`, `reconstruct.go`), 11 touch port files (`store.go`, `bus.go`,
`checkpoint.go`, journal/streaming), only **4 touch both**. Core and ports rarely
co-change — which validates the current **file-level** separation, not a missing
**module-level** boundary. Co-change is evidence to examine, never a verdict; here it
examines clean.

### 3. The house pattern for slimming event/ is extract-DOWNWARD, not split

When event/ carries weight that belongs lower, the repo moves it to a Tier-0 module and
aliases it in place:

- `record/` (ADR-0111) — `event.Type = record.Type` alias
- `id/` — `StreamRef`/`StreamType` aliases
- `metadata.Metadata[K]` — generic alias
- `go-codec` (ADR-0128) — codec extracted to a sibling repo, re-imported

Import path never breaks; 41 importers never migrate. This is the sanctioned mechanism,
and it remains available.

## Cost side (burden of proof sits on the splitter)

1. **Breaking:** a v4.x split changes import paths for 41 consumer modules (and every
   external consumer). This is a v5-class change by the repo's own API-stability rules.
2. **The non-breaking variant decouples nothing:** alias shells (à la storage/ v3.5.0)
   keep both import paths alive forever — a permanent module + release-wave cost
   (85-module workspace, tag-wave mechanics) with zero dependency isolation.
3. **Mechanical overhead:** api-stability golden, doc-check, depguard budgets,
   `testModules` in flake.nix, module-map docs — every new module pays all of them.

## What would change this verdict (falsifiers)

- A consumer population emerges that needs ports (or bus) without the event core —
  currently nonexistent.
- The core type stabilizes while ports churn independently at release-drift scale
  (versioned independently would then serve consumers).
- v5 lands and a ports extraction can ride the same breaking wave at near-zero
  marginal cost — re-evaluate then, evidence still required.

## Changelog

- 2026-09-13: ROADMAP §Non-Goals entry rewritten with this evidence (was stale
  "27 importers").
