package compat

import "errors"

// ErrBaselineFormatVersion reports a baseline file whose layout version this
// build does not understand. It is always returned wrapped with the offending
// path and the expected version, and the message names the fix
// (--update-baseline) so a stale artifact is actionable without reading code.
var ErrBaselineFormatVersion = errors.New("compat: unsupported baseline formatVersion")
