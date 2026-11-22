# go-filewatcher — Utilization Audit · go-cqrs-lite

> Audited 2026-11-22 · Library `github.com/larsartmann/go-filewatcher/v2` @ 26459ba (latest local tag v2.5.0) · Consumer go-cqrs-lite @ feec9bc74 · Group G3

**Verdict:** NOT-ADOPTED-JUSTIFIED
**Score:** 0/0 capabilities applicable = N/A (no band — no current need, and that is the honest answer)

Zero `require`s and zero hand-rolled watchers: there is nothing to audit as
duplication, because nothing in go-cqrs-lite watches files. Every cmd/ tool
is one-shot (`cqrs-gen` generates and exits; `doc-check` validates and
exits), docserver serves pre-built docs without hot reload, and the only
`fsnotify` in the module graph is indirect (via `a-h/templ`, koanf's file
provider, and `samber-do-auditlog`) with no consumer import of it. This is
the rare report where "not adopted" is simply correct today — the value is
in pre-committing the flip decision, because the first `--watch` flag anyone
adds (codegen watch is the canonical one) is exactly the moment a fleet
either adopts the shared watcher or starts re-implementing recursion,
debounce, and ENOSPC handling per tool.

## 1. What the library is

A high-performance, composable filesystem watcher built on fsnotify that
"eliminates the boilerplate of raw fsnotify": automatic recursion, 20
built-in filters with AND/OR/NOT composition, 19 middleware, global and
per-path debouncing, `.gitignore`-aware filtering, ENOSPC resilience,
NFS/FUSE polling fallback, self-healing (re-adding watches after
removals/renames), and Prometheus + OTel observability hooks
(go-filewatcher/README.md:13-45). Usage is `New(paths, opts...)` →
`Watch(ctx)` → read-only event channel carrying path, op, timestamp, size,
modtime, and optional content hash (go-filewatcher/README.md:47-52,69+).
Module path is `/v2`. Requires Go 1.26.7+ and, on Go ≤1.26,
`GOEXPERIMENT=jsonv2` (go-filewatcher/README.md:54-68) — a non-issue for
this consumer, whose modules are on go 1.27.2 where `encoding/json/v2` is
stable stdlib. HEAD 26459ba (2026-11-17); v2.5.0.

## 2. Usage census in go-cqrs-lite (evidence)

- `rg -n 'filewatcher' -g go.mod` over the repo: **zero hits**; no imports.
- `rg -n 'fsnotify|Watch\(' --type go`: the only `Watch(` hits are
  metaengine's reactive query watchers — a dataflow concept, not filesystem
  watching (`metaengine/v4/sse.go:161`, `metaengine/v4/dx.go:87,150`). No
  `fsnotify.` import exists in any non-test file.
