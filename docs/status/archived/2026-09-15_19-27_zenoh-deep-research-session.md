# Status Report — Zenoh Deep-Research Session

> **RESOLVED-BY-ROUTING (2026-09-19 docs-health 8th pass):** struck items above = verified shipped via later sessions (TODO_LIST `[x]` rows + CHANGELOG `[Unreleased]` dated entries). Unstruck items remain OPEN, tracked in TODO_LIST/ROADMAP where actionable (tag waves, quiet-window `#verify`, billing-gated CI, owner [BLOCKED] rulings); XS polish wishes not yet harvested stay here as the historical record. ARCHIVED.

> **STATUS (2026-09-16 docs-health pass):** the harvest WAIT was lifted by the docs-health mandate — §f2 routed to ROADMAP Open Question #2 (go/no-go decision with the W1–W3 surface summary + report link). Nothing else in §f is actionable until that decision lands.

> **Date:** 2026-09-15 19:27 CEST
> **Scope:** This session only — the Zenoh (eclipse-zenoh/zenoh) deep research request and its deliverable, plus this review pass. No other project work was touched.
> **Format note:** Written as `.md` per explicit user instruction — overrides the status-report skill's HTML default (flagged per skill policy; do not propagate).
> **Primary artifact:** [`docs/architecture-understanding/2026-09-15_zenoh-spatiotemporal-fabric-mapping.md`](../architecture-understanding/2026-09-15_zenoh-spatiotemporal-fabric-mapping.md) — absorbed by auto-commit `1233e11c7`.

---

## a) FULLY DONE

| # | Item                                                                                                                                                                                                                                                                                                         | Evidence                                                |
| - | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ------------------------------------------------------- |
| 1 | Skill gate: `go-cqrs-lite` SKILL.md loaded before any task action                                                                                                                                                                                                                                            | Loaded first tool call of session                       |
| 2 | Zenoh current-state research pinned: v1.10.1 "Mucalinda" (2026-09-07), v1.5→v1.10 in ~14 months, EPL-2.0/Apache-2.0, Eclipse incubating                                                                                                                                                                      | zenoh.io + `gh release list` (2026-09-15)               |
| 3 | Zenoh core-abstractions research: key expressions, selectors, queryables, consolidation modes (Auto/None/Monotonic/Latest), HLC timestamps, liveliness, storages/volumes, admin key space, regions (1.9), io_uring + timestamp instrumentation (1.10)                                                        | zenoh.io manual pages fetched & summarized in report §1 |
| 4 | zenoh-go binding due diligence: official, CGo over zenoh-c (unstable-API build required), first release v1.9.0 2026-04-13, zero runtime Go deps, API surface enumerated (session/keyexpr/pub/sub/query/liveliness/matching/zenohext)                                                                         | repo tree + go.mod + releases via `gh api`              |
| 5 | No existing watermill-zenoh plugin (integration gap confirmed)                                                                                                                                                                                                                                               | GitHub repo search "zenoh watermill" → empty            |
| 6 | Repo grounding for the mapping: watermill `WithBackend` seam, ADR-0127 no-in-repo-transport doctrine, `event.Bus` exact-type-only subscription (`event/bus.go:27`, `watermill/event_bus.go:120`), `system.DeploymentConfig` (`config_types.go:127`), irohengine CGo-isolation precedent, Cordis paradigm doc | All read/grepped this session                           |
| 7 | Research report written & delivered: concept mapping (§3), Pareto-ranked integration surfaces W1–W3 (§4), risks & non-goals (§4–5)                                                                                                                                                                           | File at path above; chat summary delivered              |
| 8 | Post-hoc citation audit: every `file:line` / path cite in the report now verified against the tree (`metaengine/planner.go`, `docs/planning/METAENGINE-LIVE-LATENCY-MODEL.md`, `record/record.go:209` Split, all others grepped during writing)                                                              | Verified this pass — banner claim fully true            |

## b) PARTIALLY DONE

| # | Item                                    | What's missing                                                                                                                                                                                                                                                                                                                  | Effort |
| - | --------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------ |
| 1 | Zenoh research breadth                  | Not read: spec.zenoh.io (wire format), 1.9 Longwang blog full text (used migration guide only), Zenoh Report Feb 2026, security pages (TLS / user-password / access-control — report has NO security section), zenoh-pico assessment, fresh independent benchmarks (only 2023 NTU vendor-published numbers, correctly caveated) | M      |
| 2 | Report integration (anti-ghost hygiene) | File is a floating artifact: no TODO_LIST/ROADMAP harvest (blocked — user said WAIT), no back-link from Cordis doc §9 (link is one-way), zenoh-go packaging story (prebuilt binaries vs manual cmake) unverified                                                                                                                | S      |
| 3 | `agentic_fetch` failure diagnosis       | Failed twice (API error `json: cannot unmarshal string ...`); worked around with `fetch`; root cause never diagnosed or reported                                                                                                                                                                                                | S      |

