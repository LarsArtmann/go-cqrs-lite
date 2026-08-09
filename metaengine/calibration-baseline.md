# Calibration Benchmark Baseline — 2026-08-08

> Machine: AMD Ryzen (32 threads), Linux
> Run via: `go test -run='^$' -bench='BenchmarkCalibration' -benchtime=1s ./...`
> CI regression threshold: 3× (fail if ns/op > 3× baseline)

| Benchmark                         | Baseline (ns/op) | Notes                                        |
| --------------------------------- | ---------------- | -------------------------------------------- |
| BenchmarkCalibration_PebbleSet-32 | 2439             | Pebble MapSet (JSON encode + LSM write)      |
| BenchmarkCalibration_PebbleGet-32 | 954              | Pebble MapGet (LSM point read + JSON decode) |

## Usage

The calibration regression CI gate compares these baselines against live benchmark
results. If any benchmark drifts more than 3× from baseline, the CI step fails.

### Updating the baseline

After intentional cost-model changes (new engine version, hardware upgrade, etc.),
re-run the benchmarks and update this file:

```bash
cd metaengine/pebbleengine && GOWORK=off go test -run='^$' -bench='BenchmarkCalibration' -benchtime=1s ./...
```

Then commit the updated baseline values.
