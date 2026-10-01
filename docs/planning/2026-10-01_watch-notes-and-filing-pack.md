# Watch notes + verified filing pack — matview v2 triage, CI watches, upstream filings

> Compiled 2026-10-01 (M20.1 + M22.1–4 + M23.1–4 of the SUPERB post-wave plan).
> FILINGS THEMSELVES ARE OWNER-GATED (github-voice + explicit approval) — this
> file records verification state and what is ready to file the moment
> approval lands.

## M20.1 — Matview v2 demand-triage (routing decision)

Verdict: **DEMAND-GATED, do not build.** The v2 surface (planned-table
matviews, filtered-view variants, multi-aggregate/DISTINCT serving,
`DropMaterializedView`, IVM write-amp otel counter, `system.Introspection()`,
two cqrs-lint rules, dedicated example) routes individually when a consumer
asks — same policy as ADR-0138 (see owner-decision pack B4). Structural
blocker stays regardless of demand: upstream defect A makes GROUPED views
unsafe beyond one transaction's rows (re-verified on v0.8.1 today, below), so
the only buildable v2 slice today is scalar-shape serving + the
`AggregateOn` seam (one-pager 2026-09-21) with grouped shapes left O(N) +
Doctor note.

## M23 verification state (2026-10-01 live re-verification)

### M23.1 — turso defects A+B (silent wrong results) — READY, approval-gated

- Draft: `docs/research/2026-09-07_turso-go-ivm-commit-failure-issue-draft.md` (everything below its first `---`), fully verified.
- **NEW 2026-10-01:** the repo pin moved to `tursogo v0.8.1` (latest) WITHOUT the runbook's ivmrepro check; run today, `-tags ivmrepro` on v0.8.1: **all three defects A/B/C still reproduce** (suite green = defects present; the 430.50 delta signature reproduced byte-for-byte). Canonical constants bumped in the same change (`TursoGoIVMVerifiedThrough=v0.8.1`, `LastVerified=2026-10-01`), `#check-turso-version` green across all live citations.
- Consequence for the filing: upstream PR tursodatabase/turso#9392 (defect-C fix) has NOT carried into v0.8.1 for these paths — the A+B issue body must cite the range **v0.7.2 … v0.8.1 (latest)**, reference #9391 (zombie-tx readback, filed) and #9392 (fix in flight, C-only), and carry the onset matrix (`docs/benchmarks/2026-09-28_ivm-defect-a-onset-matrix.md`; wall re-observed at 29k standalone).
- Remaining pre-file step when approval lands: none — the onset-matrix-on-fixed-build precondition applies only once a fix actually ships.

### M23.2 — turso-go driver (a)(b)(c) — VERIFIED STILL-LIVE on v0.8.1 (2026-10-01, source inspection)

- (a) missing `DriverContext`/`OpenConnector`: **confirmed** — zero matches in the v0.8.1 package; struct-level config still requires DSN stringification.
- (b) pure-remote connections cannot present a BYOK key: **confirmed** — no BYOK surface anywhere in v0.8.1 (sync/embedded replica remains the only Go path to Cloud BYOK).
- (c) mistyped DSN params silently ignored: **confirmed** — `parseDSN` (`driver_db.go:620`) reads only known keys via `vals.Get` and drops everything else; `encryption_hexkkey=` (typo) still opens the DB unencrypted. (The `unknown named parameter` error at `driver_db.go:768` is a different, non-DSN parser — it does not protect this path.)
- File order: (c) first (security class), then (a), then (b).

### M23.3 — md-go-validator upstream asks (owner's own repo)

- (a) relative-path baseline mode — deletes every consumer's sed re-absolutization layer (this repo's wrapper carries one).
- (b) `--save-baseline` exits 0 when the save succeeds (wrappers should not need `|| true`).
- Both verified against the wrapper experience in-repo; drafting needs the current upstream main state re-checked (owner repo, direct access).

### M23.4 — go-graph-rag consumer update

- Feedback #3 + #5 fixed and shipped in `system/v4.9.0` (`ErrRacySaveRefused`/`WithRacySave`/`ErrEventSaveNotAtomic` + 17 `# Experimental` stamps); consumer unaware. Draft: invite re-test on v4.9.0 (owner voice; `github-voice` skill at draft time).

## M22 — CI observe notes (no action until the watched events occur)

1. **First real push-gated CI runs** (M22.1): `Examples Test` job (nix eval, 10m timeout, DB-skip env), md-go-validator ci.yml leg (cold build 1–2 min), nightly `Go version contract` step, README push-leg timeout. NOTE 2026-10-01: paid CI legs remain broken on GH billing (user action, decision-pack Part C) — the free nix-based legs are the observable ones.
2. **Nightly weekly load-sweep leg** (M22.2): Sundays-only; first real fire must be confirmed in the nightly logs (check the next Sunday's run + `scripts/nightly/` timer journal).
3. **dgraph+redis shuffled CI watch** (M22.3): ~10 green runs before declaring stable; record seeds from `build/shuffle-seeds.log` per run. Tally restarts at each CI-leg change.
4. **TestEngineHealth_CatchUpUnderConcurrentApplies** (M22.4): observe-only — 15/15 green isolated (2026-09-28 receipt), no recurrence under full-suite contention since; revisit only on recurrence.
