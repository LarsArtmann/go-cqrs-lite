# go-atomic-write — Utilization Audit · go-cqrs-lite

> Audited 2026-11-22 · Library `github.com/larsartmann/go-atomic-write` @ 4f10347 (latest local tag v0.6.2) · Consumer go-cqrs-lite @ feec9bc74 · Group G3

**Verdict:** UNDERUSED-SEVERE
**Score:** 0/4 capabilities fully leveraged = 0% (low)

The library is already inside the module graph of all five repo CLI tools —
pulled indirectly by `cmdguard/v5`, which every tool imports directly — yet
not one line of consumer code imports it. Worse than unused: `cqrs-lint`
hand-rolls a do-it-yourself temp-file-rename that is exactly the library's
"DIY write + rename" comparison row (crash-durable: *Partial*; concurrent:
*no*), inside a module that already carries the real thing transitively.
And `cqrs-upgrade` rewrites the **user's source files** with bare
`os.WriteFile` — the highest-stakes non-atomic write in the whole repo.
27 non-test `os.WriteFile` sites stand where a one-liner exists.

## 1. What the library is

Crash-safe, race-free file writes: every `os.WriteFile` has three silent
failure modes — TOCTOU overwrite, crash-torn files, interleaving writers —
and the library handles all three (go-atomic-write/README.md:27-37). The
pipeline: xxhash64 fingerprint of current content → write to unique `.tmp` →
fsync temp → flock + re-verify fingerprint → single atomic `rename(2)` →
fsync directory (go-atomic-write/README.md:47-54). API: `Write` (plain
atomic), `WriteVerified(path, data, fp)` returning `ErrConcurrentModification`
when the file changed under you, `WriteIfChanged` (skips identical content),
`WriteFunc*` streaming variants, `WriteWithPerm` — per
go-atomic-write/FEATURES.md ("Core write API" table; `Write` at
atomicwrite.go:15, `WriteVerified` at :63, `WriteIfChanged` at :89). Two
dependencies (xxhash, gofrs/flock); POSIX dir-fsync + Windows
`MoveFileEx`-with-retry split (go-atomic-write/README.md:39-45). HEAD
4f10347 (2026-11-15); v0.6.2. Self-audited FEATURES.md with per-feature
test citations — a credible maturity signal.

## 2. Usage census in go-cqrs-lite (evidence)

**Indirect in exactly 5 go.mod, zero imports.**
`rg -n 'larsartmann/go-atomic-write' --type go` over the repo: no hits. The
pins:

- `cmd/doc-check/v4/go.mod:35` — v0.6.2 `// indirect`
- `cmd/api-stability/v4/go.mod:36` — v0.6.2 `// indirect`
- `cmd/cqrs-gen/v4/go.mod:35` — v0.6.2 `// indirect`
- `cmd/cqrs-lint/v4/go.mod:44` — v0.6.1 `// indirect`
- `cmd/cqrs-bench/v4/go.mod:91` — v0.6.1 `// indirect`

`go mod graph` names the requirers: `cmdguard/v5@v5.0.0 →
go-atomic-write@v0.6.1` (all five tools import cmdguard for CLI config,
e.g. `cmd/cqrs-lint/v4/main.go:13,170-174`), plus
`samber-do-auditlog@v0.12.3 → v0.6.2` in the three newer graphs — MVS
picks 0.6.2 there, 0.6.1 elsewhere. Patch-level split, self-healing on the
next tidy.

**The hand-rolled surface that exists instead** (non-test, key sites):

- `cmd/cqrs-lint/v4/baseline_file.go:100-125` — a hand-rolled atomic write:
  `os.CreateTemp(dir, ".cqrs-lint-baseline-*.tmp")` → write → close →
  `os.Rename(tmpName, path)`, with manual cleanup on each failure path
  (~25 lines). No temp-file fsync, no directory fsync, no cross-process
  lock, no fingerprint — precisely the "DIY write + rename" the library's
  comparison table grades Partial (go-atomic-write/README.md:39-45). This
  module *already has* go-atomic-write in its go.mod.
- `cmd/cqrs-upgrade/v4/util.go:12` — `osWriteFile` = bare
  `os.WriteFile(path, data, 0o600)`; used by the v4→v5 rewriters
  (`rewrite_v5.go:199`, `rewrite_v5_surface.go:150` also writes directly,
  preserving perms). These rewrite the **user's Go source files**; a crash
  mid-write leaves a torn `.go` file behind. Not in the 5-tool set — this
  tool would need the dep added (a cmd tool; budget-appropriate).
- `cmd/cqrs-gen/v4/main.go:124` — generated handler code written with bare
  `os.WriteFile(outputFile, []byte(code), 0o644)`; cqrs-gen already carries
  the dep transitively.
- `cmd/api-stability/v4/golden.go:29` — golden export list, bare write of a
  committed artifact.
