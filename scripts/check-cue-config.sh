#!/usr/bin/env bash
# check-cue-config.sh — pin system/config.cue against the Go DeploymentConfig
# surface (ADR-0165). Three checks:
#   1. cue vet — the schema parses and evaluates (comprehensions included).
#   2. golden eval — `standard` rendered from the fixture matches the
#      committed golden YAML (the CUE preset cannot silently change shape).
#   3. drift pin — the set of koanf tags in system/config_types.go +
#      system/engine_config.go equals the set of field names in the CUE
#      definitions, bidirectionally: a Go field without CUE, or CUE without
#      Go, both fail.
#
# Usage:
#   scripts/check-cue-config.sh [--self-test]
#
# Env overrides (honored by --self-test so the test never touches tracked
# files — planted-fixture pattern from calibration-gate.sh):
#   CUE_CONFIG_ROOT   root for config.cue + testdata discovery (default: repo root)
#   CUE_BIN           cue binary (default: cue from PATH)
set -uo pipefail

repo_root="$(git rev-parse --show-toplevel 2>/dev/null)" || {
	echo "check-cue-config.sh: must run inside the git repo" >&2
	exit 1
}
cd "$repo_root" || exit 1

config_root="${CUE_CONFIG_ROOT:-$repo_root}"
cue_bin="${CUE_BIN:-cue}"

config_cue="$config_root/system/config.cue"
fixture="$config_root/system/testdata/standard-fixture.cue"
golden="$config_root/system/testdata/standard-topology.golden.yaml"
go_sources=("$config_root/system/config_types.go" "$config_root/system/engine_config.go")

fail() {
	echo "check-cue-config: FAIL: $*" >&2
	exit 1
}

cue_ok() {
	command -v "$cue_bin" >/dev/null 2>&1
}

extract_go_tags() {
	rg -o 'koanf:"[A-Za-z_]+"' "${go_sources[@]}" 2>/dev/null \
		| sed 's/.*koanf:"\([A-Za-z_]*\)".*/\1/' | sort -u
}

extract_cue_fields() {
	# Field-name lines inside the CUE file: `name:` or `name?:` at any indent.
	# The two top-level entry points (deployment, standard) are values, not
	# schema fields — excluded from the pin.
	sed -n 's/^[[:space:]]*\([A-Za-z_][A-Za-z0-9_]*\)[?]*:.*$/\1/p' "$config_cue" \
		| grep -vx -e deployment -e standard \
		| sort -u
}

run_checks() {
	cue_ok || fail "cue binary not found (run via: nix run .#check-cue-config)"

	"$cue_bin" vet "$config_cue" "$fixture" >/dev/null 2>&1 \
		|| fail "cue vet rejected the schema/fixture pair"

	"$cue_bin" eval -e standard --out yaml "$config_cue" "$fixture" >"$tmp_eval" 2>/dev/null \
		|| fail "cue eval -e standard failed"

	diff -u "$golden" "$tmp_eval" >/dev/null \
		|| fail "golden drift: standard preset eval differs from $golden"

	local go_tags cue_fields
	go_tags="$(extract_go_tags)"
	cue_fields="$(extract_cue_fields)"

	local go_only cue_only
	go_only="$(comm -23 <(echo "$go_tags") <(echo "$cue_fields"))"
	cue_only="$(comm -13 <(echo "$go_tags") <(echo "$cue_fields"))"

	[ -z "$go_only" ] || fail "koanf tags missing from config.cue: $(echo "$go_only" | tr '\n' ' ')"
	[ -z "$cue_only" ] || fail "config.cue fields with no koanf tag: $(echo "$cue_only" | tr '\n' ' ')"

	echo "check-cue-config: OK (vet + golden + drift pin over $(echo "$go_tags" | wc -l) fields)"
}

self_test() {
	local fixture_dir
	fixture_dir="$(mktemp -d /tmp/cue-config-fixture.XXXXXX)" || return 1
	trap 'rm -rf "$fixture_dir"' RETURN

	mkdir -p "$fixture_dir/system/testdata"
	cp "$repo_root/system/config.cue" "$fixture_dir/system/"
	cp "$fixture" "$fixture_dir/system/testdata/"
	cp "$golden" "$fixture_dir/system/testdata/"
	cp "${go_sources[@]}" "$fixture_dir/system/"

	local fails=0
	expect_fail() { # name, mutation-cmd
		local name="$1" cmd="$2"
		cp "$fixture_dir/system/config.cue" "$fixture_dir/.config.cue.bak"
		eval "$cmd"
		if CUE_CONFIG_ROOT="$fixture_dir" CUE_BIN="$cue_bin" "$0" >/dev/null 2>&1; then
			echo "self-test: $name: expected failure, got success" >&2
			fails=$((fails + 1))
		else
			echo "self-test: $name: caught (expected)"
		fi
		mv "$fixture_dir/.config.cue.bak" "$fixture_dir/system/config.cue"
	}

	cue_ok || {
		echo "self-test: cue binary not found — install cue or run via nix" >&2
		exit 1
	}

	expect_fail "cue-field-removed" \
		"sed -i 's/^[[:space:]]*max_open_conns?:.*$//' '$fixture_dir/system/config.cue'"
	expect_fail "cue-field-added" \
		"printf 'bogus_field?: int\n' >> '$fixture_dir/system/config.cue'"
	expect_fail "go-tag-added" \
		"printf 'type driftProbe struct{ Bogus string \`koanf:\"bogus\"\` }\n' >> '$fixture_dir/system/engine_config.go'"
	expect_fail "golden-corrupted" \
		"sed -i 's/role: source-of-truth/role: events/' '$fixture_dir/system/testdata/standard-topology.golden.yaml'"

	[ "$fails" -eq 0 ] || exit 1
	echo "self-test: OK (4 fault injections caught)"
}

tmp_eval="$(mktemp /tmp/cue-config-eval.XXXXXX.yaml)" || exit 1
trap 'rm -f "$tmp_eval"' EXIT

case "${1:-}" in
--self-test)
	self_test
	;;
"")
	run_checks
	;;
*)
	echo "usage: $0 [--self-test]" >&2
	exit 2
	;;
esac
