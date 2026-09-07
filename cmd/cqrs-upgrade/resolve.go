package main

import (
	"fmt"
	"strings"

	"golang.org/x/mod/semver"
)

// versionResolver resolves the latest published version of a module. A
// variable so tests can stub it offline.
var versionResolver = resolveLatest

// resolveLatest shells out to `go list -m -versions <module>` and returns the
// highest semver tag. Resolves through the module proxy, so it works for any
// published module regardless of the local checkout state.
func resolveLatest(module string) (string, error) {
	out, err := goOut(nil, "list", "-m", "-versions", module)
	if err != nil {
		return "", fmt.Errorf("resolve latest %s: %w", module, err)
	}

	return pickLatest(strings.Fields(strings.TrimSpace(out))), nil
}

// pickLatest selects the highest version from a `go list -m -versions` field
// list (first field is the module path, the rest are tags).
func pickLatest(fields []string) string {
	if len(fields) < 2 {
		return ""
	}

	latest := ""

	for _, v := range fields[1:] {
		if semver.IsValid(v) && (latest == "" || semver.Compare(v, latest) > 0) {
			latest = v
		}
	}

	return latest
}
