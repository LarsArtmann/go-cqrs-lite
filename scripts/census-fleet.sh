#!/usr/bin/env bash
# census-fleet.sh — fleet consumer census for the v4→v5 delete-at-zero rule
# (ADR-0152 addendum 2026-10-17; playbook docs/agents/v5-fork-and-delete-playbook.md §5).
#
# Scans every go.mod under FLEET_ROOT (default ~/projects) for requires of
# github.com/larsartmann/go-cqrs-lite/* modules and emits a timestamped JSON
# artifact to docs/evidence/. Semantics match `depgraph who-uses`: go.mod
# REQUIRES (direct vs `// indirect`), NOT imports. Excluded from the scan:
# any go.mod declaring a go-cqrs-lite module path (the library itself AND full
# forks/work copies of it, e.g. eventcatalog-hub/work/src/mesh-* — a fork's
# internal requires are not fleet consumption), `.corpus-pins/` pin snapshots
# and go-cqrs-lite@<hash> dirs, vendor/ and node_modules/ trees, and testdata/
# fixtures (the intentional real-setup-v4.8.1-poisoned class must never keep a
# module alive). Duplicate CI checkouts of the same repo (e.g. ci-siblings/)
# are NOT deduplicated — counts can be inflated, never falsely zeroed; review
# the candidate list, the report shows repo paths.
#
# Modes:
#   default          scan → docs/evidence/fleet-census-<ts>.json + human summary
#   --zero-consumers print ONLY v4 tree modules with no fleet requires
#                    (delete-at-zero candidates; the in-repo coupling rg from
#                    playbook §3.2 is still required before any delete)
#   --json <path>    write JSON to <path> instead of docs/evidence/
#   --latest         re-print the summary of the newest existing artifact (no scan)
#   --self-test      fixture-tree test (planted direct/indirect/vendor requires,
#                    fault-injection assertions on the exclusion logic)
#
# Usage:
#   scripts/census-fleet.sh
#   scripts/census-fleet.sh --zero-consumers
#   FLEET_ROOT=/somewhere/else scripts/census-fleet.sh
set -euo pipefail

REPO_ROOT=$(cd "$(dirname "$0")/.." && pwd)
FLEET_ROOT="${FLEET_ROOT:-$HOME/projects}"
PREFIX="github.com/larsartmann/go-cqrs-lite"
EVIDENCE_DIR="$REPO_ROOT/docs/evidence"

MODE="scan"
JSON_OUT=""
while [ $# -gt 0 ]; do
	case "$1" in
	--zero-consumers) MODE="zero" ;;
	--json)
		shift
		JSON_OUT="${1:?--json needs a path}"
		;;
	--latest) MODE="latest" ;;
	--self-test) MODE="selftest" ;;
	*)
		echo "unknown flag: $1 (see header for usage)" >&2
		exit 2
		;;
	esac
	shift
done

scan_fleet() { # $1 = fleet root, $2 = output json path
	python3 - "$1" "$2" <<'PYEOF'
import datetime, json, os, re, sys

fleet_root, out_path = sys.argv[1], sys.argv[2]
prefix = "github.com/larsartmann/go-cqrs-lite/"
# module-path token: everything up to whitespace; version token follows.
req_re = re.compile(r"^\s*(?:require\s+)?(" + re.escape(prefix) + r"\S+)\s+(\S+)(\s+//\s*indirect)?\s*$")

modules = {}  # submodule -> [{"repo","version","indirect"}]
scanned = 0
library_excluded = False

for dirpath, dirnames, filenames in os.walk(fleet_root):
	if any(d.startswith("go-cqrs-lite") for d in dirnames):
		library_excluded = True
	dirnames[:] = [
		d for d in dirnames
		if d not in ("vendor", "node_modules", ".git", "testdata", ".corpus-pins")
		and not d.startswith("go-cqrs-lite")
	]
	if dirpath == fleet_root or "go.mod" not in filenames:
		continue
	scanned += 1
	repo = os.path.relpath(dirpath, fleet_root)
	with open(os.path.join(dirpath, "go.mod"), encoding="utf-8", errors="replace") as fh:
		lines = fh.readlines()
	for line in lines:
		m_mod = re.match(r"^module\s+(\S+)", line)
		if m_mod and (m_mod.group(1) == prefix.rstrip("/") or m_mod.group(1).startswith(prefix)):
			# the library itself or a full fork of it — never its own consumer
			scanned -= 1
			break
	else:
		for line in lines:
			if "=>" in line:  # replace lines are not usage
				continue
			m = req_re.match(line.rstrip("\n"))
			if not m:
				continue
			path, version, indirect = m.group(1), m.group(2), bool(m.group(3))
			sub = path[len(prefix):]
			modules.setdefault(sub, []).append(
				{"repo": repo, "version": version, "indirect": indirect}
			)

summary = {
	"generated_at": datetime.datetime.now().strftime("%Y-%m-%dT%H:%M:%S"),
	"fleet_root": os.path.abspath(fleet_root),
	"gomod_files_scanned": scanned,
	"library_repo_excluded": library_excluded,
	"exclusions": [
		"library self + full forks (module path = go-cqrs-lite/*)",
		"go-cqrs-lite* dirs + .corpus-pins/ snapshots",
		"vendor/ + node_modules/",
		"testdata/ fixtures",
		"replace (=>) lines",
	],
	"modules": {
		sub: {
			"direct_count": sum(1 for c in cons if not c["indirect"]),
			"indirect_count": sum(1 for c in cons if c["indirect"]),
			"consumers": cons,
		}
		for sub, cons in sorted(modules.items())
	},
}
with open(out_path, "w", encoding="utf-8") as fh:
	json.dump(summary, fh, indent=1, sort_keys=True)
	fh.write("\n")
print(out_path)
PYEOF
}

