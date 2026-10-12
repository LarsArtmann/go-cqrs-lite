Fixed on master (in tree, not in a tag yet — rides the next otel/prometheus patch cuts).

`if p == nil { return nil }` at the top of both `Shutdown` methods:

- `otel/setup.go:126` (`Provider.Shutdown`, the `p.tracerProvider.ForceFlush` first statement this issue named)
- `prometheus/exporter.go:69` (`Provider.Shutdown`)

Your diagnosis was exact — reproduced deterministically pre-fix with the same one-liner:

```go
_ = (*otel.Provider)(nil).Shutdown(context.Background())
// panic: runtime error: invalid memory address or nil pointer dereference
```

The zero behavior is the honest one you asked for: nothing to flush, nothing to shut down, return nil. Pinned by `TestProvider_Shutdown_NilReceiverNoOp` in both modules; both full suites green.