- `catalog/v4/eventcatalog/writer.go:19,68,114,186,288` +
  `index_manifest.go:262` — the catalog exporter's write layer (its own
  comment: "centralizes the strings.Builder + os.WriteFile",
  `writer.go:180`); `catalog/v4/compat/baseline.go:94`;
  `catalog/v4/architecture/exporter.go:173`;
  `catalog/v4/cmd/go-cqrs-lite-catalog/main.go:113,127,159`.
- `cmd/cqrs-lint/v4/pkg/suppression/fix.go:187` — rewrites the user-managed
  suppression file (read-modify-write, unprotected).
- `metaengine/v4/plan_diff.go:186` — `SaveManifest`, bare write inside a
  production library module (diagnostic artifact; see gap 3).

## 3. Capability inventory

| # | Capability | Grade | Evidence / reason |
|---|---|---|---|
| 1 | Plain atomic write (`Write`) | Missed | cqrs-lint hand-rolls the weaker DIY version (`cmd/cqrs-lint/v4/baseline_file.go:100-125`) while already requiring the lib via cmdguard (`cmd/cqrs-lint/v4/go.mod:44`) |
| 2 | TOCTOU-verified write (`WriteVerified` + `ErrConcurrentModification`) | Missed | suppression fix rewrites a user-managed file read-modify-write with no protection (`cmd/cqrs-lint/v4/pkg/suppression/fix.go:187`); no caller anywhere verifies a fingerprint |
| 3 | Idempotent write (`WriteIfChanged`) | Missed | catalog exporter + baselines rewrite identical bytes on every run (`catalog/v4/eventcatalog/writer.go:19+`, `catalog/v4/compat/baseline.go:94`, `cmd/api-stability/v4/golden.go:29`) — mtime churn for downstream caches and noisy git status |
| 4 | Crash durability (temp fsync + dir fsync) | Missed | the DIY path skips both (`cmd/cqrs-lint/v4/baseline_file.go:100-125`); `cqrs-upgrade` writes user sources with no protection at all (`cmd/cqrs-upgrade/v4/util.go:12`) |
| 5 | Cross-process locking (flock / LockFileEx) | N/A | today's workflows are single-writer one-shot CLIs — flip: CI matrix jobs sharing one checkout writing the same baseline/golden files |
| 6 | Streaming write (`WriteFunc*`) | N/A | every writer holds full content in memory — flip: an exporter streaming multi-MB diagrams/manifests |
| 7 | Windows rename retry (`rename_windows.go`) | N/A | fleet CI is linux-only (same posture as go-health's compatibility note) — flip: first Windows developer tooling |
| 8 | Fingerprint helpers / retry semantics | N/A | exists to serve `WriteVerified` callers; none exist yet — arrives with row 2 |

Score derivation: rows N=8, N/A rows 5–8 → **A=4**. Full=0, Partial=0,
Missed rows 1–4 (4 × 0). Score = round(100 × 0 / 4) = **0% → low band**.
Severe is the right word: the dependency is *present in five module graphs*
and leveraged zero percent, while its exact product is re-implemented
worse inside one of those five modules.

## 4. Gap analysis

1. **cqrs-lint is the flagship irony.** Its baseline writer
   (`baseline_file.go:100-125`) is a hand-maintained clone of the library's
   stage→commit pipeline minus fsync, lock, and fingerprint — 25 lines that
   exist only because nobody noticed the dependency was already there. The
   swap is `atomicwrite.Write(path, data)` and moves one line in go.mod
   from `// indirect` to direct. Zero new module-graph weight.
2. **cqrs-upgrade is the highest-stakes miss.** It edits user source files
   (`rewrite_v5.go:199`, `rewrite_v5_surface.go:150`, via `util.go:12`) —
   the one category of file in the repo that is *not* regeneratable from
   the repo itself. A crash mid-rewrite corrupts the user's working tree;
   `Write`'s temp+rename+fsync makes that structurally impossible. This
   tool is also how every fleet project migrates v4→v5, so the risk is
   fleet-wide, and so is the fix's payoff.
3. **Regeneratable artifacts are a judged, not blind, swap.** Golden files,
   catalog exports, and baselines can be regenerated; plain `WriteFile` is
   defensible there, and `catalog/v4` is a production module under dep
   budgets (two transitive deps: xxhash, flock). The capability that *does*
   pay today is `WriteIfChanged`: identical-bytes exports stop churning
   mtimes, which docserver-side caches and git status both feel. If budget
   forbids the dep in catalog/v4, the honest move is to record that as a
   deliberate decision (same posture as docserver's nonce in
   `httputil.md`), not silence. `metaengine/v4/plan_diff.go:186`
   (`SaveManifest`) stays bare for the same budget reason — it is a
   diagnostic artifact.

## 5. Version currency

Indirect pins split **3× v0.6.2 / 2× v0.6.1** against latest local tag
**v0.6.2** (HEAD 4f10347, 2026-11-15). The split is MVS fallout, not drift
by decision: `cmdguard/v5@v5.0.0` requires v0.6.1 in all five graphs;
`samber-do-auditlog@v0.12.3` requires v0.6.2 in the three that resolve
newer (see §2). Nobody is *pinning* anything — the versions are selected.
The moment any tool adopts directly, it pins v0.6.2 and the question
dissolves. Nothing in the v0.6.x line is flagged partial on POSIX in the
library's own FEATURES audit (only the Windows rename path is
`PARTIALLY_FUNCTIONAL`, untested on real hardware — irrelevant to this
linux-CI fleet).

