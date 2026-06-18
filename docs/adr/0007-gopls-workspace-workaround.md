# ADR-0007: gopls Multi-Module Workspace Workaround

## Status

Accepted

## Context

This project uses a Go workspace (`go.work`) with 14+ independent modules. The `gopls` LSP server frequently reports stale or incorrect diagnostics in this setup:

- False-positive "not in go.mod" errors for test dependencies
- Phantom "undefined" errors for cross-module types
- Stale diagnostics after `go mod tidy`

This is a known limitation of `gopls` with multi-module workspaces.

## Decision

Document the workaround rather than restructure the project. The multi-module approach is correct for this library/SDK and should not be changed.

## Workarounds

1. **Disable the `mod_tidy` analyzer**: The false positives come from gopls's
   `mod_tidy` analyzer running in workspace mode. Disable it via:
   - VS Code: `.vscode/settings.json` (included in repo root)
   - Neovim: `lspconfig.gopls.settings.gopls.analzers.gopls.mod_tidy = false`
2. **Restart gopls**: Run `:LspRestart` (Neovim) or `Go: Restart Language Server` (VS Code)
3. **Run `go mod tidy`** in the affected module directory
4. **Verify with `go build`**: If `go build ./...` passes, the LSP errors are phantom
5. **Per-module verification**: Use `GOWORK=off go build ./...` inside a module to confirm isolation

## Consequences

- No structural changes required
- Developers need to be aware of this LSP limitation
- CI runs `GOWORK=off` per-module tests to guarantee isolation
