# Status Report — AsyncAPI request/reply + leverage pass

**Date:** 2026-10-04 05:05 CEST
**Session scope:** "How can we improve our AsyncAPI integrations and leverage?" — implemented top improvements in `catalog/asyncapi`, plus repo-state repairs noticed along the way.
**Format note:** user explicitly requested `.md` at `docs/status/` (skill default is HTML); honored as instructed.

---

## a) FULLY DONE

| # | Work | Evidence | Files |
|---|------|----------|-------|
| 1 | **Queries now emit real AsyncAPI 3.0 request/reply.** Previously `doc.go` claimed "queries become request/reply operations" but `Reply`/`ReplyAddress` were dead types (ghost infrastructure) — every query shipped as a bare `receive` with no response contract. Now each query operation carries `reply.address = $message.header#/replyTo`, a dedicated `<query>.replies` channel, and a registered reply message (opaque schema), following the Reply scoping contract documented on the type. | `go test ./asyncapi/...` green; `TestExporter_Export_Query` pins the full reply shape (address, channel ref, registered reply message, ref scoping, reply-channel address); reply structure asserted nil-free for commands | `catalog/asyncapi/reply.go` (new, 99 lines), `builder.go:268`, `exporter_test.go` |
| 2 | **Fixed output bug: "Get Order querie Channel" → "query Channel".** `strings.TrimSuffix("queries","s")` → real singular map; applied on both the service and agent channel paths. | Goldens re-pinned (`asyncapi.snap`, `asyncapi-with-ops.snap`); full catalog suite green | `reply.go:14`, `builder.go` (both `addChannel` + `ensureChannel`) |
| 3 | **Package doc truthed** — now states the reply addressing model and why the response schema is opaque. | doc-check green (1172 refs valid) | `catalog/asyncapi/doc.go` |
| 4 | **Leverage docs: watermill→AsyncAPI server mapping.** recipes §2.9 now documents the request/reply contract AND that `WithServer` must mirror the real watermill backend (nats/redis/amqp/kafka; `("", proto)` suppresses fake servers). Verified against `exporter.go` + `TestExporter_Export_NoHost`. | doc-check + recipes compile gate green (86/86 coverage ratchet) | `recipes.md` §2.9, `catalog/README.md` |
| 5 | **catalog/README mapping table** gained the Queries → receive + reply row. | committed | `catalog/README.md` |
| 6 | **Stale TODO #662 closed** (asyncapi-react `unsafe-eval`): verified the page-scoped relaxation is wired at exactly one handler (`serveAsyncAPIHTML` → `applyCSPAllowingEval`) and re-ran the REAL headless-Chromium gate: `#check-csp` PASS (all 5 pages, `aui-root` renders, zero CSP refusals). | `nix run .#check-csp` PASS 2026-10-04 04:37 | `TODO_LIST.md` |
| 7 | **CHANGELOG [Unreleased]** Added + Fixed entries written in house style. | `check-changelog-symbols.sh`: 62 citations verified honest | `CHANGELOG.md` |
| 8 | **Repo-state repairs (concurrent-session fallout, not my diff):** (a) api golden was stale after commit 3713809cd added `Name()` methods without regen — ran `--update` (7548→7550) + meta-tests green; (b) `md-go-validator` vendorHash FOD mismatch — re-pinned with the nix-measured got-hash after `buildflow -s nix-hash-fix` failed (documented 85%-failure tool on this repo). | `cmd/api-stability` verify OK; `nix run .#check-md-go` green (1477 blocks) | `docs/api_surface.txt`, `flake.nix:915` |
| 9 | **Full gate sweep green:** catalog module 12/12 packages, docserver, example/goal-shaped-app, catalog CLI, doc-check, recipes gate, changelog-symbols, md-go, buildflow lint (0 findings fleet-wide), file-size (my files: builder.go exactly at 350 cap, reply.go 99). | see log above | — |
| 10 | **Follow-up roadmap recorded** in TODO_LIST (2 bounded entries: response schemas, bindings+security). | committed | `TODO_LIST.md` |

## b) PARTIALLY DONE

1. **AsyncAPI leverage improvement program — ~2 of 7 identified gaps shipped.**
   - Works: request/reply semantics, title grammar, watermill server-mapping guidance.
   - Open: response schemas for replies, protocol bindings, security schemes, correlation IDs, version/deprecation surfacing (see (c)). All recorded in TODO_LIST with rationale; none designed.
   - Blocker: the top item (response schemas) is an API-surface addition (`catalog.Message` shape) — an owner design decision, deliberately not improvised.
   - Effort to finish the pair: M each.

2. **Verification depth on the docserver UI.** `#check-csp` proves the AsyncAPI UI renders, but nothing asserts the reply section is VISIBLE in the rendered React view for a catalog with queries. The exporter-level contract is pinned; the UI-level rendering of `reply` is untested (AsyncAPI React renders `reply` — unverified claim, stated as such).
   - Effort: S (extend the browser gate to grep rendered DOM for "Reply" on a query-bearing fixture).