- fsnotify v1.10.1 appears as `// indirect` in several go.mod
  (`system/v4/go.mod:44`, `systemtest/v4/go.mod:43`,
  `systemcatalog/v4/go.mod:21`, `benchkit/v4/go.mod:58`) — `go mod graph`
  traces it to `a-h/templ@v0.3.1100` (templ's own generate-watch),
  `knadh/koanf/providers/file`, and `samber-do-auditlog`. No consumer code
  uses it.
- Candidate surfaces that *could* watch, and their current shape:
  - `cmd/cqrs-gen/v4/` is a one-shot generator (`main.go`, `generate.go`,
    `scan.go` — no watch flag, no loop).
  - `catalog/v4/docserver/` serves pre-built architecture docs
    (`catalog/v4/docserver/architecture.go:58` shells out to a `docserver
    <docsPath>/architecture` command); no reload logic, no polling.
  - `cmd/doc-check/v4/` walks and validates docs on demand.

## 3. Capability inventory

| # | Capability | Grade | Evidence / reason |
|---|---|---|---|
| 1 | Recursive fsnotify watching (`New`+`Watch`) | N/A | no tool in the repo watches files — flip: `cqrs-gen --watch` |
| 2 | Debounce/coalesce (editor 5-events-per-save) | N/A | no watch loop to debounce — flip with row 1 |
| 3 | Filters incl. `.gitignore`-aware (vendor/node_modules) | N/A | no watcher — and the repo's ignore needs would be real on day one (vendor dirs, `docs/` build output) |
| 4 | Resilience: ENOSPC, NFS/FUSE poll fallback, self-heal | N/A | no watcher — these are the parts hand-rolls omit and then hit in production |
| 5 | Observability hooks (Prometheus, OTel, health) | N/A | no watcher; ties into the go-health seam when one exists (`go-health.md`, this directory) |
| 6 | Event content-hash delivery (dedupe no-op writes) | N/A | no watcher; pairs naturally with go-atomic-write's `WriteIfChanged` on the write side |

Score derivation: rows N=6, N/A rows 1–6 → **A=0**. Score is not computable
(0/0); the honest verdict is **N/A — no applicable capability today**, which
the template's band table deliberately tolerates when justified. The
justification is the usage census above: zero watchers, zero hand-rolls,
zero candidates in current scope.

## 4. Gap analysis

1. **There is no gap today — and that is the finding.** Unlike go-health
   (a lying `/health` endpoint) or go-daemon (a hand-rolled drain with a
   wart), no consumer surface watches files badly or at all. Forcing
   adoption would be dependency-driven development.
2. **The flip is foreseeable and should be pre-decided.** The fleet's
   likeliest first watcher is `cqrs-gen --watch` (regenerate projections on
   save — the canonical Go codegen DX feature, and templ already ships its
   own watcher alongside: `a-h/templ@v0.3.1100 → fsnotify@v1.10.1` in the
   graph). Second likeliest: docserver live preview. Both are feature
   decisions, not integration debts; this report's job is to make the
   library the *default answer* for them so the implementation isn't
   re-argued per tool.
3. **The transitive cost is lower than it looks.** fsnotify is already in
   the module graph via templ/koanf/auditlog; go-filewatcher adds its own
   logic on a foundation the consumer already carries. The Go-1.26
   `GOEXPERIMENT=jsonv2` caveat does not apply on the consumer's go 1.27.2
   toolchain.

## 5. Version currency

No pin (zero requires). Latest local tag **v2.5.0**; HEAD 26459ba
(2026-11-17). The library carries an `API_STABILITY.md`, `MIGRATION.md`,
and a release-please config (go-filewatcher repo root), i.e. an explicit
compatibility policy — good signs for a future adopter. Any adoption would
target `cmd/` tools or `example/` apps only; no production library module
should grow a watcher dependency (per-module dep budgets,
`nix run .#check-arch`).

## 6. The fleet argument

File watching is the archetypal "everyone re-implements it, every
implementation is subtly broken" problem: recursion, editor-event storms,
inotify limits (ENOSPC), NFS silence, watch loss on rename. The library
exists so the fleet fixes those once. go-cqrs-lite needs none of it today —
but the owner's fleet lens says the decision that matters is the *next*
one: the first `--watch` flag anywhere in the fleet should start from
`filewatcher.New(paths, WithExtensions(".go"), WithDebounce(...))`, not
from a fresh `fsnotify.NewWatcher()` and a hand-rolled debounce map. A
one-line note in the fleet's tooling docs converts this from a per-tool
debate into a settled default.

## 7. Ranked opportunities

1. **Record the default** (now, zero code): add one line to the repo's
   tooling/contribution docs — "any watch mode uses go-filewatcher/v2, never
   raw fsnotify" — so the flip decision is pre-made.
2. **`cqrs-gen --watch` when wanted** (medium, feature work): debounce
   200–500ms, `WithExtensions(".go")`, `WithIgnoreDirs("vendor")`, gitignore
   filter; regenerate on coalesced change. The feature is the deliverable;
   the library makes it ~40 lines instead of a week of edge cases.
3. **docserver live preview** (low, feature work): watch the generated docs
   tree, re-serve on change; pairs with the write side being atomic
   (`go-atomic-write.md`, this directory) so readers never see torn files.
4. **Health feed** (low, contingent): when a watcher lands and the go-health
   seam from `go-health.md` exists, surface watcher self-heal events as a
   non-critical check.

## 8. Reconsider when

- `cqrs-gen` gains a watch/regenerate mode (row 1 flips to required).
- docserver or any docs tool gains live preview / re-run-on-change.
- Any fleet app adds config hot-reload — note koanf's file provider (already
  in the graph via a dep chain) covers simple config reload; go-filewatcher
  wins once the watched set is code/docs rather than a single config file.
- Someone proposes importing `fsnotify` directly anywhere in the fleet —
  that PR is the trigger; it should be redirected to this library.

## 9. How to verify

```
rg -n 'filewatcher' -g go.mod /home/lars/projects/go-cqrs-lite          # zero hits
rg -n 'fsnotify' --type go /home/lars/projects/go-cqrs-lite             # no direct imports
rg -n 'Watch\(' --type go /home/lars/projects/go-cqrs-lite/metaengine/v4 | head   # query watchers, not files
ls /home/lars/projects/go-cqrs-lite/cmd/cqrs-gen/v4                     # one-shot: main/generate/scan
rg -n -i 'watch|reload' /home/lars/projects/go-cqrs-lite/catalog/v4/docserver --type go  # zero
rg -n 'fsnotify' -g go.mod /home/lars/projects/go-cqrs-lite             # indirect only
(cd /home/lars/projects/go-cqrs-lite/system/v4 && go mod graph | grep fsnotify | grep -v cqrs-lite | head)
sed -n '13,52p;54,68p' /home/lars/projects/go-filewatcher/README.md
git -C /home/lars/projects/go-filewatcher log -1 --format='%H %cs'   # 26459ba 2026-11-17
git -C /home/lars/projects/go-filewatcher tag --list | sort -V | tail -1  # v2.5.0
```

Related: `go-atomic-write.md` (the write side of the same file lifecycle)
and `go-health.md`, both in this directory.
