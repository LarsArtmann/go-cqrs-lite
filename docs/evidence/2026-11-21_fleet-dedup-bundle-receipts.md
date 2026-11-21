# Fleet dedup bundle receipts (C26)

> **Status: SKELETON — fill only after the gated tranche (C18–C21, D28/D29) lands.**
> Created 2026-11-21 from the 01-28 session report §f-29 so the final receipt document's shape is
> fixed before execution; per-section placeholders below. Companion evidence:
> [the 2026-11-20 baseline + swap receipts + commit ledger](2026-11-20_fleet-idempotency-baseline.md).
> Receipt discipline: every number captured from EACH REPO'S OWN cwd via
> `scripts/fleet-dedup-receipt.sh` (the cwd-preset trap is fixed in cqrs-lint as of 2026-11-21, but
> per-repo-cwd remains the honest shape).

## 1. Bundle receipts (C18–C21)

| Repo | Bundle | Commit | Store shape after | Sweeper (cadence, first pass, stop) | Tests | F030/V007 |
| ---- | ------ | ------ | ----------------- | ----------------------------------- | ----- | --------- |

<!-- One row per gated bundle when it lands; cite the sweeper test by name. -->

## 2. Census diff (26.4)

<!-- bash scripts/census-fleet.sh; diff the v4-tree consumer counts against the ADR-0167 addendum
     numbers; the shim class should shrink to the sqlstore residue V007 owns. -->

## 3. Versions manifest (26.5)

<!-- bash scripts/check-versions-manifest.sh --check output. -->

## 4. Residue list (C12 input)

<!-- Remaining idempotency-family requires across the fleet after the bundles: expected = zero shim
     parents, zero kvstore, sqlstore only where V007's v5 window still owns them. Feeds the D30
     keep/downgrade call on the codemod. -->

2026-11-21 measured residue (9/9 receipts, per-repo cwd — full table in the baseline evidence §6b,
`docs/evidence/2026-11-20_fleet-idempotency-baseline.md`): parent-shim residue = Zlota44 only
(`idempotency/v4`, F030: 3); kvstore = zero; sqlstore = browser-history + precheck + KeyCountdown
(F030: 2/2/1) + DiscordSync (import present, no flagged sites); direct `go-idempotency` = InboxClean,
SEC, github-local-sync, SwettySwipperWeb.
