# Status Report — Watermill Deep Research & Project Skill Session

> **Date:** 2026-09-15 19:45 CEST
> **Scope:** This session only — the Watermill deep-research request (`watermill.io`, NATS page, ThreeDotsLabs/watermill) and the project-local skill it produced, plus this review pass. No other project work touched by me.
> **Format note:** `.md` per explicit user instruction — overrides the status-report skill's HTML default (same as previous report; flagged, not propagated).
> **Primary artifacts:** `.agents/skills/watermill/` (SKILL.md + `references/backends.md` + `references/internals.md`), `watermill/README.md` NATS correction, `AGENTS.md` discovery row — all absorbed by auto-commit daemon (`b6dbc0a21`/`44ce40551`/`2876a316c` era).

---

## a) FULLY DONE

| # | Item | Evidence |
| --- | --- | --- |
| 1 | Skill gates: `go-cqrs-lite` (prior turn) + `skill-creator` loaded before task actions | First tool calls of each turn |
| 2 | Watermill current-state research: core v1.5.3 (2026-08-25, active, 9.9k stars, MIT), our pins (core v1.5.3, redisstream v1.4.5), plugin majors (nats /v2 @ v2.2.0, kafka /v3, amqp /v3, sql /v4) | watermill.io pages + `gh release list` / `gh api`, 2026-09-15 |
| 3 | Core semantics researched: Message/Ack/Nack (idempotent, ctx-cancelled on ack), Router (parallel handlers, auto-ack, CloseTimeout), at-least-once contract, non-atomic multi-publish, middleware catalog, their `components/cqrs` (incl. EventGroupProcessor ordering trick), Forwarder outbox, SQL tx-publish + #311 caveat, per-backend characteristics (GoChannel/NATS/Kafka/AMQP/Redis/SQL) | 8 docs pages fetched & distilled |
| 4 | **Doc lie found & fixed**: `watermill/README.md:99` claimed "no maintained NATS JetStream plugin exists" — false since v2.1.0 (2024-08). Corrected with dated note; `watermill-nats/v2` documented as supported | Edit applied; committed via daemon |
| 5 | Project-local skill written: `.agents/skills/watermill/SKILL.md` (~230 lines: contract, powers, 12-row limits table, backend matrix, their-CQRS-vs-ours rule, canonical wiring) + `references/backends.md` (full matrix + per-backend gotchas + cheat sheet) + `references/internals.md` (module file map, `protocol.go` metadata wire contract, CatchUp mechanics, test infra) | Files written; doc-check-compliant fences |
| 6 | `AGENTS.md` "Where to Find Things" row added → skill discoverable by future sessions | Edit applied after view; committed via daemon |
| 7 | **Gate green**: `cmd/doc-check` over new files + full default superset → `exit=0`, 1154 refs valid, **zero warnings from my files** (4 union-verified alias warnings pre-exist in go-cqrs-lite references) | Ran directly, exit code checked |
| 8 | Post-hoc claim audit (before this report): "MIT" license ✓ (`gh api .license.spdx_id`), `scripts/ephemeral-nats.sh` ✓ + `ephemeral-redis.sh` ✓ (both exist), `watermill.ProcessingModeMiddleware`/middleware wrappers/WithBackend signatures/protocol keys all grepped during writing | This pass — every shipped claim now session-verified |

## b) PARTIALLY DONE

| # | Item | What's missing | Effort |
| --- | --- | --- | --- |
| 1 | Skill coverage breadth | Not covered anywhere in the skill: Delayed Messages, Requeuing After Error, FanIn/FanOut, Metrics page, Troubleshooting page; backends SQLite/Bolt/Firestore/GCP/AWS/HTTP/io marked "not verified here" in the matrix (honest but shallow) | M |
| 2 | Skill validation (skill-creator loop) | 3 eval prompts drafted & offered; with/without-skill runs NOT executed — deferred to user decision | S–M |
| 3 | Upstream plugin freshness | Verified our redisstream pin (v1.4.5) but not upstream watermill-redisstream's latest release; same for kafka/amqp/sql plugin latest | S |
| 4 | Cross-linking | go-cqrs-lite root `SKILL.md` SSE matrix + `references/advanced.md` watermill sections don't yet point at the new skill (one-way discovery via AGENTS.md only) | S |

## c) NOT STARTED

1. **NATS JetStream roundtrip test leg** — `scripts/ephemeral-nats.sh` exists and the README now documents the plugin; a `TestNatsJetStreamRoundtrip` mirroring the Redis leg is the obvious next code change.
2. Skill trigger evals (description optimization loop).
3. CHANGELOG candidate entry for the README correction (docs-only fix; policy question, see g2).
4. Advanced-topics reference file (`references/advanced.md` for the watermill skill).
5. Any verification of watermill benchmark claims (none made in the skill — deliberately; still nothing measured).

## d) TOTALLY FUCKED UP

