#!/usr/bin/env bash
# check-eventcatalog: render-validate the EventCatalog exporter output.
#
# Replaces the manual /tmp/ec-validate flow: generate a fixture catalog,
# npm install (network required, pinned by the exporter's package.json to
# @eventcatalog/core ^4), run `npx eventcatalog build`, and FAIL if the
# build errors or logs unresolved content references.
#
# Workdir override: CHECK_EVENTCATALOG_DIR=/path (kept for inspection on failure).
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
work="${CHECK_EVENTCATALOG_DIR:-$(mktemp -d "${TMPDIR:-/tmp}/eventcatalog-validate.XXXXXX")}"
log="$work/build.log"

echo "==> workdir: $work"

echo "==> 1/3 generating catalog fixture"
(cd "$root/catalog" && GOWORK=off go run -tags "goexperiment.jsonv2" ./cmd/ec-fixture "$work")

cd "$work"

echo "==> 2/3 npm install (network required)"
npm install --no-audit --no-fund --loglevel=error

echo "==> 3/3 npx eventcatalog build"
set +e
npx eventcatalog build >"$log" 2>&1
build_exit=$?
set -e
tail -20 "$log"

if [ "$build_exit" -ne 0 ]; then
	echo "FAIL: eventcatalog build exited $build_exit (full log: $log)" >&2
	exit 1
fi

if grep -qi "invalid content reference" "$log"; then
	echo "FAIL: unresolved content references in build output (full log: $log)" >&2
	exit 1
fi

echo "OK: eventcatalog build clean (log: $log)"
