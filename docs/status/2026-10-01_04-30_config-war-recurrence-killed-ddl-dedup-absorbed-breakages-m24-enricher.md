# Session Status — config-war recurrence killed, DDL dedup, six absorbed breakages fixed, M26/M15/M20-23 docs, graph unify, request-scope enricher (M24 code green, doc tail open)

> 2026-10-01, 01:45–04:30 CEST. Continuation of the SUPERB post-wave plan
> (`docs/planning/2026-09-28_22-00_SUPERB-post-wave-verification-and-backlog-pareto-plan.md`).
> Session start state: M18/M19 completed (with M19 doc tail pending), 4 owner
> questions open and unanswered. Owner input this session: generic "keep
> going" directives only — all rulings remain owner-gated and are now
> consolidated in the decision pack (below).

## Environment reality (drove most decisions)

- **Load NEVER went quiet for the composed gates**: opened briefly at 3.71 (01:45),
  bounced 9→57 (5 langserver instances woke up after my `nix fmt` reformatted
  3,048 files), sat ≥9 for the rest of the session, 47.80 at close.
- `verify-fast` hit **transient /mnt/buildcache races** (test-spawned `go build`
  under the ambient cache) — the documented ambient-env gotcha, confirmed by
  green isolated re-runs.
- Auto-commit daemon absorbed every edit batch (expected). Tree clean at each
  checkpoint.

## a) FULLY DONE (verified green in-session)

1. **M19 doc tail** — watermill skill `references/backends.md`: NATS gotchas
   entry (verbatim topic→stream names, dotted topics illegal,
   name-as-subject routing, subscribers never provision, `nats.Conn.Close`
   Closer adapter) + §3 verified-surface update; CHANGELOG M18 + M19 Added
   bullets; `check-changelog-symbols` green (41 citations); doc-check green.
2. **Duplication gate re-run → 2 new clone groups (M16.2's advisory-lock DDL
   twins) → FIXED by extraction, not annotation**: new
   `metaengine/ddl_lock.go` exports `metaengine.ExecDDLLocked(ctx, db, label,
   ddls)` (errorfamily-classified, label-prefixed — storage's superior error
   shape); `claimkit.ensureClaimsTablesLocked` and `pgengine.execDDLLocked`
   now delegate. Gate: **0 new clone groups** (baseline 187). metaengine +
   pgengine build/vet/tests green. Api golden +2 (7,518).
