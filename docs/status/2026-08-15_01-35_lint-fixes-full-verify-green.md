# Status Report: 15 Lint Fixes → First Fully GREEN Verify Gate Since ADR-0128

- **Date:** 2026-08-15 01:35 (Saturday)
- **Session scope:** Resume of the verify-gate session. Fixed the 15 remaining
  lint issues, re-ran the lint gate, ran the FULL verify gate, closed out the
  3 pending questions by events/standing instruction, updated docs + memory.
- **Branch:** `master`. Tree state at report time: clean — the auto-commit
  daemon committed the lint fixes as `444be10a7` and both status docs.

## a) FULLY DONE (this session)

1. **Recon.** Confirmed the daemon committed the prior ~115-file change set
   (`5127039da` + `875bb689b`), conflict-marker scan clean. Prior question 1
   (commit policy) is resolved by events — no clean single commit was possible
   anymore, and none is needed.
2. **15/15 lint issues fixed** (all mechanical, as diagnosed):
   - `cmd/cqrs-lint` — golines ×1 (`pkg/analyzer/module_catalog_data.go:127`)
     via `golangci-lint --fix`.
   - `idempotency/kvstore` — gci ×4 + gofumpt ×4 (coverage_test.go,
     property_test.go, store.go, store_test.go) via `--fix`.
   - `idempotency/sqlstore` — gci ×3 (property_test.go, store.go,
     store_test.go) via `--fix`.
   - SA1019 `NewMemoryStore is deprecated` ×3 (kvstore property_test.go:34,
     store_test.go:215, store_test.go:262) via widening the scoped
     `.golangci.yml` exclusion path from `middleware/.*_test\.go$` to
     `(middleware|idempotency)/.*_test\.go$` (same SA1019 MemoryStore text;
     rationale documented in the config comment — kvstore test matrices
     intentionally compare MemoryStore against the KV-backed store; the
     deprecation notice itself names tests as the sanctioned consumer).
     This implements the recommendation from the prior session's question 3.
3. **Per-module verification after fixes.** `GOWORK=off` builds green; tests
   green: kvstore 5.2s, sqlstore 51.5s, cqrs-lint pkg tests ok.
4. **`nix run .#lint` — GREEN.** Exit 0, 76 modules linted, every module
   `0 issues.` — verified from the FULL captured log (`grep -v '^0 issues\.$'`
   returned no non-clean lines), not a tail view.
5. **`nix run .#verify` — GREEN.** Exit 0, all 18 phase markers present
   (pre-flight checks, Module Coverage, Build, Vet, Test, Race, Lint,
   Check Arch, Check Depguard, Check Duplication, Check Coverage,
   API Stability, Doc Check), final line
   `✅ All verification checks passed`.
   **First genuinely green full verify gate since the ADR-0128 shim
   deletion.**
6. **Doc Check noise investigated.** Warnings for `../../flightrecorder`,
   `../../retry`, `../../idempotency` ("cannot read / no exports found") are
   the doc-checker probing for sibling checkouts of the externalized repos
   that are absent locally. References in SKILL.md/references are the correct
   external paths. Benign; cleanup item listed in f).
7. **Docs + memory maintained.** Appended the Follow-up close-out section to
   `2026-08-15_00-51_verify-gate-check-arch-root-cause.md` (resolving all 3
   questions); added the exit-capture footgun gotcha to `AGENTS.md`
   (see d5/e1).
8. **Session todos tracked end-to-end** — nothing dropped; the only remaining
   blocked item (commit) was overtaken by the daemon commit.

## b) PARTIALLY DONE

1. **kvstore SA1019 story.** Config-level exclusion done and scoped; the
   alternative (migrate kvstore tests onto the go-idempotency contract
   suite) deliberately NOT done. Exclusion is the pragmatic 5-minute answer;
   revisit decision at v5 (question g3).
2. **Doc-check warning noise.** Diagnosed and proven benign, but the stale
   invocation paths for deleted modules were not yet removed from wherever
   verify-docs passes them (did not research the caller this session).
3. **Release chain.** Verify-green unblocks tagging, but nothing tagged
   (explicit approval required — never tag without instruction).
   Consequence chain still pending: system/go.mod 5 temporary replaces +
   ~49 stale `// indirect` shim refs.

## c) NOT STARTED (carried, untouched this session)

- Tagging engine v4.0.2 ×4 (sqliteengine, badgerengine, pebbleengine,
  pgengine) + watermill/v4.5.0 — blocked on user approval.
- Removing the 5 temporary replaces from system/go.mod + GOWORK=off
  re-verification.
- `go mod tidy` sweep of ~49 stale `// indirect` shim refs.
- TODO_LIST backlog (~33 items), including: Dgraph JournalReadFrom
  off-by-one (S), cqrs-lint per-module regression tests, `.golangci.yml`
  exclusion audit, DuckDB/Row calibration benches, real Redis/NATS broker
  roundtrips, v5 Phase 8 deletions + migration guide, go-codec repo
  scaffolding, gate-hardening items (lint summary line; LAYER/DEP_BUDGET
  meta-test).

## d) TOTALLY FUCKED UP (this session's honest failures)