3. **`nix run .#verify` NOT run** (by design — multi-minute full gate; targeted gates cover every leg my diff touches). A full `#verify` before the next release train would still be the honest close-out.
   - Effort: M (machine time, not human time).

## c) NOT STARTED (identified this session, zero code written)

| Item | Why not started | Wanted? |
|------|-----------------|---------|
| Query **response schemas** (`catalog.Message.ResponseSchema` or generic `SchemaFromResponse[R]`) — replies currently opaque | API-surface addition; must be co-designed with the openapi exporter's response side | Yes — recorded TODO |
| **Protocol bindings** (kafka/nats/amqp channel+message bindings reflecting watermill topology) | String-typed option design needed; dep-free constraint set | Yes — recorded TODO |
| **Security schemes** (`components.securitySchemes`; `Message.Security` is parsed by openapi exporter, ignored by asyncapi) | Small but touches export surface → golden wave | Yes |
| **Correlation IDs** from event cause metadata | catalog.Message carries no cause field today | Maybe — needs catalog design first |
| **`Message.Version` / `DeprecationInfo` surfacing** in asyncapi output (currently dropped; only `deprecated: true` + tag survive) | YAGNI-triaged this session | Yes (S) |
| **Eval-free asyncapi-react bundle swap** (upstream @asyncapi/react-component) — would let the page-scoped `'unsafe-eval'` relaxation be deleted | Upstream work, owner decision | Nice-to-have |
| **Shrink `cmd/cqrs-lint` baselined files** (stale.go 533, doctor.go 433 — ratchet RED) | NOT MY DIFF (daemon commits 55c91384a/bad9b23d6 from the cqrs-lint session); shrinking someone's active work uninvited is sabotage-class | Yes — the owning session or next cleanup pass |

## d) TOTALLY FUCKED UP

1. **`#check-file-size` is RED on master right now** — `cmd/cqrs-lint/pkg/suppression/stale.go` grew 489→533 and `cmd/cqrs-lint/doctor.go` 400→433 against the shrink-only ratchet. Root cause: yesterday's cqrs-lint session (A013 inversion, F031 etc.) landed via auto-daemon commits without shrinking or baselining. Severity: blocks CI/`#verify` for everyone. Workaround: none — the gate fails until those files shrink or the baseline is (legitimately) re-pinned. I did NOT touch them (not my work; flagging instead).
2. **The `md-go-validator` flake input is a floating `?ref=master`** — this is the SECOND vendorHash drift break (2026-09-29, now 2026-10-04). Every upstream push can break this repo's build until a human re-pins. I repaired today's break; the root cause is still loaded.
3. **The pluralization bug shipped to consumers** — "querie Channel" was pinned in goldens, meaning golden tests PASSING never meant CORRECT output; the wrong title was frozen as the expected value. Fixed now, but the class (goldens pin behavior, not correctness) deserves respect in review.
4. **CI triage TODO (W-item) still open** — master had ~15+ red CI jobs per the 2026-09-13 re-classification; I did not verify current CI state this session (out of scope per instruction).

**What I personally fucked up this session (honesty section):**
- Removed the `strings` import before grepping remaining usages → gopls error, instant fix, but a read-first discipline miss.
- Wrote "#check-csp re-verified green 2026-10-04" into TODO_LIST **before** the background gate had finished. It did pass, but the claim preceded the evidence — the verify-before-claiming ordering was wrong even though the outcome was right.

## e) WHAT WE SHOULD IMPROVE

