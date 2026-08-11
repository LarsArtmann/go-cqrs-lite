# codec — DEPRECATED

> **This module is deprecated.** Import [`github.com/larsartmann/go-codec`](https://pkg.go.dev/github.com/larsartmann/go-codec) directly instead.

This module re-exports all symbols from [go-codec](https://pkg.go.dev/github.com/larsartmann/go-codec) as type/variable aliases for backward compatibility.

## Migration

Change your import from:

```go
import "github.com/larsartmann/go-cqrs-lite/codec/v4"
```

to:

```go
import "github.com/larsartmann/go-codec"
```

All symbols are identical (type aliases). No code changes are needed beyond the import path.