1. **Broken exit-code capture, three times.** The initial three parallel
   module lint checks used `golangci-lint ... | tail -N; echo "EXIT=$?"` —
   which printed `EXIT=0` while **11 issues were listed directly above it**.
   `$?` was tail's exit code, not golangci's. Every "EXIT=0" from those runs
   was meaningless. No wrong action resulted (the issue text was read
   correctly), but the verification theater was real.
2. **First background gate run captured nothing.** `nix run .#lint 2>&1 |
   tail -30; echo "LINT-EXIT=${PIPESTATUS[0]}"` printed `LINT-EXIT=` (empty)
   — PIPESTATUS did not behave in this shell. Cost: a full extra ~5+ minute
   lint re-run with the correct `cmd > /tmp/log 2>&1; echo $?` pattern.
3. **Almost repeated last session's partial-log failure.** First instinct
   after the background lint run was to judge from `tail -30` (all green
   modules visible). Self-corrected — re-ran with full log capture and
   `grep -v '^0 issues\.$'` — but the anti-pattern instinct fired first.
   It is not extinct; the correct pattern must be pre-loaded, not
   improvised.
4. **Imprecise count stated with confidence.** Final message said "12-file
   lint-fix diff"; the actual diff was 10 files (24 insertions, 14
   deletions) plus 1 new status doc. Sloppy number, stated without checking.
5. **Forgot the proactive-memory mandate mid-session.** Discovered the
   exit-capture footgun live (d1/d2) and did NOT write it to AGENTS.md at
   the moment of discovery — only during this report, after the user asked
   "what did you forget". Classic "I'll remember" anti-pattern; it is now
   recorded.

## e) WHAT WE SHOULD IMPROVE (systemic)

1. **Institutionalize the capture pattern** (now in AGENTS.md): never
   `cmd | tail; echo $?` — always `cmd > /tmp/x.log 2>&1; echo $?` + grep
   the full log. Apply from the FIRST verification, not after observing
   garbage exit codes.
2. **`flake.nix` #lint app should print a final summary line** (modules
   linted, total issue count, explicit exit marker). Failures can currently
   hide mid-log since the loop continues past failures. Carried from last
   session; still open.
3. **Remove deleted-module paths from the doc-check invocation** so a green
   gate prints zero noise. Warning spam trains readers to ignore output.
4. **Meta-test: every LAYER/DEP_BUDGET/TEST_INFRA_MODULES key in
   check-module-layers.sh must resolve to an existing go.mod dir** — makes
   the silent-enforcement-killer class (spaced keys, missing entries)
   structurally impossible to reintroduce. Carried.
5. **Pre-load known-broken patterns.** When a prior session documented a
   failure mode (here: PIPESTATUS), the session should start with the
   correct pattern, not rediscover it.

## f) NEXT — ordered by leverage

1. USER: approve/decline tagging engine v4.0.2 ×4 + watermill/v4.5.0
   (unblocks 2–4). Verify gate is green; this is the release checkpoint.
2. After tags: remove the 5 temporary replaces from system/go.mod, then
   `GOWORK=off go build + test` re-verification of system/.
3. After tags: `go mod tidy` sweep of the ~49 stale `// indirect` shim refs
   across go.mod files (cosmetic today; becomes real once replaces drop).
4. Remove stale `../../flightrecorder|retry|idempotency` paths from the
   doc-check invocation in flake.nix (kills Doc Check warning noise).
5. flake.nix #lint app: final summary line + explicit exit marker (e2).
6. Meta-test: LAYER/DEP_BUDGET/TEST_INFRA_MODULES keys ⇄ existing go.mod
   dirs (e4).
7. Run `nix run .#vulncheck` + `nix run .#check-arch` as pre-tag release
   checklist steps (verify covered the rest).
8. Harvest still-open items from the 2026-08-14/15 status reports into
   TODO_LIST.md and mark the reports done (docs-health HARVEST).
9. Dgraph JournalReadFrom off-by-one fix (S) — carried.
10. cqrs-lint per-module regression tests — carried.
11. `.golangci.yml` exclusion audit (all exclusions justified + minimal,
    including the one added this session) — carried.
12. DuckDB/Row calibration benchmarks — carried.
13. Real Redis/NATS broker roundtrips for the watermill adapter — carried.
14. v5 Phase 8: delete transport/http, transport/grpc, and deprecated
    shells; write the migration guide — carried.
15. go-codec repo scaffolding — carried.
16. v5 decision: keep the kvstore SA1019 exclusion permanently or migrate
    tests onto the go-idempotency contract suite (g3).
17. CHANGELOG `[Unreleased]`: confirm the ADR-0128 extraction + verify-green
    state is described (gate counts entries; content not inspected this
    session).

## g) QUESTIONS (cannot figure out myself)

1. **Tag now or batch?** Engine v4.0.2 (×4) and watermill/v4.5.0 as a
   standalone release pass now that verify is green, or batched with the
   final transport/* v4.x patches into one release pass? This gates f2–f4.
2. **If tagging now: is pushing tags + master to the remote authorized?**
   I never push without explicit instruction, and tag-release.sh assumes
   the tags reach the module proxy eventually.
3. **SA1019 exclusion permanence:** is the scoped config exclusion for
   `(middleware|idempotency)/.*_test\.go$` acceptable as the permanent
   answer, or should the kvstore test migration onto the go-idempotency
   contract suite be scheduled before v5?