summary_of() { # $1 = json path; human summary to stdout
	python3 - "$1" <<'PYEOF'
import json, sys
with open(sys.argv[1], encoding="utf-8") as fh:
	data = json.load(fh)
mods = data["modules"]
print(f"census {data['generated_at']}  fleet_root={data['fleet_root']}  go.mod scanned: {data['gomod_files_scanned']}")
ranked = sorted(mods.items(), key=lambda kv: (-(kv[1]["direct_count"] + kv[1]["indirect_count"]), kv[0]))
for sub, info in ranked:
	tag = " ZERO" if info["direct_count"] + info["indirect_count"] == 0 else ""
	print(f"  {info['direct_count']:>3} direct {info['indirect_count']:>3} indirect  {sub}{tag}")
print(f"  {len(mods)} distinct submodules required by the fleet")
PYEOF
}

zero_consumers_of() { # $1 = json path; prints tree v4 modules with no fleet consumers
	python3 - "$1" "$REPO_ROOT" <<'PYEOF'
import json, os, re, sys

with open(sys.argv[1], encoding="utf-8") as fh:
	fleet = json.load(fh)["modules"]
repo_root = sys.argv[2]
# non-candidate product dirs: the future module, tools, examples, suites
NON_CANDIDATES = {"core", "cmd", "example", "integration", "systemtest", "benchkit"}
consumed = {re.sub(r"/v\d+$", "", sub) for sub in fleet}

mods = []
for dirpath, dirnames, filenames in os.walk(repo_root):
	dirnames[:] = [d for d in dirnames if d not in ("vendor", ".git")]
	if "go.mod" not in filenames or dirpath == repo_root:
		continue
	sub = os.path.relpath(dirpath, repo_root)
	if sub.split(os.sep)[0] in NON_CANDIDATES:
		continue
	mods.append(sub)

for sub in sorted(set(mods) - consumed):
	print(sub)
PYEOF
}