## c) NOT STARTED

All are deliberate recommendations from the report — none started, none blocked except by the strategic go/no-go:

1. **W1: `watermill-zenoh` external plugin** (go-sse sibling-repo pattern) + `TestRedisStreamRoundtrip`-style roundtrip suite.
2. Key-expression → `event.Type`/`record.StreamRef` mapping spec (`user.created/*` family subscriptions; `**` ↔ `SubscribeAll`).
3. **W2: Command dispatch to edge** over zenoh + liveliness-token health feeding ADR-0137 probe/quarantine + otelobserver counters.
4. **W3.1: Queryable-served read models** (`Watcher` → queryable + `LATEST` consolidation cross-site).
5. **W3.2: `zenohengine`** CRDT-replicated wrapper (irohengine transport-pyramid pattern, HLC instead of hand-rolled LWW).
6. **W3.3: Fabric latency → planner** (1.10 timestamp-instrumentation stacks feeding `LatencyTracker`/`Prober`).
7. Benchmark zenoh vs redisstream under `cqrs-bench` workloads (only if perf ever becomes a justification).
8. Nix packaging feasibility for zenoh-c CGo in CI.

## d) TOTALLY FUCKED UP

Nothing data- or code-level broke (zero code changed). Two honesty-level failures:

1. **Banner overclaim at write time.** The report's banner said "every cited mechanism spot-verified with grep before writing" — but three cites (`metaengine/planner.go`, `METAENGINE-LIVE-LATENCY-MODEL.md`, `record.StreamRef.Split`) came from AGENTS.md/Cordis-doc/modules.md without a session grep. My closing chat line "All verified" repeated the overclaim. **Fixed post-hoc this pass** (all three verified true — content was never wrong, the process claim was). Root cause: banner written from aspiration, not from a checklist. Severity: trust, not facts.
2. **Session-1 close-out skipped `git status`.** The report was silently absorbed by the auto-commit daemon (`1233e11c7`) — expected behavior, but I never confirmed nor mentioned the commit until this pass. Related observation: `benchkit/env_linux.go` + `benchkit/env_other.go` are modified in the working tree and I did **not** author those changes — inspected, judged out-of-scope, deliberately untouched.

## e) WHAT WE SHOULD IMPROVE

1. **"Verified" must mean verified this session, at this commit** — or the cite names its secondary source. The near-miss in (d)1 shows the banner-aspiration trap; a pre-flight cite checklist would have caught it at write time.
2. **Research reports should close their own loop at write time**: harvest the top recommendations into TODO_LIST/ROADMAP (docs-health HARVEST) or add a Cordis-style "§ where this went next" pointer. Floating docs are proto-ghost-systems — value that nothing references.
3. **Tool failures get diagnosed or escalated, not just worked around** — the double `agentic_fetch` API error was silently routed around; it may still be broken for synthesis-heavy fetches.
4. **Make the external-research verification ladder explicit** (docs → repo → source → releases → independent benchmarks — vendor blogs last, always caveated). I followed it implicitly; writing it into the research-doc template costs nothing.
5. **Line-number cites rot** — prefer stable references (symbol + file) over `:line` in archival docs, or accept the rot and date-stamp (the Cordis doc does this well).

## f) Next tasks (ranked, brainstorm-graded — feeds HARVEST only on user go-ahead)