## 6. The fleet argument

Crash-safe file writes are a fix-once, benefit-everywhere utility — the
purest form of the owner's "collective improvements" thesis. The fleet's
tools write baselines, golden files, catalog exports, generated code, and —
via cqrs-upgrade — each project's *source*. Every one of those sites
currently re-implements (or skips) safety differently. The library is
already in five module graphs, so the marginal cost of the flagship fixes
is one import line each; the marginal cost of *not* fixing is that the next
torn baseline or half-written user file gets debugged per-project, forever.
This is the cheapest severe finding in this audit group.

## 7. Ranked opportunities

1. **Swap cqrs-lint's baseline writer** (high, ~30 min): replace
   `cmd/cqrs-lint/v4/baseline_file.go:100-125` with
   `atomicwrite.Write(path, data)`; promote the go.mod line to direct.
   Deletes 25 hand-maintained lines, gains fsync + lock.
2. **Make cqrs-upgrade crash-safe** (high): replace `osWriteFile`
   (`cmd/cqrs-upgrade/v4/util.go:12`) and the direct write
   (`rewrite_v5_surface.go:150`) with `atomicwrite.WriteWithPerm`/
   `Write` — user source files must never be torn. Requires adding the dep
   to `cmd/cqrs-upgrade/v4/go.mod` (cmd tool; dep budget fine).
3. **cqrs-gen: `Write` or `WriteIfChanged` for generated handlers**
   (medium): `cmd/cqrs-gen/v4/main.go:124`; dep already in this module's
   graph. `WriteIfChanged` also spares watch-mode rebuilds (see
   `go-filewatcher.md`) from no-op rewrites.
4. **`WriteIfChanged` for committed baselines/goldens** (medium):
   `catalog/v4/compat/baseline.go:94`, `cmd/api-stability/v4/golden.go:29`,
   catalog manifests — mtime stability only; record the catalog/v4 dep
   decision explicitly either way (gap 3).
5. **Leave `metaengine/v4/plan_diff.go:186` bare, with a comment** (low):
   diagnostic artifact inside a budget-constrained library module; one
   comment makes that a decision instead of an oversight.

## 8. Reconsider when

- Any tool gains a watch/daemon mode (concurrent writers become real;
  rows 5–6 flip) — cqrs-gen watch is the likeliest first mover.
- CI runs tool updates in parallel on a shared checkout (flock flips from
  N/A to required overnight).
- cqrs-upgrade grows beyond source rewrites (lock files, manifests —
  `WriteVerified` becomes the right tool for read-modify-write migrations).
- The Windows fleet-owning developer appears (row 7 flips; until then the
  lib's Windows caveat is moot).

## 9. How to verify

```
rg -n 'larsartmann/go-atomic-write' --type go /home/lars/projects/go-cqrs-lite   # zero imports
rg -n 'go-atomic-write' -g go.mod /home/lars/projects/go-cqrs-lite               # the 5 indirect pins
(cd /home/lars/projects/go-cqrs-lite/cmd/doc-check/v4 && go mod graph | grep ' go-atomic-write@' | grep -v cqrs-lite)
sed -n '100,125p' /home/lars/projects/go-cqrs-lite/cmd/cqrs-lint/v4/baseline_file.go
sed -n '10,14p'  /home/lars/projects/go-cqrs-lite/cmd/cqrs-upgrade/v4/util.go
sed -n '148,152p;197,201p' /home/lars/projects/go-cqrs-lite/cmd/cqrs-upgrade/v4/rewrite_v5_surface.go /home/lars/projects/go-cqrs-lite/cmd/cqrs-upgrade/v4/rewrite_v5.go
sed -n '124p'    /home/lars/projects/go-cqrs-lite/cmd/cqrs-gen/v4/main.go
sed -n '186,190p' /home/lars/projects/go-cqrs-lite/metaengine/v4/plan_diff.go
sed -n '16,50p'  /home/lars/projects/go-atomic-write/README.md
head -40 /home/lars/projects/go-atomic-write/FEATURES.md
git -C /home/lars/projects/go-atomic-write log -1 --format='%H %cs'   # 4f10347 2026-11-15
git -C /home/lars/projects/go-atomic-write tag --list | sort -V | tail -1  # v0.6.2
```

Related: `go-filewatcher.md` (the read side of file lifecycles) and
`go-health.md`, both in this directory.
