# ADR-0165: Standard deployment topology — a data filler, a config-format ladder, and a first-party NixOS module

| Field      | Value                                                                                                                       |
| ---------- | --------------------------------------------------------------------------------------------------------------------------- |
| Status     | Accepted                                                                                                                    |
| Date       | 2026-11-08                                                                                                                  |
| Deciders   | Lars Artmann                                                                                                                |
| Related    | ADR-0123 (single composition root), ADR-0152 (fleet-first topology), ADR-0163 (CUE verdict, different domain), ADR-0142      |
| Supersedes | nothing — blesses a default shape on top of the existing `DeploymentConfig` roles; adds no storage concepts                  |

## Context

Nearly every fleet app needs the same three-tier deployment:

1. **One command/event WAL** — the journal (source of truth).
2. **One query audit log** — a WAL of dispatched queries for audit, debug, analytics.
3. **One to N projection stores** — state caches rebuilt from commands and events.

`DeploymentConfig` already models this 1:1 (`source-of-truth`/`events`, `queries`,
`projections` roles, config_types.go). What is missing:

- Every consumer hand-writes the same three-instance YAML; nothing blesses the shape.
- There is no deployment story beyond the file: `flake.nix` exports zero `nixosModules`
  (the only NixOS usage today is internal pg/mysql test VMs).
- The owner dislikes YAML as an authoring format; CUE was raised as an alternative.

CUE prior art: ADR-0163 evaluated and rejected CUE as a *subsumption engine* for
schema-compat gating (fidelity spike 22/24). That verdict is about CUE answering
"is this shape additive?" — a different domain. Config-as-CUE is not blocked by it,
but it does not earn a runtime dependency either.

## Decision

1. **`system.Standard(d DeploymentConfig) DeploymentConfig`** — a pure DATA filler, not
   a builder. When `d.Instances` is empty and `d.Engines` is not, it stamps the canonical
   topology: a `source-of-truth` instance on the first declared engine, a `queries`
   audit instance on that same engine, and a `projections` instance pooling ALL declared
   engines. Engines, buses, durability, priorities stay operator-owned; `Standard` only
   fills `Instances` (idempotent — a config that already declares instances passes
   through untouched). This is deliberately NOT a return of the Go-code `stack` presets
   ADR-0123 killed: those hid engine choice inside bundles; `Standard` inspects and
   completes operator data.
2. **Config-format ladder** (each layer avoids the one below for authoring):
   - **Go code**: `system.Standard(d)` — typed Go, no files.
   - **NixOS**: attrsets — the module (below) generates the config; no hand-written files.
   - **CUE**: `system/config.cue` ships the `DeploymentConfig` schema WITH the standard
     topology as CUE defaults; `cue export --out yaml` (or `--out json`) feeds `LoadConfig`.
   - **YAML/JSON/env**: stays the binary's wire format (k8s-style lowest layer).
   **No runtime CUE dependency**: `cuelang.org/go` is not added to any module — it would
   make all 38 fleet consumers pay for a parser most never invoke. CUE is an optional
   authoring artifact, evaluated by the `cue` CLI at build/deploy time.
3. **`LoadConfig` gains JSON** — extension-based parser pick (`.json` → stdlib
   `encoding/json` into a koanf confmap; YAML stays the default). Zero new dependencies;
   `cue export --out json` output becomes loadable verbatim.
4. **NixOS module lives HERE** — this repo owns the `DeploymentConfig` contract, so it
   owns the module that generates it. `flake.nix` exports `nixosModules.cqrs` from
   `nix/nixos/cqrs.nix`:
   - `cqrs.deployments.<name>`: engine/instance/durability/priority attrsets mirroring
     the koanf schema, rendered to `/run/cqrs/<name>/deployment.json` per deployment.
   - **Auto-provisioning**: `engine = "postgres"` with no explicit DSN provisions a
     local `services.postgresql` database + user and points the DSN at it; `sqlite`
     defaults to a file under the deployment's `StateDirectory`. An explicit DSN always
     wins and provisions nothing.
   - The module generates config + DBs; the CONSUMING app's unit stays app-owned
     (shared hardening defaults exported as `cqrs.lib.hardenedService`), because the
     service binary is consumer property — a library must not ship a generic daemon.
5. **Proof**: `example/standard-topology/` demonstrates Go-filler + CUE export + NixOS
   module in one runnable app; a NixOS VM test exercises the module end-to-end
   (the repo already runs QEMU VMs for pg/mysql integration).

## Consequences

- The canonical topology gets ONE blessed address (`system.Standard`, `cue eval`
  defaults, NixOS submodule defaults) instead of N hand-copied YAML files.
- YAML stops being the authoring surface at every layer where something better exists;
  it remains the interchange format, which is what it is good at.
- The flake gains a public `nixosModules` output — a new consumer surface that must
  stay eval-clean on stable NixOS (guarded by the example's VM test).
- `system/config.cue` is a generated-or-verified artifact: a check script pins the CUE
  schema against the Go struct surface (field names, koanf tags) so the two cannot
  drift silently.
- If a future consumer genuinely needs in-process CUE evaluation, that is a NEW ADR
  (dep-budget review for `cuelang.org/go`), not a quiet addition.

## Implementation map

- `system/standard.go` — the filler (+ `standard_test.go`)
- `system/config_loader.go` — JSON branch
- `system/config.cue` — schema + defaults; `scripts/check-cue-config.sh` drift pin
- `nix/nixos/cqrs.nix` — the module; `flake.nix` — `nixosModules.cqrs` output
- `example/standard-topology/` — runnable demo + `example/standard-topology/vm-test.nix`