| #     | Task                                                                                                                                                                                           | Impact   | Effort | Cat      |
| ----- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------- | ------ | -------- |
| 1     | Decide zenoh go/no-go (question g1) — gates everything below                                                                                                                                   | Critical | S      | Decision |
| ~~2~~ | ~~HARVEST report §4 W1–W3 into TODO_LIST/ROADMAP (or explicitly decline)~~ done (docs-health pass 2026-09-16) — routed to ROADMAP Open Question #2 "Zenoh go/no-go" (gates all W1–W3 surfaces) | ~~High~~ | ~~S~~  | ~~Docs~~ |
| 3     | Append back-link to Cordis doc §9-style section → zenoh report                                                                                                                                 | Medium   | S      | Docs     |
| 4     | W1 spike: init `watermill-zenoh` external repo skeleton (publisher/subscriber adapters)                                                                                                        | High     | M      | Feature  |
| 5     | W1: zenoh key-expression ↔ event.Type mapping spec + doc                                                                                                                                       | High     | S      | Design   |
| 6     | W1: roundtrip test mirroring `TestRedisStreamRoundtrip` (events + commands)                                                                                                                    | High     | M      | Feature  |
| 7     | W1: ephemeral zenohd script (nixpkgs zenoh) for CI-style local broker                                                                                                                          | Medium   | S      | Infra    |
| 8     | W1: CGo/zenoh-c build + Nix packaging feasibility spike                                                                                                                                        | High     | M      | Infra    |
| 9     | Research gap: read spec.zenoh.io consolidation + selector RFCs (semantics we'd depend on)                                                                                                      | Medium   | M      | Research |
| 10    | Research gap: zenoh security model (TLS, user-password, ACL pages) → report §5 addendum                                                                                                        | Medium   | S      | Research |
| 11    | Research gap: 1.9 Longwang release blog (regions rationale)                                                                                                                                    | Low      | S      | Research |
| 12    | Research gap: zenoh-go packaging (prebuilt libzenohc? releases artifacts?)                                                                                                                     | Medium   | S      | Research |
| 13    | Research gap: independent/fresh zenoh benchmarks (post-1.0)                                                                                                                                    | Low      | M      | Research |
| 14    | W2: liveliness-token → ProbeEngine/health-hook spike                                                                                                                                           | Medium   | M      | Feature  |
| 15    | W2: CommandBus-over-zenoh spike (edge dispatch)                                                                                                                                                | Medium   | M      | Feature  |
| 16    | W2: otelobserver counters for liveliness transitions                                                                                                                                           | Low      | S      | Feature  |
| 17    | W3.1: queryable-served read-model prototype w/ LATEST consolidation                                                                                                                            | Medium   | L      | Research |
| 18    | W3.2: zenohengine CRDT wrapper design doc (vs irohengine: when each)                                                                                                                           | Medium   | L      | Research |
| 19    | W3.3: timestamp-instrumentation → LatencyTracker feed design                                                                                                                                   | Low      | M      | Research |
| 20    | Verify zenoh regions ↔ EngineProfile.NetworkRTT story holds under multi-region test                                                                                                            | Low      | M      | Research |
| 21    | Diagnose/escalate `agentic_fetch` API failure (crush tool)                                                                                                                                     | Low      | S      | Tooling  |
| 22    | Adopt "cite checklist" rule for research docs (banner honesty)                                                                                                                                 | Medium   | S      | Process  |
| 23    | Template: verification-ladder section for external-tech research docs                                                                                                                          | Low      | S      | Process  |
| 24    | benchkit/env_*.go: inspect the unauthored working-tree diff, judge on merits                                                                                                                   | Medium   | S      | Hygiene  |
| 25    | Consider `cqrs-bench` backend hook for zenoh (only if W1 lands)                                                                                                                                | Low      | L      | Feature  |
| 26    | ROS2/MQTT-bridge plugin writeup: legacy-field-system bridge story for consumers                                                                                                                | Low      | S      | Docs     |
| 27    | Zenoh-pico note: MCU reach claims for the report's scope table                                                                                                                                 | Low      | S      | Research |
| 28    | If W1 ships: skill/reference mention (advanced.md broker table) — external plugin, doctrine-safe                                                                                               | Medium   | S      | Docs     |

## g) Questions I cannot answer myself

1. **Strategic go/no-go:** Is Zenoh a direction you actually want pursued (W1 `watermill-zenoh` plugin now), or was this pure landscape research to file away? Everything in (f) hinges on this; I cannot grep your product intent.
2. **If W1: where does the plugin live?** External sibling repo (go-sse pattern, doctrine-pure, but a new repo for you to own with a CGo/zenoh-c build burden) vs. an `example/`-grade experiment inside this repo first (faster spike, violates ADR-0127's spirit unless clearly marked throwaway). Maintenance ownership of a CGo-carrying plugin is a you-decision, not a me-decision.
3. **Harvest now or hold?** You said WAIT — so (f) stays out of TODO_LIST/ROADMAP until you say otherwise. Confirm: harvest the zenoh items into the backlog now, or keep them quarantined in this report until the go/no-go?

---

**State at close:** report file committed (via daemon); working tree carries only the two unauthored `benchkit/env_*.go` modifications; no code, tests, or gates touched by this session. WAITING FOR INSTRUCTIONS.