3. **Config-war recurrence #7 CAUGHT AND REPAIRED**: a daemon auto-commit
   (d9d62b845) had re-planted `gci` into `.golangci.yml` formatters AND the
   depguard block had gone missing. Removed gci (contract #18), re-spliced
   depguard at correct indentation (the check script's auto-repair landed it
   at column 0 — fixed manually), added `watermill-nats/v2` +
   `nats-io/nats.go` to the allow list (M19's test-only deps; 143/143 direct
   deps covered), re-pinned the config hash golden. `#check-lint-config`
   green, `#verify-fast` config legs green.
4. **ADR-0149 index drift fixed** — `docs/README.md` now rows
   `0149-durable-checkpoints-and-dlq-by-events` (a prior session shipped the
   ADR unindexed; the ADR-index doc assertion caught it).
5. **SIGPIPE flaky-gate bug fixed at root** — `scripts/verify-docs.sh`'s
   `grep | head -1` under `set -o pipefail` raced SIGPIPE once CHANGELOG
   outgrew the pipe buffer (observed exit 141, intermittent); replaced with
   single-process `awk '/^## /{print; exit}'`. Same-class `|| true` hardening
   on the two `check-canonical-facts.sh` `$()` extractions.
6. **eventtest nil-payload breakage fixed** — the 2026-09-28 daemon-absorbed
   `NewEvent`→`New` sweep changed nil-payload semantics (`New` rejects nil
   interface; 3 storage/memory tests red). `MakeTimelineEvents` now passes
   `[]byte{}`. storage/memory green.
7. **cqrs-lint typedfixture re-tidied** — pin-sweep had bumped
   go-branded-id out of resolvable range again (P014 + V007 test failures);
   tidy + build green, full cqrs-lint pkg suite green (15 packages).
8. **Turso CTE pin flipped with verification** — `TestTurso_RecursiveCTEProbeFails`
   failed because upstream turso driver NOW executes recursive CTEs. Renamed
   to `...ProbeSucceeds`, dated comment records the 2026-08-30→2026-10-01
   flip; runtime probe mechanism unchanged (still protects CTE-less servers);
   tursoengine suite green (15.3s).
9. **M23.1 final verify (turso defects A+B)** — repo pin had silently moved to
   `tursogo v0.8.1` WITHOUT the runbook's ivmrepro check. Ran it: **defects
   A/B/C all still reproduce on v0.8.1** (430.50 signature byte-for-byte;
   upstream fix PR #9392 not effective in v0.8.1). Runbook step 4 executed:
   `TursoGoIVMVerifiedThrough=v0.8.1`, `LastVerified=2026-10-01`;
   `#check-turso-version` caught 7 live citations — all updated in the same
   change, gate green.
10. **M23.2 verified on v0.8.1 by source inspection** — (a) no
    DriverContext/OpenConnector, (b) zero BYOK surface, (c) `parseDSN`
    (`driver_db.go:620`) silently drops unknown params (`encryption_hexkkey=`
    still opens unencrypted; the line-768 "unknown named parameter" error is a
    different parser). All three still-live, file-order (c)→(a)→(b).
11. **M26.1+M26.2** — NEW `docs/planning/2026-10-01_v5-cut-readiness-checklist.md`:
    every v5-gated row sequenced Layer 0 (owner rulings) → 1 (quiet legs) → 2
    (pre-cut migrations) → 3 (branch) → 4-10 (deletions in cascade order,
    scan-default flip ON the v5 branch per ADR-0123/G-T14) → 11-13 (docs,
    tag). TODO_LIST v5 section header carries the pointer receipt.
12. **M15.1+M15.2+M15.4** — NEW `docs/planning/2026-10-01_owner-decision-pack.md`:
    Part A 14 quick rulings (A1-A14, each with recommendation), Part B 5
    ADR-level rulings (SingleWriter ADR-0146, Direction ADR-0147, v5-encryption
    ×4, ADR-0138 demand check, this session's M16.1/M18.4/M19 rulings), Part C
    4 user actions. Folds in the 4 open session questions. M15.3 routing
    stays owner-gated.
13. **M20.1+M22.1-4+M23 notes** — NEW
    `docs/planning/2026-10-01_watch-notes-and-filing-pack.md`: matview v2
    demand-gated verdict (with the defect-A structural blocker), 4 CI watch
    notes, filing-pack verification state (M23.1/2 above, M23.3 md-go asks,
    M23.4 go-graph-rag invite draft plan). HOLD filing — owner-gated.
14. **M21.1+M21.2 graph unify** — NEW `metaengine.GraphBFSNodes[N any]`
    generic BFS core (typed-key param, never-nil result contract); `GraphBFS`
    is now a thin string-specialized delegate (signature unchanged — NO API
    break); `graphNeighborsFallback` delegates with `typedNodeKey` preserving
    original neighbor types. nil-vs-empty unified to never-nil. metaengine +
    sqliteengine graph tests green. Api golden +1 (7,519).
15. **M21.3 ephemeral passthrough contract unified** — `ephemeral-pg.sh` (go-
    prefix special case + EXTRA_ARGS append mode REMOVED) and
    `ephemeral-dgraph.sh` (go-prefix special case removed) now follow the
    redis/nats verbatim convention: args → exec with broker env; no args →
    curated default. bash -n + shellcheck clean. Convention documented in
    `docs/agents/gotchas-testing.md`. M21.4: templ watch row verified present
    in TODO_LIST (no lever until art-dupl templ support — by design).
16. **M24.1-M24.3 (code)** — read cqrs-htmx `audit_context.go` (the explicit
    gap comment: "go-cqrs-lite does not ship a correlation enricher"); NEW
    `event/request_context.go`: `RequestScope` struct (CorrelationID,
    RequestID, IPAddress, UserAgent, ClientID) + `WithRequestScope` +
    `RequestScopeFromContext` + `RequestScopeEnricher` (skips zero fields,
    composes via `CompositeEnricher`). 6 tests green (roundtrip,
    absent/zero, full + partial enrichment, nil-on-absent, composite with
    ActorEnricher). Api golden +4 (7,523).

## b) PARTIALLY DONE

- **M24.4 (doc tail)**: core.md section + cqrs-htmx TODO note + recipes
  classification NOT yet written. Code + tests + golden are done.
- **Receipts for TODAY's work**: TODO_LIST rows + CHANGELOG entries for the
  graph unify, request-scope enricher, ephemeral unify, turso re-verification,
  SIGPIPE fix, config-war repair NOT yet written (daemon has the edits; the
  receipts are this session's own debt).
- **verify-fast final green confirmation**: last full run went red ONLY on the
  transient buildcache races + the 8 absorbed breakages — all fixed and
  individually re-verified green since; the full composed re-run (M04) has
  not happened (load). So "everything green" is per-module verified, not
  composed-verified.
- **M04**: never launched (load ≥9 after the brief 3.71 window; the
  preflight→wait-loop→verify chain was designed but not started).

## c) NOT STARTED (this session)

- M25 (mesh-demo coeffect-gate variant), F032 (named-[]byte lint rule),
  NATS second-run flake check, M05 (mysql-vm + seed replay), M11
  (calibration → SearchQuery count=5 → dgraph constants), M14.3 (per-module
  fresh-run stamps), `nix flake check`, closing `nix fmt` pass, push master.

## d) TOTALLY FUCKED UP (honest)

- **Nothing unrecoverable.** Three self-inflicted iterations worth flagging:
  1. **Two bad ULID literals in my own new test** (invalid char U, then
     25-chars) — two compile/test cycles wasted on hand-minted IDs instead of
     copying the known-good pattern; the exact "sloppy multiedit anchors"
     failure mode the last session's self-critique named.
  2. **My `nix fmt` triggered the load storm** — reformatting 3,048 files woke
     5 golangci-lint langservers that pinned load ≥9 for hours and poisoned
     the shared buildcache mid-verify-fast. I should have formatted only the
     changed modules or accepted the per-module lint runs.
  3. **Dead placeholder lines in a first draft** of `pgengine/ddl_lock.go`
     (`var _ = context.TODO`) caught on re-read before commit — sloppy write.
- Pre-existing rot found (NOT mine, all fixed by me): gci re-planted + depguard
  missing (daemon commit d9d62b845), ADR-0149 unindexed, nil-payload helper
  break, typedfixture pin rot, stale CTE pin, unverified tursogo pin bump
  (constant said pre.10, pin was v0.8.1 — a runbook breach absorbed silently).

## e) WHAT WE SHOULD IMPROVE (systemic)

1. **The daemon's config corruption needs a faster tripwire**: gci was
   re-planted and depguard vanished BETWEEN verify runs. `#verify-fast` has
   the tripwires but only runs when someone runs it — consider wiring
   `check-golangci-hash.sh` into the daemon's own pre-commit heuristic.
2. **tursogo pin bumps must be gate-locked**: the pin moved to v0.8.1 with the
   canonical constant saying pre.10 and nothing failed. `#check-turso-version`
   checks citations vs constants, NOT go.mod vs constants. Add a leg asserting
   `go.mod pin ≤ TursoGoIVMVerifiedThrough` (or force the ivmrepro check on
   pin change).
3. **ULIDs in tests should come from helpers** (idtest mint-from-seed or
   `ulid.Make`), never hand-typed — my two failed cycles are the argument.
4. **`nix fmt` on a 3k-file tree is a load weapon**: scope it (treefmt walks
   changed paths) or schedule it, especially when langserver instances are
   alive.
5. **The `sweep` flake app runs `golangci-lint --fix`** — with formatters
   enabled that rewrites imports; that is the likely mechanism behind the
   daemon's gci-era rewrites. Sweep should refuse to run while gci-style
   formatters are present (defense in depth beyond the config gate).
6. **grep|head under pipefail is a landmine class** — I fixed 3 sites; a
   one-line repo-wide audit for the pattern in set -e scripts is cheap and
   overdue.

## f) NEXT (up to 50, priority order)

1. M24.4: core.md §(enrichers) row for RequestScope trio + recipes classify + cqrs-htmx TODO note ("drop local requestContextEnricher at next bump").
2. Receipts: CHANGELOG bullets (GraphBFSNodes, RequestScope enricher, ExecDDLLocked dedup, ephemeral unify, turso v0.8.1 re-verification, SIGPIPE fix, config-war repair note) + TODO_LIST row strikes/annotations.
3. M04 composed verify (EXCLUSIVE, load1<5): preflight → can-run-composed-gate --wait-loop → `nix run .#verify`.
4. Second NATS suite run (flake check) under the nix-shell ephemeral invocation.
5. M25.1-25.3: mesh-demo system.New coeffect-gate variant + core.md §9 row + receipts.
6. F032: named-[]byte lint rule (cqrs-lint) guarding the event.New bug class.
7. M05: `nix run .#integration-mysql-vm` + shuffle-seed replay from build/shuffle-seeds.log (quiet window; check QEMU port 33070 first).
8. M11: calibration-gate → SearchQuery count=5 re-run → supersede note if >5% → dgraph constants re-anchor (one gate-passing window).
9. M14.3: per-module fresh-run stamps (quiet CPU).
10. `nix flake check` (quiet-gated).
11. turso pin-vs-constant gate leg (improvement e2).
12. grep|head audit across scripts/ (improvement e6).
13. Route owner answers from the decision pack (M15.3) when they arrive.
14. Benchkit tag wave per A13 ruling.
15. Push master per A2 ruling.
16. v5 checklist Layer 2 migrations (listing type-driven status; taskmanager off OnTombstone) — v4.x-safe, unblocked.
17. ADR-0149 body read-through vs `system.NewEngineCheckpointStore` reality (it was authored by a prior session; spot-check the citation).
18. The 3 repaired gate scripts' self-tests re-run (`#check-release-scripts`) — verify-docs.sh + canonical-facts edits carry no self-test legs themselves; confirm nothing else pins their internals.
19. mysql snapshot-migration live run (v5 checklist Layer 1) in the next quiet window.
20. Watch: first push-gated CI legs (M22.1) once billing is fixed.
21. turso A+B filing upon approval (draft + v0.8.1 range + #9391/#9392 context, onset matrix).
22. turso-go (c)(a)(b) filings upon approval (verified this session).
23. md-go-validator upstream asks (M23.3).
24. go-graph-rag consumer invite (M23.4).
25. Systemtest stress tests under -race (last session's known gap).
26. Load-watcher: background poller that pings when load1<5 for 3 consecutive minutes (this session polled by hand between tasks).
27. Re-run `#check-duplication` after M24/M25/F032 code lands.
28. Re-run doc-check over core.md/recipes after M24.4.
29. Confirm `#check-md-go` over the three new planning docs (should parse; no Go fences used).
30. Canonical-facts gate after the status-index row (this report).
31. Sweep eval: `evals/` validation via claude CLI (user action C3).
32. ERRAUDIT_PAT secret (user action C2).
33. GH billing fix (user action C1).
34. benchkit/LICENSE name (user action C4).
35. ADR-0146 SingleWriter ratification (B1).
36. ADR-0147 Direction ruling (B2).
37. v5-encryption 4 questions (B3).
38. ADR-0138 demand check (B4).
39. M16.1 clock seam ruling B-vs-C (B5.1).
40. M18.4 pebble/bbolt ruling A-D (B5.2).
41. M19 CI-leg ruling (B5.3).
42. Quiet-window strategy ruling (A14).
43. M22/Q3 report-artifact policy (A1) — this report follows the narrow-trigger shape.
44. claiming V006 advisory (A3), iroh P99 ratify (A4), T18b (a)(b) (A5/A6), DSN strict (A7), sync/embedded (A8), dgraph one-RPC (A9), CapabilityGaps→Doctor (A10), #test-examples (A11), docs-health cadence (A12).
45. v5 checklist Layer 0 blockers chase once rulings land.
46. Nightly bench verification vs baseline (next scheduled run).
47. `#verify-ci` parity run (GOWORK=off per-module matrix).
48. Post-M04: dedup (a) + CI re-record row receipts (M04.5).
49. Consider killing orphaned 22h-old langserver instances (they are the load floor).
50. Session-close: final `nix fmt` (scoped if possible), api-stability verify pass, closing index update.

## g) Questions I cannot answer myself

1. **Master push**: daemon holds ~15 auto-commits of this session's work; tags
   were already pushed by earlier sessions. Push `master` now (A2 recommends
   yes) — yes/no?
2. **Load floor**: five golangci-lint langservers (up to 22h old, several
   likely orphaned editor sessions) keep load1 ≥9 and block every quiet-gated
   gate. May I kill the stale ones (keeping the newest)?
3. **Composed-verify fallback**: if load1<5 never returns this week, do you
   want (a) the composed verify run anyway at load ~9 with the known
   cache-race risk, (b) a named off-hours window, or (c) accept per-module
   green as the release signal until a window appears?
