package main

// F031 typed-path fixture: Scan call sites whose receivers have REAL static
// types (packages.Load with NeedTypes). The bufio loop must NOT fire F031
// (non-metaengine receiver); the TypedReader Scan without WithLimit MUST fire.

import (
	"bufio"
	"context"
	"os"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// readLines is the false-positive shape from the nsfw-classifier feedback:
// a bufio.Scanner.Scan loop in a metaengine-importing project.
func readLines(f *os.File) {
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		_ = scanner.Text()
	}
}

// readAll is the coached shape: an unbounded TypedReader Scan.
func readAll(ctx context.Context, r *metaengine.TypedReader[int]) {
	_, _ = r.Scan(ctx)
}

func main() {
	readLines(os.Stdin)
	readAll(context.Background(), metaengine.NewReader[int](nil, "items"))
}
