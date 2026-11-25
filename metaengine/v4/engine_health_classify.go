package metaengine

import (
	"errors"

	errorfamily "github.com/larsartmann/go-error-family"
)

// countsTowardHealth reports whether a failure says something about the
// engine. ErrKeyTypeMismatch is raised before any backend is called (the
// input struct has no field of the query's key type), so it is a caller
// mistake and must not fail an engine over — even though errorfamily's
// fail-open default would classify it Transient.
func countsTowardHealth(err error) bool {
	if errors.Is(err, ErrKeyTypeMismatch) {
		return false
	}

	switch errorfamily.Classify(err) {
	case errorfamily.Infrastructure, errorfamily.Transient:
		return true
	default:
		return false
	}
}
