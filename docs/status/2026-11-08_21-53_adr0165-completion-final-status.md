# FINAL Status: ADR-0165 Completion Arc — Executed, One Receipt Owed (brutal-honest)

**Session:** 2026-11-08 16:15 → ~18:31 active work, status cut 21:53 CET · **Trigger:** owner "GET SHIT DONE — the WHOLE TODO LIST" over `docs/planning/2026-11-08_16-11_SUPERB-adr0165-standard-topology-completion-pareto-plan.md`.
**Companion report:** `2026-11-08_18-30_standard-topology-completion-arc-status.md` (task-level detail; its §f verify line is SUPERSEDED by this report — the full-#verify receipt did NOT land, see d-2).

> **One-line truth:** all 14 medium tasks (46 fine tasks) executed and gated — `#verify-fast` GREEN end-to-end, QEMU VM proves the peer-auth contract on hardware — but the composed full `#verify` receipt was NOT obtained (verify-chain died on a load-guard exit-code mismatch), and the work shipped as daemon `chore:` commits, not the authorized phase-boundary messages.

---

## a) FULLY DONE (verified, receipts listed)

1. **P01 — Postgres auth truth + fix.** nixpkgs pinned source: default `pg_hba` = `local all all peer` + `host 127.0.0.1/32 md5` (postgresql.nix:688-697) — the TCP-trust assumption was FICTION. `ensureDBOwnership` = plain `ALTER DATABASE OWNER TO` (no PG floor). Fix: socket DSNs (`postgres:///cqrs_<n>_<e>?host=/run/postgresql&user=...`, libpq socket dir verified in the store binary) + `cqrs_peer` pg_ident map (`cqrs-<name>` → `cqrs_<n>_<e>`) + database-scoped hba line ahead of the defaults. Zero passwords anywhere. Module header documents the service-user contract.
2. **P02 — Cross-layer round-trips.** `system/config_roundtrip_test.go` loads the NixOS-rendered JSON + the CUE golden JSON through `system.LoadConfig` verbatim; artifacts committed (`standard-topology.nixos-rendered.json`, `standard-topology.golden.json`).
3. **P03 — Committed NixOS eval test.** `nix/nixos/eval-test.nix` → flake check `cqrs-nixos-eval`: x86_64-linux AND aarch64-linux from any host; asserts rendered JSON vs the shared triple, provisioning + ownership, hba/ident lines, hardenedService fragment. Mutation-tested BOTH directions (corrupt → "drifted" → restore → green).
4. **P04 — Gate wiring.** `#check-cue-config` in `#verify`, `#verify-fast`, ci.yml (self-test + real); `cqrs-nixos-eval` as a CI step.
5. **P05 — Equivalence pin.** Shared `system/testdata/standard-instance-triple.json` asserted by the Go suite (DeepEqual), the CUE gate (jq-diff, fault #5 of 6), and the Nix eval test (fixture-loaded expected). Caught a REAL split brain: the Nix twin never stamped `collections` — fixed everywhere; fixture-corruption fails all three legs.
6. **P06 — `example/standard-topology/`.** domain/app/main/test + README ladder; standalone-green FIRST TRY (`GOWORK=off`); full 6-gate registration (go.work use, examplePaths, 4 api-stability exclusion maps, LAYER 7 + DEP_BUDGET 8, cqrs-lint catalog) + standalone-audit baseline row for the pre-release system `replace`; FEATURES row.
7. **P07 — VM test.** `nix/vm/cqrs.nix` + `cqrsServiceTest` → check `cqrs-vm` + nightly CI leg: QEMU boots, asserts rendered config, **`SELECT 1` as `cqrs-demo` over the socket (the peer-auth contract ON HARDWARE)**, ownership check, wrong-OS-user rejected with the exact map-miss log line. Green twice (also after the `file:` DSN change).
8. **P08 — Polish.** sqlite `file:` DSN unified across the ADR-0165 surface; `mdDoc` on all 18 option descriptions; README fences verified already parse-gated repo-wide by md-go (1532 blocks).
9. **P09 (partial, see b) — release hygiene.** `#verify-fast` **GREEN (exit 0)** after the 8-attempt tail-chase (§d-4); CHANGELOG `[Unreleased]` rows (Added ×2, Changed ×1; changelog-symbols 13 citations honest); canonical-facts repaired (104 go.mod claims ×6 sites, 108 recipes, status-index rows); v4.18.0 cut PREPPED, no tag (owner gate).
10. **P10 — Consumer docs.** recipes.md §2.51 "The Deployment Ladder" (compile-verified fence + catalog entry; 108/108) + FAQ "move the query audit log to its own engine".
11. **P11 — Extra pins (all six).** `standardTopology=false` → empty instances AND provisioning deliberately ON (my wrong first assert corrected — the gate taught the independence); explicit-DSN suppresses provisioning; CUE `deployment`-entry golden + gate fault #6; both-arch eval (F38); `nixosModules.default` alias equality; maintainers meta.
12. **P12 — Fuzz + property.** `FuzzLoadConfigJSON` (7 seeds incl. the NixOS artifact; 10s campaign: 70k execs, 85 interesting, zero crashers; registered as fuzz-rotation target **117/117**); rapid: `Standard` idempotent + permutation-invariant (SliceOfNDistinct; no skips).
13. **P13 — Chassis (opportunistic split).** **cqrs-htmx ADOPTED**: all 7 one-engine presets return `system.Standard(...)`; `RecommendedSplitSQLiteDeployment` stays explicit BY DESIGN (Standard pools projections over every engine — would write projections into the journal file); systemadapter + usermgmt suites GREEN (0.958s / 19.6s). **go-appkit BLOCKED on the tag** (published-version pins; site rowed).
14. **P14 — Close-out.** ADR-0165 dated addendum (nixpkgs citations, receipts, deferrals); plan reconcile banner (per-task verdicts); TODO_LIST rows updated (10 checked, 1 amended with tag-time checklist); gotchas recorded (`gotchas-testing.md` directive positioning + rapid signature; `gowork-modes.md` NEVER bare `go work sync`); this report + index row.

**Gates green at cut:** verify-fast · check-cue-config (+6-fault self-test) · cqrs-nixos-eval (both arches) · cqrs-vm (QEMU) · check-md-go (1532 blocks) · canonical-facts · check-adr-numbering (163) · check-changelog-symbols (13) · check-test-sleeps (182 frozen) · check-test-skips (242 routed) · check-duplication (0 new) · lint-scope (system, cmd/doc-check, example) · module-layers · example-standalone audit · cqrs-lint catalog + api-stability TestEvery · doc-check (1389 refs) · cqrs-htmx systemadapter + usermgmt suites.

## b) PARTIALLY DONE

1. **Full composed `#verify` receipt — OWED.** verify-chain armed 18:28, refused twice on tree-stability (my own doc edits restarted the 60s window), then the load guard REFUSED the verify step with **rc=1** — but the chain retries ONLY rc=12 → `[chain] step FAILED for real (rc=1), not retrying` (18:31). Load at 21:53 is **61** (parallel sessions hammering). verify-fast IS the composed fast gate and IS green; the missing receipt is the full chain (soak incl. bench gate, coverage floor, vulncheck-class legs).
2. **`#check-coverage`** — rides the full verify; not run standalone this session (system-module floors were green at the last wave).
3. **Authorized phase-boundary commits never landed.** The daemon won both HEAD races (16:38, and the 174-file sync commit); everything shipped as `chore:` commits. Content is green-gated; the MESSAGE trail the owner ruling wanted (real messages at phase boundaries) is missing for this arc.
4. **cqrs-htmx CHANGELOG row for the preset shape change** — presets + tests edited and suite-green, but I did not touch cqrs-htmx's own changelog/README preset docs. (Deliberate question to owner, g-3.)

## c) NOT STARTED (and why)

1. **system v4.18.0 TAG** — owner-gated ("NO tag without owner"); cut prepped.
2. **go-appkit `defaultDeployment` → `system.Standard`** — impossible until the tag (published pins, no sibling replace).
3. **Example binary inside the VM** — same tag dependency (pre-release `replace` blocks a plain `buildGoModule` package).
4. **The 266-row backlog** — owned by the 12-28 plan; deliberately untouched (verified zero overlap).

## d) TOTALLY FUCKED UP (mine, no excuses)

1. **`go work sync` — 174-file fleet churn.** I ran a fleet-wide command to register ONE module; it rewrote go.mod/go.sum across the tree (templ 1070→1100 require bumps — the exact whipsaw the flake pin exists to prevent), left sums missing so every `GOWORK=off` run failed, and the daemon committed it within minutes, forcing a surgical `git restore --source=HEAD~1` over the .mod/.sum set. Recorded as a NEVER rule in `gowork-modes.md`.
2. **Verify-chain arming discipline.** I armed the long gate and KEPT EDITING (status report, gotchas, index) — restarting the stability window repeatedly — and had not read the retry contract closely enough: the chain's retry-only-on-rc-12 does not cover the load guard's rc=1 refusal. Net: 3.5h later there is still no full-verify receipt. Correct order: finish ALL edits → go silent → arm.
3. **`git add -A` with a live parallel session** — staged a foreign status file into my phase-boundary commit; the daemon's HEAD-lock race aborted the commit, so nothing wrong SHIPPED, but the process would have committed another session's file.
4. **8-attempt verify-fast tail-chase — 4 failure classes were MY exhaust:** go.mod count 103→104 (my example; 6 doc sites + canonical facts), a Han-script fuzz seed tripping gosmopolitan, `rt.SkipNow()` without a reason (skip gate), the example's `run()` clone (accept-annotated) — plus my FIRST eval assertion encoding a wrong model (`standardTopology=false` must NOT disable provisioning). Each was caught by exactly the gate designed for it, but I burned ~5 full gate cycles discovering my own residue because I did not pre-flight the cheap gates before the composed run.
5. **Artifact churn ×3.** I wrote the NixOS artifact + goldens, then changed semantics twice more (collections in P05, `file:` prefix in P08), regenerating artifacts and re-proving the VM each time. Settle semantics FIRST, generate artifacts ONCE.
6. **The 18-30 report ships an unfulfilled promise** ("receipt appended below when it lands") — the receipt never landed; this report supersedes that line.

## e) WHAT TO IMPROVE (durable)

1. **Arm-last, then silence** — no edits after arming any wait-loop gate.
2. **Fix the retry contract:** load guard should exit 12 (or verify-chain must recognize the refusal) + a self-test fault for it — a real defect this session exposed.
3. **Cheap-gate pre-flight before every composed run** (canonical-facts, sleeps, skips, clones, lint-scope on touched modules) — the tail-chase would have been one cycle.
4. **Semantic decisions before artifact generation** (collections/DSN form were still moving in P08).
5. **Explicit-path staging only** while parallel sessions are live.
6. **Cross-repo edits need cross-repo docs** (cqrs-htmx changelog) — same edit window, not a follow-up.
7. **Directive positioning after `nix fmt`** — annotated gates must be re-run post-format (recorded in gotchas-testing.md).

## f) NEXT (ranked; 1–13 are this arc's tail, 14+ reference the standing backlog)

1. Re-arm full `#verify` via `nix run .#can-run-composed-gate -- --wait-loop && nix run .#verify` in the next quiet window; durable log to `build/verify-chain-<ts>.log`.
2. Fix verify-chain/load-guard rc contract (+ `--self-test` fault).
3. **Owner: tag system v4.18.0** — unblocks 4–6.
4. Post-tag: drop example pre-release `replace` + standalone-audit baseline row (+ go.sum tidy check).
5. Post-tag: go-appkit `defaultDeployment` → `system.Standard` + its suite.
6. Post-tag: example-binary VM leg (plain `buildGoModule` package; `F24` completion).
7. cqrs-htmx: changelog row (+ preset README if stale) for the 2→3-instance shape change.
8. cqrs-htmx release wave; sweep ITS consumers (crm, KeyHolderAI class) over the preset shape.
9. `#check-coverage` standalone (system floors after the new test files).
10. Watch tonight's Fuzz Nightly: `FuzzLoadConfigJSON` must appear in the rotation shard.
11. Authored commit restoring the message trail for this arc (a docs-only commit with a real message; history stays chore).
12. Sweep core/v5 for OTHER inert >5-line-window art-dupl directives (the fuzz twins were inert since birth — same class may hide elsewhere).
13. VM asserts: extend with config-file readability by the service user (`sudo -u cqrs-demo cat /etc/cqrs/...`).
14. CUE `deployment`-entry output → `LoadConfig` round-trip on the Go side (golden exists; loader test missing).
15. ADR-0165: add the `sot.implicit_memory` interplay note to recipes §2.51 (engines-empty passthrough) if consumers trip on it.
16–50. **The 266-row backlog** stays owned by `2026-11-08_12-28_SUPERB-full-backlog-trust-to-v5-pareto-plan.md` — deliberately not duplicated here; the standing open owner questions from prior reports (decider.StateCache race A/B, SSE flake instrument-vs-quarantine, daemon retitle, md-go-validator pin policy) all remain open and rank above anything new.

## g) QUESTIONS (cannot figure these out myself)

1. **Tag system v4.18.0 now?** The cut is prepped (CHANGELOG rows, minor-version decision per new export). Three deferred items (go-appkit adoption, example replace drop, VM binary leg) unblock on it — but the standing rule is no tag without you.
2. **Full-#verify policy tonight:** re-arm the wait-loop now and let it storm-ride (host load is 61 at 21:53 with parallel sessions live), or park it for the overnight watcher like previous waves? Both have precedent; the host is the constraint I can't decide for you.
3. **cqrs-htmx preset shape change (2→3 instances per Recommended* preset):** want me to add ITS changelog row and run a consumer sweep (crm etc.) from this side, or does cqrs-htmx's own release flow own that repo's documentation?

---

*Verification receipts cited: verify-fast exit 0 (attempt 9, `/tmp/verify-fast9.log` summary); check-cue-config "OK (vet + golden + triple pin + drift pin over 28 fields)" + "self-test: OK (6 fault injections caught)"; cqrs-vm "config + peer-auth provisioning verified" (QEMU 13.2s); cqrs-nixos-eval green both arches + mutation-proven; system/example/doc-check/cqrs-lint suites green; canonical-facts all-match. NOT obtained: full `#verify`, `#check-coverage`, aarch64 on real hardware.*
