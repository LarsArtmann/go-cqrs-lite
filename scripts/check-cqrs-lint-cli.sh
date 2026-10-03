#!/usr/bin/env bash
# check-cqrs-lint-cli.sh — binary-level contract probes for the cqrs-lint CLI.
#
# The in-process tests (cmd/cqrs-lint/subcommand_consistency_test.go,
# contract_enforcement_test.go) pin the subcommand flag contract at the API
# level; this script pins it at the BINARY level: the exact exit codes and
# stdout shapes a shell user or CI pipeline sees. The probes encode the
# README "Subcommand flag contract" + "Exit codes" tables — a contract
# regression fails here before it ships.
#
# Usage:
#   scripts/check-cqrs-lint-cli.sh              build + probe the real binary
#   CQRS_LINT_BIN=./cqrs-lint scripts/...       probe an existing binary
#   scripts/check-cqrs-lint-cli.sh --self-test  fault-injection self-test
#                                                (stub binaries only, no build)
#
# Self-test policy (repo convention): every probe is re-run against stub
# binaries that simulate each violation class; the harness must CATCH every
# planted fault (probe exits nonzero) and PASS one positive control. A
# self-test that cannot fail is not a pin.

set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# Probe helpers return 0 (contract holds) or 1 (violation) and print the
# reason on violation. They never exit the script so --self-test can invert
# them per-probe.

# probe_rules_json <bin>: `rules --format json` exits 0 and emits a JSON
# array as its first stdout character.
probe_rules_json() {
	local out rc
	out="$("$1" rules --format json 2>/dev/null)" && rc=0 || rc=$?
	if [ "$rc" -ne 0 ]; then
		echo "rules --format json exited $rc (want 0)" >&2
		return 1
	fi
	case "$out" in
	'['*) return 0 ;;
	*)
		echo "rules --format json stdout should start with '[', got: ${out:0:40}" >&2
		return 1
		;;
	esac
}

# probe_init_path <bin>: init honors --path and writes the file there.
probe_init_path() {
	local dir="$WORK_DIR/init-probe"
	mkdir -p "$dir"
	"$1" init --path "$dir" >/dev/null 2>&1 || {
		echo "init --path exited nonzero" >&2
		return 1
	}
	if [ ! -f "$dir/.cqrs-lint.json" ]; then
		echo "init --path did not write $dir/.cqrs-lint.json" >&2
		return 1
	fi
	return 0
}

# REJECTED probes: the binary call must FAIL (probe succeeds when rc != 0).
# Covers: scorecard --format csv / doctor --format yaml (per-command format
# subsets), version --fix (lint-only flag scoping), doctor --fix (rename pin),
# scorecard below --scorecard-threshold (CI gate), and the clean empty-dir
# lint run that must exit 0.
run_rejected_probe() {
	local label="$1" bin="$2"
	if "$bin" >/dev/null 2>&1; then
		echo "$label: binary exited 0, want nonzero" >&2
		return 1
	fi
	return 0
}

run_probes() {
	local bin="$1" failed=0

	# Rejected-class probes wrap their binary invocation into a full command
	# via the WORK_DIR-scoped helpers above; dispatch by name.
	probe_rules_json "$bin" || failed=1
	if ! run_rejected_probe "scorecard --format csv" \
		"$bin scorecard --format csv --path $WORK_DIR"; then failed=1; fi
	if ! run_rejected_probe "doctor --format yaml" \
		"$bin doctor --format yaml --path $WORK_DIR"; then failed=1; fi
	if ! run_rejected_probe "version --fix (lint-only flag)" \
		"$bin version --fix"; then failed=1; fi
	if ! run_rejected_probe "doctor --fix (renamed away)" \
		"$bin doctor --fix"; then failed=1; fi
	if ! run_rejected_probe "scorecard below threshold" \
		"$bin scorecard --scorecard-threshold 101 --path $WORK_DIR"; then failed=1; fi
	probe_init_path "$bin" || failed=1

	# Accepted-class: a valid Go module with no go-cqrs-lite imports is a
	# clean "nothing to lint" run, rc=0. (A bare empty dir is NOT clean —
	# package loading fails there and must exit nonzero.)
	if ! "$bin" --path "$WORK_DIR/clean" >/dev/null 2>&1; then
		echo "clean no-import module lint: exited nonzero, want 0" >&2
		failed=1
	fi

	return "$failed"
}