self_test() {
	local fixture out
	fixture=$(mktemp -d)
	out=$(mktemp -d)
	trap 'rm -rf "$fixture" "$out"' RETURN
	# planted fleet: direct require, indirect require, a vendor tree (must be
	# ignored), the library repo itself (must be ignored), a pin snapshot
	# (go-cqrs-lite@<hash> — must be ignored), a testdata fixture (must be
	# ignored), and a clean repo.
	mkdir -p "$fixture/appA" "$fixture/appB" "$fixture/appC" "$fixture/vendor-user/vendor" "$fixture/vendor-user/vendor/dep" "$fixture/go-cqrs-lite/event" "$fixture/.corpus-pins/go-cqrs-lite@deadbeef/idempotency/kvstore" "$fixture/appA/testdata/pin-fixture" "$fixture/my-fork/scheduling"
	cat >"$fixture/appA/go.mod" <<EOF
module example.com/a

go 1.27

require (
	$PREFIX/event/v4 v4.20.0
	$PREFIX/id/v4 v4.7.0 // indirect
)
EOF
	cat >"$fixture/appB/go.mod" <<EOF
module example.com/b

go 1.27

require $PREFIX/decider/v4 v4.3.1

replace $PREFIX/decider/v4 => ../go-cqrs-lite/decider
EOF
	printf 'module example.com/vu\n\ngo 1.27\n' >"$fixture/vendor-user/go.mod"
	printf 'module vendored/dep\n\nrequire %s/event/v4 v4.20.0\n' "$PREFIX" >"$fixture/vendor-user/vendor/dep/go.mod"
	printf 'module example.com/pin\n\ngo 1.27\n' >"$fixture/.corpus-pins/go-cqrs-lite@deadbeef/go.mod"
	printf 'module example.com/pinsnap\n\nrequire %s/idempotency/kvstore/v4 v4.1.0\n' "$PREFIX" >"$fixture/.corpus-pins/go-cqrs-lite@deadbeef/idempotency/kvstore/go.mod"
	printf 'module example.com/fixture\n\nrequire %s/codec/v4 v4.4.0\n' "$PREFIX" >"$fixture/appA/testdata/pin-fixture/go.mod"
	printf 'module github.com/larsartmann/go-cqrs-lite\n\ngo 1.27\n' >"$fixture/my-fork/go.mod"
	printf 'module github.com/larsartmann/go-cqrs-lite/scheduling/v4\n\ngo 1.27\n\nrequire %s/deriver/v4 v4.0.0\n' "$PREFIX" >"$fixture/my-fork/scheduling/go.mod"
	printf 'module example.com/clean\n\ngo 1.27\n' >"$fixture/appC/go.mod"
	printf 'module github.com/larsartmann/go-cqrs-lite/event/v4\n\ngo 1.27\n' >"$fixture/go-cqrs-lite/event/go.mod"

	FLEET_ROOT="$fixture" scan_fleet "$fixture" "$out/census.json" >/dev/null
	python3 - "$out/census.json" <<'PYEOF'
import json, sys
data = json.load(open(sys.argv[1]))
mods = data["modules"]
assert data["gomod_files_scanned"] == 4, f"scanned {data['gomod_files_scanned']}, want 4 (appA, appB, appC, vendor-user root)"
assert data["library_repo_excluded"], "library repo must be excluded"
ev = mods["event/v4"]
assert ev["direct_count"] == 1 and {c["repo"] for c in ev["consumers"]} == {"appA"}, ev
assert ev["indirect_count"] == 0, "vendor-user require must NOT count (vendor exclusion)"
idm = mods["id/v4"]
assert idm["indirect_count"] == 1 and idm["direct_count"] == 0, idm
dec = mods["decider/v4"]
assert dec["direct_count"] == 1 and dec["consumers"][0]["version"] == "v4.3.1", dec
assert "idempotency/kvstore/v4" not in mods, "pin snapshot (go-cqrs-lite@hash) must be excluded"
assert "codec/v4" not in mods, "testdata fixture must be excluded"
assert "deriver/v4" not in mods, "full fork of the library (module path = library path) must be excluded"
assert "appC" not in json.dumps(mods), "clean repo must not appear"
print("self-test: PASS (direct, indirect, version, replace-skip, vendor+library exclusions)")
PYEOF
	# fault injection: a scanner that lost the vendor exclusion would now pass
	local poisoned
	poisoned=$(mktemp -d)
	cp -r "$fixture/." "$poisoned/"
	rm -rf "$poisoned/vendor-user"
	mkdir -p "$poisoned/vendor-user"
	printf 'module example.com/vu\n\ngo 1.27\n\nrequire %s/event/v4 v4.1.0\n' "$PREFIX" >"$poisoned/vendor-user/go.mod"
	FLEET_ROOT="$poisoned" scan_fleet "$poisoned" "$out/poison.json" >/dev/null
	python3 - "$out/poison.json" <<'PYEOF'
import json, sys
mods = json.load(open(sys.argv[1]))["modules"]
repos = {c["repo"] for c in mods["event/v4"]["consumers"]}
assert repos == {"appA", "vendor-user"}, repos  # non-vendor require DOES count → scanner is live, not vacuous
print("self-test: PASS (fault-injection leg proves the assertions can fail)")
PYEOF
	rm -rf "$poisoned"
}

case "$MODE" in
selftest)
	self_test
	;;
latest)
	latest=$(ls -1 "$EVIDENCE_DIR"/fleet-census-*.json 2>/dev/null | sort | tail -1)
	[ -n "${latest:-}" ] || {
		echo "no fleet-census artifact in $EVIDENCE_DIR yet" >&2
		exit 1
	}
	summary_of "$latest"
	;;
*)
	ts=$(date +%Y-%m-%d_%H-%M)
	if [ "$MODE" = "zero" ]; then
		# report mode: scan to a temp file, never mint a timestamped artifact
		tmp=$(mktemp)
		trap 'rm -f "$tmp"' EXIT
		artifact=$(scan_fleet "$FLEET_ROOT" "$tmp")
		cands=$(zero_consumers_of "$artifact")
		if [ -n "$cands" ]; then
			echo "delete-at-zero candidates (0 fleet requires):"
			echo "$cands" | sed 's/^/  /'
			echo "NEXT: run the in-repo coupling check (playbook §3.2) before deleting anything."
		else
			echo "no module is at zero fleet consumers"
		fi
	else
		out="${JSON_OUT:-$EVIDENCE_DIR/fleet-census-$ts.json}"
		[ -n "$JSON_OUT" ] || mkdir -p "$EVIDENCE_DIR"
		artifact=$(scan_fleet "$FLEET_ROOT" "$out")
		summary_of "$artifact"
	fi
	;;
esac
