#!/usr/bin/env bash
# bump-cqrs-lint.sh — sync cqrs-lint go.mod and the Nix vendorHash after a
# dependency change, and verify the Nix build.
#
# Usage: scripts/bump-cqrs-lint.sh [new-version]
# Example: scripts/bump-cqrs-lint.sh 4.11.0
#
# The version argument is informational only: since the hand-maintained
# version constant was removed (resolvedVersion() reads the version from
# debug.ReadBuildInfo), there is NO source file to bump — the reported
# version comes from the git tag itself at `go install …@vX.Y.Z`.
#
# This script:
# 1. Runs `go mod tidy` in the cqrs-lint module (GOWORK=off)
# 2. Attempts `nix build .#cqrs-lint` and extracts the correct vendorHash on mismatch
# 3. Verifies the build succeeds
#
# It does NOT tag or push — do that with scripts/tag-release.sh.

set -euo pipefail

VERSION="${1:-}"
REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
LINT_DIR="$REPO_ROOT/cmd/cqrs-lint"

if [[ -n "$VERSION" && ! "$VERSION" =~ ^v?[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
	echo "ERROR: version must be semver (X.Y.Z), got: $VERSION" >&2
	exit 1
fi

VERSION="${VERSION#v}"

echo "==> Syncing cqrs-lint go.mod + Nix vendorHash${VERSION:+ (targeting v$VERSION)}"

# 1. go mod tidy
echo "==> Running go mod tidy (GOWORK=off)..."
(cd "$LINT_DIR" && GOWORK=off go mod tidy)

# 3. nix build (retry with corrected vendorHash)
echo "==> Building with nix..."
cd "$REPO_ROOT"
if nix build .#cqrs-lint --no-link 2>&1 | tee /tmp/cqrs-lint-build.log; then
	echo "  Build succeeded on first try"
else
	# Extract the correct hash from the error
	NEW_HASH=$(grep -oP 'got:\s+\Ksha256-[a-zA-Z0-9+/=]+' /tmp/cqrs-lint-build.log || true)
	if [[ -z "$NEW_HASH" ]]; then
		echo "ERROR: Build failed and could not extract vendorHash from output" >&2
		cat /tmp/cqrs-lint-build.log
		exit 1
	fi
	echo "  vendorHash mismatch — updating flake.nix"
	sed -i "s|vendorHash = \"sha256-[a-zA-Z0-9+/=]*\"|vendorHash = \"$NEW_HASH\"|" "$REPO_ROOT/flake.nix"
	echo "  Retrying build with corrected hash..."
	nix build .#cqrs-lint --no-link
fi

echo ""
echo "==> Done! Next steps:"
echo "  1. nix run .#verify"
echo "  2. ./scripts/tag-release.sh cmd/cqrs-lint v${VERSION:-<version>} \"<description>\""
echo "  3. git push origin cmd/cqrs-lint/v${VERSION:-<version>}"
echo "  4. ./scripts/tag-release.sh --smoke cmd/cqrs-lint v${VERSION:-<version>}"
