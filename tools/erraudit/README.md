# Vendored erraudit source

This directory holds the pinned erraudit source snapshot that the CI
`error-audit` job (`.github/workflows/ci.yml`) builds and runs.

**Why a tarball?** erraudit is a private GitHub module (no proxy
artifact), the runner has no `ERRAUDIT_PAT` secret, and the Forgejo
mirror (`forgejo.home.lan`) is unreachable from GitHub-hosted runners.
A compressed source tarball is credential-free, invisible to the repo's
`find . -name go.mod` per-module loops, and auditable (exact `git
archive` output of the tag — no binary blobs).

| File | Origin | SHA256 |
| --- | --- | --- |
| `erraudit-v0.6.1-src.tgz` | `git archive v0.6.1 -- cmd pkg internal go.mod go.sum LICENSE README.md` from `LarsArtmann/erraudit` | `8e84b9d2a1d6a350c9d38c6d4dfdedba0fbdb2c84db35536f4ce72f599065879` |

## Refresh (bump the pin)

From a checkout of `LarsArtmann/erraudit` at the target tag:

```bash
git archive --format=tar.gz -o tools/erraudit/erraudit-vX.Y.Z-src.tgz \
  vX.Y.Z cmd pkg internal go.mod go.sum LICENSE README.md
sha256sum tools/erraudit/erraudit-vX.Y.Z-src.tgz
# remove the old tarball, update the table above AND the checksum +
# filename in .github/workflows/ci.yml, then verify locally:
scripts/erraudit-sweep.sh <path-to-built-binary>
```

Zero-baseline expectation: 2026-09-13 (22 modules) and re-verified
2026-10-15 (104 modules) — `erraudit lint ./... --enforce-go-error-family
--type-aware` exits 0 everywhere. The sweep script is
`scripts/erraudit-sweep.sh`.
