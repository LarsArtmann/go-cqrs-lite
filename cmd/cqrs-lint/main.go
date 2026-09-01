// Command cqrs-lint is a deprecation stub. This suffix-less module path is
// retired: every v4 release was published under the /v4 module path, which
// this path can never serve. Installing from here used to silently yield
// the ancient v0.2.0 binary (60 rules); it now fails loudly instead.
package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Fprint(os.Stderr, `cqrs-lint has moved.

This module path is retired: the v4 releases carry the /v4 major-version
suffix required by the Go module proxy, and none of them were ever
servable from this unsuffixed path (the newest version it ever served is
v0.2.0, a 60-rule binary from before the path change).

Install the current release instead:

    go install github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4@latest

The binary name is unchanged (cqrs-lint). Background:
https://github.com/LarsArtmann/go-cqrs-lite/issues/20
`)
	os.Exit(1)
}