Nothing broke; gate green; no code touched. Two honesty/process failures:

1. **Repeated the claim-before-verify pattern — second session in a row.** The skill shipped with two claims written from memory and only verified afterwards in this review pass: "Watermill (…, MIT)" and `scripts/ephemeral-nats.sh` existence (inherited from the README I was editing). Both turned out TRUE — but that's luck, not process. This is the exact failure class flagged in the 19:27 report ("banner overclaim"), where the improvement rule ("verify THIS session or name the secondary source") was written down and then not applied to new-synthesis claims. Root cause: verification discipline applied to file:line cites but not to inline factual assertions (license, file existence) during drafting.
2. **Skipped `git status` at session-1 close — again.** Caught this pass; also means the concurrent-work signal arrived late again: the tree now carries `?? benchkit/repeat_test.go` (untracked, NOT authored by me; earlier in the session it was `M benchkit/env_linux.go` + `M env_other.go`). Someone/something else is actively working in `benchkit/` — I inspected, did not touch. Twice now the working tree has contained foreign changes; twice I only noticed at review time instead of at handoff time.

## e) WHAT WE SHOULD IMPROVE

1. **Institutionalize the claims checklist.** Same improvement in 2+ reports ⇒ make it a rule/skill (per section-quality-guide): during research, maintain an explicit list of every factual assertion destined for the artifact (license, version, file existence, API shape); verify ALL of them — not just `file:line` cites — BEFORE `write()`. The audit-then-fix loop works but costs a turn every time.
2. **`git status` at every artifact handoff**, not just at review time — foreign concurrent edits (benchkit, twice) need to be surfaced in the delivery message, not discovered later.
3. **Close the skill loop**: run the offered trigger evals; an unvalidated skill is a hypothesis, not a tool.
4. Skill-creator's eval-viewer flow was skipped entirely (offered as text instead) — acceptable for a first pass, but the next skill iteration should use the real loop.

## f) Next tasks (ranked; feeds HARVEST only on user go-ahead)

| # | Task | Impact | Effort | Cat |
| --- | --- | --- | --- | --- |
| 1 | Run the 3 drafted skill-eval prompts (with/without skill) and iterate on the description | High | M | Quality |
| 2 | NATS JetStream roundtrip test: `watermill-nats/v2` + `scripts/ephemeral-nats.sh`, mirroring `TestRedisStreamRoundtrip` | High | M | Feature |
| 3 | Add watermill-skill `references/advanced.md`: Delayed Messages, Requeuing After Error, FanIn/FanOut, Metrics, Troubleshooting | Medium | M | Docs |
| 4 | Cross-link: go-cqrs-lite `SKILL.md`/`advanced.md` watermill sections → `.agents/skills/watermill/` | Medium | S | Docs |
| 5 | Verify upstream latests for watermill-redisstream/kafka/amqp/sql plugins; record in backends.md | Low | S | Research |
| 6 | Deep-dive remaining backends (SQLite first — aligns with repo's SQLite-first storage story) and fill matrix cells | Medium | M | Research |
| 7 | CHANGELOG decision for the README NATS correction (see g2) | Low | S | Docs |
| 8 | Claims-checklist habit → consider a tiny `docs/agents/` note or AGENTS.md line so future sessions inherit it | Medium | S | Process |
| 9 | Inspect `benchkit/repeat_test.go` (foreign untracked file) once its author surfaces — judge on merits, don't absorb blindly | Low | S | Hygiene |
| 10 | If NATS leg lands: add `nix run .#integration-nats`-style CI job analog to `#integration-redis` | Medium | M | Infra |
| 11 | Consider recipes.md §addition: Forwarder outbox recipe using repo EventPublisher + watermill-sql tx publisher | Medium | M | Docs |
| 12 | Re-check treefmt/flake formatter coverage for `.md` (treefmt.toml absent; config may be inline in flake.nix — unverified) | Low | S | Hygiene |

## g) Questions I cannot answer myself

1. **Run the skill eval loop now?** I drafted 3 test prompts ("route ordered projections through a watermill Router on Kafka", "make events commit atomically with my MySQL writes", "browser live-updates via watermill?"). Execute with/without-skill runs now, or is the doc-check-green draft enough for today?
2. **Does the README NATS correction warrant a CHANGELOG `[Unreleased]` entry** (docs-only fix, no symbol changes — the changelog-symbols gate wouldn't apply), or is the dated in-file correction sufficient?
3. **`benchkit/repeat_test.go` is untracked and not mine** — another session/agent is clearly mid-work there (env files earlier, a new test file now). Keep `benchkit/` strictly off-limits for me, or is there coordination you want (e.g., review it once committed)?

---

**State at close:** all session artifacts committed (daemon); working tree carries only the foreign untracked `benchkit/repeat_test.go`; doc-check green over the full default scan set; no Go code changed by me (no builds/tests required beyond the doc gate). WAITING FOR INSTRUCTIONS.
