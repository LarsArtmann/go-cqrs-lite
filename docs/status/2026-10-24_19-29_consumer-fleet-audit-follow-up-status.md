# Consumer Fleet Audit Follow-Up — Session Status (2026-10-24 19:29)

> Point-in-time snapshot. One session: read the five 2026-10-24 consumer deep-dive
> audits (nsfw-classifier, CV, dnsblockd, pbx-lens, InboxClean) of go-cqrs-lite's
> catalog/bench/testing modules, verified their claims against source, fixed what
> was real, harvested what needs owner decisions.

**Session window:** ~18:15–19:29 CEST · **Base:** master `ec2a91121` → HEAD `4bb980592` (daemon-absorbed) · **Commits touching this work:** `fa6169a41`, `372bf716e`, `d021f2b65`(parallel), `2504e78e6`, `1c1d5cead`, `beea54d1c`, `d762458a9`, `4bb980592`

---

## a) FULLY DONE

| # | Item | Evidence |
| - | ---- | -------- |
| 1 | **Read all 10 audit docs** (5 research + 5 status) from nsfw-classifier, CV, dnsblockd, pbx-lens, InboxClean; extracted every library-facing finding, recommendation, and open question | agent extractions, verbatim quotes preserved in-session |
| 2 | **pgengine compile error fixed** — `typed_wal.go` passed an anonymous `interface{ SQLState() string }` to the generic `AsType` (type argument must satisfy `error`); replaced with named `error`-embedding `sqlStateCarrier`; new white-box test `typed_wal_internal_test.go` (7 cases: driver SQLSTATE, wrapped, message fallback, negative) | `metaengine/pgengine/typed_wal.go:189` · build+vet+test green · commit `2504e78e6` |
| 3 | **versions.json regenerated** (112 trains) — was stale since 10-17 (systemscenario v4.2.0 vs tag v4.2.1; pbx-lens called it "a lie") | `bash scripts/check-versions-manifest.sh --update` → `--check` exit 0 |
| 4 | **batch-release.sh now refreshes the versions manifest** — root cause of the staleness: batch waves tag without running tag-release.sh, so the manifest trailed every wave | `scripts/batch-release.sh` (guard + `--update` after the tag loop) · `nix run .#check-release-scripts` all batch legs PASS |
| 5 | **cqrs-lint `explain` accepts rule IDs** — `explain E004` used to fail with "Unknown command" (nsfw-classifier's blocked finding, confirmed live); now prints a full catalog card (ID, name, category, severity, confidence, auto-fix, DocURL, description) and fails loudly on unknown IDs pointing at `cqrs-lint rules`; no-arg mode unchanged; root help + README updated; moved to `explain_rules.go` to respect explain.go's 542-line baseline (now 501) | `cmd/cqrs-lint/explain.go`, `explain_rules.go`, tests added · full cqrs-lint suite `ok` |
| 6 | **systemscenario hollow-root warning** — CV's explicit upstream ask (§f-46/§f-49): `System` with an empty `DomainConfig` logs a WARNING naming `Adopt` as the fix; advisory `Logf` by design; `isEmptyDomain` checks all 7 wiring slots, flags only | `systemscenario/empty_domain.go` + test (warn + 6 silent variants PASS) · full module suite green · commit `1c1d5cead` |
| 7 | **benchkit README de-lied** — flagged stale by 4 of 5 audits, confirmed (led with `Factory`+`stack.Bundle`, "the deprecated v1 composition root"); rewritten: `FactoryFromSystem` quick-start first, preset path marked legacy/removed-v5; `Run`/`Compare` signatures verified against source first | `benchkit/README.md` · `run.go:26,49` |
| 8 | **SKILL.md + faq.md updated** — SKILL.md: benchkit library-lane guidance (FactoryFromSystem vs deprecated Factory, EXPERIMENTAL status) + `Adopt`/hollow-root warning in the testing paragraph; faq.md: two new entries ("My systemscenario assertions all pass — but do they test anything?" and "How do I check which version of a module I am actually on?" — per-module tags make merge-base ancestry meaningless, a burned-round lesson from nsfw-classifier) | doc-check green: "All 1356 references valid across 57 package(s)" |
| 9 | **FEATURES.md cross-links** — scenario ↔ systemscenario mutual references (CV §f-48) + empty-domain warning row in the systemscenario table | `FEATURES.md:406,411,1298` |
| 10 | **md-go-validator vendorHash re-pinned** (6th recurrence of the documented class) — the flake input tracks master, moved again (pprof 10-16 pseudo-version); fake-hash dance → pinned `sha256-0B+mcM0vCcy…`; gate green: "All 1519 code blocks valid" | `flake.nix` md-go-validator package · `nix run .#check-md-go` exit 0 |
| 11 | **6 TODO_LIST rows harvested** from the audits (before this report): catalog Registry-from-DomainConfig derivation, `scenario.Given` version-carrying decides, cqrs-lint 2.0s→7.4s timing regression, fleet case-study invites, MarkDemo untagged-API incident, GOWORK=off example-build gate ask | `TODO_LIST.md` two new sections |
| 12 | **CHANGELOG [Unreleased]** rows: 2 Added (systemscenario warning, explain rule IDs) + 3 Fixed (pgengine, versions.json/batch-release, benchkit README) — with the §d-1 correction applied before this report | `bash scripts/check-changelog-symbols.sh` → "citations are honest" |
| 13 | **Formatter + config gates** — `nix fmt` (also healed 7 pre-existing unformatted files), `check-lint-config` green, `check-nix-hashes` tripwire green, `check-versions-manifest --check` green | per-gate receipts above |

## b) PARTIALLY DONE

| # | Item | State |
| - | ---- | ----- |
| 1 | **MarkDemo incident triage** — root-caused and TODO-filed, NOT healed: `Registry.MarkDemo` landed in untagged daemon commit `5952a4562`; `example/mesh-demo` pins `catalog/v4.10.0` and calls it → `GOWORK=off go build` fails against the proxy. Healing requires a catalog tag (release = owner call). Workspace build green (`MESH_DEMO_WORKSPACE_OK`) | TODO_LIST § "Untagged-API incident" |
| 2 | **explain rule cards** — cover catalog metadata (what/severity/confidence), not the full when-it-fires rationale (that lives in rule source + RULES.md); no pointer from the card to RULES.md yet | card footer points at `cqrs-lint rules` + `explain` only |
| 3 | **Per-module test sweep over the changed set** — touched modules (pgengine targeted, systemscenario, cqrs-lint) ran green; the AGENTS.md pre-wave ritual ("plain sweep over all modules, ~10 min warm") NOT run — fine for a non-release session, required before any tag | — |
| 4 | **Pool index** — this report gets its `docs/status/README.md` row right after writing (not yet done at snapshot time) | — |
| 5 | **Versions-manifest nightly coverage** — `nightly-gates.yml` runs `--check --remote`, yet the manifest sat stale for ~a week; whether the nightly has run/fired since 10-17 is UNVERIFIED | `.github/workflows/nightly-gates.yml:133` |
| 6 | **Auditor feedback loop** — the five consumer projects each left §f HARVEST input and §g questions of their own; only go-cqrs-lite-side items were harvested here; replying to their open questions is untouched | their docs, not this repo's |

## c) NOT STARTED

| # | Item |
| - | ---- |
| 1 | `nix run .#verify` composite run (per-gate battery only; #verify-exclusivity respected, composite proof missing) |
| 2 | api-stability golden mechanical re-check (no exported-symbol change → regen skipped per procedure, but never machine-verified) |
| 3 | mysqlengine twin test — `typed_wal.go:240` uses `AsType[*mysql.MySQLError]` (correct form, builds), but has no unit test mirroring pgengine's new one |
| 4 | AGENTS.md gotcha row for the untagged-API-in-examples class (the MarkDemo lesson lives only in TODO_LIST) |
| 5 | `check-error-taxonomy` + `check-duplication` runs after the session's Go edits (no new codes/clones expected; unverified) |
| 6 | recipes.md §2.43 mention of the hollow-root warning (faq.md covers it; the recipe doesn't) |
| 7 | Empty-domain warning opt-out decision (deliberately empty boots now always log) |
| 8 | benchkit README "Testing.B integration" section still shows only the stack-preset `RunSuite` shape |
| 9 | Static cqrs-lint detection of empty-DomainConfig test boots (runtime warning shipped; linter complement not designed) |
| 10 | Answering the five auditors' own §g questions / notifying them of the upstreamed fixes |

## d) TOTALLY FUCKED UP

| # | What | Detail |
| - | ---- | ------ |
| 1 | **Shipped an overclaim, twice** — the CHANGELOG Fixed row and my session summary said the pgengine bug "failed to build for every consumer." FALSE: the broken line entered master in untagged daemon commit `d021f2b65` **today 17:12** and was fixed at 18:50 (`2504e78e6`); **no pgengine tag contains it** — tag-pinned consumers were never broken; only local-checkout riders (which is how three projects' gopls saw it). Corrected in-tree before this report, but the wrong claim left this machine in chat first — precisely the §d failure class the 12-00 report already self-reviewed ("fleet-wide rejection claim"). Blast-radius checks (tag-vs-checkout) must precede severity words like "every". |
| 2 | **A red compile diagnostic sat untriaged for the session's first half** — gopls flagged `typed_wal.go:197` in my very first tool output; I deprioritized it until the consumer audits independently surfaced it. A broken build in the project under edit should be triage #1, not a side quest. |
| 3 | **master's file-size ratchet is red (pre-existing, unfixed)** — `scheduling/sqlstore/store.go` 382, `system/adapter_event.go` 351, `metaengine/layout.go` 360 lines, all at HEAD, none baselined: `#verify` has not run since they grew. I verified all three pre-date my session and left them (unrelated-file rule), but "red on master" is red. |
| 4 | **The MarkDemo class shipped through every gate** — an untagged API used by a pinned example, invisible to CI (workspace builds), to `check-example-standalone.sh` (audits replace-directives, not API resolution), and to this session until a chance build sweep. The fleet audits' "verify at the proxy tag, not the checkout" warning was prophetic in-repo. |
| 5 | **No authored commits** — the daemon absorbed all work as `chore: auto-commit` rows; the three consumer audits independently complained about the same attribution loss. Nothing here is revertible-hostile, but history quality is zero. |

## e) WHAT WE SHOULD IMPROVE

1. **Triage order: red build diagnostics before external reading.** The compile error was visible at session start; fixing it first would have cost 15 minutes instead of surfacing mid-session.
2. **Severity words require blast-radius receipts.** "Every consumer" needed a `git tag --contains` check before writing. Same discipline as verify-external-claims, pointed at my own claims.
3. **Gate the untagged-API class** — build `examplePaths` with `GOWORK=off` (TODO row filed). One leg would have caught MarkDemo at commit time.
4. **Pin the md-go-validator flake input to a rev** — six vendorHash re-pins (09-29, 10-04, 10-11, 10-13, 10-17, 10-24) is a standing tax; master-tracking is the root cause.
5. **Per-gate batteries are not #verify.** This session ran ~9 gates green and still shipped next to a red composite (file-size). Run the chain before calling a tree healthy.
6. **Daemon attribution** — commit authored work at phase boundaries if history matters (three audits' worth of evidence this annoys consumers too).
7. **Warn-with-escape-hatch** — new advisory logs should ship with their opt-out story in the same edit, or the first harness self-test that hates the noise will ad-hoc one.

## f) TOP THINGS TO GET DONE NEXT (28; 🅷 = harvested to TODO_LIST already)

| # | Task | Why / source |
| - | ---- | ------------ |
| 1 | Cut the next catalog train (contains `Registry.MarkDemo`) and re-verify mesh-demo `GOWORK=off` | 🅷 unbuildable vs proxy today |
| 2 | Add the GOWORK=off `examplePaths` build leg to `check-example-standalone.sh` or the verify-ci matrix | 🅷 the gate that would have caught #1 |
| 3 | Split or baseline the 3 file-size offenders (`scheduling/sqlstore/store.go`, `system/adapter_event.go`, `metaengine/layout.go`) — un-reds the ratchet | §d-3, pre-existing |
| 4 | Run the full `nix run .#verify` chain on the current tree | §c-1 |
| 5 | Machine-check the api-stability golden (even with no exported change) | §c-2 |
| 6 | Add the mysqlengine unique-violation unit test (twin of pgengine's) | §c-3 |
| 7 | Investigate the cqrs-lint 2.0s→7.4s timing regression (InboxClean #29) | 🅷 |
| 8 | Design catalog Registry-from-DomainConfig derivation WITH the owner (dnsblockd's ~40-line-registry pain) | 🅷 |
| 9 | `scenario.Given` variant accepting version-carrying decide funcs (InboxClean #33) | 🅷 |
| 10 | Invite the five auditors to contribute fleet case studies (CV §f-50); answer their §g questions | 🅷 + §b-6 |
| 11 | Decide empty-domain warning opt-out (`WithAllowEmptyDomain`?) | §e-7 |
| 12 | Pin md-go-validator input to a rev (stop the 6× re-pin tax) | §e-4 |
| 13 | Verify why the nightly `check-versions-manifest --check --remote` didn't flag the stale manifest since 10-17 | §b-5 |
| 14 | AGENTS.md gotcha row: untagged-API-in-examples + tag-vs-checkout blast-radius rule | §c-4, §d-1 |
| 15 | Point explain rule cards at RULES.md / rule docs for when-it-fires rationale | §b-2 |
| 16 | Add FactoryFromSystem variant to the benchkit README RunSuite section | §c-8 |
| 17 | Mention the hollow-root warning in recipes.md §2.43 | §c-6 |
| 18 | Run `check-error-taxonomy` + `check-duplication` on the session's edits | §c-5 |
| 19 | Per-module test sweep over all modules touched since the last tag wave (pre-wave ritual) | §b-3 |
| 20 | Batch the typed_wal fix + versions.json + systemscenario warning into the next tag wave; proxy-smoke per tag | consumers only get fixes via tags |
| 21 | Re-title this session's daemon commits (or accept attribution loss deliberately) | §d-5 |
| 22 | cqrs-lint: static empty-DomainConfig detection as an advisory rule (E-series complement to the runtime warning) | §c-9 |
| 23 | FAQ: fold the version-probe entry into `cqrs-upgrade` docs (the tool automates the sweep half) | §b linkage |
| 24 | Nightly-gates: add `check-file-size` + `GOWORK=off` example builds to the nightly matrix so master-reds surface within a day | §d-3/§d-4 dwell time |
| 25 | docs/status/README.md index row for this report (doing immediately after write) | §b-4 |
| 26 | Consider `WithSuppressEmptyDomainWarning` naming review before anyone copies it (naming-review skill) | §e-7 |
| 27 | Sweep the other five audit-proposed upstream asks for owner verdicts: catalog C1 appetite (dnsblockd), docserver `WithSkipBootstrapFiles` CLI exposure (pbx-lens F5 was consumer-side), FakeStore no-journal twin decision (InboxClean) | leftover upstream asks |
| 28 | Re-run the five-audit reconciliation after the next wave: their version-currency tables all cited the stale manifest | closes the loop |

## g) THREE QUESTIONS I CANNOT FIGURE OUT MYSELF

1. **Tag now or batch?** The pgengine compile fix, versions.json refresh, batch-release manifest wiring, systemscenario warning, and explain rule-IDs are all on untagged master. Do you want a small patch/minor wave cut **now** (checkout-riders — half the fleet — get the pgengine fix immediately), or does everything ride the next scheduled wave? If a wave: include catalog (to heal mesh-demo, item f-1) or explicitly not?
2. **Warning escape hatch:** should the systemscenario empty-DomainConfig warning get an explicit opt-out `Option` (e.g. `WithAllowEmptyDomain`), or is unconditional `Logf` the intended final shape (self-tests eat the noise)?
3. **md-go-validator input policy:** pin the flake input to a rev (stability; manual bumps; kills the 6× vendorHash re-pin class) or keep tracking master and accept periodic re-pins?

---

*Report ends. Section (f) rows marked 🅷 are already in TODO_LIST.md; the rest await instruction. WAITING FOR INSTRUCTIONS.*