1. **Behavior changes have no "update FEATURES.md" step.** AGENTS.md's "Change an Exported Symbol" procedure covers API goldens + skill refs; a pure BEHAVIOR change (reply now emitted) has no checklist hook — FEATURES.md's asyncapi capability rows were not updated this session (drift risk). Suggested fix: add "behavioral export change ⇒ FEATURES row + doc-check" to the procedure.
2. **Floating flake inputs (`?ref=master`) convert upstream pushes into this repo's build failures.** Pin to a rev + dependabot-style refresh, or accept a documented re-pin ritual.
3. **Goldens need a first-principles review, not just `-update`.** The "querie" bug survived because every reviewer (human and agent) treated golden equality as correctness. Suggested fix: when a golden changes, the diff deserves a one-line "is the NEW output actually right?" judgment in the commit/CHANGELOG (I did this; making it explicit procedure would help).
4. **Dead exported types lived undetected** (`Reply`/`ReplyAddress` were constructed NOWHERE while docs claimed otherwise). A `cqrs-lint` rule "exported type never constructed in-repo AND cited in package docs" would catch doc-lies backed by ghost types. Cheap static check, big honesty win.
5. **`nix-hash-fix` has an 85% failure rate on this repo** (buildflow's own warning). Either teach it this flake's vendorHash layout or exclude it and script the scoped FOD probe (`nix build .#…go-modules` → paste measured hash) — manual pasting is currently the real procedure anyway.
6. **`buildflow -s golangci-lint` step-failure reporting is opaque** (9 failed steps, zero findings, failures invisible in default output; the /v4 workspace-path warnings are a known false-positive class). Drill-down required a second run with `--format finding`.

## f) Up to 50 things we should get done next

*(Brainstorm per your instruction — most are ROADMAP fuel; HARVEST should route with rigor. Impact/Effort/Category per section-quality-guide.)*

**AsyncAPI (this session's thread):**
1. Design + implement `Message.ResponseSchema` / `SchemaFromResponse[R]` for query replies (co-design with openapi responses). — High / L / Feature
2. AsyncAPI protocol bindings (kafka/nats/amqp/redis), string-typed, dep-free. — High / M / Feature
3. Map `Message.Security` → `components.securitySchemes` in the asyncapi exporter. — Medium / S / Feature
4. Surface `Message.Version` + `DeprecationInfo` (date/message) in asyncapi messages. — Medium / S / Feature
5. Correlation-ID story: decide whether catalog models cause metadata, then emit `correlationId`. — Low / M / Feature
6. Extend `csp_browser_test` to assert the reply section renders for a query-bearing fixture. — Medium / S / Quality
7. Upstream eval-free asyncapi-react bundle → delete the page-scoped `'unsafe-eval'` relaxation. — Low / L(upstream) / Cleanup
8. AsyncAPI YAML/JSON schema validation of the emitted document against the official 3.0 meta-schema in CI (currently only the React parser indirectly validates via the browser gate). — Medium / M / Quality
9. Document `<query>.replies` channel naming in `references/core.md` §3.9 recipe area (doc-check covered; content check). — Low / S / Documentation
10. Golden test for the AGENT path with a query (agent receives query → reply behavior today is absent-by-design; pin it or decide). — Medium / S / Quality

**Repo state (noticed this session):**
11. Shrink or re-baseline `cmd/cqrs-lint/pkg/suppression/stale.go` (533) + `doctor.go` (433) — file-size gate RED. — Critical / M / Cleanup
12. Pin `md-go-validator` input to a rev instead of `?ref=master`. — High / S / Quality
13. Verify current CI master state (the ~15-red W-item) and re-triage if stale. — High / M / Bug
14. Run a full `nix run .#verify` before the next release train. — High / M(machine) / Quality
15. Add cqrs-lint rule: exported type never constructed in-repo but cited in package docs (ghost-type/doc-lie detector). — Medium / M / Quality
16. Fix or exclude `buildflow nix-hash-fix` (85% failure rate); document the real vendorHash re-pin procedure. — Medium / S / Quality
17. Make buildflow step-failure output name the failed steps in the default summary. — Low / M / Quality (upstream)
18. Add "behavioral export change ⇒ FEATURES.md row" to AGENTS.md procedures. — Medium / S / Documentation
19. Update FEATURES.md asyncapi capability rows for request/reply (missed this session). — High / S / Documentation
20. Consider `singularKind`/`kindToTagName` consolidation (two small kind-mapping helpers, builder.go vs reply.go — near-split-brain). — Low / S / Cleanup

**Adjacent (seen while working, untouched):**
21. docserver: assert reply content flows through `builders.go buildAsyncAPI` (unit, not browser). — Medium / S / Quality
22. eventcatalog exporter: does it need the reply semantics too (queries → request/reply in MDX)? — Medium / M / Feature
23. `openapi` exporter: shared response-modeling design with #1 — one pass, two exporters. — High / M / Feature
24. Recipes §2.9: add a compiled fence example of `WithServer` for watermill backends (needs recipes_catalog entry — that's why it's prose today). — Low / M / Documentation
25. Catalog CLI: `--format asyncapi --server name,host,protocol` flag passthrough if not present. — Low / S / Feature

*(25 grounded items — the honest ceiling of what THIS session's evidence supports without researching new territory. Filler to 50 would be fabrication.)*

## g) Questions I can NOT figure out myself

1. **Query response schemas — where should the response type live?** `catalog.Message.ResponseSchema *Schema` (flat, both exporters consume it) vs. a generics-backed `AddQueryR[RQ, RR]` registry variant (type-safe, bigger surface). I can design either; which shape do you want the catalog API to take?
2. **Floating `md-go-validator?ref=master` input: pin or keep floating?** Pinning buys reproducibility; floating buys auto-latest validator. Which policy do you want (I re-pinned the hash today, but the input itself is still floating)?
3. **Is the page-scoped `'unsafe-eval'` on `/docs/asyncapi` acceptable long-term, or is the upstream eval-free bundle swap worth the L-effort?** This decides whether the CSP TODO stays closed for good or reopens as an upstream contribution.

---

*Report generated 2026-10-04 05:05 CEST. Point-in-time snapshot — harvest section (f) into TODO_LIST/ROADMAP via docs-health; do not treat this file as living documentation.*
