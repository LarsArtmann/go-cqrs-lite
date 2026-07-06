package grpc

import (
	errorfamily "github.com/larsartmann/go-error-family"
)

// familyToWire converts a Family to its lowercase wire representation.
// Delegates to Family.String(); kept as a named function for readability at call sites.
func familyToWire(f errorfamily.Family) string { return f.String() }

// wireToFamily parses a wire string back to a Family.
// Returns ok=false for empty or unrecognized strings.
// Uses ParseFamily with round-trip verification to distinguish known from unknown.
func wireToFamily(s string) (errorfamily.Family, bool) {
	if s == "" {
		return 0, false
	}

	f := errorfamily.ParseFamily(s)

	return f, s == f.String()
}

// classifyError extracts the machine-readable code and wire-family string from
// an error using the error-family taxonomy. Used by the gRPC servers to populate
// the structured error fields on the response proto.
func classifyError(err error) (string, string) {
	return errorfamily.Code(err), familyToWire(errorfamily.Classify(err))
}

// reconstructError rebuilds a typed *errorfamily.Error from the wire fields so
// callers can use errorfamily.Classify, errorfamily.IsRetryable, and errors.As
// on the client side. Falls back to a wrapped error when the server didn't send
// structured fields (backward compatibility with older servers).
func reconstructError(wrapErr error, errMsg, errCode, errFamily string) error {
	if errMsg == "" {
		return nil
	}

	family, ok := wireToFamily(errFamily)
	if !ok || errCode == "" {
		return errorfamily.Wrap(wrapErr, errorfamily.Classify(wrapErr),
			"grpc.remote_error", errMsg)
	}

	return errorfamily.New(family, errCode, errMsg)
}
