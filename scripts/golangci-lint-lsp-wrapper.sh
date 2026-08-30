#!/usr/bin/env bash
# golangci-lint-lsp wrapper: pins the disk cache so the LSP's diagnostics are
# computed against a warm, working cache instead of the broken /mnt/buildcache
# mount (or whatever GOLANGCI_LINT_CACHE leaked in from the launching shell).
#
# Background: the tool shell exports GOLANGCI_LINT_CACHE pointing at
# /mnt/buildcache (corrupted since 2026-08-16), and environment bleed made
# every LSP diagnostic render as "failed to initialize build cache". crush.json
# sets its own value, but ambient wins in some launch paths — this wrapper
# makes the pin unconditional.
#
# Install: point the LSP "command" in ~/.config/crush/crush.json at this file
# (the repo copy is scripts/golangci-lint-lsp-wrapper.sh; copy or symlink it
# somewhere on PATH, e.g. ~/.local/bin/).

set -euo pipefail

CACHE_ROOT="${HOME}/tmp/golangci-lint-cache"
mkdir -p "$CACHE_ROOT"

export GOLANGCI_LINT_CACHE="$CACHE_ROOT"
export GOLANGCI_LINT_ANALYSIS_CACHE="${CACHE_ROOT}-analysis"
mkdir -p "$GOLANGCI_LINT_ANALYSIS_CACHE"

exec golangci-lint-langserver "$@"
