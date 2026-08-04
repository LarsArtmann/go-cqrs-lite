#!/usr/bin/env bash
# enable-nspawn-support.sh — One-shot setup: enable systemd-nspawn container
# support for NixOS test driver on this host.
#
# Adds `uid-range` to system-features and enables `auto-allocate-uids` so that
# `nix build .#checks.x86_64-linux.mysql-nspawn` works (~10x faster MySQL test).
#
# MUST be run as root: sudo bash scripts/enable-nspawn-support.sh
#
# Idempotent: safe to run multiple times.
set -euo pipefail

if [ "$(id -u)" -ne 0 ]; then
    echo "ERROR: This script must be run as root ( systemd-nspawn needs CAP_SYS_ADMIN )."
    echo "  sudo bash $0"
    exit 1
fi

CONFIG_FILE="/etc/nixos/configuration.nix"
MODULE_FILE="/etc/nixos/nspawn-support.nix"

# --- Create the nspawn support module ---------------------------------------

cat > "$MODULE_FILE" <<'EOF'
# nspawn-support.nix — Enable systemd-nspawn container tests for the NixOS
# test driver. Required by: nix build .#checks.x86_64-linux.mysql-nspawn
#
# These settings allow the Nix daemon to allocate UID ranges for build
# processes that use systemd-nspawn (PID namespace isolation). Without them,
# the test derivation fails with "missing system features: uid-range".
{ ... }:
{
  nix.settings = {
    auto-allocate-uids = true;
    # Must include existing features + uid-range
    system-features = [
      "nixos-test"
      "benchmark"
      "big-parallel"
      "kvm"
      "uid-range"
    ];
    experimental-features = [
      "nix-command"
      "flakes"
      "pipe-operators"
      "auto-allocate-uids"
    ];
  };
}
EOF

echo "==> Created $MODULE_FILE"

# --- Import the module in configuration.nix (idempotent) --------------------

if grep -q "nspawn-support.nix" "$CONFIG_FILE"; then
    echo "==> $CONFIG_FILE already imports nspawn-support.nix — skipping"
else
    # Add import before the closing brace
    sed -i '/^}/i\  # Enable systemd-nspawn container test support\n  imports = [ ./nspawn-support.nix ];\n' "$CONFIG_FILE"
    echo "==> Added import to $CONFIG_FILE"
fi

# --- Rebuild ----------------------------------------------------------------

echo "==> Rebuilding NixOS (switch)..."
nixos-rebuild switch

echo ""
echo "==> Verifying uid-range is available..."
AVAILABLE=$(nix show-config 2>/dev/null | grep '^system-features' | sed 's/.*= //')
echo "    system-features: $AVAILABLE"
if echo "$AVAILABLE" | grep -qw uid-range; then
    echo "✅ uid-range is available — nspawn tests will work"
else
    echo "❌ uid-range NOT in system-features — rebuild may have failed"
    exit 1
fi

echo ""
echo "==> Done! You can now run:"
echo "    nix build .#checks.x86_64-linux.mysql-nspawn -L"
echo "    sudo nix run .#integration-mysql-nspawn"
