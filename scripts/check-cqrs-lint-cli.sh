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

# make_clean_module <dir>: a valid Go module with no go-cqrs-lite imports
# (the canonical clean-run fixture; all-relevant rows read as MISSING).
make_clean_module() {
	mkdir -p "$1"
	printf 'module example.com/clean\n\ngo 1.27\n' >"$1/go.mod"
	printf 'package main\n\nfunc main() {}\n' >"$1/main.go"
}

# probe_scorecard_waiver <bin>: a recorded waiver in <path>/.cqrs-lint.json
# moves the row to a visible WAIVED section and exits 0 — pins the
# config-file -> WAIVED chain at the binary level (unit tests bypass the
# CLI glue; the first waiver cut shipped silently broken exactly there).
probe_scorecard_waiver() {
	local dir="$WORK_DIR/waiver-probe" out rc
	make_clean_module "$dir"
	printf '{"scorecard": {"waivers": [{"key": "graph", "reason": "probe: no graph read models", "trigger": "probe end"}]}}\n' \
		>"$dir/.cqrs-lint.json"
	out="$($1 scorecard --path "$dir" 2>/dev/null)" && rc=0 || rc=$?
	if [ "$rc" -ne 0 ]; then
		echo "scorecard waiver run exited $rc (want 0)" >&2
		return 1
	fi
	if ! grep -q "WAIVED" <<<"$out" || ! grep -q "Graph" <<<"$out"; then
		echo "waived row must render in a WAIVED section, got: ${out:0:80}" >&2
		return 1
	fi
	return 0
}

# probe_scorecard_bogus_waiver <bin>: an unknown waiver key in the scored
# project's config must exit nonzero (never silently ignored).
probe_scorecard_bogus_waiver() {
	local dir="$WORK_DIR/bogus-waiver-probe"
	make_clean_module "$dir"
	printf '{"scorecard": {"waivers": [{"key": "bogus", "reason": "r"}]}}\n' \
		>"$dir/.cqrs-lint.json"
	if "$1" scorecard --path "$dir" >/dev/null 2>&1; then
		echo "bogus waiver key must exit nonzero" >&2
		return 1
	fi
	return 0
}

# scorecard_relevant_total <bin> <dir>: extract relevant_total from JSON.
scorecard_relevant_total() {
	"$1" scorecard --format json --path "$2" 2>/dev/null |
		grep -oE '"relevant_total":[0-9]+' | head -1 | grep -oE '[0-9]+'
}

# probe_scorecard_path_preset <bin>: the preset recorded in
# <path>/.cqrs-lint.json must drive relevance — a "production" preset makes
# production-restricted rows (postgres, prometheus, …) relevant, so its
# relevant_total must exceed a preset-less twin's. If the operator's cwd
# config won instead, both totals would match and this probe fails.
probe_scorecard_path_preset() {
	local plain="$WORK_DIR/preset-plain" prod="$WORK_DIR/preset-prod"
	make_clean_module "$plain"
	make_clean_module "$prod"
	printf '{"preset": "production"}\n' >"$prod/.cqrs-lint.json"

	local plain_total prod_total
	plain_total="$(scorecard_relevant_total "$1" "$plain")" || true
	prod_total="$(scorecard_relevant_total "$1" "$prod")" || true
	if ! [[ "$plain_total" =~ ^[0-9]+$ ]] || ! [[ "$prod_total" =~ ^[0-9]+$ ]]; then
		echo "could not extract relevant_total (plain=${plain_total:-none} prod=${prod_total:-none})" >&2
		return 1
	fi
	if [ "$prod_total" -le "$plain_total" ]; then
		echo "path preset must widen relevance: prod=$prod_total plain=$plain_total (want prod>plain)" >&2
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
	probe_scorecard_waiver "$bin" || failed=1
	probe_scorecard_bogus_waiver "$bin" || failed=1
	probe_scorecard_path_preset "$bin" || failed=1

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
	local tmp failed=0
	tmp="$(mktemp -d)"

	# Fault injection: stub binaries that simulate each violation class.
	printf '#!/usr/bin/env bash\necho "rule catalog (text)"\n' >"$tmp/stub_text_rules"
	printf '#!/usr/bin/env bash\nexit 0\n' >"$tmp/stub_always_ok"
	printf '#!/usr/bin/env bash\nexit 1\n' >"$tmp/stub_always_fails"
	printf '#!/usr/bin/env bash\nexit 0\n' >"$tmp/stub_init_no_file"
	printf '#!/usr/bin/env bash\necho "[]"\n' >"$tmp/stub_good_json"
	printf '#!/usr/bin/env bash\necho "MISSING only — no waived section"\n' >"$tmp/stub_no_waived"
	printf '#!/usr/bin/env bash\necho "{\"summary\":{\"relevant_total\":9}}"\n' >"$tmp/stub_fixed_total"
	printf '#!/usr/bin/env bash\necho "WAIVED (recorded refusals) — Graph Projections"\n' >"$tmp/stub_good_waived"
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
	probe_scorecard_waiver "$tmp/stub_no_waived" && {
		echo "self-test: output without a WAIVED section must fail the waiver probe" >&2
		failed=1
	}
	probe_scorecard_bogus_waiver "$tmp/stub_always_ok" && {
		echo "self-test: silently accepting a bogus waiver must fail the probe" >&2
		failed=1
	}
	probe_scorecard_path_preset "$tmp/stub_fixed_total" && {
		echo "self-test: identical relevance under a path preset must fail the preset probe" >&2
		failed=1
	}

	# Positive controls: well-behaved stubs pass.
	probe_rules_json "$tmp/stub_good_json" || {
		echo "self-test: valid json stub should pass the json probe" >&2
		failed=1
	}
	probe_scorecard_waiver "$tmp/stub_good_waived" || {
		echo "self-test: a rendered WAIVED section should pass the waiver probe" >&2
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
