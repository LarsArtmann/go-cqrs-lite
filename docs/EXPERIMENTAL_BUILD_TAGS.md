# Experimental Build Tags

> Go build tags that unlock experimental features. All are opt-in and have no effect unless explicitly enabled.

## Available Tags

| Tag             | Module  | Description                                                      | Stability    |
| --------------- | ------- | ---------------------------------------------------------------- | ------------ |
| `jsonv2`        | codec   | Use `encoding/json/v2` for marshaling instead of `encoding/json` | Experimental |
| `arenas`        | event   | Use arena allocation for high-throughput event creation          | Experimental |
| `simd`          | event   | Enable SIMD-accelerated event serialization                      | Experimental |
| `runtimesecret` | signing | Load signing keys from runtime secret stores                     | Experimental |

## Usage

```bash
# Build with jsonv2 codec
go build -tags jsonv2 ./...

# Build with multiple experimental features
go build -tags "jsonv2,arenas" ./...

# Test with experimental features
go test -tags jsonv2 ./codec/...
```

## Stability Policy

Experimental features behind build tags:

- May change API between releases without notice
- May be removed if the experiment doesn't pan out
- Are tested in CI but not part of the stability guarantee
- Are explicitly called out in release notes when changed

## Future Candidates

These tags are planned but not yet implemented:

- `streaming` — streaming event reads without materializing full slice
- `zstd` — zstd compression for stored events
