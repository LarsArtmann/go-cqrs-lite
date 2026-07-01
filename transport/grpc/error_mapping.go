package grpc

import (
	"errors"

	cqrsevent "github.com/larsartmann/go-cqrs-lite/event/v3"
)

// familyToWire maps an error-family.Family value to the lowercase string used
// on the wire. Empty string means "unclassified".
func familyToWire(f cqrsevent.Family) string {
	switch f {
	case cqrsevent.Rejection:
		return "rejection"
	case cqrsevent.Conflict:
		return "conflict"
	case cqrsevent.Transient:
		return "transient"
	case cqrsevent.Corruption:
		return "corruption"
	case cqrsevent.Infrastructure:
		return "infrastructure"
	default:
		return ""
	}
}

// wireToFamily is the inverse of familyToWire. Returns false for unknown strings.
func wireToFamily(s string) (cqrsevent.Family, bool) {
	switch s {
	case "rejection":
		return cqrsevent.Rejection, true
	case "conflict":
		return cqrsevent.Conflict, true
	case "transient":
		return cqrsevent.Transient, true
	case "corruption":
		return cqrsevent.Corruption, true
	case "infrastructure":
		return cqrsevent.Infrastructure, true
	default:
		return 0, false
	}
}

// classifyError extracts the machine-readable code and wire-family string from
// an error using the CQRS taxonomy. Used by the gRPC servers to populate the
// structured error fields on the response proto.
func classifyError(err error) (string, string) {
	family := familyToWire(cqrsevent.Classify(err))

	code := ""

	if ce, ok := errors.AsType[*cqrsevent.Error](err); ok {
		code = ce.Code()
	}

	return code, family
}

// reconstructError rebuilds a typed *event.Error from the wire fields so callers
// can use event.Classify, event.IsRetryable, and errors.As on the client side.
// Falls back to wrapErr when the server didn't send structured fields (backward
// compatibility with older servers that only populated the error string).
func reconstructError(wrapErr error, errMsg, errCode, errFamily string) error {
	if errMsg == "" {
		return nil
	}

	family, ok := wireToFamily(errFamily)
	if !ok || errCode == "" {
		// Old server or unclassified error — preserve the string-only behavior.
		return cqrsevent.Wrap(wrapErr, cqrsevent.Classify(wrapErr),
			"grpc.remote_error", errMsg)
	}

	return newClassifiedError(family, errCode, errMsg)
}

// newClassifiedError constructs a *event.Error for the given family+code+message.
func newClassifiedError(family cqrsevent.Family, code, message string) error {
	switch family {
	case cqrsevent.Rejection:
		return cqrsevent.NewRejection(code, message)
	case cqrsevent.Conflict:
		return cqrsevent.NewConflict(code, message)
	case cqrsevent.Transient:
		return cqrsevent.NewTransient(code, message)
	case cqrsevent.Corruption:
		return cqrsevent.NewCorruption(code, message)
	case cqrsevent.Infrastructure:
		return cqrsevent.NewInfrastructure(code, message)
	default:
		return cqrsevent.NewTransient(code, message)
	}
}
