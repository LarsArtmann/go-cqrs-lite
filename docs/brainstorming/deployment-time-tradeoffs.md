# Deployment-Time Tradeoffs: Why Don't We Ship Multiple Binaries?

> Brainstorming session on Storage/Memory/CPU/Latency/Correctness tradeoffs at deployment time.

## The Question

There are real tradeoffs on Storage, Memory, CPU, Latency, and Correctness. Why don't we give end users multiple binaries so they can choose these tradeoffs at deployment time?

A developer can't always make the right decision. One user deploys on a hyperscaler cloud, the next on a Pi Zero, the next on a home cluster, the next on a desktop, another on a battery-constrained laptop or IoT device.

## Why It's Rare

### Most tradeoffs are runtime knobs, not compile-time axes

Cache sizes, batch sizes, worker counts, buffer pools — these don't need a different binary, they need a config file. Shipping 5 binaries for what a `--max-memory=512MB` flag solves is overkill.

### Combinatorial explosion

4 tradeoff dimensions with 3 options each = 81 binaries. Nobody tests 81 binaries. The support matrix becomes untenable.

### Distribution friction

"Which binary do I download?" is already a hurdle. A decision matrix in your install docs kills adoption. Most projects struggle to get users past `curl | sh`.

### More choices ≠ more freedom

Every binary you ship is a support contract. Every configuration knob is a documentation burden and a bug surface area. Users want:

```
"It works. It's fast enough. It doesn't crash."
```

Not 5 binaries. Not a tuning matrix. Opinionated software that makes reasonable defaults for the 90th percentile use case.

## Projects That DO Ship Multiple Builds

| Project | Strategy | Tradeoffs |
|---|---|---|
| Go | `GOAMD64=v1,v2,v3,v4` | CPU instruction sets |
| TensorFlow | CPU-only vs GPU builds | Compute backend |
| SQLite | 100+ compile-time flags | Threading, memory model, extensions |
| Alpine Linux | musl vs glibc images | Binary size vs compatibility |
| Nix | Rebuild everything with different flags | Every dependency swappable |

## The Spectrum

```
Runtime config  ◄── 90% of tradeoffs live here
Compile flags   ◄── 9% (allocators, crypto backends, SIMD paths)
Separate binary ◄── 1% (fundamentally different environments)
```

## GOAMD64 Explained

Go's `GOAMD64` controls which CPU instructions the compiler is allowed to emit for AMD64 (x86-64) binaries.

| Level | Baseline | What you gain | Who can't run it |
|---|---|---|---|
| `v1` | Original x86-64 (2003) | Maximum compatibility | Nobody (all x86-64 CPUs) |
| `v2` | CMPXCHG16B, LAHF/SAHF, SSE4 | Better atomics, SIMD | Very old pre-2006 CPUs |
| `v3` | AVX, AVX2, BMI1/2, FMA | 256-bit SIMD, faster math | Anything before ~2013 |
| `v4` | AVX512 | 512-bit SIMD | Anything before ~2017, some AMD chips |

```bash
GOAMD64=v1 go build -o app-v1 .   # runs everywhere
GOAMD64=v3 go build -o app-v3 .   # faster on modern CPUs
```

Same source code, same binary structure, but v1 might loop to copy 16 bytes, v3 uses a single AVX2 instruction to copy 32 bytes, v4 uses a single AVX512 instruction to copy 64 bytes.

Real-world impact: most code sees <2% difference. Hot loops (math, crypto, encoding, sorting) see 10-40% faster on higher levels. Default is `v1` for maximum compatibility.

This exists as a build flag because CPU instructions are baked into machine code at compile time — you can't switch instruction sets at runtime.

## The Real Answer

### "I don't know my users' constraints" — false premise

You already make decisions for your users. You chose Go. You chose a GC. You chose your memory model. You chose your dependencies. Every library you imported baked in tradeoffs your user never got to vote on.

You don't ship "one binary" — you ship **one set of tradeoffs**. That's what software *is*.

### Different algos for different environments

What if you want different algorithms for a big memory machine vs a very constrained one (<1GB, or even 10MB)?

The answer isn't multiple binaries. It's:

1. **Make your algorithm swappable** (interface/strategy pattern)
2. **Let the caller pick** at runtime based on their knowledge of their deployment
3. **Pick sensible defaults** for the 90th percentile

This is exactly what `go-cqrs-lite` already does: `EventSink` vs `EventSource`, `MemoryStore` vs `SQLEventStore`. The tradeoff isn't in the binary — it's in the **composition**. The Pi Zero user picks `MemoryStore`. The hyperscaler user picks `SQLEventStore`. Same binary, different wiring.

### You already solved this problem

The composition-based approach is so baked into good library design that it feels obvious. The user on the constrained device picks lightweight implementations. The user on the beefy server picks the full-featured ones. Same binary, different tradeoffs — chosen at composition time, not compile time.

## Where the Industry Is Heading

- **WebAssembly** — same code, different runtimes with different resource profiles
- **Nix** — every dependency swappable at build time
- **Go build tags** — zero-allocation paths for embedded vs full-featured for cloud

The gap: there's no standard mechanism to express "I'm deploying to a constrained environment, give me the minimal build." It's all ad-hoc per project. That's a tooling problem, not a conceptual one.
