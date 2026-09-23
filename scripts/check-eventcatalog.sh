#!/usr/bin/env bash
# check-eventcatalog: render-validate the EventCatalog exporter output.
#
# Replaces the manual /tmp/ec-validate flow: generate a fixture catalog,
# npm install (network required, pinned by the exporter's package.json to
# @eventcatalog/core ^4), run `npx eventcatalog build`, and FAIL if the
# build errors or logs unresolved content references.
#
# Two fixture profiles are built in the same installed workdir:
#   1. default  — every resource kind (incl. agents; changelog pages off,
#                 see shouldEnableChangelog for the upstream core bug)
#   2. changelog — agents omitted so changelog pages render, proving the
#                 changelog.mdx sidecar files actually show up
#
# Workdir override: CHECK_EVENTCATALOG_DIR=/path (kept for inspection on failure).
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
work="${CHECK_EVENTCATALOG_DIR:-$(mktemp -d "${TMPDIR:-/tmp}/eventcatalog-validate.XXXXXX")}"
log="$work/build.log"

echo "==> workdir: $work"

echo "==> 1/4 generating catalog fixture (default profile)"
(cd "$root/catalog" && GOWORK=off go run ./cmd/ec-fixture "$work")

cd "$work"

echo "==> 2/4 npm install (network required)"
npm install --no-audit --no-fund --loglevel=error

run_build() {
	local profile="$1"
	echo "==> npx eventcatalog build ($profile profile)"
	set +e
	npx eventcatalog build >"$log" 2>&1
	local build_exit=$?
	set -e
	tail -20 "$log"

	if [ "$build_exit" -ne 0 ]; then
		echo "FAIL: eventcatalog build exited $build_exit for $profile profile (full log: $log)" >&2
		exit 1
	fi

	if grep -qi "invalid content reference" "$log"; then
		echo "FAIL: unresolved content references in $profile build output (full log: $log)" >&2
		exit 1
	fi

	if grep -q "InvalidContentEntryDataError" "$log"; then
		echo "FAIL: frontmatter schema violations in $profile build output (full log: $log)" >&2
		exit 1
	fi
}

run_build default

if [ ! -f "$work/commands/CreateOrder/changelog.mdx" ]; then
	echo "FAIL: fixture did not export changelog.mdx" >&2
	exit 1
fi

echo "==> 3/4 generating catalog fixture (changelog profile)"
(cd "$root/catalog" && GOWORK=off go run ./cmd/ec-fixture "$work" changelog)
run_build changelog

if ! grep -rq "initial release" "$work/dist/docs/commands/CreateOrder/1.0.0/changelog/"; then
	echo "FAIL: changelog profile did not render the CreateOrder changelog page" >&2
	exit 1
fi

if ! grep -q "changelog" "$work/eventcatalog.config.js"; then
	echo "FAIL: changelog profile did not enable changelog in eventcatalog.config.js" >&2
	exit 1
fi

echo "==> 4/4 semantic spot-checks (default profile artifacts)"
# Re-generate the default profile so its dist is the one left for inspection.
(cd "$root/catalog" && GOWORK=off go run ./cmd/ec-fixture "$work")
run_build default

if ! grep -q "collection: commands" "$work/channels/order-events/index.mdx"; then
	echo "FAIL: channel frontmatter lacks fully qualified message pointers" >&2
	exit 1
fi

if ! grep -rq "Create Order" "$work/dist/docs/channels/order-events/1.0.0/index.html"; then
	echo "FAIL: rendered channel page does not link its messages" >&2
	exit 1
fi

echo "OK: eventcatalog build clean (log: $log)"
