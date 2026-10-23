# Status Report — v4→v5 Phase 0 Execution + Gate Closure (delete-at-zero follow-through)

- **Generated:** 2026-10-23 15:20 CEST
- **Session scope:** execute the 10-17 report's owner-independent next steps: the 8-item gate-closure sweep, Phase 0 (plan tasks 1–3: playbook, census artifact, ghost verdicts), plan/mermaid consistency, TODO_LIST harvest. Six days of parallel sessions intervened (BDD, disk-truth, open-issues workstreams) — their state was re-verified, not assumed.
- **Owner questions from the 10-17 report §g remain UNANSWERED** (the 2026-10-18 owner stamps in TODO_LIST answered a different session's question batch). Re-posed at the end of this session, plus one new one.

## a) FULLY DONE

1. **Gate-closure sweep (10-17 report f/1–8) — all items closed:**
   - `check-versions-manifest --check` ✅ (112 trains, local tags fresh)
   - `check-readme-links` ✅ (848 targets, 0 broken) + `check-readme-deprecated` ✅ — the 10-17 dangling links had already been fixed by the 10-17 23:xx session
   - `check-adr-numbering` ✅ (158 ADRs, no duplicates, lockstep)
   - `cmd/cqrs-upgrade` audit ✅ — zero references to the deleted trio (tokens + testdata), full test suite green (`ok ... 3.473s`)
   - `api_surface.txt` 10-17 delete-window diff EYEBALLED: 123 removed exports = exactly kvstore(7) + grpc(19) + grpc/proto(54) + http(43); zero collateral; the 11 additions in the window are the parallel session's `catalog/architecture` exporter (attributable, not mine)
   - `go.work.sum`: stale `idempotency/kvstore/v4 v4.1.0/go.mod` row pruned; root `go build ./...` green; row not re-added
   - `check-md-go` ✅ (1514 blocks, 105 baselined) — re-run green after ALL of today's docs edits
   - Composite `#verify`: **deliberately not attempted** — load1/load5 = 87.6/73.6 vs the gate's <10/<10 (the 10-23 14:35 session burned an hour on this; not repeated). Constituents verified individually across 10-17 + 10-23 + today. Rowed in TODO_LIST as load-gated residue.
2. **Phase 0 task 1 — fork/delete playbook** → `docs/agents/v5-fork-and-delete-playbook.md`: fork procedure F1–F13 with the twin policy (incl. the KNOWN gate fight: TestMirrorLockstep forces registration vs Big-23-unregistered policy — task 25/47 must land first), exclusion list (7 deprecated-surface classes), delete-at-zero procedure with the load-bearing ordering (registrations → goldens → gate-check → `git rm` LAST), shared 16-row gate checklist, and the 12 guardrails.
3. **Phase 0 task 2 — census tooling** → `scripts/census-fleet.sh`: fleet-wide go.mod require scanner (direct vs indirect, replace-lines skipped) → timestamped JSON artifacts in `docs/evidence/`; `--zero-consumers` delete-trigger report (tree modules ∅ census keys, product dirs excluded); `--latest`; `--self-test` with a 3-leg fixture (planted direct/indirect/vendor/replaces + fault-injection leg proving assertions can fail). Three exclusion semantics learned the hard way (see d): vendor/testdata, `.corpus-pins`+`go-cqrs-lite*` dirs, and **library-fork detection by module path**.
4. **Phase 0 task 3 — ghost verdicts re-based** → addendum in `docs/planning/2026-10-09_ghost-verdict-memo.md` with the fork-aware census as evidence: deriver/graph drop to 0 real fleet consumers (mesh forks were their only "consumers"); verdicts per module + the **rule-domain boundary** (drivers/test-infra/presets are NOT delete candidates even at zero — delete-at-zero governs deprecated/ghost surface only). Executable-now kills identified: `metaengine/graphadapter` (zero-zero) + `otel/otlp` (one recipe rewire) — **held for owner ratification** (they carry no formal deprecation notice; the 10-17 precedent class was formally-deprecated modules).
5. **MAJOR CENSUS DISCOVERY:** `eventcatalog-hub/work/src/mesh-billing` and `mesh-orders` are **full forks of this library** (own .git, `module github.com/larsartmann/go-cqrs-lite`) — they were silently inflating EVERY who-uses count (sole "consumers" of deriver, graph, and all engine requires). Also found: `.corpus-pins/go-cqrs-lite@<hash>` snapshots, `.gotmp/tmp.*` scratch modules, `.crm-stage-dedup2`, duplicate CI checkouts (`ci-siblings/`, `eventcatalog-hub/work/src/cqrs-htmx`). census-fleet.sh excludes forks/pins/testdata; naive who-uses does not. Rowed in TODO_LIST (feeds plan task 48 + owner question).
6. **Plan consistency fixed:** mermaid D1 (delete-at-zero) removed from the Phase D subgraph (task 38 is DONE 2026-10-17; sweeps are continuous via 38b — comment node added, D0 bench-port node added, edges rewired); errata rows for tasks 1/2/3 DONE + the deriver-absorb Phase A gap.
7. **TODO_LIST harvest:** Phase 0 shipped row [x]; owner-ratification row (graphadapter+otlp); deriver-absorb gap row; composite-#verify residue row; census-artifacts row; transport delete row ticked [x] with DONE note; ruling blockquote's stale "ZERO-fleet-consumer modules" list annotated (trio deleted).
8. **Staleness fixed on sight (canonical-facts was RED, 2 findings):** AGENTS recipe-gate row "103/103" → 104/104 (a parallel session added a catalog entry); docs/status/README.md live index backfilled 10 missing rows (10-17 evening + 10-23 morning sessions never indexed their reports) + this report's row. Re-verified green.
9. **AGENTS Quick Reference:** Fleet census row added (census-fleet.sh + playbook pointer).

## b) PARTIALLY DONE

1. **Composite `#verify`** — every constituent green individually; the composed proof stays load-gated (see a/1). The 10-23 session's proposed `#verify-serial` fallback tier is still unwritten (their TODO, not mine).
2. **Ghost verdicts** — evidence + verdicts recorded; the two executable kills await owner ratification (question 4 below). No deletes executed this session (deliberate: untagged/non-deprecated modules are outside the 10-17 precedent class).

## c) NOT STARTED

- Phase A (23+1 forks) — still awaiting explicit owner GO (unchanged since the 10-17 report).
- Phase B codemod, Phase C fleet migration, Phase D/E — unchanged.
- The 10-23 14:35 session's own next steps (delivery/commit/push/issue-closing, SSE flake, cap/baseline follow-ups) — different workstream, untouched by this session.

## d) TOTALLY FUCKED UP

1. **Self-test assertion bugs (2 round trips):** fixture arithmetic (forgot vendor-user's own root go.mod: 3≠4) and fault-leg expectation (forgot the poisoned tree still contains appA). Both caught BY the self-test — the harness worked; my expectations were wrong. Fine, but both were preventable by dry-running the fixture by hand first.
2. **Census semantics shipped wrong twice before right:** v1 counted vendor trees and pin snapshots (kvstore showed "1 consumer" — `.corpus-pins` — minutes after I'd verified the trio deleted); v2 still counted library forks (deriver/graph/engine counts inflated by mesh-*). The 10-17 lesson "requires-only census misses couplings" has a mirror: **a requires census that doesn't exclude forks/pins overcounts consumers**. Both directions now exclusion-aware + self-tested.
3. **Three misleading evidence artifacts got daemon-committed** (pre-fork-exclusion JSONs). git-rm'd and replaced by one self-describing artifact (exclusion list embedded in the JSON). The daemon makes intermediate garbage durable — mint artifacts only in final form (fixed: `--zero-consumers` no longer writes to docs/evidence/).
4. **Multiedit mangled the codec TODO row** (my new rows replaced its header instead of inserting before it — old_string boundary too greedy). Caught on view, repaired same minute. The 10-17 lesson "anchored replaces for long blocks" extends: when inserting, match the FULL anchor block plus its successor's first line, and put the insert BETWEEN them.
5. First rg-based coupling scan silently matched nothing (shell quoting ate the patterns) and briefly looked like "all candidates have zero couplings" — cross-checked with plain grep before believing it. Pipe/quoting discipline again.

## e) WHAT WE SHOULD IMPROVE

- **Who-uses cross-check:** before any delete decision, run BOTH `census-fleet.sh --zero-consumers` AND `depgraph who-uses` — the two tools have different exclusion semantics; disagreement is the signal to investigate (today: kvstore/mesh-*).
- **Artifact discipline:** evidence files are commitments; scan-to-temp for report modes (now implemented).
- **The status-index gap keeps recurring** (10-17 evening + 10-23 sessions wrote reports without indexing). The canonical-facts gate catches it, but only when someone runs it. Candidate cheap fix: a pre-commit hook or extending the daemon's heuristic — rowed for owner triage, not built.

## f) NEXT (ordered)

1. Owner answers to the 4 questions below → fold rulings into plan/ADR (same-day discipline).
2. If graphadapter+otlp kills ratified: execute via playbook §3/§4 (engine-capabilities `--update` regen needed for graphadapter; recipes catalog rewire for otel/otlp).
3. If Phase A GO: task 4 (storage core fork part 1) first — but land the task-25 mirror-gate scoping in the same wave (playbook F9 fight).
4. Add deriver-absorb to the Phase A wave (row exists).
5. Composite #verify at the next quiet window.
6. Census re-run after every fleet batch (38b) — tool is ready.

## g) QUESTIONS FOR THE OWNER

1. **(carried, 10-17 §g/1) Pacing:** "slowly phase out" vs "1–2 weeks sprint" — which governs? May deployment-sensitive repos (crm/browser-history class) hold frozen v4 tags past the window?
2. **(carried, 10-17 §g/2) Twin policy:** fork = snapshot, v4 fix-only, Big-23 unregistered, 9 primitives keep sync — confirm or veto?
3. **(carried, sharpened) Fleet automation:** what creates `eventcatalog-hub/work/src/mesh-billing` + `mesh-orders` (full library forks with .git), `.corpus-pins/`, `.gotmp/tmp.*`, `.crm-stage-dedup2`, and the `ci-siblings/` duplicate checkouts? They all look like automation workdirs, they inflate every census, and I need to know what can race the Phase C codemod waves.
4. **(new) Ratify the two executable kills:** `metaengine/graphadapter` (0 fleet, 0 in-repo couplings) and `otel/otlp` (0 fleet, one recipe rewire) — both memo-KILL since 10-09, now at true zero under the fork-aware census. Delete now, or hold for the v5 cut? My recommendation: delete now (standing rule spirit; git history resurrects).
