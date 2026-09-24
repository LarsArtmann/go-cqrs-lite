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
# The exporter is non-destructive (consumers may keep their own files in the
# project), so clear its generated resource dirs between profiles — a stale
# agents/ dir would retrigger the upstream agent-changelog crash.
find "$work" -mindepth 1 -maxdepth 1 \
	\( -name node_modules -o -name dist -o -name package-lock.json -o -name build.log \) -prune -o \
	-exec rm -rf {} +
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
find "$work" -mindepth 1 -maxdepth 1 \
	\( -name node_modules -o -name dist -o -name package-lock.json -o -name build.log \) -prune -o \
	-exec rm -rf {} +
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

echo "==> 5/5 @eventcatalog/linter on the plain-refs profile"
# The render profiles above pin the @eventcatalog/core contract (composite
# "<id>-<version>" refs). The linter is the opposite consumer: it indexes by
# frontmatter ID and can never resolve composite refs, so the same fixture is
# exported again with WithPlainRefIDs and linted with the two rules the
# federation hub previously had to warn-suppress — refs/resource-exists and
# best-practices/owner-required — fully re-armed at error severity.
plain="$work-plain"
mkdir -p "$plain"
(cd "$root/catalog" && GOWORK=off go run ./cmd/ec-fixture "$plain" plain)
# --no-save: the linter is a gate tool, not an export dependency.
npm install --no-save --no-audit --no-fund --loglevel=error @eventcatalog/linter@1.1.20 >/dev/null
cat > "$plain/.eventcatalogrc.js" <<'RC'
export default {
  rules: {
    'refs/resource-exists': 'error',
    'best-practices/owner-required': 'error',
    'best-practices/summary-required': 'error',
    'refs/file-exists': 'error',
    'structure/duplicate-resource-ids': 'error',
  },
};
RC
if ! (cd "$plain" && node "$work/node_modules/@eventcatalog/linter/dist/cli/index.js" -q .); then
	echo "FAIL: @eventcatalog/linter reported problems on the plain-refs fixture" >&2
	echo "      (workdir: $plain)" >&2
	exit 1
fi

echo "OK: eventcatalog build clean + linter clean (log: $log)"
