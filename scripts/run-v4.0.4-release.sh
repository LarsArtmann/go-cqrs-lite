#!/usr/bin/env bash
# Run batch release for v4.0.4
set -euo pipefail

cd "$(git rev-parse --show-toplevel)"

./scripts/batch-release.sh \
  "catalog v4.0.4 Dependency alignment" \
  "cmd/api-stability v4.0.2 Dependency alignment" \
  "cmd/cqrs-gen v4.0.2 Dependency alignment" \
  "cmd/doc-check v4.0.1 Dependency alignment" \
  "codec v4.0.4 Dependency alignment" \
  "command v4.0.2 Dependency alignment" \
  "decider v4.0.3 Dependency alignment" \
  "dedup v4.0.1 Dependency alignment" \
  "deriver v4.0.2 Dependency alignment" \
  "dispatcher v4.0.2 Dependency alignment" \
  "encryption v4.0.3 COSE encryption support" \
  "event v4.0.4 MultiBatchEntry MultiSink COSE integration" \
  "event/v4/eventtest v0.2.1 Multi-batch test helpers" \
  "example/getting-started v4.0.2 Dependency alignment" \
  "example/taskmanager v4.0.2 Dependency alignment" \
  "graph v4.0.3 Dependency alignment" \
  "id v4.0.3 Dependency alignment" \
  "idempotency v4.0.2 Dependency alignment" \
  "idempotency/kvstore v4.0.2 Dependency alignment" \
  "integration v4.0.2 Dependency alignment" \
  "kv v4.0.3 Dependency alignment" \
  "listing v4.0.3 Dependency alignment" \
  "metadata v4.0.2 Dependency alignment" \
  "middleware v4.0.3 Dependency alignment" \
  "otel v4.0.3 Dependency alignment" \
  "projection v4.0.2 Dependency alignment" \
  "projectionhost v4.0.3 Dependency alignment" \
  "prometheus v4.0.2 Dependency alignment" \
  "query v4.0.2 Dependency alignment" \
  "retry v4.0.2 Dependency alignment" \
  "scenario v4.0.3 Dependency alignment" \
  "scheduling v4.0.3 Dependency alignment" \
  "schema v4.0.3 Dependency alignment" \
  "signing v4.0.3 COSE Sign1 implementation" \
  "snapshot v4.0.3 Dependency alignment" \
  "stack v4.0.2 Multi-database improvements" \
  "stack/bench v4.0.2 Dependency alignment" \
  "stack/memory v4.0.2 Dependency alignment" \
  "stack/pebble v4.0.2 Dependency alignment" \
  "stack/postgres v4.0.2 Dependency alignment" \
  "stack/sqlite v4.0.2 Dependency alignment" \
  "stack/turso v4.0.2 Dependency alignment" \
  "storage v4.0.3 OTel instrumentation multi-batch" \
  "storage/memory v4.0.2 Dependency alignment" \
  "storage/pebble v4.0.3 Dependency alignment" \
  "storage/turso v4.0.2 Dependency alignment" \
  "testutil v4.0.2 Dependency alignment" \
  "transport/grpc v4.0.2 Event handler refactoring" \
  "transport/http v4.0.3 Dependency alignment" \
  "watermill v4.0.4 Command bus improvements"
