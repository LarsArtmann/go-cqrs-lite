#!/usr/bin/env bash
# check-retracts-shipped.sh — a retract only reaches consumers through a RELEASE.
#
# A `retract` directive in a module's go.mod is invisible to the module proxy
# until a version of that module is TAGGED whose go.mod carries the directive;
# until then it is inert — the release-management equivalent of an unsent
# letter (the cqrs-lint v4.8.0 retract was authored on master while consumers
# kept installing the poisoned version).
#
# For every module whose WORKING-TREE go.mod declares retracts, this gate
# asserts the module's newest tag exists and its go.mod AT THAT TAG declares
# the same retract set. Violations exit 1.
#
# Run: bash scripts/check-retracts-shipped.sh
set -euo pipefail

cd "$(git rev-parse --show-toplevel)"

# extract_retracts <file>: prints the retracted version tokens, one per
# line — single-line `retract v1.2.3`, range `retract [v1, v2]`, and block
# form all reduce to their version tokens.
extract_retracts() {
	awk '
		/^[[:space:]]*retract[[:space:]]*\(/ { inblock = 1; next }
		inblock && /^[[:space:]]*\)/        { inblock = 0; next }
		{
			line = $0
			if (inblock || line ~ /^[[:space:]]*retract[[:space:]]*v[0-9]/) {
				gsub(/[\[\],]/, " ", line)
				n = split(line, w, /[[:space:]]+/)
				for (i = 1; i <= n; i++)
					if (w[i] ~ /^v[0-9]/) print w[i]
			}
		}
	' "$1"
}

violations=0
checked=0
while IFS= read -r gomod; do
	dir="${gomod#./}"
	dir="${dir%/go.mod}"
	retracts="$(grep -E '^[[:space:]]*retract' "$gomod" 2>/dev/null || true)"
	[ -z "$retracts" ] && continue
	checked=$((checked + 1))

	# The root module has BARE tags; nested modules own "<dir>/<version>".
	if [ -z "$dir" ]; then
		tag_glob='v*'
		prefix=''
	else
		tag_glob="${dir}/*"
		prefix="${dir}/"
	fi

	# Newest tag owned by THIS module (exact "<dir>/<version>" form; deeper
	# nesting belongs to nested modules, mirroring tag-release.sh --audit).
	newest="$(git tag -l "$tag_glob" | while IFS= read -r t; do
		case "${t#"$prefix"}" in */*) continue ;; esac
		printf '%s\n' "${t#"$prefix"}"
	done | sort -V | tail -1)"

	if [ -z "$newest" ]; then
		echo "FAIL  ${gomod}: declares retracts but the module has NO tags —"
		echo "      nothing ships them; cut a release carrying the retracts."
		violations=$((violations + 1))
		continue
	fi

	newest_tag="${prefix}${newest}"
	tagged_gomod="$(git show "${newest_tag}:${gomod}" 2>/dev/null || true)"
	missing="$(comm -23 \
		<(extract_retracts "$gomod" | sort -u) \
		<(extract_retracts /dev/stdin <<<"$tagged_gomod" | sort -u))"
	if [ -n "$missing" ]; then
		echo "FAIL  ${gomod}: retract(s) missing from ${newest_tag}'s go.mod —"
		echo "      inert until a release ships them. Missing:"
		printf '%s\n' "$missing" | sed 's/^/        /'
		violations=$((violations + 1))
	fi
done < <(find . -name go.mod -not -path './vendor/*' -not -path './.git/*')

echo ""
echo "check-retracts: ${checked} module(s) with retracts checked."
if [ "$violations" -gt 0 ]; then
	echo "${violations} unshipped-retract violation(s). A retract reaches the"
	echo "proxy ONLY through a release: tag a new version whose go.mod carries"
	echo "the retract directives."
	exit 1
fi
