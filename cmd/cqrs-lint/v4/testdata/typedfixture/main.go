// Command typedfixture is the committed F091 typed-path fixture: a real
// consumer module (own go.mod, replace-based schema dependency) whose
// packages.Load provides genuine type information. F090(b) and the other
// typed-confirmation rules are tested against THIS module — throwaway temp
// modules could not be pinned in CI.
package main

import (
	. "github.com/larsartmann/go-cqrs-lite/schema/v4" //nolint:dot-import // fixture: dot-import is the subject under test
)

// removedRef references a v5-removed symbol WITHOUT a qualifier — exactly the
// hole F090(a) warns about and F090(b) must attribute (V007 on this line).
var removedRef VersionedStore

// liveRef references a surviving symbol through the same dot-import — it
// must NOT fire (negative control).
var liveRef = UpcastSourceTransform

func main() {
	_, _ = removedRef, liveRef
}
