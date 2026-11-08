# example/standard-topology

The ADR-0165 demo: the smallest app where **developers declare the domain and
engines; the topology fills itself**. One command, one event, one Lookup
projection, on sqlite, composed through `system.Standard`.

```bash
go run .            # built-in engines-only default (sqlite file in cwd)
go run . -config deployment.json   # or any operator-rendered config
```

## The three-step ladder (ADR-0165)

**1. Go** — declare engines only; `Standard` fills the canonical
three-instance topology (source-of-truth + queries audit log on the WAL
engine, projections pooled over every engine):

```go
sys, err := system.New(ctx, domain(), system.Standard(system.DeploymentConfig{
	Engines: map[string]system.EngineConfig{
		"primary": {Driver: "sqlite", DSN: "file:notes.db"},
	},
}))
```

**2. CUE** — the same preset as `system/config.cue`, for operators who render
configs outside Go (`cue eval -e standard --out json`, no runtime CUE
dependency):

```cue
import "github.com/larsartmann/go-cqrs-lite/system/v4"

standard: engines: primary: {driver: "sqlite", dsn: "file:notes.db"}
```

**3. NixOS** — the `nixosModules.cqrs` flake output renders
`/etc/cqrs/demo/deployment.json` from a Nix declaration, auto-provisions the
databases (postgres peer-auth over the unix socket, no passwords in the
store), and exports a hardened systemd fragment. The unit itself stays
app-owned:

```nix
{
  inputs.cqrs.url = "github:LarsArtmann/go-cqrs-lite";

  # in your NixOS module list, after importing flake.nixosModules.cqrs:
  cqrs.deployments.demo.engines.primary.driver = "postgres";

  systemd.services.demo = cqrs.lib.hardenedService "demo" // {
    # your binary: Environment.CQRS_DEPLOYMENT_CONFIG and the cqrs-demo user,
    # group, StateDirectory, and sandboxing come from the fragment
    ServiceConfig.ExecStart = "${pkgs.hello}/bin/hello";
  };
}
```

The rendered JSON loads through the same `system.LoadConfig` this binary uses
(`CQRS_DEPLOYMENT_CONFIG` is the default `-config` value), and
`system.Standard` is a no-op pass-through when the config already declares
instances — the three steps compose.

The Go, CUE, and NixOS presets are pinned to one shared instance-triple
fixture (`system/testdata/standard-instance-triple.json`) asserted by the Go
suite, the `#check-cue-config` gate, and the `cqrs-nixos-eval` flake check,
so the twins cannot drift apart.