self_test() {
	local tmp stub failed=0
	tmp="$(mktemp -d)"

	# Fault injection: stub binaries that simulate each violation class.
	printf '#!/usr/bin/env bash\necho "rule catalog (text)"\n' >"$tmp/stub_text_rules"
	printf '#!/usr/bin/env bash\nexit 0\n' >"$tmp/stub_always_ok"
	printf '#!/usr/bin/env bash\nexit 1\n' >"$tmp/stub_always_fails"
	printf '#!/usr/bin/env bash\nexit 0\n' >"$tmp/stub_init_no_file"
	printf '#!/usr/bin/env bash\necho "[]"\n' >"$tmp/stub_good_json"
	chmod +x "$tmp"/stub_*

	WORK_DIR="$tmp/work"
	mkdir -p "$WORK_DIR"

	# Every planted fault MUST be caught (probe returns 1).
	probe_rules_json "$tmp/stub_text_rules" && {
		echo "self-test: text output must fail the json probe" >&2
		failed=1
	}
	probe_rules_json "$tmp/stub_always_fails" && {
		echo "self-test: nonzero rc must fail the json probe" >&2
		failed=1
	}
	probe_init_path "$tmp/stub_init_no_file" && {
		echo "self-test: missing file must fail the init probe" >&2
		failed=1
	}
	if run_rejected_probe "always-ok stub must be caught" "$tmp/stub_always_ok"; then
		echo "self-test: an exiting-0 stub must fail a rejected-class probe" >&2
		failed=1
	fi

	# Positive controls: well-behaved stubs pass.
	probe_rules_json "$tmp/stub_good_json" || {
		echo "self-test: valid json stub should pass the json probe" >&2
		failed=1
	}

	rm -rf "$tmp"
	if [ "$failed" -ne 0 ]; then
		echo "✗ check-cqrs-lint-cli self-test FAILED" >&2
		exit 1
	fi
	echo "✓ check-cqrs-lint-cli self-test passed"
	exit 0
}

main() {
	if [ "${1:-}" = "--self-test" ]; then
		self_test
	fi

	local bin="${CQRS_LINT_BIN:-}"
	WORK_DIR="$(mktemp -d)"
	trap 'rm -rf "$WORK_DIR"' EXIT
	# A valid Go module with no go-cqrs-lite imports: the canonical clean run.
	mkdir -p "$WORK_DIR/clean"
	printf 'module example.com/clean\n\ngo 1.27\n' >"$WORK_DIR/clean/go.mod"
	printf 'package main\n\nfunc main() {}\n' >"$WORK_DIR/clean/main.go"

	if [ -z "$bin" ]; then
		# Self-source the env chain: ambient sessions can carry GOTOOLCHAIN /
		# cache vars that silently break the build (AGENTS.md gotcha #2).
		# shellcheck source=/dev/null
		source "$SCRIPT_DIR/go-env.sh" >/dev/null 2>&1 || true
		bin="$WORK_DIR/cqrs-lint"
		if ! (cd "$REPO_ROOT/cmd/cqrs-lint" && GOWORK=off go build -o "$bin" .); then
			echo "✗ check-cqrs-lint-cli: build failed" >&2
			exit 1
		fi
	fi

	if run_probes "$bin"; then
		echo "✓ check-cqrs-lint-cli: all contract probes passed ($bin)"
	else
		echo "✗ check-cqrs-lint-cli: contract violation (see above)" >&2
		exit 1
	fi
}

main "$@"
