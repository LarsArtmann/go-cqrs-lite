#!/usr/bin/env bash
# bump-cqrs-lint.sh — sync cqrs-lint version, vendorHash, and go.mod after a version bump.
#
# Usage: scripts/bump-cqrs-lint.sh <new-version>
# Example: scripts/bump-cqrs-lint.sh 4.4.0
#
# This script:
# 1. Updates the version constant in cmd/cqrs-lint/main.go
# 2. Runs `go mod tidy` in the cqrs-lint module (GOWORK=off)
# 3. Attempts `nix build .#cqrs-lint` and extracts the correct vendorHash on mismatch
# 4. Verifies the build succeeds
#
# It does NOT tag or push — do that manually per CONTRIBUTING.md.

set -euo pipefail

VERSION="${1:?Usage: $0 <new-version>}"
REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
LINT_DIR="$REPO_ROOT/cmd/cqrs-lint"

if [[ ! "$VERSION" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
    echo "ERROR: version must be semver (X.Y.Z), got: $VERSION" >&2
    exit 1
fi

echo "==> Bumping cqrs-lint to v$VERSION"

# 1. Update version constant
sed -i "s/^const version = .*/const version = \"$VERSION\"/" "$LINT_DIR/main.go"
echo "  Updated version constant in main.go"

# 2. go mod tidy
echo "==> Running go mod tidy (GOWORK=off)..."
( cd "$LINT_DIR" && GOWORK=off go mod tidy )

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
echo "  2. git tag -a cmd/cqrs-lint/v$VERSION -m \"cqrs-lint v$VERSION\""
echo "  3. git push origin cmd/cqrs-lint/v$VERSION"
