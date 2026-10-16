# Vendored erraudit source

This directory holds the pinned erraudit source snapshot that the CI
`error-audit` job (`.github/workflows/ci.yml`) builds and runs.

**Why a tarball?** erraudit is a private GitHub module (no proxy
artifact), the runner has no `ERRAUDIT_PAT` secret, and the Forgejo
mirror (`forgejo.home.lan`) is unreachable from GitHub-hosted runners.
A compressed source tarball is credential-free, invisible to the repo's
`find . -name go.mod` per-module loops, and auditable.

**Provenance (lint build).** The tarball is NOT a pristine `git archive`:
erraudit v0.7.0 requires the private `project-discovery-sdk` (portfolio
fleet audits), which credential-free CI cannot fetch. The tarball is
crafted from the tag by erraudit's `scripts/make-ci-tarball.sh`, which
detaches the two SDK-importing files behind the `portfolio` build tag
(the `portfolio` subcommand is absent; lint/tree/analyze are unchanged)
and strips the SDK requires from go.mod. The script verifies the build
against a COLD module cache before writing the tarball — the first
(2026-10-15) tarball shipped pristine and failed CI exactly here.

| File | Origin | SHA256 |
| --- | --- | --- |
| `erraudit-v0.7.0-src.tgz` | `scripts/make-ci-tarball.sh v0.7.0` from `LarsArtmann/erraudit` (tag + portfolio detached) | `f726f94681da816ddbb3fab4a35f045da0efbc7cc7422dd4fd3be10b1aa46208` |

## Refresh (bump the pin)

From a checkout of `LarsArtmann/erraudit` at the target tag:

```bash
scripts/make-ci-tarball.sh vX.Y.Z tools/erraudit/erraudit-vX.Y.Z-src.tgz
# remove the old tarball, update the table above AND the checksum +
# filename in .github/workflows/ci.yml, then verify locally:
scripts/erraudit-sweep.sh <path-to-built-binary>
```

Zero-baseline expectation: 2026-09-13 (22 modules) and re-verified
2026-10-15 (104 modules) — `erraudit lint ./... --enforce-go-error-family
--type-aware` exits 0 everywhere. The sweep script is
`scripts/erraudit-sweep.sh`.
